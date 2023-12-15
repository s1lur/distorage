package commands

import (
	"bytes"
	"encoding/hex"
	"fmt"
	uuid2 "github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"log"
	"net/url"
	"os"
	"path"
)

func (c *Commands) GetDownloadCommand() *cli.Command {
	return &cli.Command{
		Name:    "download",
		Aliases: []string{"d"},
		Usage:   "download a file from the system",
		Action:  c.download,
	}
}

func (c *Commands) downloadFile(conn *websocket.Conn) ([]byte, error) {
	ecdsaPrivKey, err := c.crypto.ReadECDSAPrivKey()
	if err != nil {
		return nil, err
	}
	sharedKey, err := c.executePreamble(ecdsaPrivKey, conn)
	if err != nil {
		return nil, err
	}
	verification, err := c.crypto.PrepareVerification(sharedKey, ecdsaPrivKey)

	err = conn.WriteMessage(websocket.BinaryMessage, verification)
	if err != nil {
		return nil, err
	}

	mt, message, err := conn.ReadMessage()
	if mt != websocket.BinaryMessage {
		return nil, fmt.Errorf("wrong message type received: %d", mt)
	}
	if err != nil {
		return nil, err
	}
	if bytes.Equal(message, []byte{0x01, 0x90}) {
		return nil, fmt.Errorf("access denied")
	}
	if bytes.Equal(message, []byte{0x01, 0x94}) {
		return nil, fmt.Errorf("file not found on node")
	}
	return message, nil

}

// return fmt.Errorf("no nodes available for chunk #%d, sorry :(", i)

func (c *Commands) download(cCtx *cli.Context) error {
	verbosity := cCtx.Int("verbosity") //переменная отображает сколько текста вывести.
	if !cCtx.Bool("no-cleanup") {
		totalFiles, deletedFiles, err := c.Cleanup(cCtx)
		if verbosity > 0 {
			if err != nil {
				fmt.Printf("error during cleanup: %e\n", err)
			} else if totalFiles > 0 {
				fmt.Printf("successfully cleaned up %d/%d files\n", deletedFiles, totalFiles)
			}
		}
	}
	// Комментарий-план
	//1) Найти файл по UUID
	//1.5) Пробегаюсь по актуальным нодам и сверяю со свомими.
	//2) Качать чанки

	//2.5) Проверяю целостность каждого чанка

	//3) Собираю чанки в один файл
	//4) Дешеферую полученный фай
	//5) Проверяю целостность файла

	uuid, err := uuid2.Parse(cCtx.Args().First())
	if err != nil {
		return err
	}
	fileInfo, err := c.storage.GetFileInfo(uuid)

	if err != nil {
		return err
	}
	if !fileInfo.Available {
		return fmt.Errorf("file %s is already deleted", uuid)
	}

	if verbosity > 1 {
		log.Println("fetching available nodes")
	}
	nodes, err := c.server.GetAvailableNodes()
	if err != nil {
		return err
	}
	if verbosity > 1 {
		log.Printf("successfully fetched %d nodes\n", len(nodes))
	}

	var bar *progressbar.ProgressBar
	if verbosity == 1 {
		bar = progressbar.Default(int64(len(fileInfo.Chunks)))
	}
	body := make([]byte, fileInfo.Size)
	ptr := 0
	for i, chunk := range fileInfo.Chunks {
		if verbosity > 1 {
			log.Printf("fetching chunk #%d\n", i)
		}
		var chunkBody []byte
		for _, nodeAddr := range chunk.Nodes {
			nodeIp, exists := nodes[nodeAddr]
			if !exists {
				if verbosity > 1 {
					log.Printf("node %s unavailable, continuing\n", nodeAddr)
				}
				continue
			}
			nodeIp = fmt.Sprintf("%s:53591", nodeIp)

			u := url.URL{Scheme: "ws", Host: nodeIp, Path: fmt.Sprintf("/get/%s", chunk.Hash)}
			nodeURL, err := url.PathUnescape(u.String())
			if err != nil {
				if verbosity > 1 {
					log.Printf("error decoding node url: %e\n", err)
				}
				continue
			}
			if verbosity > 1 {
				log.Printf("connecting to %s", nodeURL)
			}
			conn, _, err := websocket.DefaultDialer.Dial(nodeURL, nil)
			if err != nil {
				if verbosity > 1 {
					log.Printf("dial to %s error: %e\n", nodeIp, err)
				}
				continue
			}
			defer conn.Close()
			chunkBody, err = c.downloadFile(conn)
			if err != nil {
				if verbosity > 1 {
					log.Printf("failed to receive chunk #%d from %s: %e\n", i, nodeIp, err)
				}
				chunkBody = []byte{}
				continue
			}
			if bodyHash := hex.EncodeToString(c.crypto.Hash(chunkBody)); chunk.Hash != bodyHash {
				if verbosity > 1 {
					log.Printf(
						"integrity check failed for chunk #%d from node %s: stored %s, received %s\n",
						i,
						nodeAddr,
						chunk.Hash,
						bodyHash,
					)
				}
				chunkBody = []byte{}
				continue
			}
			if verbosity == 1 {
				_ = bar.Add(1)
			}
			break
		}
		if len(chunkBody) == 0 {
			return fmt.Errorf("no nodes available for chunk #%d, sorry :(", i)
		}
		if ptr+len(chunkBody) > cap(body) {
			var tmp []byte
			tmp, body = body, make([]byte, ptr+len(chunkBody))
			copy(body, tmp)
		}
		copy(body[ptr:ptr+len(chunkBody)], chunkBody)
		ptr += len(chunkBody)
	}
	if verbosity == 1 {
		_ = bar.Finish()
	}
	if verbosity > 0 {
		fmt.Printf("successfully downloaded file, decrypting and verifying signature...\n")
	}
	aesKey, err := c.crypto.ReadAesKey()
	if err != nil {
		return err
	}
	decryptedFile, err := c.crypto.AESDecrypt(aesKey, body)
	if err != nil {
		return err
	}
	if len(decryptedFile) != fileInfo.Size { // probably redundant
		return fmt.Errorf("file size mismatch: local %d, got %d", fileInfo.Size, len(body))
	}
	if hash := hex.EncodeToString(c.crypto.Hash(decryptedFile)); fileInfo.Hash != hash {
		return fmt.Errorf("hash mismatch: local %s, got %s", fileInfo.Hash, hash)
	}

	cwd, err := os.Getwd()
	filePath := path.Join(cwd, fileInfo.Name)
	if err != nil {
		return err
	}
	if verbosity > 0 {
		fmt.Printf("signature verified, writing to %s\n", filePath)
	}
	file, err := os.Create(filePath)
	_, err = file.Write(decryptedFile)
	if err != nil {
		return err
	}
	if verbosity > 0 {
		fmt.Printf("successfully downloaded file %s, stored in CWD\n", fileInfo.Name)
	}
	return nil
}

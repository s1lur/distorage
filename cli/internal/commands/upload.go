package commands

import (
	"bytes"
	"cli/internal/entity"
	"encoding/hex"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"log"
	"net/url"
	"os"
	"path/filepath"
)

func (c *Commands) GetUploadCommand() *cli.Command {
	return &cli.Command{
		Name:    "upload",
		Aliases: []string{"u"},
		Usage:   "upload a file to the system",
		Action:  c.upload,
	}
}

func (c *Commands) uploadFile(body []byte, conn *websocket.Conn) error {
	ecdsaPrivKey, err := c.crypto.ReadECDSAPrivKey("~/.distorage/keys.json")
	if err != nil {
		return err
	}
	sharedKey, err := c.executePreamble(ecdsaPrivKey, conn)
	if err != nil {
		return err
	}
	verification, err := c.crypto.PrepareVerification(sharedKey, ecdsaPrivKey)
	msg := make([]byte, len(verification)+len(body))
	copy(msg[:len(verification)], verification)
	copy(msg[len(verification):], body)

	err = conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		return err
	}

	mt, message, err := conn.ReadMessage()
	if mt != websocket.BinaryMessage {
		return fmt.Errorf("wrong message type received: %d", mt)
	}
	if err != nil {
		return err
	}
	if !bytes.Equal(message, []byte{0xc8}) {
		return fmt.Errorf("wrong message received: %x", message)
	}
	return nil
}

func (c *Commands) upload(cCtx *cli.Context) error {
	verbosity := cCtx.Int("verbosity")
	if !cCtx.Bool("no-cleanup") {
		totalFiles, deletedFiles, err := c.Cleanup(cCtx)
		if verbosity > 0 {
			if err != nil {
				log.Printf("error during cleanup: %e\n", err)
			}
			log.Printf("successfully cleaned up %d/%d files", deletedFiles, totalFiles)
		}
	}
	// read file
	path := cCtx.Args().First()
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	contentsHash := hex.EncodeToString(c.crypto.Hash(contents))

	// encrypt file
	aesKey, err := c.crypto.ReadAesKey("~/.distorage/keys.json")
	if err != nil {
		return err
	}
	encryptedContents, err := c.crypto.AESEncrypt(aesKey, contents)
	if err != nil {
		return err
	}

	// split file into chunks
	var chunks [][]byte
	for i := 0; i < len(encryptedContents); i += CHUNK_SIZE {
		end := i + CHUNK_SIZE
		if end > len(encryptedContents) {
			end = len(encryptedContents)
		}
		chunks = append(chunks, encryptedContents[i:end])
	}

	nodes, err := c.server.GetAvailableNodes()
	if err != nil {
		return err
	}
	if verbosity > 1 {
		log.Printf("%d availble nodes, choosing %d per chunk\n", len(nodes), CHUNK_SIZE)
	}

	// upload file
	var bar *progressbar.ProgressBar
	if verbosity == 1 {
		bar = progressbar.Default(int64(len(chunks)))
	}
	chunkInfos := make([]entity.ChunkInfo, len(chunks))
	for i, chunk := range chunks {
		chunkHash := hex.EncodeToString(c.crypto.Hash(chunk))
		it := 0
		storageNodes := make([]string, c.cfg.ReplicationCount)
		for addr, ip := range nodes {
			if it >= c.cfg.ReplicationCount {
				break
			}
			u := url.URL{Scheme: "ws", Host: ip, Path: fmt.Sprintf("/store/%s", chunkHash)}
			if verbosity > 1 {
				log.Printf("connecting to %s\n", u.String())
			}
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				if verbosity > 1 {
					log.Fatalf("dial to %s error: %e\n", ip, err)
				}
				continue
			}
			defer conn.Close()
			err = c.uploadFile(chunk, conn)
			if err != nil {
				if verbosity > 1 {
					log.Fatalf("failed to upload chunk #%d to %s: %e", i, ip, err)
				}
				continue
			}
			it += 1
			storageNodes = append(storageNodes, addr)
		}
		if it == 0 {
			return fmt.Errorf("failed to upload chunk %d to any nodes, sorry :(", i)
		}
		chunkInfos = append(chunkInfos, entity.ChunkInfo{
			Number: i,
			Hash:   chunkHash,
			Nodes:  storageNodes,
		})
		if verbosity == 1 {
			_ = bar.Add(1)
		}
	}
	if verbosity == 1 {
		_ = bar.Finish()
	}
	if verbosity > 0 {
		log.Printf("successfully uploaded %d chunks\n", len(chunks))
	}

	// store info locally
	fileInfo := entity.FileInfo{
		Name:   filepath.Base(path),
		Hash:   contentsHash,
		Size:   len(contents),
		Chunks: chunkInfos,
	}

	err = c.storage.AppendFileInfo(fileInfo)
	if err != nil {
		return err
	}
	if verbosity > 0 {
		log.Printf("successfully stored info about uploaded file\n")
	}
	return nil
}

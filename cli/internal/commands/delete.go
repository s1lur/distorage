package commands

import (
	"bytes"
	"cli/internal/entity"
	"fmt"
	uuid2 "github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/schollz/progressbar/v3"
	"github.com/urfave/cli/v2"
	"log"
	"net/url"
)

func (c *Commands) GetDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"del"},
		Usage:   "delete uploaded file",
		Action:  c.delete,
	}
}

func (c *Commands) deleteFile(conn *websocket.Conn) error {
	ecdsaPrivKey, err := c.crypto.ReadECDSAPrivKey("~/.distorage/keys.json")
	if err != nil {
		return err
	}
	sharedKey, err := c.executePreamble(ecdsaPrivKey, conn)
	if err != nil {
		return err
	}
	verification, err := c.crypto.PrepareVerification(sharedKey, ecdsaPrivKey)

	err = conn.WriteMessage(websocket.BinaryMessage, verification)
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
	if !bytes.Equal(message, []byte{0xcc}) {
		return fmt.Errorf("wrong message received: %x", message)
	}
	return nil

}

func (c *Commands) delete(cCtx *cli.Context) error {
	verbosity := cCtx.Int("verbosity") //переменная отображает сколько текста вывести.
	if !cCtx.Bool("no-cleanup") {
		totalFiles, deletedFiles, err := c.Cleanup(cCtx)
		if verbosity > 0 {
			if err != nil {
				log.Printf("error during cleanup: %e\n", err)
			}
			log.Printf("successfully cleaned up %d/%d files", deletedFiles, totalFiles)
		}
	}
	// read local info about file
	uuid, err := uuid2.Parse(cCtx.Args().First())
	if err != nil {
		return err
	}
	fileInfo, err := c.storage.GetFileInfo(uuid)
	if err != nil {
		return err
	}
	fileInfo.Available = false
	if err := c.storage.UpdateFileInfo(uuid, *fileInfo); err != nil {
		return err
	}

	// get info about all available nodes
	nodes, err := c.server.GetAvailableNodes()
	if err != nil {
		return err
	}

	var bar *progressbar.ProgressBar
	if verbosity == 1 {
		bar = progressbar.Default(int64(len(fileInfo.Chunks)))
	}
	var leftChunks []entity.ChunkInfo
	// send delete request to every node
	for i, chunk := range fileInfo.Chunks {
		var leftNodes []string
		for _, nodeAddr := range chunk.Nodes {
			nodeIp, exists := nodes[nodeAddr]
			if !exists {
				if verbosity > 1 {
					log.Println("node %s unavailable, continuing", nodeAddr)
				}
				leftNodes = append(leftNodes, nodeAddr)
				continue
			}
			u := url.URL{Scheme: "ws", Host: nodeIp, Path: fmt.Sprintf("/delete/%s", chunk.Hash)}
			if verbosity > 1 {
				log.Printf("connecting to %s", u.String())
			}
			conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
			if err != nil {
				if verbosity > 1 {
					fmt.Errorf("dial error :%e", err)
				}
				leftNodes = append(leftNodes, nodeAddr)
				continue
			}
			defer conn.Close()
			err = c.deleteFile(conn)
			if err != nil {
				if verbosity > 1 {
					log.Fatalf("failed to delete chunk #%d from %s: %e", i, nodeAddr, err)
				}
				leftNodes = append(leftNodes, nodeAddr)
				continue
			}
		}
		if len(leftNodes) > 0 {
			leftChunks = append(leftChunks, entity.ChunkInfo{
				Number: chunk.Number,
				Hash:   chunk.Hash,
				Nodes:  leftNodes,
			})
		}
		if verbosity == 1 {
			_ = bar.Add(1)
		}
	}
	if verbosity == 1 {
		_ = bar.Finish()
	}
	if verbosity > 0 {
		log.Printf("successfully deleted all chunks from all available nodes\n")
	}
	if len(leftChunks) == 0 {
		err = c.storage.DeleteFileInfo(uuid)
		if err != nil {
			return err
		}
		if verbosity > 0 {
			log.Printf("successfully deleted local info about stored file\n")
		}
	} else {
		fileInfo.Chunks = leftChunks
		if err := c.storage.UpdateFileInfo(uuid, *fileInfo); err != nil {
			return err
		}
		if verbosity > 0 {
			log.Printf("not all nodes were available, info will be cleaned later\n")
		}
	}
	return nil

}

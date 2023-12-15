package commands

import (
	"cli/internal/entity"
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
	"net/url"
)

func (c *Commands) executePreamble(ecdsaPrivKey *ecdsa.PrivateKey, conn *websocket.Conn) ([]byte, error) {
	ecdsaPubKey := ecdsaPrivKey.PublicKey
	marshalledEcdsaPubKey, err := x509.MarshalPKIXPublicKey(&ecdsaPubKey)
	if err != nil {
		return nil, err
	}

	ecdhPrivKey, err := c.crypto.GenerateECDHKey()
	if err != nil {
		return nil, err
	}
	ecdhPubKey := ecdhPrivKey.PublicKey()
	marshalledPubKey, err := x509.MarshalPKIXPublicKey(ecdhPubKey)
	if err != nil {
		return nil, err
	}
	msgBody := make([]byte, len(marshalledEcdsaPubKey)+len(marshalledPubKey))
	copy(msgBody[:len(marshalledEcdsaPubKey)], marshalledEcdsaPubKey)
	copy(msgBody[len(marshalledEcdsaPubKey):], marshalledPubKey)
	err = conn.WriteMessage(websocket.BinaryMessage, msgBody)
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

	return c.crypto.ExecuteECDH(ecdhPrivKey, message)
}

func (c *Commands) Cleanup(cCtx *cli.Context) (int, int, error) {
	fileInfos, err := c.storage.GetFileInfos()
	if err != nil {
		return 0, 0, err
	}
	// get info about all available nodes
	nodes, err := c.server.GetAvailableNodes()
	if err != nil {
		return 0, 0, err
	}
	totalFiles := 0
	deletedFiles := 0
	for uuid, fileInfo := range fileInfos {
		if fileInfo.Available {
			continue
		}
		totalFiles += 1
		var leftChunks []entity.ChunkInfo
		// send delete request to every node
		for _, chunk := range fileInfo.Chunks {
			leftNodes := make([]string, 0)
			for _, nodeAddr := range chunk.Nodes {
				nodeIp, exists := nodes[nodeAddr]
				if !exists {
					leftNodes = append(leftNodes, nodeAddr)
					continue
				}
				nodeIp = fmt.Sprintf("%s:53591", nodeIp)
				u := url.URL{Scheme: "ws", Host: nodeIp, Path: fmt.Sprintf("/delete/%s", chunk.Hash)}
				nodeURL, err := url.PathUnescape(u.String())
				if err != nil {
					//if verbosity > 1 {
					//	log.Fatalf("error decoding node url: %e", err)
					//}
					continue
				}
				conn, _, err := websocket.DefaultDialer.Dial(nodeURL, nil)
				if err != nil {
					leftNodes = append(leftNodes, nodeAddr)
					continue
				}
				defer conn.Close()
				if err := c.deleteFile(conn); err != nil {
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
		}
		if len(leftChunks) == 0 {
			if err = c.storage.DeleteFileInfo(uuid); err == nil {
				deletedFiles += 1
			}
		} else {
			fileInfo.Chunks = leftChunks
			_ = c.storage.UpdateFileInfo(uuid, fileInfo)
		}
	}

	return totalFiles, deletedFiles, nil

}

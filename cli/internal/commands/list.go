package commands

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
)

func (c *Commands) GetListCommand() *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Usage:   "list all uploaded files",
		Action:  c.list,
	}
}

func (c *Commands) list(cCtx *cli.Context) error {
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
	fileInfos, err := c.storage.GetFileInfos()
	if err != nil {
		return err
	}
	// print info from config file

	for uuid, fileInfo := range fileInfos {
		fmt.Printf("Name: %s\n", fileInfo.Name)
		fmt.Printf("UUID: %s\n", uuid)
		fmt.Printf("Hash: %s\n", fileInfo.Hash)
		fmt.Printf("Size: %d\n", fileInfo.Size)
		fmt.Println("Chunks:")

		for _, chunk := range fileInfo.Chunks {
			fmt.Printf("    Number: %d\n", chunk.Number)
			fmt.Printf("    Hash: %s\n", chunk.Hash)
			fmt.Printf("    Nodes: %v\n", chunk.Nodes)
		}
	}

	return nil
}

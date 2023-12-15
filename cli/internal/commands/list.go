package commands

import (
	"fmt"
	"github.com/urfave/cli/v2"
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
				fmt.Printf("error during cleanup: %e\n", err)
			} else if totalFiles > 0 {
				fmt.Printf("successfully cleaned up %d/%d files\n", deletedFiles, totalFiles)
			}
		}
	}
	fileInfos, err := c.storage.GetFileInfos()
	if err != nil {
		return err
	}
	// print info from config file

	if len(fileInfos) == 0 {
		fmt.Printf("no files uploaded yet!\n")
		fmt.Printf("you can upload a file via\n")
		fmt.Printf("distorage upload <path>\n")
		return nil
	}

	fmt.Printf("Total stored files: %d\n", len(fileInfos))
	for uuid, fileInfo := range fileInfos {
		if !fileInfo.Available {
			continue
		}
		fmt.Println()
		fmt.Printf("Name: %s\n", fileInfo.Name)
		fmt.Printf("UUID: %s\n", uuid)
		fmt.Printf("Hash: %s\n", fileInfo.Hash)
		fmt.Printf("Size: %d\n", fileInfo.Size)
		fmt.Printf("Chunk count: %d\n", len(fileInfo.Chunks))
	}

	return nil
}

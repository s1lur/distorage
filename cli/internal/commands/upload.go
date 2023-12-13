package commands

import (
	"github.com/urfave/cli/v2"
	"os"
)

func (c *Commands) GetUploadCommand() *cli.Command {
	return &cli.Command{
		Name:    "upload",
		Aliases: []string{"u"},
		Usage:   "upload a file to the system",
		Action:  c.upload,
	}
}

func (c *Commands) upload(cCtx *cli.Context) error {
	path := cCtx.Args().First()
	contents, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// encrypt file

	// split file into chunks

	// upload file

	// store info locally
}

package commands

import (
	"github.com/urfave/cli/v2"
)

func (c *Commands) GetDownloadCommand() *cli.Command {
	return &cli.Command{
		Name:    "download",
		Aliases: []string{"d"},
		Usage:   "download a file from the system",
		Action:  c.download,
	}
}

func (c *Commands) download(cCtx *cli.Context) error {
	name := cCtx.Args().First()

	// get info about available nodes

	// fetch every chunk from an available node

	// assemble chunks

	// decrypt file

	// store file locally
}

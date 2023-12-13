package commands

import (
	"github.com/urfave/cli/v2"
)

func (c *Commands) GetDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"del"},
		Usage:   "delete uploaded file",
		Action:  c.delete,
	}
}

func (c *Commands) delete(cCtx *cli.Context) error {
	name := cCtx.Args().First()

	// get info about all available nodes

	// read local info about file

	// send delete request to every node
}

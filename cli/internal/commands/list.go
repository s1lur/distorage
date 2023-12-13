package commands

import (
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
	// print info from config file
}

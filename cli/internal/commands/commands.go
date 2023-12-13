package commands

import (
	"github.com/urfave/cli/v2"
)

type Commands struct {
}

func NewCommands() *Commands {
	return &Commands{}
}

func (c *Commands) GetCommands() []*cli.Command {
	return []*cli.Command{
		c.GetUploadCommand(),
		c.GetDownloadCommand(),
		c.GetListCommand(),
		c.GetDeleteCommand(),
	}
}

package app

import (
	c "cli/internal/commands"
	"github.com/urfave/cli/v2"
)

func InitApp() *cli.App {
	commands := c.NewCommands()
	app := &cli.App{
		Name:        "Distorage CLI",
		Description: "CLI for interacting with Distorage system",
		Commands:    commands.GetCommands(),
	}
	return app
}

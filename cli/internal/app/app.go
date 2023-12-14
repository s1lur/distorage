package app

import (
	"cli/config"
	c "cli/internal/commands"
	"cli/internal/usecase"
	"github.com/urfave/cli/v2"
)

func InitApp(cfg *config.Config) *cli.App {
	cryptoUC := usecase.NewCryptoUC()
	serverUC := usecase.NewServerUC(cfg.ServerIpAddr)
	storageUC := usecase.NewStorageUC("~/.distorage/files.json")
	commands := c.NewCommands(cfg, cryptoUC, serverUC, storageUC)

	app := &cli.App{
		Name:        "Distorage CLI",
		Description: "CLI for interacting with Distorage system",
		Commands:    commands.GetCommands(),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "verbosity",
				Value: 1,
				Usage: "set 0 for no output, 1 for minimal output, 2 for all output",
			},
			&cli.BoolFlag{
				Name:  "no-cleanup",
				Value: false,
				Usage: "use to run without cleanup",
			},
		},
	}
	return app
}

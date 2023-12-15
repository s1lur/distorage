package app

import (
	"cli/config"
	c "cli/internal/commands"
	"cli/internal/usecase"
	"github.com/urfave/cli/v2"
	"path"
)

func InitApp(cfg *config.Config, homeDir string) *cli.App {
	app := &cli.App{
		Name:        "Distorage CLI",
		Description: "CLI for interacting with Distorage system",
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
	cryptoUC := usecase.NewCryptoUC(path.Join(homeDir, ".distorage", "keys.json"))
	storageUC := usecase.NewStorageUC(path.Join(homeDir, ".distorage", "files.json"))
	if cfg != nil {
		serverUC := usecase.NewServerUC(cfg.ServerURL)
		commands := c.NewCommands(cfg, cryptoUC, serverUC, storageUC)
		app.Commands = commands.GetCommands()
	} else {
		app.Commands = c.InitCommandOnly(cryptoUC, storageUC)
	}

	return app
}

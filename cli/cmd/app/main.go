package main

import (
	"cli/config"
	a "cli/internal/app"
	"log"
	"os"
)

func main() {
	var (
		cfg *config.Config
		err error
	)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("error finding home directory\n")
	}
	if !(len(os.Args) > 1 && (os.Args[1] == "init" || os.Args[1] == "i")) {
		cfg, err = config.NewConfig(homeDir)
		if err != nil {
			log.Fatalf("Config error: %s", err)
			return
		}
	}

	app := a.InitApp(cfg, homeDir)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

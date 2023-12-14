package main

import (
	"cli/config"
	a "cli/internal/app"
	"log"
	"os"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}
	app := a.InitApp(cfg)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

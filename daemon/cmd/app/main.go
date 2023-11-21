package main

import (
	"github.com/s1lur/distorage/daemon/config"
	"github.com/s1lur/distorage/daemon/internal/app"
	"log"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}
	app.Run(cfg)
}

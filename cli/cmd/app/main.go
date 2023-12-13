package main

import (
	a "cli/internal/app"
	"log"
	"os"
)

func main() {
	app := a.InitApp()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

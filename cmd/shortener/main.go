package main

import (
	"log"

	"short-urls/internal/app"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	return app.NewShortUrlApp().Start()
}

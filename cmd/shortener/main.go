package main

import (
	"short-urls/internal/app"
)

func main() {

	shortUrlApp := app.NewShortUrlApp()
	err := shortUrlApp.Start()
	if err != nil {
		panic(err)
	}
}

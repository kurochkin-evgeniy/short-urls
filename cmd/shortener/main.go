package main

import (
	"fmt"
	"log"
	"os"

	"short-urls/internal/app"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func printBuildInfo() {
	fmt.Fprintf(os.Stdout, "Build version: %s\n", valueOrNA(buildVersion))
	fmt.Fprintf(os.Stdout, "Build date: %s\n", valueOrNA(buildDate))
	fmt.Fprintf(os.Stdout, "Build commit: %s\n", valueOrNA(buildCommit))
}

func valueOrNA(value string) string {
	if value == "" {
		return "N/A"
	}
	return value
}

func run() error {
	app, err := app.NewShortUrlApp()
	if err != nil {
		return err
	}
	return app.Start()
}

package main

import (
	"fmt"
	"os"

	"github.com/gookit/color"

	storage "github.com/juev/getpocket-collector"
	"github.com/juev/getpocket-collector/templates"
)

const storageFile = "data.json"

func main() {
	if err := run(); err != nil {
		color.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	consumerKey := os.Getenv("CONSUMER_KEY")
	if consumerKey == "" {
		return fmt.Errorf("failed to read `CONSUMER_KEY` env variable")
	}
	accessToken := os.Getenv("ACCESS_TOKEN")
	if accessToken == "" {
		return fmt.Errorf("failed to read `ACCESS_TOKEN` env variable")
	}

	data := storage.New(storageFile, consumerKey, accessToken)
	if err := data.Update(); err != nil {
		return err
	}

	userName := os.Getenv("USERNAME")
	if userName == "" {
		// if USERNAME is not setting, we use "juev" by default ;)
		userName = "juev"
	}

	if err := templates.TemplateFile(data, userName); err != nil {
		return err
	}

	return nil
}

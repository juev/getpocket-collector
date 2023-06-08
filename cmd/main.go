package main

import (
	"flag"
	"fmt"
	"os"

	storage "github.com/juev/getpocket-collector"
	"github.com/juev/getpocket-collector/templates"
)

const storageFile = "data.json"

func main() {
	var normalize bool
	flag.BoolVar(&normalize, "n", false, "only normalize database")
	flag.Parse()

	if normalize {
		if err := normalizeData(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	pocketFeedURL := os.Getenv("GETPOCKET_FEED_URL")
	if pocketFeedURL == "" {
		return fmt.Errorf("failed to read `GETPOCKET_FEED_URL` env variable")
	}

	data := storage.New(pocketFeedURL, storageFile)
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

func normalizeData() error {
	data := storage.New("", storageFile)
	if err := data.Normalize(); err != nil {
		return err
	}
	if err := data.Write(); err != nil {
		return err
	}
	return nil
}

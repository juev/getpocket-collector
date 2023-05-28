package main

import (
	"fmt"
	"os"

	storage "github.com/juev/getpocket-collector"
	"github.com/juev/getpocket-collector/templates"
)

const storageFile = "data.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data := storage.New(storageFile)
	if err := data.Proceed(); err != nil {
		return err
	}

	if err := templates.TemplateFile(data); err != nil {
		return err
	}

	return nil
}

package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/juev/getpocket-collector/storage"
)

const storageFile = "data.json"

//go:embed templates/template.tmpl
var templateString string

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	data := storage.New(storageFile)
	if err := data.Read(); err != nil {
		return fmt.Errorf("failed parse file `%s`: %v", storageFile, err)
	}

	if err := data.ParseFeed(); err != nil {
		return err
	}
	if err := data.Write(); err != nil {
		return fmt.Errorf("failed write to file `%s`: %v", storageFile, err)
	}

	return nil
}

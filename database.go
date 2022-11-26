package main

import (
	"encoding/json"
	"io"
	"os"
)

type Title string
type Url string
type Database map[Url]Title

func parseFile(fileName string) (database Database) {
	database = Database{}
	jsonFile, err := os.Open(fileName)
	if err != nil {
		exit("failed to read `%s`: %v", fileName, err)
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	_ = json.Unmarshal(byteValue, &database)
	return database
}

func writeFile(fileName string, database Database) error {
	data, err := json.Marshal(database)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, data, 0660)
}

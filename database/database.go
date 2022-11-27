package database

import (
	"encoding/json"
	"html"
	"io"
	"net/url"
	"os"
	"strings"
)

type Title string
type Url string
type Database map[Url]Title

// ParseFile read data from fileName
func ParseFile(fileName string) (database Database, err error) {
	database = Database{}
	var jsonFile *os.File
	if jsonFile, err = os.Open(fileName); err != nil {
		return nil, err
	}
	defer func(jsonFile *os.File) {
		_ = jsonFile.Close()
	}(jsonFile)

	byteValue, _ := io.ReadAll(jsonFile)
	if err = json.Unmarshal(byteValue, &database); err != nil {
		return nil, err
	}

	return database, err
}

// WriteFile for write data to fileName
func WriteFile(fileName string, database Database) error {
	data, err := json.Marshal(database)
	if err != nil {
		return err
	}

	return os.WriteFile(fileName, data, 0660)
}

// NormalizeURL should remove all utm from query
func NormalizeURL(in Url) string {
	u, err := url.Parse(string(in))
	if err != nil {
		return string(in)
	}

	query := url.Values{}
	for key, value := range u.Query() {
		if !strings.HasPrefix(key, "utm") {
			for _, v := range value {
				query.Add(key, v)
			}

		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// NormalizeTitle unescapes entities like "&lt;" to become "<"
func NormalizeTitle(in Title) string {
	return html.UnescapeString(string(in))
}

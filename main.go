package main

import (
	"bufio"
	_ "embed"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/template"

	"github.com/juev/getpocket-collector/database"
	"github.com/juev/getpocket-collector/helpers"
	"github.com/juev/getpocket-collector/rss"
)

const feedURL = "GETPOCKET_FEED_URL"
const databaseFolder = "data"
const databaseFile = "data/database.json"

//go:embed templates/template.tmpl
var content string

func main() {
	var err error
	pocketFeedURL, ok := os.LookupEnv(feedURL)
	if !ok {
		helpers.Exit("failed to read %s env variable", feedURL)
	}

	userName, ok := os.LookupEnv("USERNAME")
	if !ok {
		userName = "juev"
	}

	if _, err = os.Stat(databaseFolder); os.IsNotExist(err) {
		_ = os.Mkdir("data", 0755)
		fileData, _ := os.Create(databaseFile)
		_ = fileData.Close()
	}

	var data database.Database
	if data, err = database.ParseFile(databaseFile); err != nil {
		helpers.Exit("failed to parse file %s: %v", databaseFile, err)
	}

	var resp *http.Response
	if resp, err = helpers.ReadWithClient(pocketFeedURL); err != nil {
		helpers.Exit("failed to read %s: %v", pocketFeedURL, err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	var channel *rss.Channel
	if channel, err = rss.ParseFeed(resp); err != nil {
		helpers.Exit("cannot parse feed: %v", err)
	}

	urls := make([]string, 0, len(data))
	for k := range data {
		urls = append(urls, string(k))
	}
	sort.Strings(urls)

	_ = database.WriteFile(databaseFile, data)

	var f *os.File
	if f, err = os.Create("README.md"); err != nil {
		helpers.Exit("cannot create file: %v", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var funcMap = template.FuncMap{
		"normalizeUrl":   func(url database.Url) string { return database.NormalizeURL(url) },
		"normalizeTitle": func(title database.Title) string { return database.NormalizeTitle(title) },
	}

	temp := template.Must(template.New("links").Funcs(funcMap).Parse(content))

	r := struct {
		Title    string
		UserName string
		Content  database.Database
	}{
		Title:    channel.Title,
		UserName: userName,
		Content:  data,
	}

	var buffer strings.Builder
	if err = temp.Execute(&buffer, r); err != nil {
		helpers.Exit("failed to parse template: %v", err)
	}

	w := bufio.NewWriter(f)
	_, _ = w.WriteString(buffer.String())
	_ = w.Flush()
}

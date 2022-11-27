package main

import (
	_ "embed"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/juev/getpocket-collector/database"
	"github.com/juev/getpocket-collector/helpers"
	"github.com/juev/getpocket-collector/rss"
)

const databaseFile = "README.md"

//go:embed templates/template.tmpl
var templateString string

func main() {
	var err error
	pocketFeedURL, ok := os.LookupEnv("GETPOCKET_FEED_URL")
	if !ok {
		helpers.Exit("failed to read `GETPOCKET_FEED_URL` env variable")
	}

	if _, err = os.Stat(databaseFile); os.IsNotExist(err) {
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

	for _, item := range channel.Item {
		tt, _ := time.Parse(time.RFC1123Z, string(item.PubDate))
		pubDate, _ := time.Parse("02 Jan 2006", tt.Format("02 Jan 2006"))
		if _, ok = data[pubDate]; !ok {
			data[pubDate] = map[database.Url]database.Title{}
		}
		url := database.NormalizeURL(database.Url(item.Link))
		data[pubDate][database.Url(url)] = database.Title(item.Title)
	}

	if err = database.WriteFile(databaseFile, templateString, data, channel.Title); err != nil {
		helpers.Exit("failed to write file `%s`: %v", databaseFile, err)
	}
}

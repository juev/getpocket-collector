package main

import (
	"fmt"
	"net/http"
	"os"
	"sort"
)

const feedURL = "GETPOCKET_FEED_URL"
const databaseFolder = "data"
const databaseFile = "data/database.json"

func main() {
	var err error
	pocketFeedURL, ok := os.LookupEnv(feedURL)
	if !ok {
		exit("failed to read %s env variable", feedURL)
	}

	_, err = os.Stat(databaseFolder)
	if os.IsNotExist(err) {
		_ = os.Mkdir("data", 0755)
		fileData, _ := os.Create(databaseFile)
		fileData.Close()
	}

	database := parseFile(databaseFile)
	var resp *http.Response
	if resp, err = readWithClient(pocketFeedURL); err != nil {
		exit("failed to read %s: %v", pocketFeedURL, err)
	}
	defer resp.Body.Close()

	var channel *Channel
	if channel, err = ParseFeed(resp); err != nil {
		exit("cannot parse feed: %v", err)
	}

	for _, item := range channel.Item {
		database[Url(normalizeURL(item.Link))] = Title(item.Title)
	}
	writeFile(databaseFile, database)

	var f *os.File
	if f, err = os.Create("README.md"); err != nil {
		exit("cannot create file: %v", err)
	}
	defer f.Close()

	urls := make([]string, 0, len(database))
	for k := range database {
		urls = append(urls, string(k))
	}
	sort.Strings(urls)

	f.WriteString(fmt.Sprintf("# %s\n\n", channel.Title))
	for _, url := range urls {
		f.WriteString(fmt.Sprintf("- [%s](%s)\n", database[Url(url)], url))
	}
}

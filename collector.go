package storage

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

// Storage is main interface for json-database
type Storage interface {
	Read() error
	Write() error
	ParseFeed() error
	Proceed() error
	TemplateFile() error
}

type storage struct {
	Title    string `json:"title"`
	Updated  string `json:"updated"`
	Items    []Item `json:"items"`
	fileName string
}

// Item one link from feed
type Item struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Published string `json:"published"`
}

// New creates new storage
func New(fileName string) Storage {
	return &storage{fileName: fileName}
}

// Read data from fileName
func (s *storage) Read() (err error) {
	if _, err = os.Stat(s.fileName); os.IsNotExist(err) {
		fileData, err := os.Create(s.fileName)
		if err != nil {
			return fmt.Errorf("cannot create file `%s`: %v", s.fileName, err)
		}
		fileData.Close()
		return nil
	}

	data, err := os.ReadFile(s.fileName)
	if err != nil {
		return fmt.Errorf("cannot read file `%s`: %v", s.fileName, err)
	}

	err = json.Unmarshal(data, s)

	return
}

// Write for write data to fileName
func (s *storage) Write() (err error) {
	dataJSON, _ := json.MarshalIndent(s, "", "    ")
	f, err := os.Create(s.fileName)
	if err != nil {
		return fmt.Errorf("cannot create file `%s`: %v", s.fileName, err)
	}
	if _, err := f.Write(dataJSON); err != nil {
		return fmt.Errorf("cannot write to file `%s`: %v", s.fileName, err)
	}
	f.Close()

	return nil
}

// ParseFeed processed feed from getpocket
func (s *storage) ParseFeed() (err error) {
	pocketFeedURL := os.Getenv("GETPOCKET_FEED_URL")
	if pocketFeedURL == "" {
		return fmt.Errorf("failed to read `GETPOCKET_FEED_URL` env variable")
	}

	fp := gofeed.NewParser()
	fp.UserAgent = "getpocket-collector 1.0"
	feed, err := fp.ParseURL(pocketFeedURL)
	if err != nil {
		return fmt.Errorf("cannot parse feed: %v", err)
	}

	lastUpdate, err := time.Parse(time.RFC3339, s.Updated)

	s.Title = feed.Title
	for i := len(feed.Items) - 1; i != 0; i-- {
		el := feed.Items[i]
		if lastUpdate.Before(*el.PublishedParsed) {
			s.Items = append(s.Items, Item{
				Title:     normalizeTitle(el.Title),
				Link:      normalizeLink(el.Link),
				Published: el.PublishedParsed.Format(time.RFC3339),
			})
			s.Updated = el.PublishedParsed.Format(time.RFC3339)
		}
	}

	return nil
}

// Proceed simple function for read/parseFeed/write
func (s *storage) Proceed() (err error) {
	if err := s.Read(); err != nil {
		return err
	}

	if err := s.ParseFeed(); err != nil {
		return err
	}

	if err := s.Write(); err != nil {
		return err
	}

	return nil
}

// normalizeLink should remove all utm from query
func normalizeLink(in string) string {
	u, err := url.Parse(string(in))
	if err != nil {
		return in
	}

	query := url.Values{}
	for key, value := range u.Query() {
		if oneOff(key, []string{"v", "p", "id"}) {
			for _, v := range value {
				query.Add(key, v)
			}
		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// normalizeTitle unescapes entities like "&lt;" to become "<"
func normalizeTitle(in string) string {
	return html.UnescapeString(in)
}

func oneOff(k string, fields []string) bool {
	for _, el := range fields {
		if k == el {
			return true
		}
	}

	return false
}

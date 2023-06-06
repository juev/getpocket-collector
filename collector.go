package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// Database is main interface for json-database
type Database interface {
	Read() error
	Write() error
	ParseFeed() error
	Update() error
}

// Storage is struct for storing feed info
type Storage struct {
	Title    string `json:"title"`
	Updated  string `json:"updated"`
	Items    []Item `json:"items"`
	fileName string
	feed     string
	links    map[string]struct{}
}

// Item one link from feed
type Item struct {
	Title     string `json:"title"`
	Link      string `json:"link"`
	Published string `json:"published"`
}

// New creates new storage
func New(feed, fileName string) *Storage {
	return &Storage{
		fileName: fileName,
		feed:     feed,
	}
}

// Read data from fileName
func (s *Storage) Read() (err error) {
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

	s.links = make(map[string]struct{})
	for _, el := range s.Items {
		s.links[el.Link] = struct{}{}
	}

	return
}

// Write for write data to fileName
func (s *Storage) Write() (err error) {
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
func (s *Storage) ParseFeed() (err error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "getpocket-collector 1.0"
	feed, err := fp.ParseURL(s.feed)
	if err != nil {
		return fmt.Errorf("cannot parse feed: %v", err)
	}

	lastUpdate, err := time.Parse(time.RFC3339, s.Updated)

	s.Title = feed.Title
	for i := range feed.Items {
		el := feed.Items[len(feed.Items)-i-1]
		if lastUpdate.Before(*el.PublishedParsed) {
			link := normalizeLink(el.Link)
			title, link, err := getURL(link)
			if err != nil {
				s.Updated = el.PublishedParsed.Format(time.RFC3339)
				continue
			}
			if s.notContainsLink(link) {
				s.Items = append(s.Items, Item{
					Title:     normalizeTitle(title),
					Link:      link,
					Published: el.PublishedParsed.Format(time.RFC3339),
				})
				s.Updated = el.PublishedParsed.Format(time.RFC3339)
			}
		}
	}

	return nil
}

// Update simple function for read/parseFeed/write
func (s *Storage) Update() (err error) {
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

// Normalize just check all url for existing and update Title
func (s *Storage) Normalize() (err error) {
	if err := s.Read(); err != nil {
		return err
	}

	var items []Item
	for _, item := range s.Items[:100] {
		fmt.Printf("check url: %s\n", item.Link)
		title, finishURL, err := getURL(item.Link)
		if err != nil {
			fmt.Printf("get error: %v\n", errors.Unwrap(err))
			continue
		}
		items = append(items, Item{
			Title:     normalizeTitle(title),
			Link:      normalizeLink(finishURL),
			Published: item.Published,
		})
	}
	s.Items = items

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
		if oneOff(key, []string{"v", "p", "id", "article"}) {
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

func (s *Storage) notContainsLink(link string) bool {
	if _, ok := s.links[link]; ok {
		return false
	}

	return true
}

func getURL(url string) (title, finalURL string, err error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return title, finalURL, fmt.Errorf("cannot fetch url (%s): %v", url, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		return title, finalURL, fmt.Errorf("cannot parse html from url (%s): %v", url, err)
	}

	finalURL = resp.Request.URL.String()

	title = doc.Find("title").Text()
	if title == "" {
		title = "Untitle"
	}

	return title, finalURL, nil
}

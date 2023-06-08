package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/anaskhan96/soup"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"
)

// Database is main interface for json-database
type Database interface {
	Read() error
	Write() error
	ParseFeed() error
	Update() error
	Normalize() error
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
func (s *Storage) Write() error {
	var buf bytes.Buffer
	dec := json.NewEncoder(&buf)
	dec.SetIndent("", "    ")
	dec.SetEscapeHTML(false)
	if err := dec.Encode(s); err != nil {
		return err
	}

	f, err := os.Create(s.fileName)
	if err != nil {
		return fmt.Errorf("cannot create file `%s`: %v", s.fileName, err)
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
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
			title, link, err := getURL(el.Link)
			if err != nil {
				s.Updated = el.PublishedParsed.Format(time.RFC3339)
				continue
			}
			if s.notContainsLink(link) {
				s.Items = append(s.Items, Item{
					Title:     normalizeTitle(title),
					Link:      normalizeLink(link),
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
	var lock sync.Mutex
	group := errgroup.Group{}
	group.SetLimit(100)
	for _, item := range s.Items {
		item := item
		group.Go(
			func() error {
				fmt.Printf("check url: %s\n", item.Link)
				title, finishURL, err := getURL(item.Link)
				if err != nil {
					fmt.Printf("failed: %v\n", errors.Unwrap(err))
					return nil
				}
				lock.Lock()
				items = append(items, Item{
					Title:     normalizeTitle(title),
					Link:      normalizeLink(finishURL),
					Published: item.Published,
				})
				lock.Unlock()
				return nil
			})
	}
	_ = group.Wait()
	sort.Slice(items, sortItems(items))
	s.Items = items

	if err := s.Write(); err != nil {
		return err
	}

	return nil
}

func sortItems(s []Item) func(int, int) bool {
	return func(i, j int) bool {
		return s[i].Published < s[j].Published
	}
}

// normalizeLink should remove all utm from query
func normalizeLink(in string) string {
	u, err := url.Parse(in)
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
	in = html.UnescapeString(in)
	in = strings.Map(func(r rune) rune {
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, in)

	in = strings.ReplaceAll(in, "  ", " ")
	in = strings.TrimSpace(in)
	return in
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
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	resp, err := client.Head(url)
	if err != nil {
		return title, finalURL, fmt.Errorf("cannot fetch url (%s): %w", url, err)
	}
	defer resp.Body.Close()
	finalURL = resp.Request.URL.String()
	if b, err := io.ReadAll(resp.Body); err == nil {
		if strings.Contains(http.DetectContentType(b), "text/plain;") {
			response, err := client.Get(finalURL)
			if err != nil {
				return title, finalURL, fmt.Errorf("cannot fetch url (%s): %w", url, err)
			}
			defer response.Body.Close()
			bodyBytes, _ := io.ReadAll(response.Body)

			tt := soup.HTMLParse(string(bodyBytes)).Find("head").Find("title")
			if tt.Pointer != nil {
				title = tt.Text()
			}
		}
	}

	if title == "" {
		title = "Untitled"
	}

	return title, finalURL, nil
}

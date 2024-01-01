package storage

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/anaskhan96/soup"
	"github.com/gookit/color"
	"github.com/mmcdole/gofeed"
	"github.com/sourcegraph/conc/pool"
)

const maxRequests = 100

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
		links:    make(map[string]struct{}, 1000),
	}
}

// Read data from fileName
func (s *Storage) Read() (err error) {
	if _, err = os.Stat(s.fileName); err != nil {
		return nil
	}

	data, err := os.ReadFile(s.fileName)
	if err != nil {
		return fmt.Errorf("cannot read file `%s`: %w", s.fileName, err)
	}

	if json.Valid(data) {
		err = json.Unmarshal(data, s)
		if err != nil {
			return err
		}
	}

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

	err := os.WriteFile(s.fileName, buf.Bytes(), 0600)
	if err != nil {
		return fmt.Errorf("cannot create file `%s`: %w", s.fileName, err)
	}

	return nil
}

// ParseFeed processed feed from getpocket
func (s *Storage) ParseFeed() (err error) {
	fp := gofeed.NewParser()
	fp.UserAgent = "getpocket-collector"
	feed, err := fp.ParseURL(s.feed)
	if err != nil {
		return fmt.Errorf("cannot parse feed: %w", err)
	}

	lastUpdate, _ := time.Parse(time.RFC3339, s.Updated)

	p := pool.New().WithMaxGoroutines(maxRequests)
	ch := make(chan Item, 1)
	s.Title = feed.Title
	go func() {
		for _, el := range feed.Items {
			el := el
			p.Go(func() {
				if lastUpdate.Before(*el.PublishedParsed) {
					title, link, err := getURL(el.Link)
					if err != nil {
						return
					}
					if s.notContainsLink(link) {
						ch <- Item{
							Title:     normalizeTitle(title),
							Link:      normalizeLink(link),
							Published: el.PublishedParsed.Format(time.RFC3339),
						}
					}
				}
			})
		}
		p.Wait()
		close(ch)
	}()

	for el := range ch {
		s.Items = append(s.Items, el)
		s.links[el.Link] = struct{}{}
		elPub, _ := time.Parse(time.RFC3339, el.Published)
		sUpd, _ := time.Parse(time.RFC3339, s.Updated)
		if elPub.After(sUpd) {
			s.Updated = el.Published
		}
	}

	slices.SortFunc(s.Items, func(a, b Item) int {
		return cmp.Compare(a.Published, b.Published)
	})

	return nil
}

// Update simple function for read/parseFeed/write
func (s *Storage) Update() (err error) {
	if err = s.Read(); err != nil {
		return err
	}

	if err = s.ParseFeed(); err != nil {
		return err
	}

	if err = s.Write(); err != nil {
		return err
	}

	return nil
}

// Normalize just check all url for existing and update Title
func (s *Storage) Normalize() (err error) {
	if err = s.Read(); err != nil {
		return err
	}

	items := make([]Item, 0, len(s.Items))
	p := pool.New().WithMaxGoroutines(maxRequests)
	ch := make(chan Item, 1)
	go func() {
		for _, item := range s.Items {
			item := item
			p.Go(func() {
				color.Printf("check url: %s\n", item.Link)
				title, finishURL, err := getURL(item.Link)
				if err != nil {
					color.Printf("failed: %v\n", errors.Unwrap(err))
					return
				}
				ch <- Item{
					Title:     normalizeTitle(title),
					Link:      normalizeLink(finishURL),
					Published: item.Published,
				}
			})
		}
		p.Wait()
		close(ch)
	}()

	for item := range ch {
		items = append(items, item)
	}

	slices.SortFunc(items, func(a, b Item) int {
		return cmp.Compare(a.Published, b.Published)
	})
	s.Items = items

	if err = s.Write(); err != nil {
		return err
	}

	return nil
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
	client := http.Client{Timeout: time.Second}
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("User-Agent", "getpocket-collector")
	response, err := client.Do(request)
	if err != nil {
		return title, finalURL, fmt.Errorf("cannot fetch url (%s): %w", url, err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)
	finalURL = response.Request.URL.String()
	if body, err := io.ReadAll(response.Body); err == nil {
		tt := soup.HTMLParse(string(body)).Find("head").Find("title")
		if tt.Pointer != nil {
			title = tt.Text()
		}
	}

	if title == "" {
		title = "Untitled"
	}

	return title, finalURL, nil
}

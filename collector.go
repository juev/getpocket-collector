package storage

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
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
	links    sync.Map
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
		s.links.Store(el.Link, struct{}{})
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
						title = el.Title
						link = el.Link
					}
					ch <- Item{
						Title:     normalizeTitle(title),
						Link:      normalizeLink(link),
						Published: el.PublishedParsed.Format(time.RFC3339),
					}
				}
			})
		}
		p.Wait()
		close(ch)
	}()

	for el := range ch {
		if s.notContainsLink(el.Link) {
			s.Items = append(s.Items, el)
			s.links.Store(el.Link, struct{}{})
			elPub, _ := time.Parse(time.RFC3339, el.Published)
			sUpd, _ := time.Parse(time.RFC3339, s.Updated)
			if elPub.After(sUpd) {
				s.Updated = el.Published
			}
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
				title, finishURL, err := getURL(item.Link)
				if err != nil {
					color.Printf("failed normilize link (%s): %s\n", item.Link, err)
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
	if _, ok := s.links.Load(link); ok {
		return false
	}

	return true
}

func getURL(addr string) (title, finalURL string, err error) {
	parsedURL, err := url.Parse(addr)
	if err != nil {
		return "", "", err
	}

	if parsedURL.Hostname() == "github.com" {
		title = `GitHub - ` + strings.TrimPrefix(parsedURL.Path, `/`)
		finalURL = addr
		return title, finalURL, nil
	}

	client := http.Client{Timeout: 15 * time.Second}
	request, _ := http.NewRequest("GET", addr, nil)
	request.Header.Set("User-Agent", `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36`)
	request.Header.Add("Accept", `text/html,application/xhtml+xml,application/xml`)

	response, err := client.Do(request)
	if err != nil {
		return title, finalURL, fmt.Errorf("cannot fetch url (%s): %w", addr, err)
	}
	defer func(body io.ReadCloser) {
		_ = body.Close()
	}(response.Body)
	if response.StatusCode != http.StatusOK {
		return title, finalURL, fmt.Errorf("statusCode is %d", response.StatusCode)
	}
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

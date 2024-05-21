package storage

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/imroc/req/v3"
)

const maxRequests = 100

// Storage is struct for storing getpocket items
type Storage struct {
	Title       string        `json:"title"`
	Updated     string        `json:"updated"`
	Items       []StorageItem `json:"items"`
	fileName    string
	consumerKey string
	accessToken string
	since       string
	links       sync.Map
}

// StorageItem one link from feed
type StorageItem struct {
	Title     string `json:"resolved_title"`
	Link      string `json:"resolved_url"`
	Excerpt   string `json:"excerpt"`
	Published string `json:"time_added"`
}

type PocketJson struct {
	List  map[string]StorageItem `json:"list"`
	Error error                  `json:"error"`
}

// New creates new storage
func New(fileName, consumerKey, accessToken string) *Storage {
	return &Storage{
		fileName:    fileName,
		consumerKey: consumerKey,
		accessToken: accessToken,
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

	s.since = s.Updated

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

// PocketParse processed feed from getpocket
func (s *Storage) PocketParse(bodyBytes []byte) (err error) {
	var pocketJson PocketJson

	if err := json.Unmarshal(bodyBytes, &pocketJson); err != nil {
		return fmt.Errorf("failed to unmarchal response from getpocket: %w", err)
	}

	s.Title = "My Reading List: Unread"
	for _, el := range pocketJson.List {
		if s.notContainsLink(el.Link) {
			// convert time
			d, _ := strconv.ParseInt(el.Published, 10, 64)
			// convert empty title
			if el.Title == "" {
				el.Title = "Untitled"
			}
			s.Items = append(s.Items, StorageItem{
				Title:     el.Title,
				Link:      el.Link,
				Excerpt:   normalizeExcerpt(el.Excerpt),
				Published: time.Unix(d, 0).Format(time.RFC3339),
			})
			s.links.Store(el.Link, struct{}{})
		}
	}

	slices.SortFunc(s.Items, func(a, b StorageItem) int {
		return cmp.Compare(a.Published, b.Published)
	})

	s.Updated = time.Unix(time.Now().Unix(), 0).Format(time.RFC3339)

	return nil
}

// Update simple function for read/parseFeed/write
func (s *Storage) Update() error {
	if err := s.Read(); err != nil {
		return err
	}

	client := req.C().
		SetUserAgent("getpocket-collector").
		SetTimeout(15 * time.Second)

	response, err := client.R().
		SetQueryParams(map[string]string{
			"consumer_key": s.consumerKey,
			"access_token": s.accessToken,
			"since":        s.since,
		}).
		Get("https://getpocket.com/v3/get")

	if err != nil {
		return err
	}

	fmt.Println(response.String())
	if err := s.PocketParse(response.Bytes()); err != nil {
		return err
	}

	if err := s.Write(); err != nil {
		return err
	}

	return nil
}

func (s *Storage) notContainsLink(link string) bool {
	if _, ok := s.links.Load(link); ok {
		return false
	}

	return true
}

// normalizeExcerpt unescapes entities like "&lt;" to become "<"
func normalizeExcerpt(in string) string {
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

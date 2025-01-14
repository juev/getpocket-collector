package storage

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

// Collector is struct for storing getpocket items
type Collector struct {
	Title       string `json:"title"`
	Updated     string `json:"updated"`
	Items       []Item `json:"items"`
	fileName    string
	consumerKey string
	accessToken string
	since       string
	links       sync.Map
}

// Item one link from getpocket
type Item struct {
	Title     string `json:"title,omitempty"`
	Link      string `json:"link,omitempty"`
	Published string `json:"published,omitempty"`
}

// PocketJSON to store response from the getpocket api
type PocketJSON struct {
	List  map[string]PocketJSONItem `json:"list"`
	Error string                    `json:"error"`
	Since int64                     `json:"since"`
}

// PocketJSONItem to store an element from the getpocket api
type PocketJSONItem struct {
	Title     string `json:"resolved_title"`
	Link      string `json:"resolved_url"`
	Published string `json:"time_added"`
}

// New creates new storage
func New(fileName, consumerKey, accessToken string) *Collector {
	return &Collector{
		fileName:    fileName,
		consumerKey: consumerKey,
		accessToken: accessToken,
	}
}

// Read data from fileName
func (s *Collector) Read() (err error) {
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

	if s.Updated != "" {
		t, err := time.Parse(time.RFC3339, s.Updated)
		if err != nil {
			return err
		}

		s.since = strconv.FormatInt(t.Unix(), 10)
	}

	return
}

// Write for write data to fileName
func (s *Collector) Write() error {
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
func (s *Collector) PocketParse(bodyBytes []byte) (err error) {
	var pocketJSON PocketJSON
	if err := json.Unmarshal(bodyBytes, &pocketJSON); err != nil {
		return fmt.Errorf("failed to unmarchal response from getpocket: %w", err)
	}

	s.Title = "My Reading List: Unread"
	for _, el := range pocketJSON.List {
		if s.notContainsLink(el.Link) {
			d, _ := strconv.ParseInt(el.Published, 10, 64)
			// convert empty title
			if el.Title == "" {
				el.Title = "Untitled"
			}
			s.Items = append(s.Items, Item{
				Title:     el.Title,
				Link:      el.Link,
				Published: time.Unix(d, 0).Format(time.RFC3339),
			})
			s.links.Store(el.Link, struct{}{})
		}
	}

	slices.SortFunc(s.Items, func(a, b Item) int {
		return cmp.Compare(a.Published, b.Published)
	})

	s.Updated = time.Unix(pocketJSON.Since, 0).Format(time.RFC3339)

	return nil
}

// Update simple function for read/parseFeed/write
func (s *Collector) Update() error {
	if err := s.Read(); err != nil {
		return err
	}

	defaultCount := 30
	requestBody := map[string]string{
		"consumer_key": s.consumerKey,
		"access_token": s.accessToken,
		"detailType":   "simple",
		"state":        "unread",
		"count":        strconv.Itoa(defaultCount),
		"total":        "1",
	}
	if s.since != "" {
		requestBody["since"] = s.since
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return err
	}

	request, _ := http.NewRequest(
		http.MethodPost,
		"https://getpocket.com/v3/get",
		io.NopCloser(bytes.NewBuffer(jsonData)))
	request.Header.Set("User-Agent", `getpocket-collector`)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return fmt.Errorf("status code: %d", response.StatusCode)
	}

	var body []byte
	if body, err = io.ReadAll(response.Body); err != nil {
		return err
	}

	if gjson.GetBytes(body, "error").String() != "" {
		return fmt.Errorf("error: %s", gjson.GetBytes(body, "error").String())
	}

	if err = s.PocketParse(body); err != nil {
		return err
	}

	if err = s.Write(); err != nil {
		return err
	}

	return nil
}

func (s *Collector) notContainsLink(link string) bool {
	if _, ok := s.links.Load(link); ok {
		return false
	}

	return true
}

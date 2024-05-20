package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
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
	Title     string `json:"title"`
	Link      string `json:"link"`
	Excerpt   string `json:"excerpt"`
	Published string `json:"published"`
}

type PocketJson struct {
	List  map[string]Item `json:"list"`
	Error error           `json:"error"`
}

type Item struct {
	Title   string `json:"resolved_title"`
	URL     string `json:"resolved_url"`
	Excerpt string `json:"excerpt"`
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

	fmt.Println(string(bodyBytes))

	if err := json.Unmarshal(bodyBytes, &pocketJson); err != nil {
		return fmt.Errorf("failed to unmarchal response from getpocket: %w", err)
	}

	for _, el := range pocketJson.List {
		if s.notContainsLink(el.URL) {
			s.Items = append(s.Items, StorageItem{
				Title:   el.Title,
				Link:    el.URL,
				Excerpt: el.Excerpt,
			})
			s.links.Store(el.URL, struct{}{})
		}
	}

	return nil
}

// Update simple function for read/parseFeed/write
func (s *Storage) Update() error {
	if err := s.Read(); err != nil {
		return err
	}

	bodyBytes, err := s.requestToPocket()
	if err != nil {
		return err
	}

	if err := s.PocketParse(bodyBytes); err != nil {
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

func (s *Storage) requestToPocket() (bodyBytes []byte, err error) {
	req, _ := http.NewRequest(http.MethodGet, "https://getpocket.com/v3/get", nil)

	q := req.URL.Query()
	q.Add("consumer_key", s.consumerKey)
	q.Add("access_token", s.accessToken)
	q.Add("since", s.since)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to getpocket: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to make request to getpocket: got status %s", resp.Status)
	}

	bodyBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return bodyBytes, nil
}

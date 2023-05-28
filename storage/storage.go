package storage

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"
	"os"
	"time"

	"github.com/mmcdole/gofeed"
)

type Title string
type Link string

type Storage interface {
	Read() (err error)
	Write() (err error)
	ParseFeed() (err error)
}

type storage struct {
	Title    Title  `json:"title"`
	Updated  string `json:"updated"`
	Items    []Item `json:"items"`
	fileName string
}

type Item struct {
	Title     Title  `json:"title"`
	Link      Link   `json:"link"`
	Published string `json:"published"`
}

func New(fileName string) Storage {
	return &storage{fileName: fileName}
}

// Read data from fileName
func (s *storage) Read() (err error) {
	// если файла нет, создаем его
	// возвращаем пустую структуру без ошибки
	if _, err = os.Stat(s.fileName); os.IsNotExist(err) {
		fileData, _ := os.Create(s.fileName)
		fileData.Close()
		return nil
	}

	dataFile, err := os.Open(s.fileName)
	if err != nil {
		return err
	}
	defer dataFile.Close()

	data, err := io.ReadAll(dataFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, s)

	return
}

// Write for write data to fileName
func (s *storage) Write() (err error) {
	dataJson, _ := json.MarshalIndent(s, "", "    ")
	f, err := os.Create(s.fileName)
	if err != nil {
		return fmt.Errorf("cannot create file: %v", err)
	}
	f.Write(dataJson)
	f.Close()

	return nil
}

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

	s.Title = Title(feed.Title)
	for _, el := range feed.Items {
		if lastUpdate.Before(*el.PublishedParsed) {
			s.Items = append(s.Items, Item{
				Title:     normalizeTitle(el.Title),
				Link:      normalizeLink(el.Link),
				Published: el.Published,
			})
		}
	}

	return nil
}

// func TemplateFile() (err error) {
// 	userName, ok := os.LookupEnv("USERNAME")
// 	if !ok {
// 		// if USERNAME is not setting, we use "juev" by default ;)
// 		userName = "juev"
// 	}

// 	var funcMap = template.FuncMap{
// 		"normalizeUrl":   func(url string) string { return NormalizeURL(url) },
// 		"normalizeTitle": func(title string) string { return NormalizeTitle(title) },
// 	}

// 	temp := template.Must(template.New("links").Funcs(funcMap).Parse(content))

// 	r := struct {
// 		Title    string
// 		UserName string
// 		Content  Storage
// 	}{
// 		Title:    "sdf",
// 		UserName: userName,
// 		Content:  data,
// 	}

// 	var buffer strings.Builder
// 	if err = temp.Execute(&buffer, r); err != nil {
// 		return err
// 	}

// 	w := bufio.NewWriter(f)
// 	_, _ = w.WriteString(buffer.String())
// 	_ = w.Flush()

// 	return nil
// }

// normalizeLink should remove all utm from query
func normalizeLink(in string) Link {
	u, err := url.Parse(string(in))
	if err != nil {
		return Link(in)
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
	return Link(u.String())
}

// normalizeTitle unescapes entities like "&lt;" to become "<"
func normalizeTitle(in string) Title {
	return Title(html.UnescapeString(in))
}

func oneOff(k string, fields []string) bool {
	for _, el := range fields {
		if k == el {
			return true
		}
	}

	return false
}

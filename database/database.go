package database

import (
	"bufio"
	_ "embed"
	"html"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/juev/getpocket-collector/helpers"
)

type Title string
type Url string
type Database map[time.Time]map[Url]Title

// ParseFile read data from fileName
func ParseFile(fileName string) (database Database, err error) {
	database = Database{}
	var mdFile *os.File
	if mdFile, err = os.Open(fileName); err != nil {
		return nil, err
	}
	defer func(fileName *os.File) {
		_ = fileName.Close()
	}(mdFile)

	// для даты строка вида: `### {{ $date }}`
	// для ссылки строка вида: `- [{{ $title }}]({{ $url }})`
	var re *regexp.Regexp
	if re, err = regexp.Compile(`### (?P<date>.*)|- \[(?P<title>.*)]\((?P<url>.*)\)`); err != nil {
		return nil, err
	}

	fileScanner := bufio.NewScanner(mdFile)
	fileScanner.Split(bufio.ScanLines)
	var currentDate time.Time
	for fileScanner.Scan() {
		parts := re.FindStringSubmatch(fileScanner.Text())
		if parts == nil {
			continue
		}
		result := make(map[string]string)
		for i, name := range re.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = parts[i]
			}
		}
		if result == nil {
			continue
		}
		// после обработки в result мы будем иметь мапу с тремя заполненными значениями
		// map[date: title: url:]
		if result["date"] != "" {
			currentDate, _ = time.Parse("02 Jan 2006", result["date"])
			database[currentDate] = map[Url]Title{}
			continue
		}
		if result["url"] != "" {
			database[currentDate][Url(result["url"])] = Title(result["title"])
		}
	}
	return database, err
}

// WriteFile for write data to fileName
func WriteFile(fileName string, content string, data Database, title string) (err error) {
	userName, ok := os.LookupEnv("USERNAME")
	if !ok {
		// if USERNAME is not setting, we use "juev" by default ;)
		userName = "juev"
	}

	var f *os.File
	if f, err = os.Create(fileName); err != nil {
		helpers.Exit("cannot create file: %v", err)
	}
	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	var funcMap = template.FuncMap{
		"normalizeUrl":   func(url Url) string { return NormalizeURL(url) },
		"normalizeTitle": func(title Title) string { return NormalizeTitle(title) },
	}

	temp := template.Must(template.New("links").Funcs(funcMap).Parse(content))

	r := struct {
		Title    string
		UserName string
		Content  Database
	}{
		Title:    title,
		UserName: userName,
		Content:  data,
	}

	var buffer strings.Builder
	if err = temp.Execute(&buffer, r); err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	_, _ = w.WriteString(buffer.String())
	_ = w.Flush()

	return nil
}

// NormalizeURL should remove all utm from query
func NormalizeURL(in Url) string {
	u, err := url.Parse(string(in))
	if err != nil {
		return string(in)
	}

	query := url.Values{}
	for key, value := range u.Query() {
		if !strings.HasPrefix(key, "utm") {
			for _, v := range value {
				query.Add(key, v)
			}

		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// NormalizeTitle unescapes entities like "&lt;" to become "<"
func NormalizeTitle(in Title) string {
	return html.UnescapeString(string(in))
}

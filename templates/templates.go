package templates

import (
	// embed templates to string variables
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	storage "github.com/juev/getpocket-collector"
)

//go:embed template.tmpl
var templateString string

type Data struct {
	Title    string
	UserName string
	Content  *storage.Storage
	Count    int
}

func TemplateFile(s *storage.Storage, userName string) (err error) {
	temp, err := template.New("links").Parse(templateString)
	if err != nil {
		return err
	}

	var (
		weekNumber  string
		currentWeek string
	)
	r := Data{
		UserName: userName,
	}
	weekItems := &storage.Storage{
		Title: s.Title,
	}
	for _, item := range s.Items {
		t, err := time.Parse(time.RFC3339, item.Published)
		if err != nil {
			return err
		}
		// новая неделя должна начинаться с 1:00 субботы
		year, week := t.Add(47 * time.Hour).ISOWeek()
		currentWeek = fmt.Sprintf("%d-%d", year, week)
		if weekNumber != currentWeek {
			if weekNumber != "" {
				writeTemplate(&r, weekNumber, weekItems, temp)
			}
			weekNumber = currentWeek
			weekItems.Items = []storage.StorageItem{}
		}
		weekItems.Items = append(weekItems.Items, item)
	}

	writeTemplate(&r, weekNumber, weekItems, temp)

	// Update README.md
	r.Count = len(s.Items)
	writeTemplate(&r, "", weekItems, temp)

	return nil
}

func writeTemplate(r *Data, weekNumber string, weekItems *storage.Storage, temp *template.Template) (err error) {
	r.Title = weekNumber
	r.Content = weekItems
	fileName := "data/" + weekNumber + ".md"

	if weekNumber == "" {
		r.Title = weekItems.Title
		fileName = "README.md"
	}

	var buffer strings.Builder
	if err = temp.Execute(&buffer, r); err != nil {
		return err
	}

	f, err := create(fileName)
	if err != nil {
		return fmt.Errorf("cannot create file `%s`: %v", fileName, err)
	}

	f.Write([]byte(buffer.String()))
	f.Close()

	return nil
}

func create(p string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(p), 0770); err != nil {
		return nil, err
	}
	return os.Create(p)
}

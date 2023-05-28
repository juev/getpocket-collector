package storage

import (
	_ "embed"
	"fmt"
	"os"
	"strings"
	"text/template"
)

//go:embed templates/template.tmpl
var templateString string

func (s *storage) TemplateFile() (err error) {
	userName := os.Getenv("USERNAME")
	if userName == "" {
		// if USERNAME is not setting, we use "juev" by default ;)
		userName = "juev"
	}

	temp := template.Must(template.New("links").Parse(templateString))

	r := struct {
		UserName string
		Content  Storage
	}{
		UserName: userName,
		Content:  s,
	}

	var buffer strings.Builder
	if err = temp.Execute(&buffer, r); err != nil {
		return err
	}

	f, err := os.Create("README.md")
	if err != nil {
		return fmt.Errorf("cannot create file `README.md`: %v", err)
	}

	f.Write([]byte(buffer.String()))
	f.Close()

	return nil
}

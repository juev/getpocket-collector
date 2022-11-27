package rss

import "time"

const wordpressDateFormat = "Mon, 02 Jan 2006 15:04:05 -0700"

type Date string

// Parse (Date function) and returns Time, error
func (d Date) Parse() (time.Time, error) {
	t, err := d.ParseWithFormat(wordpressDateFormat)
	if err != nil {
		t, err = d.ParseWithFormat(time.RFC822) // RSS 2.0 spec
		if err != nil {
			t, err = d.ParseWithFormat(time.RFC3339) // Atom
		}
	}
	return t, err
}

// ParseWithFormat (Date function), takes a string and returns Time, error
func (d Date) ParseWithFormat(format string) (time.Time, error) {
	return time.Parse(format, string(d))
}

// Format (Date function), takes a string and returns string, error
func (d Date) Format(format string) (string, error) {
	t, err := d.Parse()
	if err != nil {
		return "", err
	}
	return t.Format(format), nil
}

// MustFormat (Date function), take a string and returns string
func (d Date) MustFormat(format string) string {
	s, err := d.Format(format)
	if err != nil {
		return err.Error()
	}
	return s
}

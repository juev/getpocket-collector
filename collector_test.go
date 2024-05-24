package storage

import (
	"os"
	"testing"
)

func TestStorage_PocketParse(t *testing.T) {
	data, err := os.ReadFile("testdata/response.json")
	if err != nil {
		t.Errorf("cannot read file")
	}

	s := &Collector{}

	if err := s.PocketParse(data); err != nil {
		t.Errorf("Collector.PocketParse() error = %v", err)
	}
}

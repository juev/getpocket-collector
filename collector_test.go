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

	s := &Storage{}

	if err := s.PocketParse(data); err != nil {
		t.Errorf("Storage.PocketParse() error = %v", err)
	}
}

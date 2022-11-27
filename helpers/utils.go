package helpers

import (
	"fmt"
	"net/http"
	"os"
)

// ReadWithClient do GET request with client
func ReadWithClient(url string) (response *http.Response, err error) {
	var req *http.Request
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		return nil, err
	}

	req.Header.Set("user-agent", "feed:parser:v0.1 (by github.com/juev)")

	if response, err = http.DefaultClient.Do(req); err != nil {
		return nil, err
	}
	return response, nil
}

// Exit just exit with format message
func Exit(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

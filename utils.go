package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func readWithClient(url string) (response *http.Response, err error) {
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

func exit(format string, a ...any) {
	fmt.Printf(format, a...)
	os.Exit(1)
}

// normalizeURL should remove all utm from query
func normalizeURL(in string) string {
	u, err := url.Parse(in)
	if err != nil {
		return in
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

package client

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Response struct {
	StatusCode int
	Body       string
	Header     http.Header
	Err        error
}

func Request(request *http.Request) (*Response, error) {
	var response Response

	opt := func() error {
		return operation(request, &response)
	}

	if err := backoff.Retry(opt, backoff.NewExponentialBackOff()); err != nil {
		return nil, err
	}

	return &response, nil
}

func operation(request *http.Request, response *Response) error {
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	r, err := client.Do(request)
	if err != nil {
		response.Err = err
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer r.Body.Close()

	if r.StatusCode == http.StatusUnauthorized {
		err = fmt.Errorf("unauthorized")
		response.Err = err
		return backoff.Permanent(err)
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		response.Err = err
		return fmt.Errorf("failed to read response body: %w", err)
	}
	response.Body = string(bodyBytes)

	response.StatusCode = r.StatusCode
	response.Header = r.Header
	response.Err = nil

	return nil
}

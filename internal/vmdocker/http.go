package vmdocker

import (
	"io"
	"net/http"
	"time"
)

type HTTPGetter func(url string) (int, error)

func NewHTTPStatusGetter(client *http.Client) HTTPGetter {
	return func(url string) (int, error) {
		resp, err := client.Get(url)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	}
}

func DefaultHTTPGetter() HTTPGetter {
	client := &http.Client{Timeout: 2 * time.Second}
	return NewHTTPStatusGetter(client)
}

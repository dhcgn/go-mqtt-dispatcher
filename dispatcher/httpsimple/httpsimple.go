package httpsimple

import (
	"fmt"
	"io"
	"net/http"
)

var (
	HttpGetOverrideForTesting = func(url string) (resp *http.Response, err error) {
		return http.Get(url)
	}
)

func GetHttpPayload(url string) ([]byte, error) {
	resp, err := HttpGetOverrideForTesting(url)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return body, nil
}

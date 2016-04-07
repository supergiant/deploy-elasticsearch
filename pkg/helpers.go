package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func httpRequest(baseURL string, verb string, path string, body []byte) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", baseURL, path)
	req, err := http.NewRequest(verb, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	resp, err := new(http.Client).Do(req)
	if err != nil {
		return nil, err
	}
	if status := resp.Status; status[:2] != "20" {
		return nil, fmt.Errorf("Request failed with status %s", status)
	}
	return resp, nil
}

func postBody(verb string, url string, path string, in interface{}) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	_, err = httpRequest(url, verb, path, body)
	return err
}

func waitFor(t time.Duration, i time.Duration, fn func(d time.Duration) (bool, error)) error {
	start := time.Now()
	for {
		elapsed := time.Since(start)

		if ok, err := fn(elapsed); err != nil {
			return err
		} else if ok {
			return nil
		}

		if elapsed < t {
			time.Sleep(i)
		} else {
			return errors.New("Timed out waiting")
		}
	}
}

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	serverURL  = "http://127.0.0.1:8080"
	workers    = 5
	iterations = 50
)

var httpClient = &http.Client{Timeout: 5 * time.Second}

func main() {
	shortIDs := make([]string, 0, 50)
	for i := range 50 {
		id := createURL(fmt.Sprintf("https://example.com/seed/%d/path", i))
		if id != "" {
			shortIDs = append(shortIDs, id)
		}
	}

	var wg sync.WaitGroup
	var errors atomic.Int64

	for w := range workers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := range iterations {
				switch (workerID + i) % 3 {
				case 0:
					createURL(fmt.Sprintf("https://example.com/worker/%d/%d", workerID, i))
				case 1:
					if len(shortIDs) > 0 {
						followURL(shortIDs[(workerID+i)%len(shortIDs)])
					}
				default:
					createURL(fmt.Sprintf("https://load.test/%d/%d", workerID, i))
				}
			}
		}(w)
	}

	wg.Wait()
	fmt.Printf("load finished, errors: %d\n", errors.Load())
}

func createURL(original string) string {
	resp, err := httpClient.Post(serverURL+"/", "text/plain", bytes.NewBufferString(original))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return ""
	}
	shortURL := string(body)
	if len(shortURL) > len(serverURL)+1 {
		return shortURL[len(serverURL)+1:]
	}
	return ""
}

func followURL(id string) {
	resp, err := httpClient.Get(serverURL + "/" + id)
	if err != nil {
		return
	}
	resp.Body.Close()
}

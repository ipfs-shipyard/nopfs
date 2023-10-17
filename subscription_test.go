package nopfs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func createTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate serving a file from the server
		data := []byte("This is the remote content.")
		contentLength := len(data)

		// Get the "Range" header from the request
		rangeHeader := r.Header.Get("Range")

		if rangeHeader != "" {
			var start, end int

			// Check for a range request in the format "bytes %d-"
			if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-", &start); n == 1 {
				// Handle open end range requests
				if start >= contentLength {
					http.Error(w, "Invalid Range header", http.StatusRequestedRangeNotSatisfiable)
					return
				}
				end = contentLength - 1
			} else if n, _ := fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end); n == 2 {
				// Check for valid byte range
				if start < 0 || end >= contentLength || start > end {
					http.Error(w, "Invalid Range header", http.StatusRequestedRangeNotSatisfiable)
					return
				}
			} else {
				http.Error(w, "Invalid Range header", http.StatusBadRequest)
				return
			}

			// Calculate the content range and length for the response
			contentRange := fmt.Sprintf("bytes %d-%d/%d", start, end, contentLength)
			w.Header().Set("Content-Range", contentRange)
			w.Header().Set("Content-Length", fmt.Sprint(end-start+1))
			w.WriteHeader(http.StatusPartialContent)

			// Write the selected byte range to the response
			_, _ = w.Write(data[start : end+1])
		} else {
			// If no "Range" header, serve the entire content
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", contentLength-1, contentLength))
			w.Header().Set("Content-Length", fmt.Sprint(contentLength))
			w.WriteHeader(http.StatusPartialContent)

			_, _ = w.Write(data)
		}
	}))
}

func TestHTTPSubscriber(t *testing.T) {
	remoteServer := createTestServer()
	defer remoteServer.Close()

	localFile := "test-local-file.txt"
	defer os.Remove(localFile)

	subscriber, err := NewHTTPSubscriber(remoteServer.URL, localFile, 500*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	// Allow some time for subscription to run
	time.Sleep(2 * time.Second)
	subscriber.Stop()

	localFileContent, err := os.ReadFile(localFile)
	if err != nil {
		t.Errorf("Error reading local file: %v", err)
	}

	expectedContent := "This is the remote content."
	if string(localFileContent) != expectedContent {
		t.Errorf("Local file content is incorrect. Got: %s, Expected: %s", string(localFileContent), expectedContent)
	}
}

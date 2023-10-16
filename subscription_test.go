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
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(data)-1, len(data)))
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(data)
	}))
}

func TestHTTPSubscriber(t *testing.T) {
	remoteServer := createTestServer()
	defer remoteServer.Close()

	localFile := "test-local-file.txt"
	defer os.Remove(localFile)

	subscriber := NewHTTPSubscriber(remoteServer.URL, localFile, 500*time.Millisecond)
	go subscriber.Subscribe()

	// Allow some time for subscription to run
	time.Sleep(time.Second)
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

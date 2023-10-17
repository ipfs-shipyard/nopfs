package nopfs

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// HTTPSubscriber represents a type that subscribes to a remote URL and appends data to a local file.
type HTTPSubscriber struct {
	remoteURL   string
	localFile   string
	interval    time.Duration
	stopChannel chan struct{}
}

// NewHTTPSubscriber creates a new Subscriber instance with the given parameters.
func NewHTTPSubscriber(remoteURL, localFile string, interval time.Duration) (*HTTPSubscriber, error) {
	logger.Infof("Subscribing to remote denylist: %s", remoteURL)
	f, err := os.OpenFile(localFile, os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return &HTTPSubscriber{
		remoteURL:   remoteURL,
		localFile:   localFile,
		interval:    interval,
		stopChannel: make(chan struct{}, 1),
	}, nil
}

// Subscribe starts the subscription process.
func (s *HTTPSubscriber) Subscribe() {
	timer := time.NewTimer(0)

	for {
		select {
		case <-s.stopChannel:
			logger.Infof("Stopping subscription on: %s", s.localFile)
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			s.downloadAndAppend()
			timer.Reset(s.interval)
		}
	}
}

// Stop stops the subscription process.
func (s *HTTPSubscriber) Stop() {
	s.stopChannel <- struct{}{}
}

func (s *HTTPSubscriber) downloadAndAppend() {
	localFile, err := os.OpenFile(s.localFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logger.Error(err)
	}
	defer localFile.Close()

	// Get the file size of the local file
	localFileInfo, err := localFile.Stat()
	if err != nil {
		logger.Error(err)
	}

	localFileSize := localFileInfo.Size()

	// Create a HTTP GET request with the Range header to download only the missing bytes
	req, err := http.NewRequest("GET", s.remoteURL, nil)
	if err != nil {
		logger.Error(err)
	}

	rangeHeader := fmt.Sprintf("bytes=%d-", localFileSize)
	req.Header.Set("Range", rangeHeader)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusPartialContent:
		_, err = io.Copy(localFile, resp.Body)
		if err != nil {
			logger.Error(err)
		}
		logger.Infof("%s: appended %d bytes", s.localFile, resp.ContentLength)
	case (resp.StatusCode >= http.StatusBadRequest &&
		resp.StatusCode != http.StatusRequestedRangeNotSatisfiable) ||
		resp.StatusCode >= http.StatusInternalServerError:
		logger.Errorf("%s: server returned with unexpected code %d", s.localFile, resp.StatusCode)
		// error is ignored, we continued subscribed
	}
}

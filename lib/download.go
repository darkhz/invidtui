package lib

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
)

var downloadLock sync.Mutex

// GetDownload gets the video's response body and the file name to be saved to.
func GetDownload(id, itag, filename string, ctx context.Context) (*http.Response, *os.File, error) {
	var authToken []string

	token := GetToken()
	if token != "" {
		authToken = append(authToken, token)
	}

	url, _ := url.Parse(getLatestURL(id, itag))
	param := url.RequestURI()
	client := &Client{
		host: GetClient().host,
		client: &http.Client{
			Transport: http.DefaultTransport,
		},
	}

	res, err := client.GetRequest(ctx, param, authToken...)
	if err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(filepath.Join(DownloadFolder(), filename), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	return res, file, err
}

// DownloadFolder returns the download directory.
func DownloadFolder() string {
	downloadLock.Lock()
	defer downloadLock.Unlock()

	return downloadFolder
}

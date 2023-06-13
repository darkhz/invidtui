package invidious

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
)

// DownloadParams returns parameters that are used to download a file.
func DownloadParams(ctx context.Context, id, itag, filename string) (*http.Response, *os.File, error) {
	dir := cmd.GetOptionValue("download-dir")

	uri, err := url.Parse(getLatestURL(id, itag))
	if err != nil {
		return nil, nil, fmt.Errorf("Video: Cannot parse download URL")
	}

	res, err := client.Get(ctx, uri.RequestURI(), client.Token())
	if err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(filepath.Join(dir, filename), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, err
	}

	return res, file, err
}

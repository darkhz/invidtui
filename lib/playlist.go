package lib

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
)

// PlaylistResult stores the playlist data.
type PlaylistResult struct {
	Title       string          `json:"title"`
	PlaylistID  string          `json:"playlistId"`
	Author      string          `json:"author"`
	Description string          `json:"description"`
	VideoCount  int             `json:"videoCount"`
	ViewCount   int             `json:"viewCount"`
	Videos      []PlaylistVideo `json:"videos"`
}

// PlaylistVideo stores the playlist's video data.
type PlaylistVideo struct {
	Title         string `json:"title"`
	VideoID       string `json:"videoId"`
	Author        string `json:"author"`
	LengthSeconds int    `json:"lengthSeconds"`
}

var (
	plistid    string
	plistpage  int
	plistMutex sync.Mutex

	// PlistCtx is used here and the UI playlist code
	// to detect if the user tried to cancel the playlist
	// loading.
	PlistCtx context.Context

	// PlistCancel is used to cancel the playlist loading.
	PlistCancel context.CancelFunc
)

const playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos"

// Playlist gets the playlist with the given ID and returns a PlaylistResult.
// If id is blank, it indicates that more results are to be loaded for the
// same playlist ID (stored in plistid). When cancel is true, it will stop loading
// the playlist.
func (c *Client) Playlist(id string, cancel bool) (PlaylistResult, error) {
	var result PlaylistResult

	if c == nil {
		return PlaylistResult{}, nil
	}

	if PlistCtx != nil {
		PlistCancel()

		if cancel {
			return PlaylistResult{}, nil
		}
	}

	if id == "" {
		incPlistPage()
	} else {
		setPlistPage(1)
		plistid = id
	}

	PlistCtx, PlistCancel = context.WithCancel(context.Background())

	query := "playlists/" + plistid + playlistFields + "&page=" + getPlistPage()

	res, err := c.ClientRequest(PlistCtx, query)
	if err != nil {
		return PlaylistResult{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return PlaylistResult{}, err
	}

	return result, nil
}

// LoadPlaylist takes a playlist ID, determines whether to play
// video or just audio (according to the audio parameter), and
// appropriately loads the URLs into mpv.
func LoadPlaylist(id string, audio bool) error {
	var err error

	playlist, err := GetClient().Playlist(id, false)
	if err != nil {
		return err
	}

	for _, p := range playlist.Videos {
		select {
		case <-PlistCtx.Done():
			return nil

		default:
		}

		LoadVideo(p.VideoID, audio)
	}

	return nil
}

func getPlistPage() string {
	pageMutex.Lock()
	defer pageMutex.Unlock()

	return strconv.Itoa(plistpage)
}

func setPlistPage(pg int) {
	pageMutex.Lock()
	defer pageMutex.Unlock()

	plistpage = pg
}

func incPlistPage() {
	pageMutex.Lock()
	defer pageMutex.Unlock()

	plistpage++
}

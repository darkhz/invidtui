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
	AuthorID      string `json:"authorId"`
	LengthSeconds int    `json:"lengthSeconds"`
}

var (
	plistid     string
	plistpage   int
	plistMutex  sync.Mutex
	plistCtx    context.Context
	plistCancel context.CancelFunc
)

const playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos"

// Playlist gets the playlist with the given ID and returns a PlaylistResult.
// If id is blank, it indicates that more results are to be loaded for the
// same playlist ID (stored in plistid). When cancel is true, it will stop loading
// the playlist.
func (c *Client) Playlist(id string) (PlaylistResult, error) {
	var result PlaylistResult

	if c == nil {
		return PlaylistResult{}, nil
	}

	PlaylistCancel()

	if id == "" {
		incPlistPage()
	} else {
		setPlistPage(1)
		plistid = id
	}

	query := "playlists/" + plistid + playlistFields + "&page=" + getPlistPage()

	res, err := c.ClientRequest(plistCtx, query)
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

	playlist, err := GetClient().Playlist(id)
	if err != nil {
		return err
	}

	for _, p := range playlist.Videos {
		select {
		case <-videoCtx.Done():
			return videoCtx.Err()

		default:
		}

		LoadVideo(p.VideoID, audio)
	}

	return nil
}

// PlaylistCtx returns the playlist context.
func PlaylistCtx() context.Context {
	return plistCtx
}

// PlaylistCancel cancels and renews the playlist context.
func PlaylistCancel() {
	if plistCtx != nil {
		plistCancel()
	}

	plistCtx, plistCancel = context.WithCancel(context.Background())
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

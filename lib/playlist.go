package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
)

// PlaylistResult stores the playlist data.
type PlaylistResult struct {
	Title       string          `json:"title"`
	PlaylistID  string          `json:"playlistId"`
	Author      string          `json:"author"`
	AuthorID    string          `json:"authorId"`
	Description string          `json:"description"`
	VideoCount  int             `json:"videoCount"`
	ViewCount   int64           `json:"viewCount"`
	Videos      []PlaylistVideo `json:"videos"`
}

// PlaylistVideo stores the playlist's video data.
type PlaylistVideo struct {
	Title         string `json:"title"`
	VideoID       string `json:"videoId"`
	Author        string `json:"author"`
	AuthorID      string `json:"authorId"`
	IndexID       string `json:"indexId"`
	LengthSeconds int64  `json:"lengthSeconds"`
}

var (
	plistid    string
	plistpage  int
	plistMutex sync.Mutex
)

const playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos&hl=en"

// Playlist gets the playlist with the given ID and returns a PlaylistResult.
// If id is blank, it indicates that more results are to be loaded for the
// same playlist ID (stored in plistid). If auth is true, it will load playlists
// with an authorization token.
func (c *Client) Playlist(id string, auth bool) (PlaylistResult, error) {
	var authToken []string
	var result PlaylistResult

	if c == nil {
		return PlaylistResult{}, nil
	}

	if id == "" {
		incPlistPage()
	} else {
		setPlistPage(1)
		plistid = id
	}

	query := "playlists/" + plistid + playlistFields + "&page=" + getPlistPage()
	if auth {
		query = "auth/" + query
		authToken = append(authToken, GetToken())
	}

	res, err := c.ClientRequest(PlaylistCtx(), query, authToken...)
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

// AuthPlaylists lists all playlists associated with an authorization token.
func (c *Client) AuthPlaylists() ([]PlaylistResult, error) {
	var result []PlaylistResult

	res, err := c.ClientRequest(PlaylistCtx(), "auth/playlists/", GetToken())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return nil, err
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
		case <-videoCtx.Done():
			return videoCtx.Err()

		default:
		}

		LoadVideo(p.VideoID, audio)
	}

	return nil
}

// CreatePlaylist creates a new playlist.
func (c *Client) CreatePlaylist(title, privacy string) error {
	createFormat := fmt.Sprintf(
		`{"title": "%s", "privacy": "%s"}`,
		title, privacy,
	)
	_, err := c.ClientSend("auth/playlists/", createFormat, GetToken())

	return err
}

// EditPlaylist edits a playlist's properties.
func (c *Client) EditPlaylist(id, title, description, privacy string) error {
	editFormat := fmt.Sprintf(
		`{"title": "%s", "description": "%s", "privacy": "%s"}`,
		title, description, privacy,
	)
	_, err := c.ClientPatch("auth/playlists/"+id, editFormat, GetToken())

	return err
}

// RemovePlaylist removes a playlist.
func (c *Client) RemovePlaylist(id string) error {
	_, err := c.ClientDelete("auth/playlists/"+id, GetToken())

	return err
}

// AddPlaylistVideo adds a video to the playlist.
func (c *Client) AddPlaylistVideo(plid, videoId string) error {
	videoFormat := fmt.Sprintf(`{"videoId":"%s"}`, videoId)
	_, err := c.ClientSend("auth/playlists/"+plid+"/videos", videoFormat, GetToken())

	return err
}

// RemovePlaylistVideo removes a video from the playlist.
func (c *Client) RemovePlaylistVideo(plid, index string) error {
	_, err := c.ClientDelete("auth/playlists/"+plid+"/videos/"+index, GetToken())

	return err
}

// PlaylistCtx returns the playlist context.
func PlaylistCtx() context.Context {
	return ClientCtx()
}

// PlaylistCancel cancels and renews the playlist context.
func PlaylistCancel() {
	ClientCancel()
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

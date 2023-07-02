package invidious

import (
	"fmt"
	"strconv"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/utils"
)

const playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos&hl=en"

// PlaylistData stores information about a playlist.
type PlaylistData struct {
	Title       string          `json:"title"`
	PlaylistID  string          `json:"playlistId"`
	Author      string          `json:"author"`
	AuthorID    string          `json:"authorId"`
	Description string          `json:"description"`
	VideoCount  int             `json:"videoCount"`
	ViewCount   int64           `json:"viewCount"`
	Videos      []PlaylistVideo `json:"videos"`
}

// PlaylistVideo stores information about a video in the playlist.
type PlaylistVideo struct {
	Title         string `json:"title"`
	VideoID       string `json:"videoId"`
	Author        string `json:"author"`
	AuthorID      string `json:"authorId"`
	IndexID       string `json:"indexId"`
	LengthSeconds int64  `json:"lengthSeconds"`
}

// Playlist retrieves a playlist and its videos.
func Playlist(id string, auth bool, page int) (PlaylistData, error) {
	var data PlaylistData

	query := "playlists/" + id + playlistFields + "&page=" + strconv.Itoa(page)
	if auth {
		query = "auth/" + query
	}

	res, err := client.Fetch(client.Ctx(), query, client.Token())
	if err != nil {
		return PlaylistData{}, err
	}
	defer res.Body.Close()

	err = utils.JSON().NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return PlaylistData{}, err
	}

	return data, nil
}

// UserPlaylists retrieves the user's playlists.
func UserPlaylists() ([]PlaylistData, error) {
	var data []PlaylistData

	res, err := client.Fetch(client.Ctx(), "auth/playlists/", client.Token())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	err = utils.JSON().NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// CreatePlaylist creates a playlist for the user.
func CreatePlaylist(title, privacy string) error {
	createFormat := fmt.Sprintf(
		`{"title": "%s", "privacy": "%s"}`,
		title, privacy,
	)
	_, err := client.Send("auth/playlists/", createFormat, client.Token())

	return err
}

// EditPlaylist edits a user's playlist properties.
func EditPlaylist(id, title, description, privacy string) error {
	editFormat := fmt.Sprintf(
		`{"title": "%s", "description": "%s", "privacy": "%s"}`,
		title, description, privacy,
	)
	_, err := client.Modify("auth/playlists/"+id, editFormat, client.Token())

	return err
}

// RemovePlaylist removes a user's playlist.
func RemovePlaylist(id string) error {
	_, err := client.Remove("auth/playlists/"+id, client.Token())

	return err
}

// AddVideoToPlaylist adds a video to the user's playlist.
func AddVideoToPlaylist(plid, videoID string) error {
	videoFormat := fmt.Sprintf(`{"videoId":"%s"}`, videoID)
	_, err := client.Send("auth/playlists/"+plid+"/videos", videoFormat, client.Token())

	return err
}

// RemoveVideoFromPlaylist removes a video from the user's  playlist.
func RemoveVideoFromPlaylist(plid, index string) error {
	_, err := client.Remove("auth/playlists/"+plid+"/videos/"+index, client.Token())

	return err
}

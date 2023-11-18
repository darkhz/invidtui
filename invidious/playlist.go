package invidious

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/utils"
	"github.com/goccy/go-json"
)

const playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos&hl=en"

// PlaylistData stores information about a playlist.
type PlaylistData struct {
	Title       string          `json:"title"`
	PlaylistID  string          `json:"playlistId"`
	Author      string          `json:"author"`
	AuthorID    string          `json:"authorId"`
	Description string          `json:"description"`
	VideoCount  int64           `json:"videoCount"`
	ViewCount   int64           `json:"viewCount"`
	Videos      []PlaylistVideo `json:"videos"`
}

// PlaylistVideo stores information about a video in the playlist.
type PlaylistVideo struct {
	Title         string `json:"title"`
	Author        string `json:"author"`
	Index         int32  `json:"index"`
	IndexID       string `json:"indexId"`
	VideoID       string `json:"videoId"`
	AuthorID      string `json:"authorId"`
	LengthSeconds int64  `json:"lengthSeconds"`
}

// Playlist retrieves a playlist and its videos.
func Playlist(id string, auth bool, page int, ctx ...context.Context) (PlaylistData, error) {
	var data PlaylistData

	query := "playlists/" + id + "?page=" + strconv.Itoa(page)
	if auth {
		query = "auth/" + query
	}

	fetchCtx := client.Ctx()
	if ctx != nil {
		fetchCtx = ctx[0]
	}

	res, err := client.Fetch(fetchCtx, query, client.Token())
	if err != nil {
		return PlaylistData{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)
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

	err = json.NewDecoder(res.Body).Decode(&data)
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

// GeneratePlaylist generates a playlist file.
func GeneratePlaylist(file string, list []VideoData, appendToFile bool) (string, error) {
	var skipped int
	var entries string
	var fileEntries map[string]struct{}

	if len(list) == 0 {
		return "", fmt.Errorf("Playlist Generator: No videos found")
	}

	if appendToFile {
		fileEntries = make(map[string]struct{})

		existingFile, err := os.Open(file)
		if err != nil {
			return "", fmt.Errorf("Playlist Generator: Unable to open playlist")
		}

		scanner := bufio.NewScanner(existingFile)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			fileEntries[line] = struct{}{}
		}
	}

	if !appendToFile {
		entries += "#EXTM3U\n\n"
		entries += "# Autogenerated by invidtui. DO NOT EDIT.\n\n"
	} else {
		entries += "\n"
	}

	for i, data := range list {
		murl, err := encodeVideoURI(getLatestURL(data.VideoID, ""), utils.FormatDuration(data.LengthSeconds), data)
		if err != nil {
			continue
		}

		filename := murl.String()

		if appendToFile && fileEntries != nil {
			if _, ok := fileEntries[filename]; ok {
				skipped++
				continue
			}
		}

		entries += "#EXTINF:," + data.Title + "\n"
		entries += filename + "\n"

		if i != len(list)-1 {
			entries += "\n"
		}
	}

	if skipped == len(list) {
		return "", fmt.Errorf("Playlist Generator: No new items in playlist to append")
	}

	return entries, nil
}

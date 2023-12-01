package invidious

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/resolver"
	"github.com/darkhz/invidtui/utils"
	"github.com/etherlabsio/go-m3u8/m3u8"
)

const (
	PlaylistEntryPrefix = "invidtui.video."

	playlistFields = "?fields=title,playlistId,author,description,videoCount,viewCount,videos&hl=en"
)

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

	err = resolver.DecodeJSONReader(res.Body, &data)
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

	err = resolver.DecodeJSONReader(res.Body, &data)
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
//
//gocyclo:ignore
func GeneratePlaylist(file string, list []VideoData, flags int, appendToFile bool) (string, int, error) {
	var skipped int
	var ignored []m3u8.Item
	var fileEntries map[string]struct{}

	if len(list) == 0 {
		return "", flags, fmt.Errorf("Playlist Generator: No videos found")
	}

	playlist := m3u8.NewPlaylist()

	flags |= os.O_TRUNC
	if (flags & os.O_APPEND) != 0 {
		flags ^= os.O_APPEND
	}

	if appendToFile {
		fileEntries = make(map[string]struct{})

		existing, err := m3u8.ReadFile(file)
		if err != nil {
			return "", flags, err
		}

		for _, e := range existing.Items {
			var id string
			var item m3u8.Item

			add := true

			switch v := e.(type) {
			case *m3u8.SessionDataItem:
				if v.DataID == "" || !strings.HasPrefix(v.DataID, PlaylistEntryPrefix) {
					continue
				}

				utils.DecodeSessionData(*v.Value, func(prop, value string) {
					switch prop {
					case "id":
						id = value

					case "authorId":
						if value == "" {
							add = false
							ignored = append(ignored, v)
						}
					}
				})

				item = v

			case *m3u8.SegmentItem:
				if strings.HasPrefix(v.Segment, "#") {
					add = false
					ignored = append(ignored, v)
				}

				segment := strings.TrimPrefix(v.Segment, "#")
				uri, err := utils.IsValidURL(segment)
				if err != nil {
					continue
				}

				id = uri.Query().Get("id")
				if id == "" {
					id, _ = CheckLiveURL(segment, true)
				}

				item = v
			}

			if add && item != nil {
				playlist.Items = append(playlist.Items, item)
			}

			if id != "" {
				fileEntries[id] = struct{}{}
			}
		}
	}

	for _, data := range list {
		var filename, length string

		if data.VideoID == "" {
			continue
		}

		if appendToFile && fileEntries != nil {
			if _, ok := fileEntries[data.VideoID]; ok {
				skipped++
				continue
			}
		}

		if data.LiveNow {
			filename = data.HlsURL
			length = "Live"
		} else {
			filename = getLatestURL(data.VideoID, "")
			length = utils.FormatDuration(data.LengthSeconds)
		}

		if data.MediaType == "" {
			data.MediaType = "Audio"
		}

		value := fmt.Sprintf(
			"id=%s,title=%s,author=%s,authorId=%s,length=%s,mediatype=%s",
			data.VideoID, url.QueryEscape(data.Title),
			url.QueryEscape(data.Author), data.AuthorID, length,
			data.MediaType,
		)
		comment := fmt.Sprintf(
			"%s - %s",
			data.Title, data.Author,
		)

		session := m3u8.SessionDataItem{
			DataID: PlaylistEntryPrefix + data.VideoID,
			Value:  &value,
			URI:    &filename,
		}
		segment := m3u8.SegmentItem{
			Duration: float64(data.LengthSeconds),
			Segment:  filename,
			Comment:  &comment,
		}

		if data.Author == "" && data.AuthorID == "" {
			segment.Segment = "# " + filename
			ignored = append(ignored, []m3u8.Item{&session, &segment}...)
			continue
		}

		playlist.Items = append(playlist.Items, []m3u8.Item{&session, &segment}...)
	}
	if ignored != nil {
		playlist.Items = append(playlist.Items, ignored...)
	}

	if appendToFile && skipped == len(list) {
		return "", flags, fmt.Errorf("Playlist Generator: No new items in playlist to append")
	}

	return playlist.String(), flags, nil
}

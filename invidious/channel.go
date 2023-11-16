package invidious

import (
	"github.com/darkhz/invidtui/client"

	"github.com/goccy/go-json"
)

const channelFields = "?fields=title,authorId,author,description,viewCount&hl=en"

// ChannelData stores channel related data.
type ChannelData struct {
	Title        string          `json:"title"`
	ChannelID    string          `json:"authorId"`
	Author       string          `json:"author"`
	Description  string          `json:"description"`
	ViewCount    int64           `json:"viewCount"`
	Continuation string          `json:"continuation"`
	Videos       []PlaylistVideo `json:"videos"`
	Playlists    []PlaylistData  `json:"playlists"`
}

// Channel retrieves information about a channel.
func Channel(id, stype, params string, channel ...ChannelData) (ChannelData, error) {
	var err error
	var query string
	var data ChannelData

	client.Cancel()

	if channel != nil {
		data = channel[0]
		goto GetData
	}

	query = "channels/" + id + channelFields

	// Get the channel data first.
	data, err = decodeChannelData(query)
	if err != nil {
		return ChannelData{}, err
	}

GetData:
	// Then get the data associated with the provided channel type (stype).
	query = "channels/" + id + "/" + stype + params

	d, err := decodeChannelData(query)
	if err != nil {
		return ChannelData{}, err
	}

	data.Videos = d.Videos
	data.Playlists = d.Playlists
	data.Continuation = d.Continuation

	return data, nil
}

// ChannelVideos retrieves video information from a channel.
func ChannelVideos(id, continuation string) (ChannelData, error) {
	params := "?fields=videos,continuation"
	if continuation != "" {
		params += "&continuation=" + continuation
	}

	return Channel(id, "videos", params)
}

// ChannelPlaylists loads only the playlists present in the channel.
func ChannelPlaylists(id, continuation string) (ChannelData, error) {
	params := "?fields=playlists,continuation"
	if continuation != "" {
		params += "&continuation=" + continuation
	}

	return Channel(id, "playlists", params)
}

// ChannelSearch searches for a query string in the channel.
func ChannelSearch(id, searchText string, page int) ([]SearchData, int, error) {
	return Search("channel", searchText, nil, page, id)
}

// decodeChannelData sends a channel query, parses and returns the response.
func decodeChannelData(query string) (ChannelData, error) {
	var data ChannelData

	res, err := client.Fetch(client.Ctx(), query)
	if err != nil {
		return ChannelData{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)

	return data, err
}

package invidious

import (
	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/resolver"
)

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
func Channel(id, stype, continuation string, channel ...ChannelData) (ChannelData, error) {
	var err error
	var query string
	var data ChannelData

	client.Cancel()

	if channel != nil {
		data = channel[0]
		goto GetData
	}

	query = "channels/" + id

	// Get the channel data first.
	data, err = decodeChannelData(query)
	if err != nil {
		return ChannelData{}, err
	}

GetData:
	// Then get the data associated with the provided channel type (stype).
	query = "channels/" + id + "/" + stype
	if continuation != "" {
		query += "?continuation=" + continuation
	}

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
	return Channel(id, "videos", continuation)
}

// ChannelPlaylists loads only the playlists present in the channel.
func ChannelPlaylists(id, continuation string) (ChannelData, error) {
	return Channel(id, "playlists", continuation)
}

// ChannelReleases loads only the releases present in the channel.
func ChannelReleases(id, continuation string) (ChannelData, error) {
	return Channel(id, "releases", continuation)
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

	err = resolver.DecodeJSONReader(res.Body, &data)

	return data, err
}

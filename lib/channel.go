package lib

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
)

// ChannelResult stores the channel data.
type ChannelResult struct {
	Title       string           `json:"title"`
	ChannelID   string           `json:"authorId"`
	Author      string           `json:"author"`
	Description string           `json:"description"`
	ViewCount   int              `json:"viewCount"`
	Videos      []PlaylistVideo  `json:"videos"`
	Playlists   []PlaylistResult `json:"playlists"`
}

var (
	chanpage  int
	chanid    string
	chantype  string
	chanMutex sync.Mutex
)

const channelFields = "?fields=title,authorId,author,description,viewCount"

// Channel gets the playlist with the given ID and returns a ChannelResult.
// If id is blank, it indicates that more results are to be loaded for the
// same channel ID (stored in plistid). When cancel is true, it will stop loading
// the channel.
func (c *Client) Channel(id, stype, params string, cancel bool) (ChannelResult, error) {
	var result ChannelResult

	if c == nil {
		return ChannelResult{}, nil
	}

	if PlistCtx != nil {
		PlistCancel()

		if cancel {
			return ChannelResult{}, nil
		}
	}

	// We use the same context as the Playlist because only one of
	// either Playlist or Channel is supposed to load at a time. We
	// do not want both of them to load separately/simultaneously,
	// since only one screen is shown (the channel screen or the playlist screen).
	// For example, if a user loads a channel, and then immediately
	// attempts to load the playlist, there is no point in completely
	// loading the channel contents, since the user wants to view the playlist
	// contents immediately.
	PlistCtx, PlistCancel = context.WithCancel(context.Background())

	if id != "" {
		chanid = id
		chantype = stype

		query := "channels/" + chanid + channelFields

		res, err := c.chandecode(query, "channels")
		if err != nil {
			return ChannelResult{}, err
		}

		result = res.(ChannelResult)
	}

	query := "channels/" + chanid + "/" + chantype + params

	res, err := c.chandecode(query, chantype)
	if err != nil {
		return ChannelResult{}, err
	}

	switch chantype {
	case "videos":
		result.Videos = append(result.Videos, res.([]PlaylistVideo)...)

	case "playlists":
		result.Playlists = append(result.Playlists, res.([]PlaylistResult)...)
	}

	return result, nil
}

// chandecode sends a request along with the query parameter, and decodes
// the response into the appropriate dectype (videos, playlists, channels).
func (c *Client) chandecode(query, dectype string) (interface{}, error) {
	var ret interface{}
	var vres []PlaylistVideo
	var pres, cres ChannelResult

	res, err := c.ClientRequest(PlistCtx, query)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	switch dectype {
	case "videos":
		err = json.NewDecoder(res.Body).Decode(&vres)
		ret = vres

	case "playlists":
		err = json.NewDecoder(res.Body).Decode(&pres)
		ret = pres.Playlists

	case "channels":
		err = json.NewDecoder(res.Body).Decode(&cres)
		ret = cres
	}
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// ChannelVideos loads only the videos present in the channel.
func (c *Client) ChannelVideos(id string, cancel bool) (ChannelResult, error) {
	if id == "" {
		incChanPage()
	} else {
		setChanPage(1)
	}

	return c.Channel(id, "videos", videoFields+"&page="+getChanPage(), cancel)
}

// ChannelPlaylists loads only the playlists present in the channel.
func (c *Client) ChannelPlaylists(id string, cancel bool) (ChannelResult, error) {
	return c.Channel(id, "playlists", "?fields=playlists", cancel)
}

func getChanPage() string {
	chanMutex.Lock()
	defer chanMutex.Unlock()

	return strconv.Itoa(chanpage)
}

func setChanPage(pg int) {
	chanMutex.Lock()
	defer chanMutex.Unlock()

	chanpage = pg
}

func incChanPage() {
	chanMutex.Lock()
	defer chanMutex.Unlock()

	chanpage++
}
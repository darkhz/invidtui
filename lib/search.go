package lib

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"sync"
)

// SearchResult stores the search result data.
type SearchResult struct {
	Type          string `json: "type"`
	Title         string `json: "title"`
	AuthorID      string `json: "authorId"`
	VideoID       string `json: "videoId"`
	PlaylistID    string `json: "playlistId"`
	Author        string `json: "author"`
	PublishedText string `json: "publishedText"`
	Description   string `json: "description"`
	VideoCount    int    `json: "videoCount"`
	SubCount      int    `json: "subCount"`
	LengthSeconds int    `json: "lengthSeconds"`
	LiveNow       bool   `json: "liveNow"`
}

var (
	page         int
	pageMutex    sync.Mutex
	searchCtx    context.Context
	searchCancel context.CancelFunc
)

const searchField = "&fields=type,title,videoId,playlistId,author,authorId,publishedText,description,videoCount,subCount,lengthSeconds,videos,liveNow"

// Search searches for the given string and returns a SearchResult slice.
// It queries for two pages of results, and keeps a track of the number of
// pages currently returned. If the getmore parameter is true, it will add
// two more pages to the already tracked page number, and return the result.
func (c *Client) Search(stype, text string, getmore bool, chanid ...string) ([]SearchResult, error) {
	var oldpg, newpg int
	var results []SearchResult

	setpg := func(i int) {
		if chanid != nil {
			setChanPage(i, true)
		} else {
			setPage(i)
		}
	}

	getpg := func() int {
		if chanid != nil {
			return getChanPage(true)
		}

		return getPage()
	}

	if searchCtx != nil {
		searchCancel()
	}

	if !getmore {
		setpg(0)
	} else {
		oldpg = getpg()
	}

	searchCtx, searchCancel = context.WithCancel(context.Background())

	for newpg = oldpg + 1; newpg <= oldpg+2; newpg++ {
		var s []SearchResult

		query := "?q=" + url.QueryEscape(text) + searchField +
			"&page=" + strconv.Itoa(newpg)

		if chanid != nil {
			query = "channels/search/" + chanid[0] + query
		} else {
			query = "search" + query + "&type=" + stype
		}

		res, err := c.ClientRequest(searchCtx, query)
		if err != nil {
			return nil, err
		}

		err = json.NewDecoder(res.Body).Decode(&s)
		if err != nil {
			return nil, err
		}

		results = append(results, s...)

		res.Body.Close()
	}

	setpg(newpg)

	return results, nil
}

func getPage() int {
	pageMutex.Lock()
	defer pageMutex.Unlock()

	return page
}

func setPage(pg int) {
	pageMutex.Lock()
	defer pageMutex.Unlock()

	page = pg
}

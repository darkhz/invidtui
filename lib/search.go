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
	VideoCount    int    `json: "videoCount"`
	LengthSeconds int    `json: "lengthSeconds"`
}

var (
	page         int
	pageMutex    sync.Mutex
	searchCtx    context.Context
	searchCancel context.CancelFunc
)

const searchField = "&fields=type,title,videoId,playlistId,author,authorId,publishedText,videoCount,lengthSeconds,videos"

// Search searches for the given string and returns a SearchResult slice.
// It queries for two pages of results, and keeps a track of the number of
// pages currently returned. If the getmore parameter is true, it will add
// two more pages to the already tracked page number, and return the result.
func (c *Client) Search(stype, text string, getmore bool) ([]SearchResult, error) {
	var oldpg, newpg int
	var results []SearchResult

	if searchCtx != nil {
		searchCancel()
	}

	if !getmore {
		setPage(0)
	} else {
		oldpg = getPage()
	}

	searchCtx, searchCancel = context.WithCancel(context.Background())

	for newpg = oldpg + 1; newpg <= oldpg+2; newpg++ {
		var s []SearchResult

		query := "search?q=" + url.QueryEscape(text) + searchField +
			"&page=" + strconv.Itoa(newpg) + "&type=" + stype

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

	setPage(newpg)

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

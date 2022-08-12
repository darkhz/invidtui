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
	IndexID       string `json: "indexId"`
	PublishedText string `json: "publishedText"`
	Description   string `json: "description"`
	VideoCount    int    `json: "videoCount"`
	SubCount      int    `json: "subCount"`
	LengthSeconds int    `json: "lengthSeconds"`
	LiveNow       bool   `json: "liveNow"`
}

var (
	page      int
	pageMutex sync.Mutex

	paramMutex   sync.Mutex
	searchParams map[string]string
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

	SearchCancel()

	if !getmore {
		setpg(0)
	} else {
		oldpg = getpg()
	}

	for newpg = oldpg + 1; newpg <= oldpg+2; newpg++ {
		var s []SearchResult

		query := "?q=" + url.QueryEscape(text) + searchField +
			"&page=" + strconv.Itoa(newpg)

		if chanid != nil {
			query = "channels/search/" + chanid[0] + query
		} else {
			query = "search" + query + "&type=" + stype

			for param, val := range searchParams {
				if val == "" {
					continue
				}

				query += "&" + param + "=" + val
			}
		}

		res, err := c.ClientRequest(SearchCtx(), query)
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

// SearchCtx returns the search context.
func SearchCtx() context.Context {
	return ClientCtx()
}

// SearchCancel cancels and renews the search context.
func SearchCancel() {
	ClientCancel()
}

// SetSearchParams sets the search parameters.
func SetSearchParams(params map[string]string) {
	paramMutex.Lock()
	defer paramMutex.Unlock()

	searchParams = params
}

// GetSearchParams gets the search parameters.
func GetSearchParams() map[string]string {
	paramMutex.Lock()
	defer paramMutex.Unlock()

	return searchParams
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

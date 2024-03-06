package invidious

import (
	"net/url"
	"strconv"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/resolver"
)

// SearchData stores information about a search result.
type SearchData struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	AuthorID      string `json:"authorId"`
	VideoID       string `json:"videoId"`
	PlaylistID    string `json:"playlistId"`
	Author        string `json:"author"`
	IndexID       string `json:"indexId"`
	ViewCountText string `json:"viewCountText"`
	PublishedText string `json:"publishedText"`
	Duration      string `json:"duration"`
	Description   string `json:"description"`
	VideoCount    int64  `json:"videoCount"`
	SubCount      int    `json:"subCount"`
	LengthSeconds int64  `json:"lengthSeconds"`
	LiveNow       bool   `json:"liveNow"`

	Timestamp *int64
}

// SuggestData stores search suggestions.
type SuggestData struct {
	Query       string   `json:"query"`
	Suggestions []string `json:"suggestions"`
}

// Search retrieves search results according to the provided query.
func Search(stype, text string, parameters map[string]string, page int, ucid ...string) ([]SearchData, int, error) {
	var newpg int
	var data []SearchData

	client.Cancel()

	for newpg = page + 1; newpg <= page+2; newpg++ {
		query := "?q=" + url.QueryEscape(text) +
			"&page=" + strconv.Itoa(newpg)

		if stype == "channel" && ucid != nil {
			query = "channels/search/" + ucid[0] + query
		} else {
			query = "search" + query + "&type=" + stype
		}

		for param, val := range parameters {
			if val == "" {
				continue
			}

			query += "&" + param + "=" + val
		}

		res, err := client.Fetch(client.Ctx(), query)
		if err != nil {
			return nil, newpg, err
		}

		s := []SearchData{}
		err = resolver.DecodeJSONReader(res.Body, &s)
		if err != nil {
			return nil, newpg, err
		}

		data = append(data, s...)

		res.Body.Close()
	}

	return data, newpg, nil
}

// SearchSuggestions retrieves search suggestions.
func SearchSuggestions(text string) (SuggestData, error) {
	var data SuggestData

	client.Cancel()

	query := "search/suggestions?q=" + url.QueryEscape(text)

	res, err := client.Fetch(client.Ctx(), query)
	if err != nil {
		return SuggestData{}, err
	}
	defer res.Body.Close()

	err = resolver.DecodeJSONReader(res.Body, &data)
	if err != nil {
		return SuggestData{}, err
	}

	return data, nil
}

package invidious

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/darkhz/invidtui/client"
)

const searchField = "&fields=type,title,videoId,playlistId,author,authorId,publishedText,description,videoCount,subCount,lengthSeconds,videos,liveNow&hl=en"

// SearchData stores information about a search result.
type SearchData struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	AuthorID      string `json:"authorId"`
	VideoID       string `json:"videoId"`
	PlaylistID    string `json:"playlistId"`
	Author        string `json:"author"`
	IndexID       string `json:"indexId"`
	PublishedText string `json:"publishedText"`
	Duration      string `json:"duration"`
	Description   string `json:"description"`
	VideoCount    int    `json:"videoCount"`
	SubCount      int    `json:"subCount"`
	LengthSeconds int64  `json:"lengthSeconds"`
	LiveNow       bool   `json:"liveNow"`
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
		query := "?q=" + url.QueryEscape(text) + searchField +
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
		err = json.NewDecoder(res.Body).Decode(&s)
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

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return SuggestData{}, err
	}

	return data, nil
}

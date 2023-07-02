package invidious

import (
	"strconv"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/utils"
)

// FeedData stores videos in the user's feed.
type FeedData struct {
	Videos []FeedVideos `json:"videos"`
}

// FeedVideos stores information about a video in the user's feed.
type FeedVideos struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	VideoID       string `json:"videoId"`
	LengthSeconds int64  `json:"lengthSeconds"`
	Author        string `json:"author"`
	AuthorID      string `json:"authorId"`
	AuthorURL     string `json:"authorUrl"`
	PublishedText string `json:"publishedText"`
	ViewCount     int64  `json:"viewCount"`
}

// Feed retrieves videos from a user's feed.
func Feed(page int) (FeedData, error) {
	var data FeedData

	query := "auth/feed?hl=en&page=" + strconv.Itoa(page)

	res, err := client.Fetch(client.Ctx(), query, client.Token())
	if err != nil {
		return FeedData{}, err
	}
	defer res.Body.Close()

	err = utils.JSON().NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return FeedData{}, err
	}

	return data, nil
}

package lib

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
)

// FeedResult stores the feed data.
type FeedResult struct {
	Videos []FeedVideos `json:"videos"`
}

// FeedVideos stores the feed video data.
type FeedVideos struct {
	Type          string `json:"type"`
	Title         string `json:"title"`
	VideoID       string `json:"videoId"`
	LengthSeconds int    `json:"lengthSeconds"`
	Author        string `json:"author"`
	AuthorID      string `json:"authorId"`
	AuthorURL     string `json:"authorUrl"`
	Published     int    `json:"published"`
	PublishedText string `json:"publishedText"`
	ViewCount     int    `json:"viewCount"`
}

var (
	feedPage  int
	feedMutex sync.Mutex
)

// Feed gets the user's feed. If getmore is set, more feed results are loaded.
func (c *Client) Feed(getmore bool) (FeedResult, error) {
	var result FeedResult

	if getmore {
		incFeedPage()
	} else {
		resetFeedPage()
	}

	query := "auth/feed?page=" + getFeedPage()
	res, err := c.ClientRequest(context.Background(), query, GetToken())
	if err != nil {
		return FeedResult{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return FeedResult{}, err
	}

	return result, nil
}

func getFeedPage() string {
	feedMutex.Lock()
	defer feedMutex.Unlock()

	return strconv.Itoa(feedPage)
}

func resetFeedPage() {
	feedMutex.Lock()
	defer feedMutex.Unlock()

	feedPage = 1
}

func incFeedPage() {
	feedMutex.Lock()
	defer feedMutex.Unlock()

	feedPage++
}

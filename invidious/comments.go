package invidious

import (
	"github.com/darkhz/invidtui/client"
	"github.com/goccy/go-json"
)

// CommentsData stores comments and its continuation data.
type CommentsData struct {
	Comments     []CommentData `json:"comments"`
	Continuation string        `json:"continuation"`
}

// CommentData stores information about a comment.
type CommentData struct {
	Verified             bool         `json:"verified"`
	Author               string       `json:"author"`
	AuthorID             string       `json:"authorId"`
	AuthorURL            string       `json:"authorUrl"`
	Content              string       `json:"content"`
	PublishedText        string       `json:"publishedText"`
	LikeCount            int          `json:"likeCount"`
	CommentID            string       `json:"commentId"`
	AuthorIsChannelOwner bool         `json:"authorIsChannelOwner"`
	Replies              CommentReply `json:"replies"`
}

// CommentReply stores information about comment replies.
type CommentReply struct {
	ReplyCount   int    `json:"replyCount"`
	Continuation string `json:"continuation"`
}

// Comments retrieves comments for a video.
func Comments(id string, continuation ...string) (CommentsData, error) {
	var data CommentsData

	client.Cancel()

	query := "comments/" + id + "?hl=en"
	if continuation != nil {
		query += "&continuation=" + continuation[0]
	}

	res, err := client.Fetch(client.Ctx(), query)
	if err != nil {
		return CommentsData{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		return CommentsData{}, err
	}

	return data, nil
}

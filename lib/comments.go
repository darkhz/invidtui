package lib

import (
	"context"
	"encoding/json"
)

type CommentResult struct {
	Comments     []CommentsInfo `json:"comments"`
	Continuation string         `json:"continuation"`
}

type CommentsInfo struct {
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

type CommentReply struct {
	ReplyCount   int    `json:"replyCount"`
	Continuation string `json:"continuation"`
}

var (
	commentCtx    context.Context
	commentCancel context.CancelFunc
)

func (c *Client) Comments(id string, continuation ...string) (CommentResult, error) {
	var result CommentResult

	CommentCancel()

	query := "comments/" + id
	if continuation != nil {
		query += "?continuation=" + continuation[0]
	}

	res, err := c.ClientRequest(CommentCtx(), query)
	if err != nil {
		return CommentResult{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return CommentResult{}, err
	}

	return result, nil
}

func CommentCtx() context.Context {
	return commentCtx
}

func CommentCancel() {
	if commentCtx != nil {
		commentCancel()
	}

	commentCtx, commentCancel = context.WithCancel(context.Background())
}

package lib

import (
	"context"
	"encoding/json"
)

// CommentResult stores the comments.
type CommentResult struct {
	Comments     []CommentsInfo `json:"comments"`
	Continuation string         `json:"continuation"`
}

// CommentsInfo stores the comment information.
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

// CommentReply stores the comment reply count and continuation.
type CommentReply struct {
	ReplyCount   int    `json:"replyCount"`
	Continuation string `json:"continuation"`
}

var (
	commentCtx    context.Context
	commentCancel context.CancelFunc
)

// Comments gets the comments for a video ID.
func (c *Client) Comments(id string, continuation ...string) (CommentResult, error) {
	var result CommentResult

	CommentCancel()

	query := "comments/" + id + "?hl=en"
	if continuation != nil {
		query += "&continuation=" + continuation[0]
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

// CommentCtx returns the comment context.
func CommentCtx() context.Context {
	return commentCtx
}

// CommentCancel cancels and renews the comment context.
func CommentCancel() {
	if commentCtx != nil {
		commentCancel()
	}

	commentCtx, commentCancel = context.WithCancel(context.Background())
}

package ui

import (
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

var commentsLock *semaphore.Weighted

// ShowComments shows comments for the selected video.
func ShowComments() {
	if commentsLock == nil {
		commentsLock = semaphore.NewWeighted(1)
	}

	InfoMessage("Loading comments", true)

	go App.QueueUpdateDraw(func() {
		showComments()
	})
}

// showComments loads the comment viewer.
func showComments() {
	info, err := getListReference()
	if err != nil {
		ErrorMessage(err)
		return
	}

	if info.Type == "playlist" || info.Type == "channel" {
		return
	}

	comments, err := lib.GetClient().Comments(info.VideoID)
	if err != nil {
		ErrorMessage(err)
		return
	}

	InfoMessage("Loaded comments", false)

	title := tview.NewTextView()
	title.SetDynamicColors(true)
	title.SetText("[white::bu]Comments")
	title.SetTextAlign(tview.AlignCenter)
	title.SetBackgroundColor(tcell.ColorDefault)

	rootNode := tview.NewTreeNode("[blue::bu]" + info.Title).
		SetSelectable(false)

	CommentsView := tview.NewTreeView()
	CommentsView.SetRoot(rootNode)
	CommentsView.SetCurrentNode(rootNode)
	CommentsView.SetGraphics(false)
	CommentsView.SetBackgroundColor(tcell.ColorDefault)
	CommentsView.SetSelectedStyle(tcell.Style{}.
		Foreground(tcell.Color16).
		Background(tcell.ColorWhite),
	)
	CommentsView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			closeCommentView()
		}

		switch event.Rune() {
		case ' ':
			node := CommentsView.GetCurrentNode()
			if node.GetLevel() > 2 {
				node.GetParent().SetExpanded(!node.GetParent().IsExpanded())
			}
		}

		return event
	})
	CommentsView.SetSelectedFunc(func(node *tview.TreeNode) {
		var selectedNode, removeNode *tview.TreeNode

		continuation, ok := node.GetReference().(string)
		if !ok {
			return
		}

		if node.GetLevel() == 2 && len(node.GetChildren()) > 0 {
			var toggle string

			expanded := node.IsExpanded()
			if expanded {
				toggle = "Show"
			} else {
				toggle = "Hide"
			}

			node.SetExpanded(!expanded)
			node.SetText("-- " + toggle + " comments --")

			return
		}

		if node.GetLevel() > 2 || node.GetParent() == rootNode {
			selectedNode = node.GetParent()
			removeNode = node
		} else {
			selectedNode = node
		}

		go loadSubComments(
			CommentsView,
			selectedNode, removeNode,
			info.VideoID, continuation,
		)
	})

	commentsFlex := tview.NewFlex().
		AddItem(title, 1, 0, false).
		AddItem(CommentsView, 10, 10, true).
		SetDirection(tview.FlexRow)
	commentsFlex.SetBackgroundColor(tcell.ColorDefault)

	for _, comment := range comments.Comments {
		addCommentNode(rootNode, comment)
	}

	addCommentContinuation(rootNode, comments)

	MPage.AddAndSwitchToPage(
		"comments",
		statusmodal(commentsFlex, CommentsView),
		true,
	).ShowPage("ui")

	App.SetFocus(CommentsView)
}

// loadSubComments loads the subcomments.
func loadSubComments(view *tview.TreeView, selNode, rmNode *tview.TreeNode, videoID, continuation string) {
	if !commentsLock.TryAcquire(1) {
		InfoMessage("Comments are still loading", false)
		return
	}
	defer commentsLock.Release(1)

	showNode := selNode
	if rmNode != nil {
		showNode = rmNode
	}

	App.QueueUpdateDraw(func() {
		showNode.SetText("-- Loading comments --")
	})

	subcomments, err := lib.GetClient().Comments(videoID, continuation)
	if err != nil {
		ErrorMessage(err)
		App.QueueUpdateDraw(func() {
			selNode.SetText("-- Reload --")
		})

		return
	}

	App.QueueUpdateDraw(func() {
		for i, comment := range subcomments.Comments {
			current := addCommentNode(selNode, comment)
			if i == 0 {
				view.SetCurrentNode(current)
			}
		}

		showNode.SetText("-- Hide comments --")

		if rmNode != nil {
			selNode.RemoveChild(rmNode)
		}

		addCommentContinuation(selNode, subcomments)
	})
}

// closeCommentView closes the comment viewer.
func closeCommentView() {
	if pg, _ := MPage.GetFrontPage(); pg != "comments" {
		return
	}

	exitFocus()
	popupStatus(false)
	lib.CommentCancel()
}

// addCommentNode adds a comment node.
func addCommentNode(node *tview.TreeNode, comment lib.CommentsInfo) *tview.TreeNode {
	commentNode := tview.NewTreeNode("- [purple::bu]" + comment.Author)
	for _, line := range splitLines(comment.Content) {
		commentNode.AddChild(
			tview.NewTreeNode(" " + line).
				SetSelectable(false).
				SetIndent(1),
		)
	}

	if comment.Replies.ReplyCount > 0 {
		commentNode.AddChild(
			tview.NewTreeNode("-- Load " + strconv.Itoa(comment.Replies.ReplyCount) + " replies --").
				SetReference(comment.Replies.Continuation),
		)
	}

	node.AddChild(commentNode)
	node.AddChild(
		tview.NewTreeNode("").
			SetSelectable(false),
	)

	return commentNode
}

// addCommentContinuation checks if there are more comments and adds a continuation button.
func addCommentContinuation(node *tview.TreeNode, comments lib.CommentResult) {
	if comments.Continuation == "" {
		return
	}

	node.AddChild(
		tview.NewTreeNode("-- Load more replies --").
			SetSelectable(true).
			SetReference(comments.Continuation),
	)
}

// splitLines splits the comment's content into different lines.
func splitLines(line string) []string {
	var currPos int
	var lines []string
	var joinedString string

	split := strings.Split(line, " ")

	for i, w := range split {
		joinedString += w + " "

		if len(joinedString) >= 60 {
			lines = append(lines, joinedString)
			joinedString = ""

			currPos = i
		}
	}

	if lines == nil || currPos < len(split) {
		lines = append(lines, joinedString)
	}

	return lines
}

package ui

import (
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

func ShowComments() {
	InfoMessage("Loading comments", true)

	go App.QueueUpdateDraw(func() {
		showComments()
	})
}

func showComments() {
	info, err := getListReference()
	if err != nil {
		ErrorMessage(err)
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
		var selectedNode *tview.TreeNode

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
			selectedNode.RemoveChild(node)
		} else {
			selectedNode = node
		}

		selectedNode.SetText("-- Loading comments --")

		subcomments, err := lib.GetClient().Comments(info.VideoID, continuation)
		if err != nil {
			ErrorMessage(err)
			selectedNode.SetText("-- Reload --")

			return
		}

		for i, comment := range subcomments.Comments {
			current := addCommentNode(selectedNode, comment)
			if i == 0 {
				CommentsView.SetCurrentNode(current)
			}
		}

		selectedNode.SetText("-- Hide comments --")

		addCommentContinuation(selectedNode, subcomments)
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

func closeCommentView() {
	if pg, _ := MPage.GetFrontPage(); pg != "comments" {
		return
	}

	exitFocus()
	popupStatus(false)
	lib.CommentCancel()
}

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

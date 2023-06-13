package view

import (
	"strconv"

	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// CommentsView describes the layout for a comments view.
type CommentsView struct {
	init      bool
	currentID string

	modal *app.Modal
	view  *tview.TreeView
	root  *tview.TreeNode

	lock *semaphore.Weighted
}

// Comments stores the properties of the comments view.
var Comments CommentsView

// Init initializes the comments view.
func (c *CommentsView) Init() {
	c.view = tview.NewTreeView()
	c.view.SetGraphics(false)
	c.view.SetBackgroundColor(tcell.ColorDefault)
	c.view.SetSelectedStyle(tcell.Style{}.
		Foreground(tcell.Color16).
		Background(tcell.ColorWhite),
	)
	c.view.SetSelectedFunc(c.selectorHandler)
	c.view.SetInputCapture(c.Keybindings)

	c.root = tview.NewTreeNode("")

	c.modal = app.NewModal("comments", "Comments", c.view, 40, 0)

	c.lock = semaphore.NewWeighted(1)

	c.init = true
}

// Show shows the comments view.
func (c *CommentsView) Show() {
	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}
	if info.Type != "video" {
		return
	}

	c.Init()

	go c.Load(info.VideoID, info.Title)
}

// Load loads the comments from the given video.
func (c *CommentsView) Load(id, title string) {
	if !c.lock.TryAcquire(1) {
		app.ShowInfo("Comments are still loading", false)
		return
	}
	defer c.lock.Release(1)

	app.ShowInfo("Loading comments", true)

	comments, err := inv.Comments(id)
	if err != nil {
		app.ShowError(err)
		return
	}

	c.currentID = id

	app.ShowInfo("Loaded comments", false)

	app.UI.QueueUpdateDraw(func() {
		c.root.SetText("[blue::bu]" + title)
		for _, comment := range comments.Comments {
			c.addComment(c.root, comment)
		}
		c.addContinuation(c.root, comments.Continuation)

		c.view.SetRoot(c.root)
		c.view.SetCurrentNode(c.root)

		c.modal.Show(true)
		app.UI.SetFocus(c.view)
	})
}

// Subcomments loads the subcomments for the currently selected comment.
func (c *CommentsView) Subcomments(selected, removed *tview.TreeNode, continuation string) {
	if !c.lock.TryAcquire(1) {
		app.ShowInfo("Comments are still loading", false)
		return
	}
	defer c.lock.Release(1)

	showNode := selected
	if removed != nil {
		showNode = removed
	}

	app.UI.QueueUpdateDraw(func() {
		showNode.SetText("-- Loading comments --")
	})

	subcomments, err := inv.Comments(c.currentID, continuation)
	if err != nil {
		app.ShowError(err)
		app.UI.QueueUpdateDraw(func() {
			selected.SetText("-- Reload --")
		})

		return
	}

	app.UI.QueueUpdateDraw(func() {
		for i, comment := range subcomments.Comments {
			current := c.addComment(selected, comment)
			if i == 0 {
				c.view.SetCurrentNode(current)
			}
		}

		showNode.SetText("-- Hide comments --")

		if removed != nil {
			selected.RemoveChild(removed)
		}

		c.addContinuation(selected, subcomments.Continuation)
	})
}

// Close closes the comments view.
func (c *CommentsView) Close() {
	if c.modal == nil {
		return
	}

	c.modal.Exit(false)
	app.SetPrimaryFocus()
}

// Keybindings describes the keybindings for the comments view.
func (c *CommentsView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEscape:
		c.Close()
	}

	switch event.Rune() {
	case ' ':
		node := c.view.GetCurrentNode()
		if node.GetLevel() > 2 {
			node.GetParent().SetExpanded(!node.GetParent().IsExpanded())
		}
	}

	return event
}

// selectorHandler generates subcomments for the selected comment.
func (c *CommentsView) selectorHandler(node *tview.TreeNode) {
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

	if node.GetLevel() > 2 || node.GetParent() == c.root {
		selectedNode = node.GetParent()
		removeNode = node
	} else {
		selectedNode = node
	}

	go c.Subcomments(selectedNode, removeNode, continuation)
}

// addComments adds the provided comment to the comment node.
func (c *CommentsView) addComment(node *tview.TreeNode, comment inv.CommentData) *tview.TreeNode {
	authorInfo := "- [purple::bu]" + comment.Author + "[-:-:-]"
	authorInfo += " [grey::b]" + utils.FormatPublished(comment.PublishedText) + "[-:-:-]"
	if comment.Verified {
		authorInfo += " [aqua::b](Verified)[-:-:-]"
	}
	if comment.AuthorIsChannelOwner {
		authorInfo += " [plum::b](Owner)"
	}
	authorInfo += " [red::b](" + strconv.Itoa(comment.LikeCount) + " likes)"

	commentNode := tview.NewTreeNode(authorInfo)
	for _, line := range utils.SplitLines(comment.Content) {
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

// addContinuation adds a button under a comments to load more subcomments.
func (c *CommentsView) addContinuation(node *tview.TreeNode, continuation string) {
	if continuation == "" {
		return
	}

	node.AddChild(
		tview.NewTreeNode("-- Load more replies --").
			SetSelectable(true).
			SetReference(continuation),
	)
}

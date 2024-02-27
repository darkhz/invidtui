package view

import (
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
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

	property theme.ThemeProperty
	builder  theme.ThemeTextBuilder

	lock *semaphore.Weighted
}

// Comments stores the properties of the comments view.
var Comments CommentsView

// Init initializes the comments view.
func (c *CommentsView) Init() {
	c.property = theme.ThemeProperty{
		Context: theme.ThemeContextComments,
		Item:    theme.ThemePopupBackground,
	}

	c.builder = theme.NewTextBuilder(c.property.Context)

	c.view = theme.NewTreeView(c.property)
	c.view.SetGraphics(false)
	c.view.SetInputCapture(c.Keybindings)
	c.view.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextComments, c.view)
	})

	c.root = tview.NewTreeNode("")

	c.modal = app.NewModal("comments", "Comments", c.view, 40, 0, c.property)

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

// IsOpen returns if the comments view is open.
func (c *CommentsView) IsOpen() bool {
	return c.modal != nil && c.modal.Open
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
		c.setNodeText(c.root, theme.ThemeVideo, tview.Escape(title))
		for _, comment := range comments.Comments {
			c.addComment(c.root, comment)
		}
		c.addContinuation(c.root, comments.Continuation)

		c.view.SetRoot(c.root)
		c.view.SetCurrentNode(c.root)

		c.modal.Show(true)
		app.SetPrimaryFocus()
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

	item := theme.ThemeText

	app.ConditionalDraw(func() bool {
		c.setNodeText(showNode, item, "-- Loading comments --")

		return c.IsOpen()
	})

	subcomments, err := inv.Comments(c.currentID, continuation)
	if err != nil {
		app.ShowError(err)
		app.ConditionalDraw(func() bool {
			c.setNodeText(selected, item, "-- Reload --")

			return c.IsOpen()
		})

		return
	}

	app.ConditionalDraw(func() bool {
		for i, comment := range subcomments.Comments {
			current := c.addComment(selected, comment)
			if i == 0 {
				c.view.SetCurrentNode(current)
			}
		}

		c.setNodeText(showNode, item, "-- Hide comments --")

		if removed != nil {
			selected.RemoveChild(removed)
		}

		c.addContinuation(selected, subcomments.Continuation)

		return c.IsOpen()
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
	switch keybinding.KeyOperation(event, keybinding.KeyContextComments) {
	case keybinding.KeyCommentReplies:
		node := c.view.GetCurrentNode()
		c.selectorHandler(node)

		return nil

	case keybinding.KeyClose:
		c.Close()
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
		c.setNodeText(node, theme.ThemeText, "-- "+toggle+" comments --")

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
	c.builder.Format(theme.ThemeAuthor, "author", "- %s", tview.Escape(comment.Author))
	c.builder.Format(theme.ThemePublished, "published", " %s", utils.FormatPublished(comment.PublishedText))
	if comment.Verified {
		c.builder.Format(theme.ThemeAuthorVerified, "verified", " %s", "(Verified)")
	}
	if comment.AuthorIsChannelOwner {
		c.builder.Format(theme.ThemeAuthorOwner, "owner", " %s", "(Owner)")
	}
	c.builder.Format(theme.ThemeLikes, "likes", " (%d likes)", comment.LikeCount)

	commentNode := tview.NewTreeNode(c.builder.Get())
	for _, line := range tview.WordWrap(tview.Escape(comment.Content), 60) {
		c.builder.Format(theme.ThemeComment, "comment", " %s", line)
		commentNode.AddChild(
			tview.NewTreeNode(c.builder.Get()).
				SetSelectable(false).
				SetIndent(1),
		)
	}

	if comment.Replies.ReplyCount > 0 {
		c.builder.Format(theme.ThemeText, "replies", "-- Load  %d replies --", comment.Replies.ReplyCount)
		commentNode.AddChild(
			tview.NewTreeNode(c.builder.Get()).
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
		c.setNodeText(tview.NewTreeNode(""), theme.ThemeText, "-- Load more replies --").
			SetSelectable(true).
			SetReference(continuation),
	)
}

func (c *CommentsView) setNodeText(node *tview.TreeNode, item theme.ThemeItem, text string) *tview.TreeNode {
	c.builder.Append(item, "node", text)
	node.SetText(c.builder.Get())

	return node
}

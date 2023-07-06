package view

import (
	"strings"

	"github.com/darkhz/invidtui/cmd"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

const bannerText = `
   (_)____  _   __ (_)____/ // /_ __  __ (_)
  / // __ \| | / // // __  // __// / / // /
 / // / / /| |/ // // /_/ // /_ / /_/ // /
/_//_/ /_/ |___//_/ \__,_/ \__/ \__,_//_/
`

// BannerView describes the layout of a banner view.
type BannerView struct {
	flex *tview.Flex

	init, shown bool
}

// Banner stores the banner view properties.
var Banner BannerView

// Name returns the name of the banner view.
func (b *BannerView) Name() string {
	return "Start"
}

// Init intializes the banner view.
func (b *BannerView) Init() bool {
	if b.init {
		return true
	}

	b.shown = true
	b.setup()

	b.init = true

	return true
}

// Exit closes the banner view.
func (b *BannerView) Exit() bool {
	b.shown = false

	return true
}

// Tabs describes the tab layout for the banner view.
func (b *BannerView) Tabs() app.Tab {
	return app.Tab{}
}

// Primitive returns the primitive for the banner view.
func (b *BannerView) Primitive() tview.Primitive {
	return b.flex
}

// Keybindings describes the banner view's keybindings.
func (b *BannerView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event, "Search") {
	case "SearchQuery":
		Search.Query()
	}

	return event
}

// setup sets up the banner view.
func (b *BannerView) setup() {
	lines := strings.Split(bannerText, "\n")
	bannerWidth := 0
	bannerHeight := len(lines)

	bannerBox := tview.NewTextView()
	bannerBox.SetDynamicColors(true)
	bannerBox.SetBackgroundColor(tcell.ColorDefault)
	bannerBox.SetText("[::b]" + bannerText)

	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	for _, line := range lines {
		if len(line) > bannerWidth {
			bannerWidth = len(line)
		}
	}

	b.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(box, 0, 7, false).
		AddItem(tview.NewFlex().
			AddItem(box, 0, 1, false).
			AddItem(bannerBox, bannerWidth, 1, true).
			AddItem(box, 0, 1, false), bannerHeight, 1, true).
		AddItem(box, 0, 7, false)
	b.flex.SetBackgroundColor(tcell.ColorDefault)
	b.flex.SetInputCapture(b.Keybindings)
	bannerBox.SetFocusFunc(func() {
		app.SetContextMenu(b.Name(), b.flex)
	})

	b.shown = true
}

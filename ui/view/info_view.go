package view

import (
	"strings"

	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// InfoView describes the layout for a playlist/channel page.
// It displays a title, description and the entries.
type InfoView struct {
	flex               *tview.Flex
	title, description *tview.TextView

	primitive tview.Primitive
}

// Init initializes the info view.
func (i *InfoView) Init(primitive tview.Primitive) {
	i.flex = tview.NewFlex().
		SetDirection(tview.FlexRow)
	i.flex.SetBackgroundColor(tcell.ColorDefault)

	i.title = tview.NewTextView()
	i.title.SetDynamicColors(true)
	i.title.SetTextAlign(tview.AlignCenter)
	i.title.SetBackgroundColor(tcell.ColorDefault)

	i.description = tview.NewTextView()
	i.description.SetDynamicColors(true)
	i.description.SetTextAlign(tview.AlignCenter)
	i.description.SetBackgroundColor(tcell.ColorDefault)

	i.primitive = primitive
}

// Set sets the title and description of the info view.
func (i *InfoView) Set(title, description string) {
	var descSize int

	_, _, pageWidth, _ := app.UI.Pages.GetRect()

	descText := strings.ReplaceAll(description, "\n", " ")
	descLength := len(descText)
	if descLength > 0 {
		descSize = 2

		if descLength >= pageWidth {
			descSize++
		}
	}

	i.flex.Clear()
	i.flex.AddItem(i.title, 1, 0, false)
	i.flex.AddItem(app.HorizontalLine(), 1, 0, false)
	if descLength > 0 {
		i.flex.AddItem(i.description, descSize, 0, false)
		i.flex.AddItem(app.HorizontalLine(), 1, 0, false)
	}
	i.flex.AddItem(i.primitive, 0, 10, true)

	i.title.SetText("[::bu]" + title)
	i.description.SetText(descText)
}

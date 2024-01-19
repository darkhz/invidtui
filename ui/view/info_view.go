package view

import (
	"strings"

	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/tview"
)

// InfoView describes the layout for a playlist/channel page.
// It displays a title, description and the entries.
type InfoView struct {
	flex               *tview.Flex
	title, description *tview.TextView

	primitive tview.Primitive
	property  theme.ThemeProperty
}

// Init initializes the info view.
func (i *InfoView) Init(primitive tview.Primitive, property theme.ThemeProperty) {
	i.flex = theme.NewFlex(property).
		SetDirection(tview.FlexRow)

	i.title = theme.NewTextView(property)
	i.title.SetTextAlign(tview.AlignCenter)

	i.description = theme.NewTextView(property)
	i.description.SetTextAlign(tview.AlignCenter)

	i.primitive = primitive
	i.property = property
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
	i.flex.AddItem(app.HorizontalLine(i.property), 1, 0, false)
	if descLength > 0 {
		i.flex.AddItem(i.description, descSize, 0, false)
		i.flex.AddItem(app.HorizontalLine(i.property), 1, 0, false)
	}
	i.flex.AddItem(i.primitive, 0, 10, true)

	i.title.SetText(theme.SetTextStyle(
		"title",
		title,
		i.property.Context,
		theme.ThemeTitle,
	))
	i.description.SetText(theme.SetTextStyle(
		"description",
		descText,
		i.property.Context,
		theme.ThemeDescription,
	))
}

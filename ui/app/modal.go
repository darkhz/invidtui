package app

import (
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/tview"
)

// Modal stores a layout to display a floating modal.
type Modal struct {
	Name          string
	Open          bool
	Height, Width int

	attach, menu                            bool
	regionX, regionY, pageHeight, pageWidth int

	Flex  *tview.Flex
	Table *tview.Table

	y *tview.Flex
	x *tview.Flex

	property theme.ThemeProperty
}

var modals []*Modal

// NewModal returns a modal. If a primitive is not provided,
// a table is attach to it.
func NewModal(name, title string, item tview.Primitive, height, width int, property theme.ThemeProperty) *Modal {
	var table *tview.Table

	modalTitle := theme.NewTextView(property)
	modalTitle.SetText(
		theme.SetTextStyle(
			"title", title,
			property.Context, theme.ThemeTitle,
		),
	)
	modalTitle.SetTextAlign(tview.AlignCenter)

	if item == nil {
		table = theme.NewTable(property)
		table.SetSelectable(true, false)

		item = table
	}

	flex := theme.NewFlex(property)
	flex.SetBorder(true)
	flex.SetDirection(tview.FlexRow)

	box := theme.NewBox(property)

	flex.AddItem(modalTitle, 1, 0, false)
	flex.AddItem(box, 1, 0, false)
	flex.AddItem(item, 0, 1, true)

	return &Modal{
		Name:  name,
		Flex:  flex,
		Table: table,

		Height: height,
		Width:  width,

		property: property,
	}
}

// NewMenuModal returns a menu modal.
func NewMenuModal(name string, regionX, regionY int, property theme.ThemeProperty) *Modal {
	table := theme.NewTable(property)
	table.SetBorder(true)
	table.SetSelectable(true, false)

	flex := theme.NewFlex(property)
	flex.SetDirection(tview.FlexRow)
	flex.AddItem(table, 0, 1, true)

	return &Modal{
		Name:  name,
		Table: table,
		Flex:  flex,

		menu:    true,
		regionX: regionX,
		regionY: regionY,

		property: property,
	}
}

// Show shows the modal. If attachToStatus is true, the modal will
// attach to the top part of the status bar rather than float in the middle.
func (m *Modal) Show(attachToStatus bool) {
	var x, y, xprop, xattach, yattach int

	if len(modals) > 0 && modals[len(modals)-1].Name == m.Name {
		return
	}

	switch {
	case m.menu:
		xprop = 1
		x, y = m.regionX, m.regionY

	case attachToStatus:
		m.attach = true
		xattach, yattach = 1, 1

	default:
		xattach = 1
	}

	m.Open = true

	m.y = theme.NewFlex(m.property).
		SetDirection(tview.FlexRow).
		AddItem(nil, y, yattach, false).
		AddItem(m.Flex, m.Height, 0, true).
		AddItem(nil, yattach, 0, false)

	m.x = theme.NewFlex(m.property).
		SetDirection(tview.FlexColumn).
		AddItem(nil, x, xattach, false).
		AddItem(m.y, m.Width, 0, true).
		AddItem(nil, xprop, xattach, false)

	UI.Area.AddAndSwitchToPage(m.Name, m.x, true)
	for _, modal := range modals {
		UI.Area.ShowPage(modal.Name)
	}
	UI.Area.ShowPage("ui")

	UI.SetFocus(m.Flex)

	modals = append(modals, m)
	ResizeModal()
}

// Exit exits the modal.
func (m *Modal) Exit(focusInput bool) {
	if m == nil {
		return
	}

	m.Open = false
	m.pageWidth = 0
	m.pageHeight = 0

	UI.Area.RemovePage(m.Name)

	for i, modal := range modals {
		if modal == m {
			modals[i] = modals[len(modals)-1]
			modals = modals[:len(modals)-1]

			break
		}
	}

	if focusInput {
		UI.SetFocus(UI.Status.InputField)
		return
	}

	SetPrimaryFocus()
}

// ResizeModal resizes the modal according to the current screen dimensions.
//
//gocyclo:ignore
func ResizeModal() {
	var drawn bool

	for _, modal := range modals {
		_, _, pageWidth, pageHeight := UI.Region.GetInnerRect()
		_, _, _, mh := UI.MenuLayout.GetRect()

		if modal == nil || !modal.Open ||
			(modal.pageHeight == pageHeight && modal.pageWidth == pageWidth) {
			continue
		}

		modal.pageHeight = pageHeight
		modal.pageWidth = pageWidth

		if modal.attach {
			pageHeight /= 2
		}

		height := modal.Height
		width := modal.Width
		if height >= pageHeight {
			height = pageHeight
		}
		if width >= pageWidth {
			width = pageWidth
		}

		switch {
		case modal.attach:
			switch {
			case playerShown() && modal.y.GetItemCount() == 3:
				modal.y.AddItem(nil, 2, 0, false)

			case !playerShown() && modal.y.GetItemCount() > 3:
				modal.y.RemoveItemIndex(modal.y.GetItemCount() - 1)
			}

			modal.y.ResizeItem(modal.Flex, pageHeight, 0)
			modal.x.ResizeItem(modal.y, pageWidth, 0)

		default:
			var x, y int

			if modal.menu {
				x, y = modal.regionX, modal.regionY
			} else {
				x = (pageWidth - modal.Width) / 2
				y = mh + 1
			}

			modal.y.ResizeItem(modal.Flex, height, 0)
			modal.y.ResizeItem(nil, y, 0)

			modal.x.ResizeItem(modal.y, width, 0)
			modal.x.ResizeItem(nil, x, 0)
		}

		drawn = true
	}

	if drawn {
		go UI.Draw()
	}
}

// playerShown returns whether the player is shown or not.
func playerShown() bool {
	return UI.Layout.GetItemCount() > 5
}

package app

import (
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Modal stores a layout to display a floating modal.
type Modal struct {
	Name          string
	Open          bool
	Height, Width int

	attach                bool
	pageHeight, pageWidth int

	Flex  *tview.Flex
	Table *tview.Table

	y *tview.Flex
	x *tview.Flex
}

var modals []*Modal

// NewModal returns a modal. If a primitive is not provided,
// a table is attach to it.
func NewModal(name, title string, item tview.Primitive, height, width int) *Modal {
	var table *tview.Table

	modalTitle := tview.NewTextView()
	modalTitle.SetDynamicColors(true)
	modalTitle.SetText("[::bu]" + title)
	modalTitle.SetTextAlign(tview.AlignCenter)
	modalTitle.SetBackgroundColor(tcell.ColorDefault)

	if item == nil {
		table = tview.NewTable()
		table.SetSelectorWrap(true)
		table.SetSelectable(true, false)
		table.SetBackgroundColor(tcell.ColorDefault)

		item = table
	}

	flex := tview.NewFlex()
	flex.SetBorder(true)
	flex.SetDirection(tview.FlexRow)

	box := tview.NewBox()
	box.SetBackgroundColor(tcell.ColorDefault)

	flex.AddItem(modalTitle, 1, 0, false)
	flex.AddItem(box, 1, 0, false)
	flex.AddItem(item, 0, 1, true)
	flex.SetBackgroundColor(tcell.ColorDefault)

	return &Modal{
		Name:  name,
		Flex:  flex,
		Table: table,

		Height: height,
		Width:  width,
	}
}

// Show shows the modal. If attachToStatus is true, the modal will
// attach to the top part of the status bar rather than float in the middle.
func (m *Modal) Show(attachToStatus bool) {
	var attach int

	if attachToStatus {
		m.attach = true
		attach++
	}

	m.Open = true

	m.y = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, attach, false).
		AddItem(m.Flex, m.Height, 0, true).
		AddItem(nil, attach, 0, false)

	m.x = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(m.y, m.Width, 0, true).
		AddItem(nil, 0, 1, false)

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
func ResizeModal() {
	var drawn bool

	for _, modal := range modals {
		_, _, pageWidth, pageHeight := UI.Pages.GetInnerRect()

		if modal == nil || !modal.Open ||
			(modal.pageHeight == pageHeight && modal.pageWidth == pageWidth) {
			continue
		}

		modal.pageHeight = pageHeight
		modal.pageWidth = pageWidth

		if modal.attach {
			pageHeight /= 4
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

			modal.y.ResizeItem(modal.Flex, 16, 0)
			modal.x.ResizeItem(modal.y, pageWidth, 0)

		default:
			x := (pageWidth - modal.Width) / 2
			y := (pageHeight - modal.Height) / 2

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

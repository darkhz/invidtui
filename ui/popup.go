package ui

import (
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

type popupModal struct {
	width      int
	open       bool
	playing    bool
	table      *tview.Table
	modal      *tview.Flex
	origFlex   *tview.Flex
	statusFlex *tview.Flex
}

var popup popupModal

// popupStatus sets the popup state.
func popupStatus(status bool) {
	if !status {
		popup.width = -1
	}

	popup.open = status
}

// resizePopup detects if the screen is resized, and resizes the popup
// accordingly. This function is placed in App's BeforeDrawFunc, where
// it can resize the popup when the terminal is resized.
func resizePopup(width int) {
	if !popup.open || popup.width == width {
		return
	}

	resizemodal()

	popup.width = width
}

// resizemodal gets the current width and height of the screen, and resizes
// the popup modal.
func resizemodal() {
	height := popup.table.GetRowCount()

	_, _, screenWidth, screenHeight := UIFlex.GetRect()
	screenHeight /= 4

	if height > screenHeight {
		height = screenHeight
	}

	pad := 1
	playing := isPlaying()
	if popup.playing != playing {
		if playing {
			pad += 2
		}

		popup.modal.RemoveItemIndex(popup.modal.GetItemCount() - 1)
		popup.modal.AddItem(nil, pad, 1, false)

		popup.playing = playing
	}

	popup.origFlex.ResizeItem(popup.table, height, 0)
	popup.modal.ResizeItem(popup.origFlex, height, 0)
	popup.statusFlex.ResizeItem(popup.modal, screenWidth, 0)
}

// statusmodal creates a new popup modal.
func statusmodal(v, t tview.Primitive) tview.Primitive {
	_, _, _, screenHeight := UIFlex.GetRect()
	screenHeight /= 4

	pad := 1
	playing := isPlaying()
	if playing {
		pad += 2
	}

	vbox := getVbox()

	stmodal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(vbox, 1, 0, false).
		AddItem(v, screenHeight, 1, false).
		AddItem(nil, 1, 0, false).
		AddItem(vbox, 1, 0, false).
		AddItem(nil, pad, 1, false).
		SetDirection(tview.FlexRow)

	stflex := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(stmodal, 10, 1, false).
		AddItem(nil, 0, 1, false)

	popup.playing = playing

	popup.modal = stmodal
	popup.table = t.(*tview.Table)

	popup.statusFlex = stflex
	popup.origFlex = v.(*tview.Flex)

	popupStatus(true)
	resizemodal()

	return stflex
}

func getVbox() *tview.Box {
	return tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault).
		SetDrawFunc(func(
			screen tcell.Screen,
			x, y, width, height int) (int, int, int, int) {

			centerY := y + height/2
			for cx := x; cx < x+width; cx++ {
				screen.SetContent(
					cx,
					centerY,
					tview.BoxDrawingsLightHorizontal,
					nil,
					tcell.StyleDefault.Foreground(tcell.ColorWhite),
				)
			}

			return x + 1,
				centerY + 1,
				width - 2,
				height - (centerY + 1 - y)
		})
}

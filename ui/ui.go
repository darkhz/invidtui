package ui

import (
	"fmt"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	// App contains the application.
	App *tview.Application

	// UIFlex contains the arranged UI elements.
	UIFlex *tview.Flex

	// Pages enables switching between pages.
	Pages *tview.Pages

	appSuspend  bool
	detectClose chan struct{}
)

const initMessage = "Invidtui loaded. Press / to search."

// SetupUI sets up the UI and starts the application.
func SetupUI() error {
	setupPrimitives()

	Pages = tview.NewPages()
	Pages.AddPage("main", UIFlex, true, true)

	App = tview.NewApplication()
	App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlC:
			return nil

		case tcell.KeyCtrlZ:
			appSuspend = true
		}

		return event
	})

	App.SetBeforeDrawFunc(func(t tcell.Screen) bool {
		width, _ := t.Size()

		suspendUI(t)
		resizePopup(width)
		resizeListEntries(width)

		return false
	})

	InfoMessage(initMessage, true)

	detectClose = make(chan struct{})
	go detectMPVClose()

	if err := App.SetRoot(Pages, true).Run(); err != nil {
		panic(err)
	}

	return nil
}

// StopUI stops the application.
func StopUI() {
	close(detectClose)

	StopPlayer()
	App.Stop()
}

// suspendUI suspends the application.
func suspendUI(t tcell.Screen) {
	if !appSuspend {
		return
	}

	lib.SuspendApp(t)

	appSuspend = false
}

// setupPrimitives sets up the display elements and positions
// each element appropriately.
func setupPrimitives() {
	SetupList()
	SetupInputBox()
	SetupMessageBox()
	SetupStatus()
	SetupPlayer()
	SetupFileBrowser()
	SetupPlaylist()

	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	UIFlex = tview.NewFlex().
		AddItem(ResultsList, 0, 10, true).
		AddItem(box, 1, 0, false).
		AddItem(Status, 1, 0, false).
		SetDirection(tview.FlexRow)

	UIFlex.SetBackgroundColor(tcell.ColorDefault)
}

// confirmQuit shows a confirmation message before exiting.
func confirmQuit() {
	qfunc := func(text string) {
		if text == "y" {
			StopUI()
		} else {
			App.SetFocus(ResultsList)
			Status.SwitchToPage("messages")
		}
	}

	SetInput("Quit? (y/n)", 1, qfunc, nil)
}

// detectMPVClose detects if MPV has exited unexpectedly,
// and stops the application.
func detectMPVClose() {
	lib.GetMPV().WaitUntilClosed()

	select {
	case _, ok := <-detectClose:
		if !ok {
			return
		}

	default:
	}

	StopUI()
	fmt.Printf("\rMPV has exited")
}

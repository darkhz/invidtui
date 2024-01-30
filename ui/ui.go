package ui

import (
	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/menu"
	"github.com/darkhz/invidtui/ui/player"
	"github.com/darkhz/invidtui/ui/popup"
	"github.com/darkhz/invidtui/ui/view"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// SetupUI sets up the UI and starts the application.
func SetupUI() {
	app.Setup()
	app.InitMenu(menu.Items)
	app.SetResizeHandler(Resize)
	app.SetGlobalKeybindings(Keybindings)

	instance := utils.GetHostname(client.Instance())
	msg := "Instance '" + instance + "' selected. "
	msg += "Press / to search."

	app.ShowInfo(msg, true)
	go detectPlayerClose()

	player.ParseQuery()
	view.Search.ParseQuery()

	player.Start()
	view.SetView(&view.Banner)

	_, focusedItem := app.UI.Pages.GetFrontPage()

	if err := app.UI.SetRoot(app.UI.Area, true).SetFocus(focusedItem).Run(); err != nil {
		cmd.PrintError("UI: Could not start", err)
	}
}

// StopUI stops the application.
func StopUI(skip ...struct{}) {
	app.Stop(skip...)
	player.Stop()
}

// Resize handles the resizing of the app and its components.
func Resize(screen tcell.Screen) {
	width, _ := screen.Size()

	app.ResizeModal()
	player.Resize(width)
}

// Keybindings defines the global keybindings for the application.
func Keybindings(event *tcell.EventKey) *tcell.EventKey {
	operation := keybinding.KeyOperation(event, keybinding.KeyContextApp, keybinding.KeyContextDashboard, keybinding.KeyContextDownloads)

	focused := app.UI.GetFocus()
	if _, ok := focused.(*tview.InputField); ok && operation != "Menu" {
		goto Event
	}

	if player.Keybindings(event) == nil {
		return nil
	}

	switch operation {
	case keybinding.KeyMenu:
		app.FocusMenu()
		return nil

	case keybinding.KeyDashboard:
		view.Dashboard.EventHandler()

	case keybinding.KeySuspend:
		app.UI.Suspend = true

	case keybinding.KeyCancel:
		client.Cancel()
		client.SendCancel()

		view.Comments.Close()
		app.ShowInfo("Loading canceled", false)

	case keybinding.KeyDownloadView:
		view.Downloads.View()

	case keybinding.KeyDownloadOptions:
		go view.Downloads.ShowOptions()

	case keybinding.KeyInstancesList:
		go popup.ShowInstancesList()

	case keybinding.KeyTheme:
		go popup.ShowThemes()

	case keybinding.KeyQuit:
		StopUI()
	}

Event:
	return tcell.NewEventKey(event.Key(), event.Rune(), event.Modifiers())
}

// detectPlayerClose detects if the player has exited abruptly.
func detectPlayerClose() {
	mp.Player().WaitClosed()
	mp.Player().Exit()

	select {
	case <-app.UI.Closed.Done():
		return

	default:
	}

	StopUI(struct{}{})

	cmd.PrintError("Player has exited")
}

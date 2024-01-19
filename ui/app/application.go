package app

import (
	"context"
	"sync"

	"github.com/darkhz/invidtui/platform"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Application describes the layout of the app.
type Application struct {
	MenuLayout *tview.Flex
	Menu, Tabs *tview.TextView

	Area           *tview.Pages
	Pages          *tview.Pages
	Layout, Region *tview.Flex

	Status      Status
	FileBrowser FileBrowser

	Suspend bool
	Closed  context.Context
	Exit    context.CancelFunc

	resize func(screen tcell.Screen)

	lock sync.Mutex

	*tview.Application
}

// UI stores the application data.
var UI Application

// Setup sets up the application
func Setup() {
	property := theme.ThemeProperty{
		Context: theme.ThemeContextApp,
		Item:    theme.ThemeBackground,
	}

	box := theme.NewBox(property)

	UI.Status.Setup()

	UI.Menu, UI.Tabs =
		theme.NewTextView(property.SetContext(theme.ThemeContextMenu)),
		theme.NewTextView(property.SetContext(theme.ThemeContextMenu))
	UI.Tabs.SetWrap(false)
	UI.Tabs.SetTextAlign(tview.AlignRight)
	UI.Menu.SetHighlightedFunc(MenuHighlightHandler)
	UI.Menu.SetInputCapture(MenuKeybindings)

	UI.MenuLayout = theme.NewFlex(property.SetContext(theme.ThemeContextMenu)).
		SetDirection(tview.FlexColumn).
		AddItem(UI.Menu, 0, 1, false).
		AddItem(UI.Tabs, 0, 1, false)

	UI.Pages = theme.NewPages(property)
	UI.Pages.SetChangedFunc(func() {
		MenuExit()
	})

	UI.Region = theme.NewFlex(property).
		AddItem(UI.Pages, 0, 1, true)

	UI.Layout = theme.NewFlex(property).
		SetDirection(tview.FlexRow).
		AddItem(UI.MenuLayout, 1, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(UI.Region, 0, 10, false).
		AddItem(box, 1, 0, false).
		AddItem(UI.Status.Pages, 1, 0, false)

	UI.Area = theme.NewPages(property)
	UI.Area.AddPage("ui", UI.Layout, true, true)
	UI.Area.SetChangedFunc(func() {
		pg, _ := UI.Area.GetFrontPage()
		if pg == "ui" || pg == "menu" {
			return
		}

		MenuExit()
	})

	UI.Closed, UI.Exit = context.WithCancel(context.Background())

	UI.Application = tview.NewApplication()
	UI.SetAfterDrawFunc(func(screen tcell.Screen) {
		UI.resize(screen)
		suspend(screen)
	})
}

// SetPrimaryFocus sets the focus to the appropriate primitive.
func SetPrimaryFocus() {
	if pg, _ := UI.Status.GetFrontPage(); pg == "input" {
		UI.SetFocus(UI.Status.InputField)
		return
	}

	if len(modals) > 0 {
		UI.SetFocus(modals[len(modals)-1].Flex)
		return
	}

	UI.SetFocus(UI.Pages)
}

// SetResizeHandler sets the resize handler for the app.
func SetResizeHandler(resize func(screen tcell.Screen)) {
	UI.resize = resize
}

// SetGlobalKeybindings sets the keybindings for the app.
func SetGlobalKeybindings(kb func(event *tcell.EventKey) *tcell.EventKey) {
	UI.SetInputCapture(kb)
}

// Stop stops the application.
func Stop(skip ...struct{}) {
	UI.lock.Lock()
	defer UI.lock.Unlock()

	if skip == nil {
		UI.Exit()
	}

	UI.Status.Cancel()
	UI.Stop()
}

// suspend suspends the app.
func suspend(t tcell.Screen) {
	if !UI.Suspend {
		return
	}

	platform.Suspend(t)

	UI.Suspend = false
}

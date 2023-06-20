package app

import (
	"sync"

	"github.com/darkhz/invidtui/platform"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Application describes the layout of the app.
type Application struct {
	MenuLayout *tview.Flex
	Menu, Tabs *tview.TextView

	Area   *tview.Pages
	Layout *tview.Flex
	Pages  *tview.Pages

	Status      Status
	FileBrowser FileBrowser

	SelectedStyle tcell.Style
	ColumnStyle   tcell.Style

	Suspend bool
	Closed  chan struct{}

	resize func(screen tcell.Screen)

	lock sync.Mutex

	*tview.Application
}

// UI stores the application data.
var UI Application

// Setup sets up the application
func Setup() {
	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	UI.Status.Setup()

	UI.SelectedStyle = tcell.Style{}.
		Foreground(tcell.ColorBlue).
		Background(tcell.ColorWhite).
		Attributes(tcell.AttrBold)

	UI.ColumnStyle = tcell.Style{}.
		Attributes(tcell.AttrBold)

	UI.Menu, UI.Tabs = tview.NewTextView(), tview.NewTextView()
	UI.Menu.SetWrap(false)
	UI.Menu.SetRegions(true)
	UI.Tabs.SetWrap(false)
	UI.Tabs.SetRegions(true)
	UI.Tabs.SetDynamicColors(true)
	UI.Menu.SetDynamicColors(true)
	UI.Tabs.SetTextAlign(tview.AlignRight)
	UI.Menu.SetBackgroundColor(tcell.ColorDefault)
	UI.Tabs.SetBackgroundColor(tcell.ColorDefault)
	UI.Menu.SetHighlightedFunc(MenuHighlightHandler)
	UI.Menu.SetInputCapture(MenuKeybindings)
	UI.MenuLayout = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(UI.Menu, 0, 1, false).
		AddItem(UI.Tabs, 0, 1, false)
	UI.MenuLayout.SetBackgroundColor(tcell.ColorDefault)

	UI.Pages = tview.NewPages()
	UI.Pages.SetChangedFunc(func() {
		MenuExit()
	})

	UI.Layout = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(UI.MenuLayout, 1, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(UI.Pages, 0, 10, false).
		AddItem(box, 1, 0, false).
		AddItem(UI.Status.Pages, 1, 0, false)
	UI.Layout.SetBackgroundColor(tcell.ColorDefault)

	UI.Area = tview.NewPages()
	UI.Area.AddPage("ui", UI.Layout, true, true)
	UI.Area.SetChangedFunc(func() {
		pg, _ := UI.Area.GetFrontPage()
		if pg == "ui" || pg == "menu" {
			return
		}

		MenuExit()
	})

	UI.Closed = make(chan struct{})

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
		close(UI.Closed)
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

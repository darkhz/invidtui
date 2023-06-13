package app

import (
	"sync"

	"github.com/darkhz/invidtui/platform"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Application describes the layout of the app.
type Application struct {
	MenuLayout           *tview.Flex
	MenuButton, MenuTabs *tview.TextView

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

	UI.MenuButton, UI.MenuTabs = tview.NewTextView(), tview.NewTextView()
	UI.MenuButton.SetText("Menu")
	UI.MenuButton.SetWrap(false)
	UI.MenuButton.SetRegions(true)
	UI.MenuTabs.SetWrap(false)
	UI.MenuTabs.SetRegions(true)
	UI.MenuTabs.SetDynamicColors(true)
	UI.MenuButton.SetDynamicColors(true)
	UI.MenuTabs.SetTextAlign(tview.AlignRight)
	UI.MenuButton.SetBackgroundColor(tcell.ColorDefault)
	UI.MenuTabs.SetBackgroundColor(tcell.ColorDefault)
	UI.MenuLayout = tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(UI.MenuButton, 6, 0, false).
		AddItem(UI.MenuTabs, 0, 1, false)
	UI.MenuLayout.SetBackgroundColor(tcell.ColorDefault)

	UI.Pages = tview.NewPages()

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

	UI.Closed = make(chan struct{})

	UI.Application = tview.NewApplication()
	UI.SetAfterDrawFunc(func(screen tcell.Screen) {
		UI.resize(screen)
		suspend(screen)
	})
}

// SetPrimaryFocus sets the focus to the appropriate primitive.
func SetPrimaryFocus() {
	if currentModal != nil {
		UI.SetFocus(currentModal.Flex)
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

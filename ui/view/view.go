package view

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// View describes a view.
type View interface {
	Name() string
	Tabs() app.Tab
	Init() bool
	Exit() bool

	Keybindings(event *tcell.EventKey) *tcell.EventKey
	Primitive() tview.Primitive
}

var views []View

// SetView sets the current view.
func SetView(viewIface View, noappend ...struct{}) {
	if !viewIface.Init() {
		return
	}

	app.SetTab(viewIface.Tabs())
	app.UI.Pages.AddAndSwitchToPage(viewIface.Name(), viewIface.Primitive(), true)
	app.SetPrimaryFocus()

	for _, iface := range views {
		if iface == viewIface && noappend == nil {
			return
		}
	}
	if noappend != nil {
		return
	}

	views = append(views, viewIface)
}

// CloseView closes the current view.
func CloseView() {
	vlen := len(views)

	if !views[vlen-1].Exit() {
		return
	}

	if vlen > 1 {
		vlen--
		views = views[:vlen]
	}

	SetView(views[vlen-1], struct{}{})

	app.SetPrimaryFocus()
}

// PreviousView returns the view before the one currently displayed.
func PreviousView() View {
	if len(views) < 2 {
		return nil
	}

	return views[len(views)-2]
}

// GetCurrentView returns the current view.
func GetCurrentView() View {
	return views[len(views)-1]
}

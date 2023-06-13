package view

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
)

// View describes a view.
type View interface {
	Name() string
	Tabs() app.Tab
	Init() bool
	Exit() bool

	Primitive() tview.Primitive
}

var views []View

// SetView sets the current view.
func SetView(viewIface View, noappend ...struct{}) {
	if !viewIface.Init() {
		return
	}

	for _, iface := range views {
		if iface == viewIface && noappend == nil {
			return
		}
	}

	app.SetTab(viewIface.Tabs())
	app.UI.Pages.AddAndSwitchToPage(viewIface.Name(), viewIface.Primitive(), true)
	app.SetPrimaryFocus()

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
}

// GetCurrentView returns the current view.
func GetCurrentView() View {
	return views[len(views)-1]
}

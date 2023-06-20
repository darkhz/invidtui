package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/view"
)

var downloadItems = &app.Menu{
	Title: "Downloads",
	Options: []*app.MenuOption{
		{
			Title:  "Select option",
			MenuID: "Select",
		},
		{
			Title:   "Cancel Download",
			MenuID:  "Cancel",
			Visible: downloadViewVisible,
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

func downloadViewVisible() bool {
	d := view.Downloads

	return d != view.DownloadsView{} &&
		d.Primitive().HasFocus()
}

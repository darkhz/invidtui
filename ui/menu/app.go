package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/view"
)

var appItems = &app.Menu{
	Title: "App",
	Options: []*app.MenuOption{
		{
			Title:  "Open Dashboard",
			MenuID: "Dashboard",
		},
		{
			Title:  "Cancel Loading",
			MenuID: "Cancel",
		},
		{
			Title:  "Suspend",
			MenuID: "Suspend",
		},
		{
			Title:   "Show Downloads",
			MenuID:  "DownloadView",
			Visible: downloadView,
		},
		{
			Title:   "Download Options",
			MenuID:  "DownloadOptions",
			Visible: downloadOptions,
		},
		{
			Title:  "List Instances",
			MenuID: "InstancesList",
		},

		{
			Title:  "Quit",
			MenuID: "Quit",
		},
	},
}

func downloadView() bool {
	d := view.Downloads

	return d != view.DownloadsView{} &&
		!d.Primitive().HasFocus()
}

func downloadOptions() bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "video"
}

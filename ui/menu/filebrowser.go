package menu

import "github.com/darkhz/invidtui/ui/app"

var fileBrowserItems = &app.Menu{
	Title: "Files",
	Options: []*app.MenuOption{
		{
			Title:  "Select dir",
			MenuID: "CDFwd",
		},
		{
			Title:  "Go back",
			MenuID: "CDBack",
		},
		{
			Title:  "Toggle hidden",
			MenuID: "ToggleHidden",
		},
	},
}

package menu

import "github.com/darkhz/invidtui/ui/app"

var queueItems = &app.Menu{
	Title: "Queue",
	Options: []*app.MenuOption{
		{
			Title:  "Play/Replace",
			MenuID: "Play",
		},
		{
			Title:  "Save Queue",
			MenuID: "Save",
		},
		{
			Title:  "Append To Queue",
			MenuID: "Append",
		},
		{
			Title:  "Delete",
			MenuID: "Delete",
		},
		{
			Title:  "Move",
			MenuID: "Move",
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

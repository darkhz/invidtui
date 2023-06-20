package menu

import "github.com/darkhz/invidtui/ui/app"

var commentItems = &app.Menu{
	Title: "Comments",
	Options: []*app.MenuOption{
		{
			Title:  "Expand replies",
			MenuID: "Replies",
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

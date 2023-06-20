package menu

import (
	"github.com/darkhz/invidtui/ui/app"
)

var startItems = &app.Menu{
	Title: "Start",
	Options: []*app.MenuOption{
		{
			Title:  "Search",
			MenuID: "Search",
		},
	},
}

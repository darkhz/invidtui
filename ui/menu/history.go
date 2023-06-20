package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/player"
)

var historyItems = &app.Menu{
	Title: "History",
	Options: []*app.MenuOption{
		{
			Title:   "Search history",
			MenuID:  "Query",
			Visible: historyInputFocused,
		},
		{
			Title:   "View Channel Videos",
			MenuID:  "ChannelVideos",
			Visible: isVideo,
		},
		{
			Title:   "View Channel Playlists",
			MenuID:  "ChannelPlaylists",
			Visible: isVideo,
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

func historyInputFocused() bool {
	return !player.IsHistoryInputFocused()
}

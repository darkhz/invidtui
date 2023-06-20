package menu

import "github.com/darkhz/invidtui/ui/app"

var channelItems = &app.Menu{
	Title: "Channel",
	Options: []*app.MenuOption{
		{
			Title:  "Switch page",
			MenuID: "Switch",
		},
		{
			Title:  "Load more",
			MenuID: "LoadMore",
		},
		{
			Title:  "Search",
			MenuID: "Query",
		},
		{
			Title:   "View Playlist",
			MenuID:  "Playlist",
			Visible: isPlaylist,
		},
		{
			Title:   "Add To",
			MenuID:  "AddTo",
			Visible: isVideoOrPlaylist,
		},
		{
			Title:   "Show Comments",
			MenuID:  "Comments",
			Visible: isVideo,
		},
		{
			Title:   "Show Link",
			MenuID:  "Link",
			Visible: isVideo,
		},
		{
			Title:   "Download video",
			MenuID:  "DownloadOptions",
			Visible: isVideo,
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

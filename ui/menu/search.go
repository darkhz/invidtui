package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/view"
)

var searchItems = &app.Menu{
	Title: "Search",
	Options: []*app.MenuOption{
		{
			Title:   "Start Search",
			MenuID:  "Start",
			Visible: searchInputFocused,
		},
		{
			Title:   "Query",
			MenuID:  "Query",
			Visible: searchTableFocused,
		},
		{
			Title:   "Load More",
			MenuID:  "Start",
			Visible: searchTableFocused,
		},
		{
			Title:   "Switch Search Mode",
			MenuID:  "SwitchMode",
			Visible: searchInputFocused,
		},
		{
			Title:   "Get Suggestions",
			MenuID:  "Suggestions",
			Visible: searchInputFocused,
		},
		{
			Title:   "Set Search Parameters",
			MenuID:  "Parameters",
			Visible: searchInputFocused,
		},
		{
			Title:   "View Playlist",
			MenuID:  "Playlist",
			Visible: isPlaylist,
		},
		{
			Title:   "View Channel Videos",
			MenuID:  "ChannelVideos",
			Visible: isChannel,
		},
		{
			Title:   "View Channel Playlist",
			MenuID:  "ChannelPlaylists",
			Visible: isChannel,
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
			Title:   "Add Video To",
			MenuID:  "AddVideo",
			Visible: isVideo,
		},
		{
			Title:   "Download Video",
			MenuID:  "DownloadOptions",
			Visible: isVideo,
		},
	},
}

func searchInputFocused() bool {
	return app.UI.Status.InputField.HasFocus()
}

func searchTableFocused() bool {
	return view.Search.Primitive().HasFocus()
}

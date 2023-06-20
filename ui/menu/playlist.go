package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/view"
)

var playlistItems = &app.Menu{
	Title: "Playlist",
	Options: []*app.MenuOption{
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
			Title:   "Add To Playlist",
			MenuID:  "AddToPlaylist",
			Visible: playlistAddTo,
		},
		{
			Title:   "Remove From Playlist",
			MenuID:  "RemoveFromPlaylist",
			Visible: playlistRemoveFrom,
		},
		{
			Title:  "Load More",
			MenuID: "LoadMore",
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

func playlistAddTo() bool {
	return isVideo() && !view.Dashboard.IsFocused()
}

func playlistRemoveFrom() bool {
	prev := view.PreviousView()
	if prev == nil {
		return false
	}

	return isVideo() && prev.Name() == view.Dashboard.Name()
}

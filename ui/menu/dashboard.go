package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/view"
)

var dashboardItems = &app.Menu{
	Title: "Dashboard",
	Options: []*app.MenuOption{
		{
			Title:   "Switch page",
			MenuID:  "Switch",
			Visible: isDashboardFocused,
		},
		{
			Title:   "Reload Dashboard",
			MenuID:  "Reload",
			Visible: isDashboardFocused,
		},
		{
			Title:   "Load More",
			MenuID:  "LoadMore",
			Visible: isDashboardFocused,
		},
		{
			Title:   "Add To",
			MenuID:  "AddVideo",
			Visible: isDashboardVideo,
		},
		{
			Title:   "Show Comments",
			MenuID:  "Comments",
			Visible: isDashboardVideo,
		},
		{
			Title:   "Show Playlist",
			MenuID:  "Playlist",
			Visible: isDashboardPlaylist,
		},
		{
			Title:   "Create Playlist",
			MenuID:  "Create",
			Visible: createPlaylist,
		},
		{
			Title:   "Edit playlist",
			MenuID:  "Edit",
			Visible: editPlaylist,
		},
		{
			Title:   "Show Channel videos",
			MenuID:  "ChannelVideos",
			Visible: isDashboardSubscription,
		},
		{
			Title:   "Show Channel playlists",
			MenuID:  "ChannelPlaylists",
			Visible: isDashboardSubscription,
		},
		{
			Title:   "Delete",
			MenuID:  "Remove",
			Visible: isRemovable,
		},
		{
			Title:  "Exit",
			MenuID: "Exit",
		},
	},
}

func isDashboardFocused() bool {
	focused := view.Dashboard.IsFocused()
	if focused {
		tabs := view.Dashboard.Tabs()

		return focused && tabs.Selected != "auth"
	}

	return false
}

func isDashboardVideo() bool {
	return isDashboardFocused() && isVideo()
}

func isDashboardPlaylist() bool {
	return isDashboardFocused() && isPlaylist()
}

func createPlaylist() bool {
	return isDashboardFocused() && view.Dashboard.CurrentPage() == "playlists"
}

func editPlaylist() bool {
	return isDashboardFocused() && isPlaylist()
}

func isDashboardSubscription() bool {
	return isDashboardFocused() && isChannel()
}

func isRemovable() bool {
	return isDashboardPlaylist() || isDashboardSubscription()
}

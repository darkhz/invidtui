package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/player"
	"github.com/darkhz/invidtui/ui/view"
)

func add(menuType string) bool {
	switch menuType {
	case "Channel":
		return isVideoOrPlaylist(menuType)

	case "Playlist":
		return playlistAddTo(menuType)
	}

	return isVideo(menuType)
}

func remove(menuType string) bool {
	switch menuType {
	case "Playlist":
		return playlistRemoveFrom(menuType)
	}

	return isDashboardPlaylist(menuType) || isDashboardSubscription(menuType)

}

func query(menuType string) bool {
	switch menuType {
	case "History":
		return !player.IsHistoryInputFocused()

	case "Search":
		return !searchInputFocused(menuType)
	}

	return true
}

func searchInputFocused(menuType string) bool {
	return app.UI.Status.InputField.HasFocus()
}

func downloadView(menuType string) bool {
	d := view.Downloads

	return d != view.DownloadsView{} &&
		!d.Primitive().HasFocus()
}

func downloadOptions(menuType string) bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "video"
}

func isVideo(menuType string) bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "video"
}

func isPlaylist(menuType string) bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "playlist"
}

func isChannel(menuType string) bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "channel"
}

func isVideoOrPlaylist(menuType string) bool {
	return isVideo(menuType) || isPlaylist(menuType)
}

func isDashboardFocused(menuType string) bool {
	focused := view.Dashboard.IsFocused()
	if focused {
		tabs := view.Dashboard.Tabs()

		return focused && tabs.Selected != "auth"
	}

	return false
}

func isDashboardPlaylist(menuType string) bool {
	return isDashboardFocused(menuType) && isPlaylist(menuType)
}

func createPlaylist(menuType string) bool {
	return isDashboardFocused(menuType) && view.Dashboard.CurrentPage(menuType) == "playlists"
}

func editPlaylist(menuType string) bool {
	return isDashboardFocused(menuType) && isPlaylist(menuType)
}

func isDashboardSubscription(menuType string) bool {
	return isDashboardFocused(menuType) && isChannel(menuType)
}

func downloadViewVisible(menuType string) bool {
	d := view.Downloads

	return d != view.DownloadsView{} &&
		d.Primitive().HasFocus()
}

func playerQueue(menuType string) bool {
	return !player.IsQueueEmpty() && !player.IsQueueFocused()
}

func infoShown(menuType string) bool {
	return isPlaying(menuType) && player.IsInfoShown()
}

func isPlaying(menuType string) bool {
	return player.IsPlayerShown()
}

func playlistAddTo(menuType string) bool {
	return isVideo(menuType) && !view.Dashboard.IsFocused()
}

func playlistRemoveFrom(menuType string) bool {
	prev := view.PreviousView()
	if prev == nil {
		return false
	}

	return isVideo(menuType) && prev.Name() == view.Dashboard.Name()
}

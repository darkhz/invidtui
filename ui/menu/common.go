package menu

import "github.com/darkhz/invidtui/ui/app"

func isVideo() bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "video"
}

func isPlaylist() bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "playlist"
}

func isChannel() bool {
	info, err := app.FocusedTableReference()

	return err == nil && info.Type == "channel"
}

func isVideoOrPlaylist() bool {
	return isVideo() || isPlaylist()
}

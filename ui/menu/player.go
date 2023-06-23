package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/player"
)

var playerItems = &app.Menu{
	Title: "Player",
	Options: []*app.MenuOption{
		{
			Title:  "Open Playlist",
			MenuID: "Open",
		},
		{
			Title:  "Show History",
			MenuID: "History",
		},
		{
			Title:   "Show Queue",
			MenuID:  "Queue",
			Visible: playerQueue,
		},
		{
			Title:   "Track Information",
			MenuID:  "Info",
			Visible: infoShown,
		},
		{
			Title:   "Queue Audio",
			MenuID:  "QueueAudio",
			Visible: isVideo,
		},
		{
			Title:   "Queue Video",
			MenuID:  "QueueVideo",
			Visible: isVideo,
		},
		{
			Title:   "Play Audio",
			MenuID:  "PlayAudio",
			Visible: isVideo,
		},
		{
			Title:   "Play Video",
			MenuID:  "PlayVideo",
			Visible: isVideo,
		},
		{
			Title:  "Play audio from URL",
			MenuID: "AudioURL",
		},
		{
			Title:  "Play video from URL",
			MenuID: "VideoURL",
		},
	},
}

func playerQueue() bool {
	return !player.IsQueueEmpty() && !player.IsQueueFocused()
}

func infoShown() bool {
	return player.IsInfoShown()
}

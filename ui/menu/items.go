package menu

import "github.com/darkhz/invidtui/ui/app"

// Items describes the menu items for the menu names.
var Items = map[string]*app.Menu{
	"App":       appItems,
	"Start":     startItems,
	"Files":     fileBrowserItems,
	"Playlist":  playlistItems,
	"Comments":  commentItems,
	"Downloads": downloadItems,
	"Search":    searchItems,
	"Channel":   channelItems,
	"Dashboard": dashboardItems,
	"Player":    playerItems,
	"Queue":     queueItems,
	"History":   historyItems,
}

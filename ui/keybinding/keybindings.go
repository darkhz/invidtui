package keybinding

import (
	"github.com/gdamore/tcell/v2"
)

// KeyData stores the metadata for the key.
type KeyData struct {
	Title   string
	Context KeyContext
	Kb      Keybinding
	Global  bool
}

// Keybinding stores the keybinding.
type Keybinding struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

// Key describes the application keybinding type.
type Key string

// The different application keybinding types.
const (
	KeyMenu                    Key = "Menu"
	KeyCancel                  Key = "Cancel"
	KeySuspend                 Key = "Suspend"
	KeyInstancesList           Key = "InstancesList"
	KeyTheme                   Key = "Theme"
	KeyQuit                    Key = "Quit"
	KeySearchStart             Key = "SearchStart"
	KeySearchSuggestions       Key = "SearchSuggestions"
	KeySearchSwitchMode        Key = "SearchSwitchMode"
	KeySearchParameters        Key = "SearchParameters"
	KeySearchHistoryReverse    Key = "SearchHistoryReverse"
	KeySearchHistoryForward    Key = "SearchHistoryForward"
	KeySearchSuggestionReverse Key = "SearchSuggestionReverse"
	KeySearchSuggestionForward Key = "SearchSuggestionForward"
	KeyDashboard               Key = "Dashboard"
	KeyDashboardReload         Key = "DashboardReload"
	KeyDashboardCreatePlaylist Key = "DashboardCreatePlaylist"
	KeyDashboardEditPlaylist   Key = "DashboardEditPlaylist"
	KeyFilebrowserDirForward   Key = "FilebrowserDirForward"
	KeyFilebrowserDirBack      Key = "FilebrowserDirBack"
	KeyFilebrowserToggleHidden Key = "FilebrowserToggleHidden"
	KeyFilebrowserNewFolder    Key = "FilebrowserNewFolder"
	KeyFilebrowserRename       Key = "FilebrowserRename"
	KeyDownloadChangeDir       Key = "DownloadChangeDir"
	KeyDownloadView            Key = "DownloadView"
	KeyDownloadOptions         Key = "DownloadOptions"
	KeyDownloadCancel          Key = "DownloadCancel"
	KeyQueue                   Key = "Queue"
	KeyQueuePlayMove           Key = "QueuePlayMove"
	KeyQueueSave               Key = "QueueSave"
	KeyQueueAppend             Key = "QueueAppend"
	KeyQueueDelete             Key = "QueueDelete"
	KeyQueueMove               Key = "QueueMove"
	KeyQueueCancel             Key = "QueueCancel"
	KeyFetcher                 Key = "Fetcher"
	KeyFetcherReload           Key = "FetcherReload"
	KeyFetcherCancel           Key = "FetcherCancel"
	KeyFetcherReloadAll        Key = "FetcherReloadAll"
	KeyFetcherCancelAll        Key = "FetcherCancelAll"
	KeyFetcherClearCompleted   Key = "FetcherClearCompleted"
	KeyPlayerOpenPlaylist      Key = "PlayerOpenPlaylist"
	KeyPlayerHistory           Key = "PlayerHistory"
	KeyPlayerQueueAudio        Key = "PlayerQueueAudio"
	KeyPlayerQueueVideo        Key = "PlayerQueueVideo"
	KeyPlayerPlayAudio         Key = "PlayerPlayAudio"
	KeyPlayerPlayVideo         Key = "PlayerPlayVideo"
	KeyPlayerInfo              Key = "PlayerInfo"
	KeyPlayerInfoChangeQuality Key = "PlayerInfoChangeQuality"
	KeyPlayerSeekForward       Key = "PlayerSeekForward"
	KeyPlayerSeekBackward      Key = "PlayerSeekBackward"
	KeyPlayerStop              Key = "PlayerStop"
	KeyPlayerToggleLoop        Key = "PlayerToggleLoop"
	KeyPlayerToggleShuffle     Key = "PlayerToggleShuffle"
	KeyPlayerToggleMute        Key = "PlayerToggleMute"
	KeyPlayerTogglePlay        Key = "PlayerTogglePlay"
	KeyPlayerPrev              Key = "PlayerPrev"
	KeyPlayerNext              Key = "PlayerNext"
	KeyPlayerVolumeIncrease    Key = "PlayerVolumeIncrease"
	KeyPlayerVolumeDecrease    Key = "PlayerVolumeDecrease"
	KeyPlayerInfoScrollUp      Key = "PlayerInfoScrollUp"
	KeyPlayerInfoScrollDown    Key = "PlayerInfoScrollDown"
	KeyComments                Key = "Comments"
	KeyCommentReplies          Key = "CommentReplies"
	KeySwitchTab               Key = "SwitchTab"
	KeyPlaylist                Key = "Playlist"
	KeyPlaylistSave            Key = "PlaylistSave"
	KeyChannelVideos           Key = "ChannelVideos"
	KeyChannelPlaylists        Key = "ChannelPlaylists"
	KeyAudioURL                Key = "AudioURL"
	KeyQuery                   Key = "Query"
	KeyVideoURL                Key = "VideoURL"
	KeyLink                    Key = "Link"
	KeyAdd                     Key = "Add"
	KeyRemove                  Key = "Remove"
	KeySelect                  Key = "Select"
	KeyLoadMore                Key = "LoadMore"
	KeyClose                   Key = "Close"
)

// KeyContext describes the context where the keybinding is
// supposed to be applied in.
type KeyContext string

// The different context types for keybindings.
const (
	KeyContextApp       KeyContext = "App"
	KeyContextPlayer    KeyContext = "Player"
	KeyContextCommon    KeyContext = "Common"
	KeyContextSearch    KeyContext = "Search"
	KeyContextDashboard KeyContext = "Dashboard"
	KeyContextFiles     KeyContext = "Files"
	KeyContextDownloads KeyContext = "Downloads"
	KeyContextQueue     KeyContext = "Queue"
	KeyContextFetcher   KeyContext = "Fetcher"
	KeyContextComments  KeyContext = "Comments"
	KeyContextStart     KeyContext = "Start"
	KeyContextPlaylist  KeyContext = "Playlist"
	KeyContextChannel   KeyContext = "Channel"
	KeyContextHistory   KeyContext = "History"
)

var (
	// OperationKeys matches the operation name (or the menu ID) with the keybinding.
	OperationKeys = map[Key]*KeyData{
		KeyMenu: {
			Title:   "Menu",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyRune, 'm', tcell.ModAlt},
			Global:  true,
		},
		KeySuspend: {
			Title:   "Suspend",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyCtrlZ, ' ', tcell.ModCtrl},
			Global:  true,
		},
		KeyCancel: {
			Title:   "Cancel Loading",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyCtrlX, ' ', tcell.ModCtrl},
			Global:  true,
		},
		KeyTheme: {
			Title:   "Apply Theme",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyRune, 'T', tcell.ModNone},
			Global:  true,
		},
		KeyInstancesList: {
			Title:   "List Instances",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyRune, 'o', tcell.ModNone},
			Global:  true,
		},
		KeyQuit: {
			Title:   "Quit",
			Context: KeyContextApp,
			Kb:      Keybinding{tcell.KeyRune, 'Q', tcell.ModNone},
			Global:  true,
		},
		KeySearchStart: {
			Title:   "Start Search",
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		KeySearchSuggestions: {
			Title:   "Get Suggestions",
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyTab, ' ', tcell.ModNone},
		},
		KeySearchSwitchMode: {
			Title:   "Switch Search Mode",
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyCtrlE, ' ', tcell.ModCtrl},
		},
		KeySearchParameters: {
			Title:   "Set Search Parameters",
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyRune, 'e', tcell.ModAlt},
		},
		KeySearchHistoryReverse: {
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModNone},
		},
		KeySearchHistoryForward: {
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModNone},
		},
		KeySearchSuggestionReverse: {
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModCtrl},
		},
		KeySearchSuggestionForward: {
			Context: KeyContextSearch,
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModCtrl},
		},
		KeyDashboard: {
			Title:   "Dashboard",
			Context: KeyContextDashboard,
			Kb:      Keybinding{tcell.KeyCtrlD, ' ', tcell.ModCtrl},
		},
		KeyDashboardReload: {
			Title:   "Reload Dashboard",
			Context: KeyContextDashboard,
			Kb:      Keybinding{tcell.KeyCtrlT, ' ', tcell.ModCtrl},
		},
		KeyDashboardCreatePlaylist: {
			Title:   "Create Playlist",
			Context: KeyContextDashboard,
			Kb:      Keybinding{tcell.KeyRune, 'c', tcell.ModNone},
		},
		KeyDashboardEditPlaylist: {
			Title:   "Edit playlist",
			Context: KeyContextDashboard,
			Kb:      Keybinding{tcell.KeyRune, 'e', tcell.ModNone},
		},
		KeyFilebrowserDirForward: {
			Title:   "Go forward",
			Context: KeyContextFiles,
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModCtrl},
		},
		KeyFilebrowserDirBack: {
			Title:   "Go back",
			Context: KeyContextFiles,
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModCtrl},
		},
		KeyFilebrowserToggleHidden: {
			Title:   "Toggle hidden",
			Context: KeyContextFiles,
			Kb:      Keybinding{tcell.KeyCtrlG, ' ', tcell.ModCtrl},
		},
		KeyFilebrowserNewFolder: {
			Title:   "New folder",
			Context: KeyContextFiles,
			Kb:      Keybinding{tcell.KeyCtrlN, ' ', tcell.ModCtrl},
		},
		KeyFilebrowserRename: {
			Title:   "Rename",
			Context: KeyContextFiles,
			Kb:      Keybinding{tcell.KeyCtrlB, ' ', tcell.ModCtrl},
		},
		KeyDownloadChangeDir: {
			Title:   "Change download directory",
			Context: KeyContextDownloads,
			Kb:      Keybinding{tcell.KeyRune, 'Y', tcell.ModAlt},
		},
		KeyDownloadView: {
			Title:   "Show Downloads",
			Context: KeyContextDownloads,
			Kb:      Keybinding{tcell.KeyRune, 'Y', tcell.ModNone},
		},
		KeyDownloadOptions: {
			Title:   "Download Video",
			Context: KeyContextDownloads,
			Kb:      Keybinding{tcell.KeyRune, 'y', tcell.ModNone},
		},
		KeyDownloadCancel: {
			Title:   "Cancel Download",
			Context: KeyContextDownloads,
			Kb:      Keybinding{tcell.KeyRune, 'x', tcell.ModNone},
		},
		KeyQueue: {
			Title:   "Show Queue",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyRune, 'q', tcell.ModNone},
		},
		KeyQueuePlayMove: {
			Title:   "Play/Replace",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		KeyQueueSave: {
			Title:   "Save Queue",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyCtrlS, ' ', tcell.ModCtrl},
		},
		KeyQueueAppend: {
			Title:   "Append To Queue",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyCtrlA, ' ', tcell.ModCtrl},
		},
		KeyQueueDelete: {
			Title:   "Delete",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyRune, 'd', tcell.ModNone},
		},
		KeyQueueMove: {
			Title:   "Move",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyRune, 'M', tcell.ModNone},
		},
		KeyQueueCancel: {
			Title:   "Cancel Loading",
			Context: KeyContextQueue,
			Kb:      Keybinding{tcell.KeyRune, 'X', tcell.ModNone},
		},
		KeyFetcher: {
			Title:   "Show Media Fetcher",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'f', tcell.ModNone},
		},
		KeyFetcherReload: {
			Title:   "Reload",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'e', tcell.ModNone},
		},
		KeyFetcherCancel: {
			Title:   "Cancel",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'x', tcell.ModNone},
		},
		KeyFetcherReloadAll: {
			Title:   "Reload All",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'E', tcell.ModNone},
		},
		KeyFetcherCancelAll: {
			Title:   "Cancel All",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'X', tcell.ModNone},
		},
		KeyFetcherClearCompleted: {
			Title:   "Clear",
			Context: KeyContextFetcher,
			Kb:      Keybinding{tcell.KeyRune, 'c', tcell.ModNone},
		},
		KeyPlayerOpenPlaylist: {
			Title:   "Open Playlist",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyCtrlO, ' ', tcell.ModCtrl},
			Global:  true,
		},
		KeyPlayerHistory: {
			Title:   "Show History",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'h', tcell.ModAlt},
			Global:  true,
		},
		KeyPlayerQueueAudio: {
			Title:   "Queue Audio",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'a', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerQueueVideo: {
			Title:   "Queue Video",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'v', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerPlayAudio: {
			Title:   "Play Audio",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'A', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerPlayVideo: {
			Title:   "Play Video",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'V', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerInfo: {
			Title:   "Track Information",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, ' ', tcell.ModAlt},
			Global:  true,
		},
		KeyPlayerInfoChangeQuality: {
			Title:   "Change Image Quality",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, ':', tcell.ModAlt},
			Global:  true,
		},
		KeyPlayerSeekForward: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRight, ' ', tcell.ModCtrl},
			Global:  true,
		},
		KeyPlayerSeekBackward: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyLeft, ' ', tcell.ModCtrl},
			Global:  true,
		},
		KeyPlayerStop: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'S', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerToggleLoop: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'l', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerToggleShuffle: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 's', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerToggleMute: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'm', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerTogglePlay: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, ' ', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerPrev: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, '<', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerNext: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, '>', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerVolumeIncrease: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, '=', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerVolumeDecrease: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, '-', tcell.ModNone},
			Global:  true,
		},
		KeyPlayerInfoScrollUp: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModCtrl | tcell.ModAlt},
			Global:  true,
		},
		KeyPlayerInfoScrollDown: {
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModCtrl | tcell.ModAlt},
			Global:  true,
		},
		KeyAudioURL: {
			Title:   "Play audio from URL",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'b', tcell.ModNone},
		},
		KeyVideoURL: {
			Title:   "Play video from URL",
			Context: KeyContextPlayer,
			Kb:      Keybinding{tcell.KeyRune, 'B', tcell.ModNone},
		},
		KeyPlaylistSave: {
			Title:   "Save Playlist",
			Context: KeyContextPlaylist,
			Kb:      Keybinding{tcell.KeyCtrlS, ' ', tcell.ModCtrl},
		},
		KeyComments: {
			Title:   "Show Comments",
			Context: KeyContextComments,
			Kb:      Keybinding{tcell.KeyRune, 'C', tcell.ModNone},
		},
		KeyCommentReplies: {
			Title:   "Expand replies",
			Context: KeyContextComments,
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		KeySwitchTab: {
			Title:   "Switch tab",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyTab, ' ', tcell.ModNone},
		},
		KeyPlaylist: {
			Title:   "Show Playlist",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, 'i', tcell.ModNone},
		},
		KeyChannelVideos: {
			Title:   "Show Channel videos",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, 'u', tcell.ModNone},
		},
		KeyChannelPlaylists: {
			Title:   "Show Channel playlists",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, 'U', tcell.ModNone},
		},
		KeyQuery: {
			Title:   "Query",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, '/', tcell.ModNone},
		},
		KeyLink: {
			Title:   "Show Link",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, ';', tcell.ModNone},
		},
		KeyAdd: {
			Title:   "Add",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, '+', tcell.ModNone},
		},
		KeyRemove: {
			Title:   "Remove",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, '_', tcell.ModNone},
		},
		KeyLoadMore: {
			Title:   "Load more",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyRune, '.', tcell.ModNone},
		},
		KeySelect: {
			Title:   "Select",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		KeyClose: {
			Title:   "Close page",
			Context: KeyContextCommon,
			Kb:      Keybinding{tcell.KeyEscape, ' ', tcell.ModNone},
		},
	}

	// Keys match the keybinding to the key type.
	Keys map[KeyContext]map[Keybinding]Key

	translateKeys = map[string]string{
		"Pgup":      "PgUp",
		"Pgdn":      "PgDn",
		"Pageup":    "PgUp",
		"Pagedown":  "PgDn",
		"Upright":   "UpRight",
		"Downright": "DownRight",
		"Upleft":    "UpLeft",
		"Downleft":  "DownLeft",
		"Prtsc":     "Print",
		"Backspace": "Backspace2",
	}
)

// OperationData returns the key data associated with
// the provided keyID and operation name.
func OperationData(operation Key) *KeyData {
	return OperationKeys[operation]
}

// KeyOperation returns the operation name for the provided keyID
// and the keyboard event.
func KeyOperation(event *tcell.EventKey, keyContexts ...KeyContext) Key {
	if Keys == nil {
		Keys = make(map[KeyContext]map[Keybinding]Key)
		for keyName, key := range OperationKeys {
			if Keys[key.Context] == nil {
				Keys[key.Context] = make(map[Keybinding]Key)
			}

			Keys[key.Context][key.Kb] = keyName
		}
	}

	ch := event.Rune()
	if event.Key() != tcell.KeyRune {
		ch = ' '
	}

	kb := Keybinding{event.Key(), ch, event.Modifiers()}

	for _, contexts := range [][]KeyContext{
		keyContexts,
		{
			KeyContextApp,
			KeyContextCommon,
			KeyContextPlayer,
		},
	} {
		for _, context := range contexts {
			if operation, ok := Keys[context][kb]; ok {
				return operation
			}
		}
	}

	return ""
}

// KeyName formats and returns the key's name.
func KeyName(kb Keybinding) string {
	if kb.Key == tcell.KeyRune {
		keyname := string(kb.Rune)
		if kb.Rune == ' ' {
			keyname = "Space"
		}

		if kb.Mod&tcell.ModAlt != 0 {
			keyname = "Alt+" + keyname
		}

		return keyname
	}

	return tcell.NewEventKey(kb.Key, kb.Rune, kb.Mod).Name()
}

// KeyEvent returns an event for the provided key.
func KeyEvent(k Key) *tcell.EventKey {
	kb := OperationData(k).Kb
	return tcell.NewEventKey(kb.Key, kb.Rune, kb.Mod)
}

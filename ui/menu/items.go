package menu

import (
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
)

// Items describes the menu items.
var Items = &app.MenuData{
	Items: map[keybinding.KeyContext][]keybinding.Key{
		keybinding.KeyContextApp: {
			keybinding.KeyDashboard,
			keybinding.KeyCancel,
			keybinding.KeySuspend,
			keybinding.KeyDownloadView,
			keybinding.KeyDownloadOptions,
			keybinding.KeyInstancesList,
			keybinding.KeyTheme,
			keybinding.KeyQuit,
		},
		keybinding.KeyContextStart: {
			keybinding.KeyQuery,
		},
		keybinding.KeyContextFiles: {
			keybinding.KeyFilebrowserDirForward,
			keybinding.KeyFilebrowserDirBack,
			keybinding.KeyFilebrowserToggleHidden,
			keybinding.KeyFilebrowserNewFolder,
			keybinding.KeyFilebrowserRename,
			keybinding.KeyClose,
		},
		keybinding.KeyContextPlaylist: {
			keybinding.KeyComments,
			keybinding.KeyLink,
			keybinding.KeyAdd,
			keybinding.KeyRemove,
			keybinding.KeyLoadMore,
			keybinding.KeyPlaylistSave,
			keybinding.KeyDownloadOptions,
			keybinding.KeyClose,
		},
		keybinding.KeyContextComments: {
			keybinding.KeyCommentReplies,
			keybinding.KeyClose,
		},
		keybinding.KeyContextDownloads: {
			keybinding.KeySelect,
			keybinding.KeyDownloadChangeDir,
			keybinding.KeyDownloadCancel,
			keybinding.KeyClose,
		},
		keybinding.KeyContextSearch: {
			keybinding.KeySearchStart,
			keybinding.KeyQuery,
			keybinding.KeyLoadMore,
			keybinding.KeySearchSwitchMode,
			keybinding.KeySearchSuggestions,
			keybinding.KeySearchParameters,
			keybinding.KeyComments,
			keybinding.KeyLink,
			keybinding.KeyPlaylist,
			keybinding.KeyChannelVideos,
			keybinding.KeyChannelPlaylists,
			keybinding.KeyChannelReleases,
			keybinding.KeyAdd,
			keybinding.KeyDownloadOptions,
		},
		keybinding.KeyContextChannel: {
			keybinding.KeySwitchTab,
			keybinding.KeyLoadMore,
			keybinding.KeyQuery,
			keybinding.KeyPlaylist,
			keybinding.KeyAdd,
			keybinding.KeyComments,
			keybinding.KeyLink,
			keybinding.KeyDownloadOptions,
			keybinding.KeyClose,
		},
		keybinding.KeyContextDashboard: {
			keybinding.KeySwitchTab,
			keybinding.KeyDashboardReload,
			keybinding.KeyLoadMore,
			keybinding.KeyAdd,
			keybinding.KeyComments,
			keybinding.KeyPlaylist,
			keybinding.KeyDashboardCreatePlaylist,
			keybinding.KeyDashboardEditPlaylist,
			keybinding.KeyChannelVideos,
			keybinding.KeyChannelPlaylists,
			keybinding.KeyChannelReleases,
			keybinding.KeyRemove,
			keybinding.KeyClose,
		},
		keybinding.KeyContextPlayer: {
			keybinding.KeyPlayerOpenPlaylist,
			keybinding.KeyQueue,
			keybinding.KeyFetcher,
			keybinding.KeyPlayerHistory,
			keybinding.KeyPlayerInfo,
			keybinding.KeyPlayerInfoChangeQuality,
			keybinding.KeyPlayerQueueAudio,
			keybinding.KeyPlayerQueueVideo,
			keybinding.KeyPlayerPlayAudio,
			keybinding.KeyPlayerPlayVideo,
			keybinding.KeyAudioURL,
			keybinding.KeyVideoURL,
		},
		keybinding.KeyContextQueue: {
			keybinding.KeyQueuePlayMove,
			keybinding.KeyQueueSave,
			keybinding.KeyQueueAppend,
			keybinding.KeyPlayerQueueAudio,
			keybinding.KeyPlayerQueueVideo,
			keybinding.KeyQueueDelete,
			keybinding.KeyQueueMove,
			keybinding.KeyQueueCancel,
			keybinding.KeyComments,
			keybinding.KeyClose,
		},
		keybinding.KeyContextFetcher: {
			keybinding.KeyFetcherReload,
			keybinding.KeyFetcherCancel,
			keybinding.KeyFetcherReloadAll,
			keybinding.KeyFetcherCancelAll,
			keybinding.KeyFetcherClearCompleted,
		},
		keybinding.KeyContextHistory: {
			keybinding.KeyQuery,
			keybinding.KeyChannelVideos,
			keybinding.KeyChannelPlaylists,
			keybinding.KeyChannelReleases,
			keybinding.KeyComments,
			keybinding.KeyClose,
		},
	},
	Visible: map[keybinding.Key]func(menuType string) bool{
		keybinding.KeyDownloadChangeDir:       downloadView,
		keybinding.KeyDownloadView:            downloadView,
		keybinding.KeyDownloadOptions:         downloadOptions,
		keybinding.KeyComments:                isVideo,
		keybinding.KeyLink:                    isVideo,
		keybinding.KeyDownloadCancel:          downloadViewVisible,
		keybinding.KeyAdd:                     add,
		keybinding.KeyRemove:                  remove,
		keybinding.KeyPlaylist:                isPlaylist,
		keybinding.KeyChannelVideos:           isVideoOrChannel,
		keybinding.KeyChannelPlaylists:        isVideoOrChannel,
		keybinding.KeyChannelReleases:         isVideoOrChannel,
		keybinding.KeyQuery:                   query,
		keybinding.KeySearchStart:             searchInputFocused,
		keybinding.KeySearchSwitchMode:        searchInputFocused,
		keybinding.KeySearchSuggestions:       searchInputFocused,
		keybinding.KeySearchParameters:        searchInputFocused,
		keybinding.KeyDashboardReload:         isDashboardFocused,
		keybinding.KeyDashboardCreatePlaylist: createPlaylist,
		keybinding.KeyDashboardEditPlaylist:   editPlaylist,
		keybinding.KeyQueue:                   playerQueue,
		keybinding.KeyPlayerInfo:              isPlaying,
		keybinding.KeyPlayerInfoChangeQuality: infoShown,
		keybinding.KeyPlayerQueueAudio:        queueMedia,
		keybinding.KeyPlayerQueueVideo:        queueMedia,
		keybinding.KeyPlayerPlayAudio:         isVideo,
		keybinding.KeyPlayerPlayVideo:         isVideo,
	},
}

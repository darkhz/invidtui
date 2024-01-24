package theme

import (
	"github.com/darkhz/tview"
)

// ThemeProperty describes a theme property.
type ThemeProperty struct {
	Version int

	Context ThemeContext
	Item    ThemeItem
}

// ThemeContext describes the type of context to apply the color into.
type ThemeContext string

// The different context types for themes.
const (
	ThemeContextApp       ThemeContext = "App"
	ThemeContextMenu      ThemeContext = "Menu"
	ThemeContextStatusBar ThemeContext = "StatusBar"
	ThemeContextInstances ThemeContext = "Instances"
	ThemeContextLinks     ThemeContext = "Links"

	ThemeContextPlayerInfo ThemeContext = "PlayerInfo"
	ThemeContextPlayer     ThemeContext = "Player"
	ThemeContextSearch     ThemeContext = "Search"
	ThemeContextDashboard  ThemeContext = "Dashboard"
	ThemeContextFiles      ThemeContext = "Files"
	ThemeContextDownloads  ThemeContext = "Downloads"
	ThemeContextQueue      ThemeContext = "Queue"
	ThemeContextFetcher    ThemeContext = "Fetcher"
	ThemeContextComments   ThemeContext = "Comments"
	ThemeContextStart      ThemeContext = "Start"
	ThemeContextPlaylist   ThemeContext = "Playlist"
	ThemeContextChannel    ThemeContext = "Channel"
	ThemeContextHistory    ThemeContext = "History"
)

// ThemeItem describes a theme item.
type ThemeItem string

// The different item types for themes.
const (
	ThemeText            ThemeItem = "Text"
	ThemeBorder          ThemeItem = "Border"
	ThemeTitle           ThemeItem = "Title"
	ThemeTabs            ThemeItem = "Tabs"
	ThemeBackground      ThemeItem = "Background"
	ThemePopupBackground ThemeItem = "PopupBackground"

	ThemeInputLabel ThemeItem = "InputLabel"
	ThemeInputField ThemeItem = "InputField"

	ThemeListLabel   ThemeItem = "ListLabel"
	ThemeListField   ThemeItem = "ListField"
	ThemeListOptions ThemeItem = "ListOptions"

	ThemeFormButton  ThemeItem = "FormButton"
	ThemeFormLabel   ThemeItem = "FormLabel"
	ThemeFormField   ThemeItem = "FormField"
	ThemeFormOptions ThemeItem = "FormOptions"

	ThemeSelector           ThemeItem = "Selector"
	ThemeNormalModeSelector ThemeItem = "NormalModeSelector"
	ThemeMoveModeSelector   ThemeItem = "MoveModeSelector"

	ThemeInfoMessage  ThemeItem = "InfoMessage"
	ThemeErrorMessage ThemeItem = "ErrorMessage"

	ThemeProgressBar  ThemeItem = "ProgressBar"
	ThemeProgressText ThemeItem = "ProgressText"

	ThemeTagStatusBar ThemeItem = "TagStatusBar"
	ThemeTagFetching  ThemeItem = "TagFetching"
	ThemeTagLoading   ThemeItem = "TagLoading"
	ThemeTagAdding    ThemeItem = "TagAdding"
	ThemeTagStopped   ThemeItem = "TagStopped"
	ThemeTagError     ThemeItem = "TagError"
	ThemeTagPlaying   ThemeItem = "TagPlaying"
	ThemeTagChanged   ThemeItem = "TagChanged"

	ThemeName        ThemeItem = "Name"
	ThemeDescription ThemeItem = "Description"
	ThemeKeybinding  ThemeItem = "Keybinding"

	ThemeDirectory ThemeItem = "Directory"
	ThemeFile      ThemeItem = "File"
	ThemePath      ThemeItem = "Path"

	ThemeVideo          ThemeItem = "Video"
	ThemePlaylist       ThemeItem = "Playlist"
	ThemeAuthor         ThemeItem = "Author"
	ThemeAuthorOwner    ThemeItem = "AuthorOwner"
	ThemeAuthorVerified ThemeItem = "AuthorVerified"
	ThemeTotalVideos    ThemeItem = "TotalVideos"

	ThemeShuffle       ThemeItem = "Shuffle"
	ThemeLoop          ThemeItem = "Loop"
	ThemeVolume        ThemeItem = "Volume"
	ThemeDuration      ThemeItem = "Duration"
	ThemeTotalDuration ThemeItem = "TotalDuration"
	ThemePause         ThemeItem = "Pause"
	ThemePlay          ThemeItem = "Play"
	ThemeBuffer        ThemeItem = "Buffer"
	ThemeMute          ThemeItem = "Mute"
	ThemeStop          ThemeItem = "Stop"

	ThemeChannel     ThemeItem = "Channel"
	ThemeComment     ThemeItem = "Comment"
	ThemeViews       ThemeItem = "Views"
	ThemeLikes       ThemeItem = "Likes"
	ThemeSubscribers ThemeItem = "Subscribers"
	ThemePublished   ThemeItem = "Published"

	ThemeInstanceURI  ThemeItem = "InstanceURI"
	ThemeInvidiousURI ThemeItem = "InvidiousURI"
	ThemeYoutubeURI   ThemeItem = "YoutubeURI"

	ThemeMediaInfo       ThemeItem = "MediaInfo"
	ThemeMediaSize       ThemeItem = "MediaSize"
	ThemeMediaType       ThemeItem = "MediaType"
	ThemeVideoResolution ThemeItem = "VideoResolution"
	ThemeVideoFPS        ThemeItem = "VideoFPS"
	ThemeAudioSampleRate ThemeItem = "AudioSampleRate"
	ThemeAudioChannels   ThemeItem = "AudioChannels"
)

// ThemeScopes store the ThemeItem scopes for each ThemeContext.
var ThemeScopes = map[ThemeContext]map[ThemeItem]struct{}{
	ThemeContextApp: {
		ThemeAuthor:          struct{}{},
		ThemeBackground:      struct{}{},
		ThemeBorder:          struct{}{},
		ThemeChannel:         struct{}{},
		ThemeDescription:     struct{}{},
		ThemeDuration:        struct{}{},
		ThemeErrorMessage:    struct{}{},
		ThemeFormButton:      struct{}{},
		ThemeFormField:       struct{}{},
		ThemeFormLabel:       struct{}{},
		ThemeFormOptions:     struct{}{},
		ThemeInfoMessage:     struct{}{},
		ThemeInputField:      struct{}{},
		ThemeInputLabel:      struct{}{},
		ThemeInstanceURI:     struct{}{},
		ThemeLikes:           struct{}{},
		ThemeListField:       struct{}{},
		ThemeListLabel:       struct{}{},
		ThemeListOptions:     struct{}{},
		ThemeMediaType:       struct{}{},
		ThemePlaylist:        struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeProgressBar:     struct{}{},
		ThemeProgressText:    struct{}{},
		ThemePublished:       struct{}{},
		ThemeSelector:        struct{}{},
		ThemeSubscribers:     struct{}{},
		ThemeTabs:            struct{}{},
		ThemeTagStatusBar:    struct{}{},
		ThemeText:            struct{}{},
		ThemeTitle:           struct{}{},
		ThemeTotalDuration:   struct{}{},
		ThemeTotalVideos:     struct{}{},
		ThemeVideo:           struct{}{},
		ThemeViews:           struct{}{},
	},
	ThemeContextChannel: {
		ThemeBackground:    struct{}{},
		ThemeBorder:        struct{}{},
		ThemeDescription:   struct{}{},
		ThemeMediaType:     struct{}{},
		ThemePlaylist:      struct{}{},
		ThemeSelector:      struct{}{},
		ThemeTabs:          struct{}{},
		ThemeTitle:         struct{}{},
		ThemeTotalDuration: struct{}{},
		ThemeTotalVideos:   struct{}{},
		ThemeVideo:         struct{}{},
	},
	ThemeContextComments: {
		ThemeAuthor:          struct{}{},
		ThemeBorder:          struct{}{},
		ThemeAuthorOwner:     struct{}{},
		ThemeAuthorVerified:  struct{}{},
		ThemeComment:         struct{}{},
		ThemeLikes:           struct{}{},
		ThemePublished:       struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeText:            struct{}{},
		ThemeTitle:           struct{}{},
		ThemeVideo:           struct{}{},
	},
	ThemeContextDashboard: {
		ThemeBorder:          struct{}{},
		ThemeBackground:      struct{}{},
		ThemeChannel:         struct{}{},
		ThemeFormButton:      struct{}{},
		ThemeFormField:       struct{}{},
		ThemeFormLabel:       struct{}{},
		ThemeFormOptions:     struct{}{},
		ThemeInputField:      struct{}{},
		ThemeInputLabel:      struct{}{},
		ThemeInstanceURI:     struct{}{},
		ThemeListField:       struct{}{},
		ThemeListLabel:       struct{}{},
		ThemeListOptions:     struct{}{},
		ThemePlaylist:        struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTabs:            struct{}{},
		ThemeText:            struct{}{},
		ThemeTotalDuration:   struct{}{},
		ThemeTotalVideos:     struct{}{},
		ThemeVideo:           struct{}{},
	},
	ThemeContextDownloads: {
		ThemeAudioChannels:   struct{}{},
		ThemeAudioSampleRate: struct{}{},
		ThemeBackground:      struct{}{},
		ThemeBorder:          struct{}{},
		ThemeMediaInfo:       struct{}{},
		ThemeMediaSize:       struct{}{},
		ThemeMediaType:       struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeProgressBar:     struct{}{},
		ThemeProgressText:    struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTitle:           struct{}{},
		ThemeVideoFPS:        struct{}{},
		ThemeVideoResolution: struct{}{},
	},
	ThemeContextFetcher: {
		ThemeAuthor:          struct{}{},
		ThemeBorder:          struct{}{},
		ThemeErrorMessage:    struct{}{},
		ThemeInfoMessage:     struct{}{},
		ThemeMediaType:       struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeProgressText:    struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTagAdding:       struct{}{},
		ThemeTagError:        struct{}{},
		ThemeTagStatusBar:    struct{}{},
		ThemeTitle:           struct{}{},
		ThemeVideo:           struct{}{},
	},
	ThemeContextFiles: {
		ThemeBorder:          struct{}{},
		ThemeDirectory:       struct{}{},
		ThemeFile:            struct{}{},
		ThemeInputField:      struct{}{},
		ThemeInputLabel:      struct{}{},
		ThemePath:            struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTitle:           struct{}{},
	},
	ThemeContextHistory: {
		ThemeBorder:          struct{}{},
		ThemeInputField:      struct{}{},
		ThemeInputLabel:      struct{}{},
		ThemeMediaType:       struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTitle:           struct{}{},
		ThemeVideo:           struct{}{},
	},
	ThemeContextInstances: {
		ThemeBackground:      struct{}{},
		ThemeBorder:          struct{}{},
		ThemeInstanceURI:     struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeTagChanged:      struct{}{},
		ThemeTitle:           struct{}{},
	},
	ThemeContextLinks: {
		ThemeBorder:          struct{}{},
		ThemeInvidiousURI:    struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
		ThemeText:            struct{}{},
		ThemeTitle:           struct{}{},
		ThemeYoutubeURI:      struct{}{},
	},
	ThemeContextMenu: {
		ThemeBackground:      struct{}{},
		ThemeBorder:          struct{}{},
		ThemeDescription:     struct{}{},
		ThemeKeybinding:      struct{}{},
		ThemeName:            struct{}{},
		ThemePopupBackground: struct{}{},
		ThemeSelector:        struct{}{},
	},
	ThemeContextPlayer: {
		ThemeBackground:    struct{}{},
		ThemeBuffer:        struct{}{},
		ThemeDuration:      struct{}{},
		ThemeLoop:          struct{}{},
		ThemeMediaType:     struct{}{},
		ThemePause:         struct{}{},
		ThemePlay:          struct{}{},
		ThemeProgressBar:   struct{}{},
		ThemeSelector:      struct{}{},
		ThemeShuffle:       struct{}{},
		ThemeTitle:         struct{}{},
		ThemeTotalDuration: struct{}{},
		ThemeVolume:        struct{}{},
	},
	ThemeContextPlayerInfo: {
		ThemeAuthor:      struct{}{},
		ThemeBackground:  struct{}{},
		ThemeBorder:      struct{}{},
		ThemeDescription: struct{}{},
		ThemeLikes:       struct{}{},
		ThemeListField:   struct{}{},
		ThemeListLabel:   struct{}{},
		ThemeListOptions: struct{}{},
		ThemePublished:   struct{}{},
		ThemeSubscribers: struct{}{},
		ThemeTitle:       struct{}{},
		ThemeViews:       struct{}{},
	},
	ThemeContextPlaylist: {
		ThemeAuthor:        struct{}{},
		ThemeBackground:    struct{}{},
		ThemeBorder:        struct{}{},
		ThemeSelector:      struct{}{},
		ThemeTabs:          struct{}{},
		ThemeTotalDuration: struct{}{},
		ThemeTotalVideos:   struct{}{},
		ThemeVideo:         struct{}{},
	},
	ThemeContextQueue: {
		ThemeAuthor:             struct{}{},
		ThemeBorder:             struct{}{},
		ThemeMediaType:          struct{}{},
		ThemeMoveModeSelector:   struct{}{},
		ThemeNormalModeSelector: struct{}{},
		ThemePopupBackground:    struct{}{},
		ThemeSelector:           struct{}{},
		ThemeTagFetching:        struct{}{},
		ThemeTagLoading:         struct{}{},
		ThemeTagPlaying:         struct{}{},
		ThemeTagStopped:         struct{}{},
		ThemeTitle:              struct{}{},
		ThemeTotalDuration:      struct{}{},
		ThemeVideo:              struct{}{},
	},
	ThemeContextSearch: {
		ThemeAuthor:        struct{}{},
		ThemeBackground:    struct{}{},
		ThemeBorder:        struct{}{},
		ThemeChannel:       struct{}{},
		ThemeFormButton:    struct{}{},
		ThemeFormField:     struct{}{},
		ThemeFormLabel:     struct{}{},
		ThemeFormOptions:   struct{}{},
		ThemeInputField:    struct{}{},
		ThemeInputLabel:    struct{}{},
		ThemePlaylist:      struct{}{},
		ThemeSelector:      struct{}{},
		ThemeTabs:          struct{}{},
		ThemeTitle:         struct{}{},
		ThemeText:          struct{}{},
		ThemeTotalDuration: struct{}{},
		ThemeTotalVideos:   struct{}{},
		ThemeVideo:         struct{}{},
	},
	ThemeContextStart: {
		ThemeText:       struct{}{},
		ThemeBackground: struct{}{},
	},
	ThemeContextStatusBar: {
		ThemeBackground:   struct{}{},
		ThemeErrorMessage: struct{}{},
		ThemeInfoMessage:  struct{}{},
		ThemeInputField:   struct{}{},
		ThemeInputLabel:   struct{}{},
		ThemeTagStatusBar: struct{}{},
	},
}

// SetItem sets the ThemeItem for the ThemeProperty.
func (t ThemeProperty) SetItem(item ThemeItem) ThemeProperty {
	t.Item = item

	return t
}

// SetContext sets the ThemeContext for the ThemeProperty.
func (t ThemeProperty) SetContext(context ThemeContext) ThemeProperty {
	t.Context = context

	return t
}

// UpdateThemeVersion updates the theme version.
func UpdateThemeVersion() {
	ThemeVersion++
}

// SetThemeProperty updates the ThemeProperty's version and applies the theme for its primitive.
func SetThemeProperty(primitive tview.Primitive, property *ThemeProperty) {
	if property.Version == ThemeVersion {
		return
	}

	property.Version += 1

	applyTheme(primitive, *property)
}

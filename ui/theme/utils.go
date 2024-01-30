package theme

import (
	"fmt"
	"strings"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// DefaultThemeConfig is the default theme configuration in the HJSON format.
var DefaultThemeConfig = `# The InvidTUI 'default' theme file

/*
Documentation:
--------------
    A theme file consists of a collection of:
    - Contexts
    - Items
    - Parameters
    arranged in the HJSON format.

    Each context can have multiple items and parameters attached to it.
    Note that contexts and items are case-sensitive.

    Contexts
    --------
    The currently available contexts are:

    "App", "Channel", "Comments", "Dashboard", "Downloads",
    "Fetcher", "Files", "History", "Instances", "Links", "Menu",
    "Player", "PlayerInfo", "Playlist", "Queue", "Search", "Start", "StatusBar"

    The "App" context is a global context, where all items common to the other contexts
    can be defined within the "App" context. Or, all common items can be defined in the "App"
    context, and can be overriden from the other contexts.

    For example, consider this configuration:
    {
        App: {
            Background: bg:blue
        }
        Channel: {
            Background: bg:black
        }
    }

    This means that, for all other pages (not popups, more on that later), the background color will be blue,
    but for the channel page, the background color will be black, since it overrides the global background
    setting.

    Items
    -----
    The currently available items are:

    "AudioChannels", "AudioSampleRate", "Author", "AuthorOwner", "AuthorVerified", "Background",
    "Border", "Buffer", "Channel", "Comment", "Description", "Directory",
    "Duration", "ErrorMessage", "File", "FormButton", "FormField", "FormLabel",
    "InfoMessage", "InputField", "InputLabel", "InstanceURI", "InvidiousURI", "Keybinding",
    "Likes", "ListField", "ListLabel", "ListOptions", "Loop", "MediaInfo",
    "MediaSize", "MediaType", "MoveModeSelector", "Name", "NormalModeSelector", "Path",
    "Pause", "Play", "Playlist", "PopupBackground", "PopupBorder", "ProgressBar",
    "ProgressText", "Published", "Selector", "Shuffle", "Subscribers", "Tabs",
    "TagAdding", "TagChanged", "TagError", "TagFetching", "TagLoading", "TagPlaying",
    "TagStatusBar", "TagStopped", "Text", "Title", "TotalDuration", "TotalVideos",
    "Video", "VideoFPS", "VideoResolution", "Views", "Volume", "YoutubeURI"

    Out of these, the common items (which can be defined across all contexts) are:

    "Author", "Background", "Border", "FormButton", "Channel", "FormLabel", "FormField",
    "Description", "Duration", "ErrorMessage", "InfoMessage",
    "InputField", "InputLabel", "InstanceURI", "Likes",
    "ListField", "ListLabel", "ListOptions", "MediaType",
    "Playlist", "PopupBackground", "PopupBorder", "ProgressBar", "ProgressText",
    "Published", "Selector", "Subscribers", "Tabs", "TagStatusBar", "Text",
    "Title", "TotalDuration", "TotalVideos", "Video", "Views"

    # Parameters
    ------------
    Each item must be associated with a theme parameter.
    The syntax for a parameter is:

    attr:<attributeA>, <attributeB>; fg:<fgColor>; bg:<bgColor>

    Note that the 'attr', 'fg' and 'bg' keywords can be defined in any order.
    - Each keyword definition must be separated by a semicolon (;)
    - Every attribute definition must be separated by a comma(,)

    The available attributes are: bold, dim, italic, reverse and underline.
    For the foreground and background colors, any color/CSS name can be specified.

    For example:
    {
        App: {
            Video: attr:bold, underline; fg:yellow; bg:blue
        }
    }

    This means that, all video titles across all pages/popups will be with:
    - A yellow foreground color
    - A blue background color
    - Bold and underlined attributes

    # Background:Border and PopupBackground:PopupBorder
    ---------------------------------------------------
    A certain distinction is present when defining the 'Background' and 'PopupBackground' items.
    - 'Background' and 'Border' applies to pages only, while
    - 'PopupBackground' and 'PopupBorder' applies to popups.

    'PopupBackground' and 'PopupBorder' applies to the following contexts:
    "Files", "Fetcher", "Queue", "Instances", "Links", "Comments", "Menu" (menu options popup),
    "Downloads" (download options popup), "Search" (search parameters/suggestions popups)

    Otherwise the 'Background' and 'Border' items apply.
*/

{
  App: {
    Author: attr:bold; fg:purple
    Background: bg:black
    Border: attr:bold; fg:white
    FormButton: attr:bold; bg:blue; fg:white
    Channel: attr:bold; fg:blue
    FormField: attr:bold; bg:blue; fg:white
    FormLabel: attr:bold; fg:white
    Description: attr:bold; fg:white
    Duration: attr:bold; fg:white
    ErrorMessage: attr:bold; fg:red
    InfoMessage: attr:bold; fg:white
    InputField: bg:blue; fg:white
    InputLabel: attr:underline; fg:white
    InstanceURI: attr:bold; fg:blue
    Likes: attr:bold; fg:red
    ListField: bg:blue; fg:white
    ListLabel: attr:bold; fg:white
    ListOptions: attr:bold; fg:white
    MediaType: attr:bold; fg:pink
    Playlist: attr:bold; fg:blue
    PopupBackground: bg:black
    PopupBorder: attr:bold; fg:white
    ProgressBar: attr:bold; fg:white
    ProgressText: attr:bold; fg:white
    Published: attr:bold; fg:aqua
    Selector: attr:bold; bg:white; fg:black
    Subscribers: attr:bold; fg:purple
    Tabs: attr:bold; fg:aqua
    TagStatusBar: fg:black; bg:yellow
    Text: attr:bold; fg:white
    Title: attr:bold,underline; fg:white
    TotalDuration: attr:bold; fg:pink
    TotalVideos: attr:bold; fg:pink
    Video: attr:bold; fg:blue
    Views: attr:bold; fg:pink
  }
  Channel: {
    Description: attr:bold; fg:white
    MediaType: attr:bold; fg:pink
    Playlist: attr:bold; fg:blue
    Title: attr:bold,underline; fg:white
    TotalDuration: attr:bold; fg:pink
    TotalVideos: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Comments: {
    Author: attr:bold; fg:purple
    AuthorOwner: attr:bold; fg:plum
    AuthorVerified: attr:bold; fg:aqua
    Comment: attr:bold; fg:white
    Likes: attr:bold; fg:red
    Published: attr:bold; fg:grey
    Text: attr:bold; fg:white
    Title: attr:bold; fg:blue
    Video: attr:bold; fg:blue
  }
  Dashboard: {
    Channel: attr:bold; fg:blue
    InputField: bg:blue; fg:white
    InputLabel: attr:bold; fg:white
    InstanceURI: attr:bold; fg:blue
    ListField: bg:blue; fg:white
    ListLabel: attr:bold; fg:white
    ListOptions: attr:bold; bg:grey; fg:white
    Playlist: attr:bold; fg:blue
    Text: attr:bold; fg:white
    TotalDuration: attr:bold; fg:pink
    TotalVideos: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Downloads: {
    AudioChannels: attr:bold; fg:grey
    AudioSampleRate: attr:bold; fg:lightpink
    MediaInfo: attr:bold; fg:red
    MediaSize: attr:bold; fg:blue
    MediaType: attr:bold; fg:pink
    PopupBackground: bg:black
    ProgressBar: attr:bold; fg:white
    ProgressText: attr:bold; fg:white
    VideoFPS: attr:bold; fg:yellow
    VideoResolution: attr:bold; fg:green
  }
  Fetcher: {
    Author: attr:bold; fg:purple
    ErrorMessage: attr:bold; fg:red
    InfoMessage: attr:bold; fg:white
    MediaType: attr:bold; fg:pink
    ProgressText: attr:bold; fg:white
    TagAdding: attr:bold; bg:yellow; fg:black
    TagError: attr:bold; bg:red; fg:white
    TagStatusBar: attr:bold; fg:white
    Video: attr:bold; fg:blue
  }
  Files: {
    Directory: attr:bold; fg:blue
    File: attr:bold; fg:white
    InputField: bg:black
    InputLabel: attr:bold; fg:white
    Path: attr:bold,underline; fg:white
    PopupBackground: bg:black
    Title: attr:underline; fg:white
  }
  History: {
    InputField: bg:blue; fg:white
    InputLabel: attr:bold; fg:white
    MediaType: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Instances: {
    Background: bg:black
    InstanceURI: attr:bold; fg:blue
    PopupBackground: bg:black
    TagChanged: attr:bold; fg:white
    Title: attr:bold; fg:white
  }
  Links: {
    InvidiousURI: attr:bold; fg:blue
    PopupBackground: bg:black
    Text: attr:bold,underline; fg:white
    YoutubeURI: attr:bold; fg:red
  }
  Menu: {
    Background: bg:black
    Description: attr:bold; fg:white
    Keybinding: attr:bold; fg:white
    Name: attr:bold; fg:white
    PopupBackground: bg:black
  }
  Player: {
    Buffer: attr:bold; fg:white
    Duration: attr:bold; fg:white
    Loop: attr:bold; fg:white
    MediaType: attr:bold; fg:pink
    Pause: attr:bold; fg:white
    Play: attr:bold; fg:white
    ProgressBar: attr:bold; fg:white
    Shuffle: attr:bold; fg:white
    Title: attr:bold; fg:white
    TotalDuration: attr:bold; fg:white
    Volume: attr:bold; fg:white
  }
  PlayerInfo: {
    Author: attr:bold,underline; fg:purple
    Description: attr:bold; fg:white
    Likes: attr:bold; fg:red
    ListField: fg:white; bg:black
    ListLabel: attr:bold; fg:green
    ListOptions: attr:bold; fg:orange
    Published: attr:bold; fg:aqua
    Subscribers: attr:bold; fg:plum
    Views: attr:bold; fg:pink
  }
  Playlist: {
    Author: attr:bold; fg:purple
    TotalDuration: attr:bold; fg:pink
    TotalVideos: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Queue: {
    Author: attr:bold; fg:purple
    MediaType: attr:bold; fg:pink
    MoveModeSelector: attr:bold; fg:aqua
    NormalModeSelector: attr:bold; fg:white
    TagFetching: attr:bold; bg:yellow; fg:black
    TagLoading: attr:bold; bg:yellow; fg:black
    TagPlaying: attr:bold; fg:white; bg:green
    TagStopped: attr:bold; bg:red; fg:white
    TotalDuration: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Search: {
    Author: attr:bold; fg:purple
    FormButton: attr:bold; bg:blue; fg:white
    Channel: attr:bold; fg:blue
    FormField: attr:bold; bg:grey; fg:white
    FormLabel: attr:bold; fg:white
    Playlist: attr:bold; fg:blue
    Text: attr:bold; fg:white
    TotalDuration: attr:bold; fg:pink
    TotalVideos: attr:bold; fg:pink
    Video: attr:bold; fg:blue
  }
  Start: {
    Text: attr:bold; fg:white
  }
  StatusBar: {
    ErrorMessage: attr:bold; fg:red
    InfoMessage: attr:bold; fg:white
    InputField: bg:black
    InputLabel: attr:bold; fg:white
    TagStatusBar: fg:black; bg:yellow
  }
}
`

// GetThemedRegions appends style tags for each region and returns the text.
func GetThemedRegions(text string) string {
	return tview.ReplaceRegionStyles(
		text,
		func(region string) string {
			_, theme, ok := GetThemeRegion(region)
			if !ok {
				return ""
			}

			split := strings.Split(theme, ",")
			if len(split) != 2 {
				return ""
			}

			context, item :=
				ThemeContext(split[0]),
				ThemeItem(split[1])

			_, tag, ok := GetThemeSetting(ThemeProperty{
				Context: context,
				Item:    item,
			})
			if !ok {
				return ""
			}

			return tag
		},
	)
}

// GetThemeRegion returns the region name from a region tag.
func GetThemeRegion(name string) (string, string, bool) {
	s := strings.Split(name, ";")
	if len(s) == 0 {
		return "", "", false
	}

	return s[len(s)-1], s[0], true
}

// FormatRegion formats and returns the ThemeContext, ThemeItem and region name into a region tag.
func FormatRegion(region string, context ThemeContext, item ThemeItem) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "%s,%s;%s", context, item, region)

	return sb.String()
}

// SetTextStyle sets style tags for the provided text.
func SetTextStyle(region, text string, context ThemeContext, item ThemeItem) string {
	builder := NewTextBuilder(context)
	builder.Append(item, region, text)

	return builder.Get()
}

// GetLabel returns a theme formatted label for primitives.
func GetLabel(property ThemeProperty, label string, spaced bool) string {
	if label == "" {
		return ""
	}

	builder := NewTextBuilder(property.Context)
	builder.Append(property.Item, "label", label)
	if spaced {
		style, _, ok := GetThemeSetting(property)
		_, bg, _ := style.Decompose()
		if ok {
			for name, color := range tcell.ColorNames {
				if bg == color {
					fmt.Fprintf(&builder, "[:%s:] ", name)
					break
				}
			}
		}
	}

	return builder.Get()
}

// GetThemedLabel checks whether the label is formatted and reformats it, or returns a newly formatted label.
func GetThemedLabel(property ThemeProperty, label string, spaced bool) string {
	if label == "" {
		return ""
	}

	if label[0] == '[' {
		return GetThemedRegions(label)
	}

	return GetLabel(property, label, spaced)
}

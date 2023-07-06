package cmd

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

// KeyData stores the metadata for the key.
type KeyData struct {
	Title, Context string
	Kb             Keybinding
}

// Keybinding stores the keybinding.
type Keybinding struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

var (
	// OperationKeys matches the operation name (or the menu ID)
	// with the keybinding.
	OperationKeys = map[string]*KeyData{
		"Menu": {
			Title:   "Menu",
			Context: "App",
			Kb:      Keybinding{tcell.KeyRune, 'm', tcell.ModAlt},
		},
		"Suspend": {
			Title:   "Suspend",
			Context: "App",
			Kb:      Keybinding{tcell.KeyCtrlZ, ' ', tcell.ModCtrl},
		},
		"Cancel": {
			Title:   "Cancel Loading",
			Context: "App",
			Kb:      Keybinding{tcell.KeyCtrlX, ' ', tcell.ModCtrl},
		},
		"InstancesList": {
			Title:   "List Instances",
			Context: "App",
			Kb:      Keybinding{tcell.KeyRune, 'o', tcell.ModNone},
		},
		"Quit": {
			Title:   "Quit",
			Context: "App",
			Kb:      Keybinding{tcell.KeyRune, 'Q', tcell.ModNone},
		},
		"SearchQuery": {
			Title:   "Query",
			Context: "Search",
			Kb:      Keybinding{tcell.KeyRune, '/', tcell.ModNone},
		},
		"SearchStart": {
			Title:   "Start Search",
			Context: "Search",
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		"SearchSuggestions": {
			Title:   "Get Suggestions",
			Context: "Search",
			Kb:      Keybinding{tcell.KeyTab, ' ', tcell.ModNone},
		},
		"SearchSwitchMode": {
			Title:   "Switch Search Mode",
			Context: "Search",
			Kb:      Keybinding{tcell.KeyCtrlE, ' ', tcell.ModCtrl},
		},
		"SearchParameters": {
			Title:   "Set Search Parameters",
			Context: "Search",
			Kb:      Keybinding{tcell.KeyRune, 'e', tcell.ModAlt},
		},
		"SearchHistoryReverse": {
			Context: "Search",
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModNone},
		},
		"SearchHistoryForward": {
			Context: "Search",
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModNone},
		},
		"SearchSuggestionReverse": {
			Context: "Search",
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModCtrl},
		},
		"SearchSuggestionForward": {
			Context: "Search",
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModCtrl},
		},
		"Dashboard": {
			Title:   "Dashboard",
			Context: "Dashboard",
			Kb:      Keybinding{tcell.KeyCtrlD, ' ', tcell.ModCtrl},
		},
		"DashboardReload": {
			Title:   "Reload Dashboard",
			Context: "Dashboard",
			Kb:      Keybinding{tcell.KeyCtrlT, ' ', tcell.ModCtrl},
		},
		"DashboardCreatePlaylist": {
			Title:   "Create Playlist",
			Context: "Dashboard",
			Kb:      Keybinding{tcell.KeyRune, 'c', tcell.ModNone},
		},
		"DashboardEditPlaylist": {
			Title:   "Edit playlist",
			Context: "Dashboard",
			Kb:      Keybinding{tcell.KeyRune, 'e', tcell.ModNone},
		},
		"FilebrowserDirForward": {
			Title:   "Select dir",
			Context: "Files",
			Kb:      Keybinding{tcell.KeyRight, ' ', tcell.ModNone},
		},
		"FilebrowserDirBack": {
			Title:   "Go back",
			Context: "Files",
			Kb:      Keybinding{tcell.KeyLeft, ' ', tcell.ModNone},
		},
		"FilebrowserToggleHidden": {
			Title:   "Toggle hidden",
			Context: "Files",
			Kb:      Keybinding{tcell.KeyCtrlG, ' ', tcell.ModCtrl},
		},
		"DownloadView": {
			Title:   "Show Downloads",
			Context: "Downloads",
			Kb:      Keybinding{tcell.KeyRune, 'Y', tcell.ModNone},
		},
		"DownloadOptions": {
			Title:   "Download Video",
			Context: "Downloads",
			Kb:      Keybinding{tcell.KeyRune, 'y', tcell.ModNone},
		},
		"DownloadOptionSelect": {
			Title:   "Select Option",
			Context: "Downloads",
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		"DownloadCancel": {
			Title:   "Cancel Download",
			Context: "Downloads",
			Kb:      Keybinding{tcell.KeyRune, 'x', tcell.ModNone},
		},
		"Queue": {
			Title:   "Show Queue",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyRune, 'q', tcell.ModNone},
		},
		"QueuePlayMove": {
			Title:   "Play/Replace",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		"QueueSave": {
			Title:   "Save Queue",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyCtrlS, ' ', tcell.ModCtrl},
		},
		"QueueAppend": {
			Title:   "Append To Queue",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyCtrlA, ' ', tcell.ModCtrl},
		},
		"QueueDelete": {
			Title:   "Delete",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyRune, 'd', tcell.ModNone},
		},
		"QueueMove": {
			Title:   "Move",
			Context: "Queue",
			Kb:      Keybinding{tcell.KeyRune, 'M', tcell.ModNone},
		},
		"PlayerOpenPlaylist": {
			Title:   "Open Playlist",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyCtrlO, ' ', tcell.ModCtrl},
		},
		"PlayerHistory": {
			Title:   "Show History",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'h', tcell.ModAlt},
		},
		"PlayerQueueAudio": {
			Title:   "Queue Audio",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'a', tcell.ModNone},
		},
		"PlayerQueueVideo": {
			Title:   "Queue Video",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'v', tcell.ModNone},
		},
		"PlayerPlayAudio": {
			Title:   "Play Audio",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'A', tcell.ModNone},
		},
		"PlayerPlayVideo": {
			Title:   "Play Video",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'V', tcell.ModNone},
		},
		"PlayerInfo": {
			Title:   "Track Information",
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, ' ', tcell.ModAlt},
		},
		"PlayerSeekForward": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRight, ' ', tcell.ModNone},
		},
		"PlayerSeekBackward": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyLeft, ' ', tcell.ModNone},
		},
		"PlayerStop": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'S', tcell.ModNone},
		},
		"PlayerToggleLoop": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'l', tcell.ModNone},
		},
		"PlayerToggleShuffle": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 's', tcell.ModNone},
		},
		"PlayerToggleMute": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, 'm', tcell.ModNone},
		},
		"PlayerTogglePlay": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, ' ', tcell.ModNone},
		},
		"PlayerPrev": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, '<', tcell.ModNone},
		},
		"PlayerNext": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, '>', tcell.ModNone},
		},
		"PlayerVolumeIncrease": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, '=', tcell.ModNone},
		},
		"PlayerVolumeDecrease": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyRune, '-', tcell.ModNone},
		},
		"PlayerInfoScrollUp": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyUp, ' ', tcell.ModCtrl | tcell.ModAlt},
		},
		"PlayerInfoScrollDown": {
			Context: "Player",
			Kb:      Keybinding{tcell.KeyDown, ' ', tcell.ModCtrl | tcell.ModAlt},
		},
		"Comments": {
			Title:   "Show Comments",
			Context: "Comments",
			Kb:      Keybinding{tcell.KeyRune, 'C', tcell.ModNone},
		},
		"CommentReplies": {
			Title:   "Expand replies",
			Context: "Comments",
			Kb:      Keybinding{tcell.KeyRune, ' ', tcell.ModNone},
		},
		"Switch": {
			Title:   "Switch page",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyTab, ' ', tcell.ModNone},
		},
		"Playlist": {
			Title:   "Show Playlist",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, 'i', tcell.ModNone},
		},
		"ChannelVideos": {
			Title:   "Show Channel videos",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, 'u', tcell.ModNone},
		},
		"ChannelPlaylists": {
			Title:   "Show Channel playlists",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, 'U', tcell.ModNone},
		},
		"AudioURL": {
			Title:   "Play audio from URL",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, 'b', tcell.ModNone},
		},
		"VideoURL": {
			Title:   "Play video from URL",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, 'B', tcell.ModNone},
		},
		"Link": {
			Title:   "Show Link",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, ';', tcell.ModNone},
		},
		"Add": {
			Title:   "Add",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, '+', tcell.ModNone},
		},
		"Remove": {
			Title:   "Remove",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyRune, '_', tcell.ModNone},
		},
		"LoadMore": {
			Title:   "Load more",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyEnter, ' ', tcell.ModNone},
		},
		"Exit": {
			Title:   "Exit",
			Context: "Common",
			Kb:      Keybinding{tcell.KeyEscape, ' ', tcell.ModNone},
		},
	}

	// Keys match the keybinding to the operation name.
	Keys map[string]map[Keybinding]string
)

// OperationData returns the key data associated with
// the provided keyID and operation name.
func OperationData(operation string) *KeyData {
	return OperationKeys[operation]
}

// KeyOperation returns the operation name for the provided keyID
// and the keyboard event.
func KeyOperation(event *tcell.EventKey, contexts ...string) string {
	if Keys == nil {
		Keys = make(map[string]map[Keybinding]string)
		for keyName, key := range OperationKeys {
			if Keys[key.Context] == nil {
				Keys[key.Context] = make(map[Keybinding]string)
			}

			Keys[key.Context][key.Kb] = keyName
		}
	}

	ch := event.Rune()
	if event.Key() != tcell.KeyRune {
		ch = ' '
	}

	kb := Keybinding{event.Key(), ch, event.Modifiers()}

	for _, context := range contexts {
		if operation, ok := Keys[context][kb]; ok {
			return operation
		}
	}

	if common, ok := Keys["Common"][kb]; ok {
		return common
	}

	return ""
}

// parseKeybindings parses the keybindings from the configuration.
func parseKeybindings() {
	if !config.Exists("keybindings") {
		return
	}

	kbMap := config.StringMap("keybindings")
	if len(kbMap) == 0 {
		return
	}

	keyNames := make(map[string]tcell.Key)
	for key, names := range tcell.KeyNames {
		keyNames[names] = key
	}

	for keyType, key := range kbMap {
		checkBindings(keyType, key, keyNames)
	}
}

// checkBindings validates the provided keybinding.
//
//gocyclo:ignore
func checkBindings(keyType, key string, keyNames map[string]tcell.Key) {
	var runes []rune
	var keys []tcell.Key

	if _, ok := OperationKeys[keyType]; !ok {
		printer.Error(fmt.Sprintf("Config: Invalid key type %s", keyType))
	}

	keybinding := Keybinding{
		Key:  tcell.KeyRune,
		Rune: ' ',
		Mod:  tcell.ModNone,
	}

	tokens := strings.FieldsFunc(key, func(c rune) bool {
		return unicode.IsSpace(c) || c == '+'
	})

	for _, token := range tokens {
		if len(token) > 1 {
			token = strings.Title(token)
		} else if len(token) == 1 {
			keybinding.Rune = rune(token[0])
			runes = append(runes, keybinding.Rune)

			continue
		}

		switch token {
		case "Ctrl":
			keybinding.Mod |= tcell.ModCtrl

		case "Alt":
			keybinding.Mod |= tcell.ModAlt

		case "Shift":
			keybinding.Mod |= tcell.ModShift

		case "Space", "Plus":
			keybinding.Rune = ' '
			if token == "Plus" {
				keybinding.Rune = '+'
			}

			runes = append(runes, keybinding.Rune)

		default:
			if key, ok := keyNames[token]; ok {
				keybinding.Key = key
				keybinding.Rune = ' '
				keys = append(keys, keybinding.Key)
			}
		}
	}

	if keys != nil && runes != nil || len(runes) > 1 || len(keys) > 1 {
		printer.Error(
			fmt.Sprintf("Config: More than one key entered for %s (%s)", keyType, key),
		)
	}

	if keybinding.Mod&tcell.ModShift != 0 {
		keybinding.Rune = unicode.ToUpper(keybinding.Rune)

		if unicode.IsLetter(keybinding.Rune) {
			keybinding.Mod &^= tcell.ModShift
		}
	}

	if keybinding.Mod&tcell.ModCtrl != 0 {
		var modKey string

		switch {
		case len(keys) > 0:
			if key, ok := tcell.KeyNames[keybinding.Key]; ok {
				modKey = key
			}

		case len(runes) > 0:
			if keybinding.Rune == ' ' {
				modKey = "Space"
			} else {
				modKey = string(keybinding.Rune)
			}
		}

		if modKey != "" {
			modKey = "Ctrl-" + modKey
			if key, ok := keyNames[modKey]; ok {
				keybinding.Key = key
				keys = append(keys, keybinding.Key)
			}
		}
	}

	if keys == nil && runes == nil {
		printer.Error(
			fmt.Sprintf("Config: No key specified or invalid keybinding for %s (%s)", keyType, key),
		)
	}

	keydata := OperationKeys[keyType]

	for existing, data := range OperationKeys {
		if data.Kb == keybinding && data.Title != keydata.Title {
			if data.Context == keydata.Context {
				goto KeyError
			}

			for _, context := range []string{
				"App",
				"Player",
			} {
				if data.Context == context || keydata.Context == context {
					goto KeyError
				}
			}

			continue

		KeyError:
			printer.Error(
				fmt.Sprintf("Config: Keybinding for %s will override %s (%s)", keyType, existing, key),
			)
		}
	}

	OperationKeys[keyType].Kb = keybinding
}

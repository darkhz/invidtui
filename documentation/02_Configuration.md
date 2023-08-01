
Generally, invidtui will work out-of-the-box, with no configuration required.

In case you need to specify settings without using command-line options, a config file can be used.

Typing `invidtui --help` will show you the location of the config file.<br />

The configuration file is in the [HJSON](https://hjson.github.io/) format.
You can use the [--generate](03_Usage/01_Command_Line_Options.md#generate) flag to generate the configuration.<br /><br />
For example:
```
{
  download-dir: "~/Downloads"
  ffmpeg-path: ffmpeg
  force-instance: "invidious.protokolla.fi"
  mpv-path: mpv
  num-retries: 10
  video-res: 720p
  ytdl-path: youtube-dl
  keybindings: {
      Menu: Alt+m
  }
}
```
# Keybindings
The keybinding configuration is a list of `KeybindingType: keybinding` values.<br />

While defining keybindings, the global keybindings must not conflict with the non-global ones.<br />
It is possible to have duplicate keybindings amongst various non-global keybindings.

For example, this is allowed:
```
keybindings: {
	QueuePlayMove: Alt+E
	SearchSwitchMode: Alt+E
}
```
But this isn't:
```
keybindings: {
	Menu: Alt+m
	Dashboard: Alt+m
}
```

## Modifiers
The modifiers currently supported for keybindings are `Ctrl`, `Alt` and `Shift`. `Shift` should only be used in rare cases.
For example, instead of :
- `Shift+a`, type `A`
- `Alt+Shift+e`, type `Alt+E`

and so on.

For the space and the '+' characters, type Space and Plus respectively.
For example,
```
keybindings: {
	PlayerInfo: Ctrl+Space
	Add: Ctrl+Plus
}
```

## Types
Note that some keybinding combinations may be valid, but may not work due to the way your terminal/environment handles it.

The keybinding types are as follows:
| Type                    | Global | Default Keybinding | Description                                                                      |
|-------------------------|--------|--------------------|----------------------------------------------------------------------------------|
| Menu                    | Yes    | Alt+m              | Menu                                                                             |
| Cancel                  | Yes    | Ctrl+X             | Cancel Loading                                                                   |
| Suspend                 | Yes    | Ctrl+Z             | Suspend                                                                          |
| Quit                    | Yes    | Q                  | Quit                                                                             |
| InstancesList           | Yes    | o                  | List Instances                                                                   |
| Close                   | Yes    | Esc                | Close page/popup                                                                 |
| PlayerToggleShuffle     | Yes    | s                  | Toggle shuffle                                                                   |
| PlayerStop              | Yes    | S                  | Stop playback                                                                    |
| PlayerOpenPlaylist      | Yes    | Ctrl+O             | Open Playlist                                                                    |
| PlayerVolumeIncrease    | Yes    | =                  | Increase volume                                                                  |
| PlayerInfoScrollUp      | Yes    | Alt+Ctrl+Up        | Scroll player information up                                                     |
| PlayerQueueAudio        | Yes    | a                  | Queue Audio                                                                      |
| PlayerVolumeDecrease    | Yes    | -                  | Decrease volume                                                                  |
| PlayerTogglePlay        | Yes    | Space              | Play/pause track                                                                 |
| PlayerHistory           | Yes    | Alt+h              | Show History                                                                     |
| PlayerSeekBackward      | Yes    | Alt+Left           | Seek track one step backwards                                                    |
| PlayerSeekForward       | Yes    | Alt+Right          | Seek track one step forward                                                      |
| PlayerToggleLoop        | Yes    | l                  | Toggle repeat modes                                                              |
| PlayerQueueVideo        | Yes    | v                  | Queue Video                                                                      |
| PlayerNext              | Yes    | >                  | Next track                                                                       |
| PlayerPrev              | Yes    | <                  | Previous track                                                                   |
| PlayerInfoScrollDown    | Yes    | Alt+Ctrl+Down      | Scroll player information down                                                   |
| PlayerInfo              | Yes    | Alt+Space          | Track Information                                                                |
| PlayerToggleMute        | Yes    | m                  | Toggle mute                                                                      |
| PlayerPlayAudio         | Yes    | A                  | Play Audio                                                                       |
| PlayerPlayVideo         | Yes    | V                  | Play Video                                                                       |
| Query                   | Yes    | /                  | Query for a search term, used in the search/channel pages and the history popup. |
| VideoURL                | Yes    | B                  | Play video from URL                                                              |
| Remove                  | Yes    | _                  | Remove from playlist/Unsubscribe from channel                                    |
| LoadMore                | Yes    | Enter              | Load more results                                                                |
| ChannelVideos           | Yes    | u                  | Show Channel videos                                                              |
| ChannelPlaylists        | Yes    | U                  | Show Channel playlists                                                           |
| AudioURL                | Yes    | b                  | Play audio from URL                                                              |
| Link                    | Yes    | ;                  | Show Link                                                                        |
| Switch                  | Yes    | Tab                | Switch page                                                                      |
| Add                     | Yes    | Plus               | Add to playlist/subscribe to channel                                             |
| Playlist                | Yes    | i                  | Show Playlist                                                                    |
| Comments                | No     | C                  | Show Comments                                                                    |
| CommentReplies          | No     | Enter              | Expand replies                                                                   |
| DashboardReload         | No     | Ctrl+T             | Reload Dashboard                                                                 |
| Dashboard               | No     | Ctrl+D             | Dashboard                                                                        |
| DashboardCreatePlaylist | No     | c                  | Create Playlist                                                                  |
| DashboardEditPlaylist   | No     | e                  | Edit playlist                                                                    |
| DownloadOptions         | No     | y                  | Download Video                                                                   |
| DownloadView            | No     | Y                  | Show Downloads                                                                   |
| DownloadOptionSelect    | No     | Enter              | Select Download Option                                                           |
| DownloadCancel          | No     | x                  | Cancel Download                                                                  |
| FilebrowserDirBack      | No     | Left               | Go back                                                                          |
| FilebrowserDirForward   | No     | Right              | Select dir                                                                       |
| FilebrowserToggleHidden | No     | Ctrl+G             | Toggle hidden                                                                    |
| FilebrowserNewFolder    | No     | Ctrl+N             | Create new folder                                                                |
| FilebrowserRename       | No     | Ctrl+B             | Rename item                                                                      |
| QueueAppend             | No     | Ctrl+A             | Append To Queue                                                                  |
| QueueSave               | No     | Ctrl+S             | Save Queue                                                                       |
| QueueDelete             | No     | d                  | Delete                                                                           |
| QueueMove               | No     | M                  | Move                                                                             |
| Queue                   | No     | q                  | Show Queue                                                                       |
| QueuePlayMove           | No     | Enter              | Play/Replace                                                                     |
| SearchSwitchMode        | No     | Ctrl+E             | Switch Search Mode                                                               |
| SearchParameters        | No     | Alt+e              | Set Search Parameters                                                            |
| SearchSuggestions       | No     | Tab                | Get Suggestions                                                                  |
| SearchStart             | No     | Enter              | Start Search                                                                     |
| SearchHistoryForward    | No     | Down               | Get the next search history entry                                                |
| SearchSuggestionForward | No     | Ctrl+Down          | Select the next search suggestion                                                |
| SearchHistoryReverse    | No     | Up                 | Get the previous search history entry                                            |
| SearchSuggestionReverse | No     | Ctrl+Up            | Select the previous search suggestion                                            |


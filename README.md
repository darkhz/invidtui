


[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/invidtui)](https://goreportcard.com/report/github.com/darkhz/invidtui)
# invidtui

![demo](demo/demo.gif)

invidtui is an invidious client, which fetches data from invidious instances and displays a user interface in the terminal(TUI), and allows for selecting and playing Youtube audio and video.

Currently, it is tested on Linux and Windows, and it should work on MacOS.

## Features
- Play audio or video
- Search with history support
- Control the video resolution
- Ability to open, view, edit and save m3u8 playlists
- Automatically queries the invidious API and selects the best instance

## Requirements
- MPV
- Youtube-dl

## Installation
You can install the package either via the following command:
```go install github.com/darkhz/invidtui@latest ```

or check the Releases page and download the binary that matches your OS and architecture.

## Usage

    invidtui [<flags>]

    Flags:
      --video-res="720p"  Set the default video resolution.
      --close-instances   Close all currently running instances.

## Keybindings
|Operation                                        |Key                          |
|-------------------------------------------------|-----------------------------|
|Search                                           |<kbd>/</kbd>                 |
|Open playlist                                    |<kbd>p</kbd>                 |
|Open saved playlist                              |<kbd>Ctrl</kbd>+<kbd>o</kbd> |
|Save current playlist                            |<kbd>Ctrl</kbd>+<kbd>s</kbd> |
|Add audio to the playlist                        |<kbd>a</kbd>                 |
|Add audio to the playlist and play               |<kbd>Shift</kbd>+<kbd>a</kbd>|
|Add video to the playlist                        |<kbd>v</kbd>                 |
|Add video to the playlist and play               |<kbd>Shift</kbd>+<kbd>v</kbd>|
|Move an item in playlist                         |<kbd>m</kbd>                 |
|Delete an item in playlist                       |<kbd>d</kbd>                 |
|Pause/unpause                                    |<kbd>Space</kbd>             |
|Seek forward                                     |<kbd>Right</kbd>             |
|Seek backward                                    |<kbd>Left</kbd>              |
|Switch to previous track                         |<kbd><</kbd>                 |
|Switch to next track                             |<kbd>></kbd>                 |
|Cycle shuffle mode (shuffle-playlist)            |<kbd>s</kbd>                 |
|Cycle repeat modes (repeat-file, repeat-playlist)|<kbd>l</kbd>                 |
|Stop player                                      |<kbd>Shift</kbd>+<kbd>s</kbd>|
|Suspend                                          |<kbd>Ctrl</kbd>+<kbd>Z</kbd> |
|Quit                                             |<kbd>q</kbd>                 |

## Additional Notes
- Since Youtube video titles may have many unicode characters (emojis for example), it is recommended to install **noto-fonts** and its variants (noto-fonts-emoji for example). Refer to your distro's documentation on how to install them. On Arch Linux for instance, you can install the fonts using pacman:
  `pacman -S noto-fonts noto-fonts-emoji noto-fonts-extra`<br/>

- For the video mode, only MP4 videos will be played, and currently there is no way to modify this behavior. This will change in later versions.

- The close-instances option should mainly be used if another invidtui instance may be using the socket, if there was an application crash, or if an error pops up like this: ``` Error: Socket exists at /home/test/.config/invidtui/socket, is another instance running?```.

- On Windows, using invidtui in Powershell/CMD will work, but use Windows Terminal for best results.

## Bugs
- Video streams from an invidious instance that are other than 720p or 360p can't currently be played properly when loaded from a saved playlist (only video will be played, audio won't), since we need to merge the audio and video streams, and I have yet to find a way to do that via the m3u8 playlist spec.

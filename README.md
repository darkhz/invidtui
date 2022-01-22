

[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/invidtui)](https://goreportcard.com/report/github.com/darkhz/invidtui)
# invidtui

![demo](demo/demo.gif)

invidtui is an invidious client, which fetches data from invidious and displays a user interface in the terminal(TUI), and allows for selecting and playing Youtube audio and video.

Currently, it is tested only on Linux.

## Features
- Playlist viewer
- Play audio or video
- Search with history support
- Control the video resolution
- Automatically queries the invidious API and selects the best instance

## Requirements
- MPV
- Youtube-dl

## Installation
` go install github.com/darkhz/invidtui@latest `

## Usage

    invidtui [<flags>]

    Flags:
      --video-res="720p"  Set the default video resolution.
      --close-instances   Close all currently running instances.
*Note:* --close-instances should mainly be used if another invidtui instance may be using the socket, or if an error pops up like this:<br/>
``` Error: Socket exists at /home/test/.config/invidtui/socket, is another instance running? ```

And you want to ensure that all invidtui instances are closed before launching a new one.

## Keybindings
|Operation                                        |Key                          |
|-------------------------------------------------|-----------------------------|
|Search                                           |<kbd>/</kbd>                 |
|Open playlist                                    |<kbd>p</kbd>                 |
|Add audio to the playlist                        |<kbd>a</kbd>                 |
|Add audio to the playlist and play               |<kbd>A</kbd>                 |
|Add video to the playlist                        |<kbd>v</kbd>                 |
|Add video to the playlist and play               |<kbd>V</kbd>                 |
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

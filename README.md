


[![Go Report Card](https://goreportcard.com/badge/github.com/darkhz/invidtui)](https://goreportcard.com/report/github.com/darkhz/invidtui)
# invidtui

![demo](demo/demo.gif)

invidtui is an invidious client, which fetches data from invidious instances and displays a user interface in the terminal(TUI), and allows for selecting and playing Youtube audio and video.

Currently, it is tested on Linux and Windows, and it should work on MacOS.

## Features
- Play audio or video
- Control the video resolution
- Ability to open, view, edit and save m3u8 playlists
- Automatically queries the invidious API and selects the best instance
- Search for and browse videos, playlists and channels, with history support

## Requirements
- MPV
- Youtube-dl
- FFMpeg

## Installation
You can install the package either via the following command:<br />
```go install github.com/darkhz/invidtui@latest ```

or check the Releases page and download the binary that matches your OS and architecture.

## Usage

    invidtui [<flags>]

    Flags:
      --video-res="720p"        Set the default video resolution.
      --close-instances         Close all currently running instances.
      --mpv-path="mpv"          Specify path to the mpv executable.
      --ytdl-path="youtube-dl"  Specify path to youtube-dl executable or its forks (yt-dlp, yt-dtlp_x86)
      --num-retries=100         Set the number of retries for connecting to the socket.

## Configuration file
Generally, invidtui will work out-of-the-box, with no configuration required.<br />

In case you need to specify settings without using command-line options, a config file can be used.<br />

Typing `invidtui --help` will show you the location of the config file.<br />
Config file definitions are in the form of a simple `name=value` or `name value` pair.<br /><br />
For example:
```
video-res=720p
mpv-path=/home/user/mycustompath/mpv
ytdl-path=/home/user/mycustompath/ytdl
num-retries=10
use-current-instance
```

## Keybindings

### Search

> <kbd>/</kbd><br /> Show search input. To search a channel from the main screen
> immediately instead of loading it first, press <kbd>Alt</kbd>+<kbd>/</kbd>.<br />
>
> <kbd>Ctrl</kbd> + <kbd>e</kbd><br /> Switch between search modes
> (video, playlist, channel)<br />

### Playlist Queue

> <kbd>p</kbd><br /> Open playlist queue. This control will work across
> all pages.<br />
>
> <kbd>Ctrl</kbd>+<kbd>o</kbd><br /> Open saved playlist. This control will work across
> all pages.<br />
>
> <kbd>Ctrl</kbd>+<kbd>a</kbd><br /> Append from a playlist file to the playlist queue<br />
>
> <kbd>Ctrl</kbd>+<kbd>s</kbd><br /> Save current playlist queue<br />
>
> <kbd>Shift</kbd>+<kbd>m</kbd><br /> Move an item in playlist queue. To cancel a move,
> just press <kbd>Enter</kbd> in the same position the move operation
> was started.<br />
>
> <kbd>d</kbd><br /> Delete an item in playlist queue<br />

### Player
Note: These controls will work across all pages (search, playlist or channel pages)<br /><br />

> <kbd>Space</kbd><br /> Pause/unpause<br />
>
> <kbd>=</kbd><br /> Increase volume<br />
>
> <kbd>-</kbd><br /> Decrease volume<br />
>
> <kbd>Right</kbd><br /> Seek forward<br />
>
> <kbd>Left</kbd><br /> Seek backward<br />
>
> <kbd><</kbd><br /> Switch to previous track<br />
>
> <kbd>></kbd><br /> Switch to next track<br />
>
> <kbd>s</kbd><br /> Cycle shuffle mode (shuffle-playlist)<br />
>
> <kbd>m</kbd><br /> Cycle mute mode<br />
>
> <kbd>l</kbd><br /> Cycle repeat modes (repeat-file,
> repeat-playlist)<br />
>
> <kbd>Shift</kbd>+<kbd>s</kbd><br /> Stop player<br />

### Application

> <kbd>Ctrl</kbd>+<kbd>Z</kbd><br /> Suspend<br />
>
> <kbd>q</kbd><br /> Quit<br />


### Page-based Keybindings

> <kbd>i</kbd><br />
> This control works on the search and channel playlist pages.<br />
> Fetches the Youtube playlist contents from the currently selected entry and displays it in a separate playlist page. <br />
> In case you have exited this page, you can come back to it by pressing <kbd>Alt</kbd>+<kbd>i</kbd> instead of reloading the playlist again.<br/>
>
> <kbd>u</kbd><br />
> This control works on the search page.<br />
> Fetches only videos from a Youtube channel (from the currently selected entry) and displays it in a separate channel video page.<br />
> <kbd>Shift</kbd>+<kbd>u</kbd> fetches only playlists from a Youtube channel and displays it in a separate channel playlist page.
> In case you have exited<br /> this page, you can come back to it by pressing <kbd>Alt</kbd>+<kbd>u</kbd> instead of reloading the channel again.<br />
>
> <kbd>Enter</kbd><br />
> This control works on the search, playlist, channel video and channel playlist pages.<br />
> Fetches more results.<br />
>
> <kbd>Tab</kbd><br />
> This control works on the channel video, channel playlist and channel search pages<br />
> Switches the channel page being shown.<br />
>
> <kbd>/</kbd><br />
> This control works on the search and channel search pages.<br />
> Refer to the search keybindings above.<br />
>
> <kbd>a</kbd><br />
> This control works on the search, playlist and channel video list pages.<br />
> Fetches audio of the currently selected entry and adds it to the playlist.<br />
> If the selected entry is a playlist, all the playlist contents will be loaded into<br />
> the playlist queue as audio.
> To immediately play after adding to playlist, press <kbd>Shift</kbd>+<kbd>a</kbd>.<br/>
>
> <kbd>v</kbd><br />
> This control works on the search, playlist and channel video pages<br/>
> Fetches video of the currently selected entry and adds it to the playlist.<br />
> If the selected entry is a playlist, all the playlist contents will be loaded into<br />
> the playlist queue as video.
> To immediately play after adding to playlist, press <kbd>Shift</kbd>+<kbd>v</kbd>.<br/>
>
> <kbd>Ctrl</kbd>+<kbd>x</kbd><br />
> Cancel the fetching of playlist or channel contents (in case it takes a long time,<br/>
> due to slow network speeds for example).<br/>
>
> <kbd>Esc</kbd><br />
> Exit the current page.<br/>

## Additional Notes
- Since Youtube video titles may have many unicode characters (emojis for example), it is recommended to install **noto-fonts** and its variants (noto-fonts-emoji for example). Refer to your distro's documentation on how to install them. On Arch Linux for instance, you can install the fonts using pacman:
  `pacman -S noto-fonts noto-fonts-emoji noto-fonts-extra`<br/>

- For the video mode, only MP4 videos will be played, and currently there is no way to modify this behavior. This will change in later versions.

- The close-instances option should mainly be used if another invidtui instance may be using the socket, if there was an application crash, or if an error pops up like this: ``` Error: Socket exists at /home/test/.config/invidtui/socket, is another instance running?```.

- The use-current-instance option can be used in cases where a playlist file has to be loaded, but the URLs in the playlist point to a slow invidious instance. The playlist media can instead be retrieved from a fast instance (automatically selected by invidtui).

- On Windows, using invidtui in Powershell/CMD will work, but use Windows Terminal for best results.

- For certain videos where the duration is shown as "00:00", but the published date is greater than 0s, it is most likely that the video is a live stream. Due to certain inconsistencies with the invidious API, such videos are not shown as live streams in the search results, but will show when playing.

- Since invidtui relies on specially crafted URLs to load and display media properly, it is not recommended to edit the autogenerated playlist.

package lib

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkhz/mpvipc"
)

// Connector stores the mpvipc connection data.
type Connector struct {
	conn *mpvipc.Connection
}

var (
	loop   string
	socket string
	mpvcmd *exec.Cmd
	mpvctl *Connector

	monitorMutex sync.Mutex
	monitorMap   map[int]string
	mpvInfoChan  chan int
	mpvErrorChan chan int

	// MPVErrors is a channel to receive mpv error messages.
	MPVErrors chan string

	// MPVFileLoaded is a channel to receive file-loaded events.
	MPVFileLoaded chan struct{}
)

// NewConnector returns a Connector with an active mpvipc connection.
func NewConnector(conn *mpvipc.Connection) *Connector {
	return &Connector{
		conn: conn,
	}
}

// GetMPV returns the currently active mpvipc instance.
func GetMPV() *Connector {
	return mpvctl
}

// MPVStart loads the mpv executable, and connects to the socket.
func MPVStart() error {
	var err error

	socket, err = ConfigPath("socket")
	if err != nil {
		return err
	}

	mpvctl, err = MPVConnect(socket, true)
	if err != nil {
		return err
	}

	MPVErrors = make(chan string, 100)
	MPVFileLoaded = make(chan struct{}, 100)
	go mpvctl.eventListener()

	mpvInfoChan = make(chan int, 100)
	mpvErrorChan = make(chan int, 100)
	monitorMap = make(map[int]string)
	go monitorStart()

	mpvctl.Set("pause", "yes")
	mpvctl.Set("pause", "no")

	mpvctl.Call("keybind", "q", "")
	mpvctl.Call("keybind", "Ctrl+q", "")
	mpvctl.Call("keybind", "Shift+q", "")

	return nil
}

// MPVConnect attempts to connect to the mpv instance.
func MPVConnect(socket string, mpvexec bool) (*Connector, error) {
	if mpvexec {
		mpvcmd = exec.Command(
			mpvpath,
			"--idle",
			"--keep-open",
			"--no-terminal",
			"--really-quiet",
			"--no-input-terminal",
			"--user-agent="+userAgent,
			"--input-ipc-server="+socket,
			"--script-opts=ytdl_hook-ytdl_path="+ytdlpath,
		)

		err := mpvcmd.Start()
		if err != nil {
			return nil, fmt.Errorf("Error: Could not start mpv")
		}
	}

	conn := mpvipc.NewConnection(socket)
	for i := 1; i < connretries; i++ {
		err := conn.Open()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		return NewConnector(conn), nil
	}

	return nil, fmt.Errorf("Error: Could not connect to socket")
}

// CloseInstances sends a quit command to instances running on the socket.
func CloseInstances(socket string) {
	conn := mpvipc.NewConnection(socket)
	if err := conn.Open(); err != nil {
		return
	}

	NewConnector(conn).MPVStop(false)

	time.Sleep(2 * time.Second)
}

// WaitUntilClosed waits until a connection is closed.
func (c *Connector) WaitUntilClosed() {
	c.conn.WaitUntilClosed()
}

// MPVStop sends a quit command to the mpv executable.
func (c *Connector) MPVStop(rm bool) {
	if c.IsClosed() {
		return
	}

	c.Call("quit")

	if !rm {
		return
	}

	os.Remove(sockPath)
}

// Call sends a command to the mpv instance.
func (c *Connector) Call(args ...interface{}) (interface{}, error) {
	if c.IsClosed() {
		return nil, fmt.Errorf("Connection closed")
	}

	value, err := c.conn.Call(args...)

	return value, err
}

// Get gets a property from the mpv instance.
func (c *Connector) Get(prop string) (interface{}, error) {
	if c.IsClosed() {
		return nil, fmt.Errorf("Connection closed")
	}

	value, err := c.conn.Get(prop)

	return value, err
}

// Set sets a property in the mpv instance.
func (c *Connector) Set(prop string, value interface{}) error {
	if c.IsClosed() {
		return fmt.Errorf("Connection closed")
	}

	err := c.conn.Set(prop, value)

	return err
}

// LoadFile loads the given file into mpv along with the relevant metadata.
// If the files parameter contains more than one filename argument, it
// will consider the first entry as the video file and the second entry as
// the audio file, set the relevant options and pass them to mpv.
func (c *Connector) LoadFile(title string, duration int, liveaudio bool, files ...string) error {
	options := "title='" + title

	if duration > 0 {
		options += "',length=" + strconv.Itoa(duration)
	}

	if liveaudio {
		options += ",vid=no"
	}

	if len(files) == 2 {
		options += ",audio-file=" + files[1]
	}

	files[0] += "&options=" + url.QueryEscape(options)
	_, err := c.Call("loadfile", files[0], "append-play", options)
	if err != nil {
		return fmt.Errorf("Unable to load %s", title)
	}

	addToMonitor(title)

	return nil
}

// LoadPlaylist loads a playlist file. If replace is false, it appends the loaded
// playlist to the current playlist, otherwise it replaces the current playlist.
func (c *Connector) LoadPlaylist(plpath string, replace bool) error {
	if replace {
		c.Call("playlist-clear")
		c.Call("playlist-remove", "current")

		clearMonitor()
	}

	pl, err := os.Open(plpath)
	if err != nil {
		return fmt.Errorf("Unable to open %s", plpath)
	}

	// We implement a simple playlist parser instead of relying on
	// the m3u8 package here, since that package deals with mainly
	// HLS playlists, and it seems to panic when certain EXTINF fields
	// are blank. With this method, we can parse the URLs from the playlist
	// directly, and pass the relevant options to mpv as well.
	scanner := bufio.NewScanner(pl)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		var title, options string

		line := scanner.Text()

		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		data := GetDataFromURL(line)
		if t := data.Get("title"); t != "" {
			title = t
		}
		if o := data.Get("options"); o != "" {
			options, _ = url.QueryUnescape(o)
		}
		if l := data.Get("length"); l == "Live" {
			audio := data.Get("mediatype") == "Audio"
			if refresh := refreshLiveURL(line, audio); refresh {
				continue
			}
		}

		if currInstance {
			uri, err := url.ParseRequestURI(line)
			if err != nil {
				return fmt.Errorf("Unable to parse playlist URL")
			}

			requri := uri.RequestURI()
			line = GetClient().host + requri
		}

		c.Call("loadfile", line, "append-play", options)

		addToMonitor(title)
	}

	return nil
}

// IsPaused checks if mpv is paused.
func (c *Connector) IsPaused() bool {
	paused, err := c.Get("pause")
	if err != nil {
		return false
	}

	return paused.(bool)
}

// IsShuffle checks if the playlist is shuffled.
func (c *Connector) IsShuffle() bool {
	shuffle, err := c.Get("shuffle")
	if err != nil {
		return false
	}

	return shuffle.(bool)
}

// IsMuted checks if playback is muted.
func (c *Connector) IsMuted() bool {
	mute, err := c.Get("mute")
	if err != nil {
		return false
	}

	return mute.(bool)
}

// IsEOF checks if an already loaded file has finished playback.
func (c *Connector) IsEOF() bool {
	eof, err := c.Get("eof-reached")
	if err != nil {
		return false
	}

	return eof.(bool)
}

// IsIdle checks if mpv is currently idle.
func (c *Connector) IsIdle() bool {
	idle, err := c.Get("core-idle")
	if err != nil {
		return false
	}

	return idle.(bool)
}

// IsBuffering checks if the media is buffering.
func (c *Connector) IsBuffering() bool {
	buf, err := c.Get("paused-for-cache")
	if err != nil {
		return true
	}

	return buf.(bool)
}

// IsClosed checks if mpv has exited.
func (c *Connector) IsClosed() bool {
	return c.conn.IsClosed()
}

// MediaType determines if currently playing file is of
// audio or video type.
func (c *Connector) MediaType() string {
	_, err := c.Get("height")
	if err != nil {
		return "Audio"
	}

	return "Video"
}

// LoopType determines if the loop option is set, and
// determines if it is one of loop-file or loop-playlist.
func (c *Connector) LoopType() string {
	lf, err := c.Call("get_property_string", "loop-file")
	if err != nil {
		return ""
	}

	lp, err := c.Call("get_property_string", "loop-playlist")
	if err != nil {
		return ""
	}

	if lf == "yes" || lf == "inf" {
		return "loop-file"
	}

	if lp == "yes" || lp == "inf" {
		return "loop-playlist"
	}

	return ""
}

// TimePosition returns the current position in the file.
func (c *Connector) TimePosition() int {
	timepos, err := c.Get("playback-time")
	if err != nil {
		return 0
	}

	return int(timepos.(float64))
}

// Duration returns the total duration of the file.
func (c *Connector) Duration() int {
	duration, err := c.Get("duration")
	if err != nil {
		duration, err = c.Get("options/length")
		if err != nil {
			return 0
		}

		time, err := strconv.Atoi(duration.(string))
		if err != nil {
			return 0
		}

		return time
	}

	return int(duration.(float64))
}

// Volume returns the current volume.
func (c *Connector) Volume() int {
	vol, err := c.Get("volume")
	if err != nil {
		return -1
	}

	return int(vol.(float64))
}

// PlaylistData return the current playlist data.
func (c *Connector) PlaylistData() string {
	list, err := c.Call("get_property_string", "playlist")
	if err != nil {
		return ""
	}

	return list.(string)
}

// PlaylistCount returns the total amount of files in the playlist.
func (c *Connector) PlaylistCount() int {
	count, err := c.Get("playlist-count")
	if err != nil {
		return 0
	}

	return int(count.(float64))
}

// PlaylistPos returns the current position of the file in the playlist.
func (c *Connector) PlaylistPos() int {
	pos, err := c.Get("playlist-playing-pos")
	if err != nil {
		return 0
	}

	return int(pos.(float64))
}

// PlaylistTitle returns the title, or filename of the playlist entry if
// title is not available.
func (c *Connector) PlaylistTitle(pos int) string {
	pltitle, _ := c.Call("get_property_string", "playlist/"+strconv.Itoa(pos)+"/title")

	if pltitle == nil {
		plfile, _ := c.Call("get_property_string", "playlist/"+strconv.Itoa(pos)+"/filename")

		if plfile == nil {
			return "-"
		}

		return plfile.(string)
	}

	return pltitle.(string)
}

// SetMediaTitle force sets the media title for the currently
// playing track.
func (c *Connector) SetMediaTitle(title string) {
	c.Set("force-media-title", title)
}

// SetPlaylistPos sets the playlist position.
func (c *Connector) SetPlaylistPos(pos int) {
	c.Set("playlist-pos", pos)
}

// PlaylistDelete deletes an entry from the playlist.
func (c *Connector) PlaylistDelete(entry int) {
	c.Call("playlist-remove", entry)
}

// PlaylistMove moves an entry to a different index in the playlist.
func (c *Connector) PlaylistMove(a, b int) {
	c.Call("playlist-move", a, b)
}

// PlaylistClear clears the playlist.
func (c *Connector) PlaylistClear() {
	c.Call("playlist-clear")

	clearMonitor()
}

// PlaylistPlayLatest plays the latest entry in the playlist.
func (c *Connector) PlaylistPlayLatest() {
	c.Set("playlist-pos", c.PlaylistCount()-1)

	c.Play()
}

// CyclePaused toggles between pause and play states.
func (c *Connector) CyclePaused() {
	if c.IsEOF() && c.IsPaused() {
		c.Call("seek", 0, "absolute-percent")
	}

	c.Call("cycle", "pause")
}

// CycleShuffle cycles the playlist's shuffle state.
func (c *Connector) CycleShuffle() {
	c.Call("cycle", "shuffle")
}

// CycleMute toggles the playback mute state.
func (c *Connector) CycleMute() {
	c.Call("cycle", "mute")
}

// CycleLoop toggles between looping a file, playlist or none.
func (c *Connector) CycleLoop() {
	switch c.LoopType() {
	case "":
		c.Set("loop-file", "yes")
		c.Set("loop-playlist", "no")

	case "loop-file":
		c.Set("loop-file", "no")
		c.Set("loop-playlist", "yes")

	case "loop-playlist":
		c.Set("loop-file", "no")
		c.Set("loop-playlist", "no")
	}
}

// Play starts the playback.
func (c *Connector) Play() {
	c.Set("pause", "no")
}

// Stop stops the playback.
func (c *Connector) Stop() {
	c.Call("stop")
}

// VolumeIncrease increases the volume.
func (c *Connector) VolumeIncrease() {
	vol := c.Volume()
	if vol == -1 {
		return
	}

	c.Set("volume", vol+1)
}

// VolumeDecrease decreases the volume.
func (c *Connector) VolumeDecrease() {
	vol := c.Volume()
	if vol == -1 {
		return
	}

	c.Set("volume", vol-1)
}

// SeekForward seeks the track forward.
func (c *Connector) SeekForward() {
	c.Call("seek", 1)
}

// SeekBackward seeks the track backward.
func (c *Connector) SeekBackward() {
	c.Call("seek", -1)
}

// Next plays the next item in the playlist.
func (c *Connector) Next() {
	c.Call("playlist-next")
}

// Prev plays the previous item in the playlist.
func (c *Connector) Prev() {
	c.Call("playlist-prev")
}

// monitorStart starts the playlist monitor.
func monitorStart() {
	for {
		select {
		case id, ok := <-mpvErrorChan:
			if !ok {
				return
			}

			monitorMutex.Lock()

			title := monitorMap[id]
			delete(monitorMap, id)

			monitorMutex.Unlock()

			select {
			case MPVErrors <- title:
			default:
			}

		}
	}
}

// addToMonitor adds a filename to the monitor.
func addToMonitor(name string) {
	select {
	case id, _ := <-mpvInfoChan:
		monitorMutex.Lock()
		defer monitorMutex.Unlock()

		monitorMap[id] = name

	default:
	}
}

// clearMonitor clears the monitor data.
func clearMonitor() {
	monitorMutex.Lock()
	defer monitorMutex.Unlock()

	monitorMap = make(map[int]string)
}

// eventListener listens for events from the mpv instance.
func (c *Connector) eventListener() {
	events, stopListening := c.conn.NewEventListener()

	shutdown := func() {
		c.conn.Close()
		close(MPVErrors)
		close(mpvInfoChan)
		close(mpvErrorChan)
		stopListening <- struct{}{}
	}

	c.Call("observe_property", 1, "shutdown")

	for {
		select {
		case event, ok := <-events:
			if !ok {
				shutdown()
				return
			}

			switch event.Name {
			case "start-file":
				if len(event.ExtraData) > 0 {
					val := event.ExtraData["playlist_entry_id"]

					mpvInfoChan <- int(val.(float64))
				}

			case "end-file":
				if len(event.ExtraData) > 0 {
					err := event.ExtraData["file_error"]
					val := event.ExtraData["playlist_entry_id"]

					if err != nil && val != nil {
						if err.(string) != "" {
							mpvErrorChan <- int(val.(float64))
						}
					}
				}

			case "file-loaded":
				MPVFileLoaded <- struct{}{}

			case "shutdown":
				shutdown()
				return
			}
		}
	}
}

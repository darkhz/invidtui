package mediaplayer

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/darkhz/invidtui/resolver"
	"github.com/darkhz/mpvipc"
)

// MPV describes the mpv player.
type MPV struct {
	socket string

	*mpvipc.Connection
}

var mpv MPV

// Init initializes and sets up MPV.
func (m *MPV) Init(execpath, ytdlpath, numretries, useragent, socket string) error {
	if err := m.connect(
		execpath, ytdlpath,
		numretries, useragent, socket,
	); err != nil {
		return err
	}

	go m.eventListener()

	m.Call("keybind", "q", "")
	m.Call("keybind", "Ctrl+q", "")
	m.Call("keybind", "Shift+q", "")

	return nil
}

// Exit tells MPV to exit.
func (m *MPV) Exit() {
	m.Call("quit")
	m.Connection.Close()

	os.Remove(m.socket)
}

// Exited returns whether MPV has exited or not.
func (m *MPV) Exited() bool {
	return m.Connection == nil || m.Connection.IsClosed()
}

// SendQuit sends a quit signal to the provided socket.
func (m *MPV) SendQuit(socket string) {
	conn := mpvipc.NewConnection(socket)
	if err := conn.Open(); err != nil {
		return
	}

	conn.Call("quit")
	time.Sleep(1 * time.Second)
}

// LoadFile loads the provided files into MPV. When more than one file is provided,
// the first file is treated as a video stream and the second file is attached as an audio stream.
func (m *MPV) LoadFile(title string, duration int64, audio bool, files ...string) error {
	if files == nil {
		return fmt.Errorf("MPV: Unable to load empty fileset")
	}

	if audio {
		m.Call("set_property", "video", "0")
	}

	options := []string{}
	if duration > 0 {
		options = append(options, "length="+strconv.FormatInt(duration, 10))
	}
	if len(files) == 2 {
		options = append(options, "audio-file="+files[1])
	}

	_, err := m.Call("loadfile", files[0], "replace", strings.Join(options, ","))
	if err != nil {
		return fmt.Errorf("MPV: Unable to load %s", title)
	}

	if !audio {
		m.Call("set_property", "video", "1")
	}

	return nil
}

// Play start the playback.
func (m *MPV) Play() {
	m.Set("pause", "no")
}

// Stop stops the playback.
func (m *MPV) Stop() {
	m.Call("stop")
}

// SeekForward seeks the track forward by 1s.
func (m *MPV) SeekForward() {
	m.Call("seek", 1)
}

// SeekBackward seeks the track backward by 1s.
func (m *MPV) SeekBackward() {
	m.Call("seek", -1)
}

// Position returns the seek position.
func (m *MPV) Position() int64 {
	var position float64

	timepos, err := m.Get("playback-time")
	if err == nil {
		m.store(timepos, &position)
	}

	return int64(position)
}

// Duration returns the total duration of the track.
func (m *MPV) Duration() int64 {
	var duration float64

	dur, err := m.Get("duration")
	if err != nil {
		var sdur string

		dur, err = m.Get("options/length")
		if err != nil {
			return 0
		}

		m.store(dur, &sdur)

		time, err := strconv.ParseInt(sdur, 10, 64)
		if err != nil {
			return 0
		}

		return time
	}

	m.store(dur, &duration)

	return int64(duration)
}

// Paused returns whether playback is paused or not.
func (m *MPV) Paused() bool {
	var paused bool

	pause, err := m.Get("pause")
	if err == nil {
		m.store(pause, &paused)
	}

	return paused
}

// TogglePaused toggles pausing the playback.
func (m *MPV) TogglePaused() {
	if m.Finished() && m.Paused() {
		m.Call("seek", 0, "absolute-percent")
	}

	m.Call("cycle", "pause")
}

// Muted returns whether playback is muted.
func (m *MPV) Muted() bool {
	var muted bool

	mute, err := m.Get("mute")
	if err == nil {
		m.store(mute, &muted)
	}

	return muted
}

// ToggleMuted toggles muting of the playback.
func (m *MPV) ToggleMuted() {
	m.Call("cycle", "mute")
}

// SetLoopMode sets the loop mode.
func (m *MPV) SetLoopMode(mode RepeatMode) {
	switch mode {
	case RepeatModeOff:
		m.Set("loop-file", "no")
		m.Set("loop-playlist", "no")

	case RepeatModeFile:
		m.Set("loop-file", "yes")
		m.Set("loop-playlist", "no")

	case RepeatModePlaylist:
		m.Set("loop-file", "no")
		m.Set("loop-playlist", "yes")
	}
}

// Idle returns if the player is idle.
func (m *MPV) Idle() bool {
	var idle bool

	ci, err := m.Get("core-idle")
	if err == nil {
		m.store(ci, &idle)
	}

	return idle
}

// Finished returns if the playback has finished.
func (m *MPV) Finished() bool {
	var finished bool

	eof, err := m.Get("eof-reached")
	if err == nil {
		m.store(eof, &finished)
	}

	return finished
}

// Buffering returns if the player is buffering.
func (m *MPV) Buffering() bool {
	var buffering bool

	buf, err := m.Get("paused-for-cache")
	if err == nil {
		m.store(buf, &buffering)
	}

	return buffering
}

// Volume returns the volume.
func (m *MPV) Volume() int {
	var volume float64

	vol, err := m.Get("volume")
	if err != nil {
		return -1
	}

	m.store(vol, &volume)

	return int(volume)
}

// VolumeIncrease increments the volume by 1.
func (m *MPV) VolumeIncrease() {
	vol := m.Volume()
	if vol == -1 {
		return
	}

	m.Set("volume", vol+1)
}

// VolumeDecrease decreases the volume by 1.
func (m *MPV) VolumeDecrease() {
	vol := m.Volume()
	if vol == -1 {
		return
	}

	m.Set("volume", vol-1)
}

// WaitClosed waits for MPV to exit.
func (m *MPV) WaitClosed() {
	m.Connection.WaitUntilClosed()
}

// Call send a command to MPV.
func (m *MPV) Call(args ...interface{}) (interface{}, error) {
	if m.Exited() {
		return nil, fmt.Errorf("MPV: Connection closed")
	}

	return m.Connection.Call(args...)
}

// Get gets a property from the mpv instance.
func (m *MPV) Get(prop string) (interface{}, error) {
	if m.Exited() {
		return nil, fmt.Errorf("MPV: Connection closed")
	}

	return m.Connection.Get(prop)
}

// Set sets a property in the mpv instance.
func (m *MPV) Set(prop string, value interface{}) error {
	if m.Exited() {
		return fmt.Errorf("MPV: Connection closed")
	}

	return m.Connection.Set(prop, value)
}

// connect launches MPV and starts a new connection via the provided socket.
func (m *MPV) connect(mpvpath, ytdlpath, numretries, useragent, socket string) error {
	command := exec.Command(
		mpvpath,
		"--idle",
		"--keep-open",
		"--no-terminal",
		"--really-quiet",
		"--no-input-terminal",
		"--user-agent="+useragent,
		"--input-ipc-server="+socket,
		"--script-opts=ytdl_hook-ytdl_path="+ytdlpath,
	)

	if err := command.Start(); err != nil {
		return fmt.Errorf("MPV: Could not start")
	}

	conn := mpvipc.NewConnection(socket)
	retries, _ := strconv.Atoi(numretries)
	for i := 0; i <= retries; i++ {
		err := conn.Open()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		m.socket = socket
		m.Connection = conn

		return nil
	}

	return fmt.Errorf("MPV: Could not connect to socket")
}

// store applies the property value into the given data container.
func (m *MPV) store(prop, apply interface{}) {
	var data []byte

	err := resolver.EncodeSimpleBytes(&data, prop)
	if err == nil {
		resolver.DecodeSimpleBytes(data, apply)
	}
}

// eventListener listens for MPV events.
//
//gocyclo:ignore
func (m *MPV) eventListener() {
	events, stopListening := m.Connection.NewEventListener()

	defer func() { stopListening <- struct{}{} }()

	m.Call("observe_property", 1, "eof-reached")

	for event := range events {
		mediaEvent := EventNone

		if event.ID == 1 {
			if eof, ok := event.Data.(bool); ok && eof {
				mediaEvent = EventEnd
			}
		}

		switch event.Name {
		case "start-file":
			m.Set("pause", "yes")
			m.Set("pause", "no")

			mediaEvent = EventStart

		case "end-file":
			if event.Reason == "eof" {
				mediaEvent = EventEnd
			}

			if len(event.ExtraData) > 0 {
				var errorText string

				m.store(event.ExtraData["file_error"], &errorText)

				if errorText != "" {
					mediaEvent = EventError
				}
			}

		case "file-loaded":
			mediaEvent = EventInProgress

		}

		select {
		case Event <- mediaEvent:
		default:
		}
	}
}

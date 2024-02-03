package mediaplayer

import "sync"

// MediaPlayer describes a media player.
type MediaPlayer interface {
	Init(properties MediaPlayerProperties) error
	Exit()
	Exited() bool
	SendQuit(socket string)

	LoadFile(title string, duration int64, liveaudio bool, files [2]string) error

	Play()
	Stop()
	SeekForward()
	SeekBackward()
	Position() int64
	Duration() int64

	Paused() bool
	TogglePaused()

	Muted() bool
	ToggleMuted()

	SetLoopMode(mode RepeatMode)

	Idle() bool
	Finished() bool

	Buffering() bool
	BufferPercentage() int

	Volume() int
	VolumeIncrease()
	VolumeDecrease()

	WaitClosed()

	Call(args ...interface{}) (interface{}, error)
	Get(prop string) (interface{}, error)
	Set(prop string, value interface{}) error
}

// MediaPlayerSettings stores the media player's settings.
type MediaPlayerSettings struct {
	current string
	handler func(e MediaEvent)

	mutex sync.Mutex
}

// MediaPlayerProperties stores the media player's properties.
type MediaPlayerProperties struct {
	PlayerPath, YtdlPath, UserAgent, SocketPath string
	NumRetries                                  string
	CloseInstances                              bool
}

type MediaEvent int

const (
	EventNone MediaEvent = iota
	EventEnd
	EventStart
	EventInProgress
	EventError
)

type RepeatMode int

const (
	RepeatModeOff RepeatMode = iota
	RepeatModeFile
	RepeatModePlaylist
)

var (
	settings MediaPlayerSettings

	players = map[string]MediaPlayer{
		"mpv": &mpv,
	}
)

// Init launches the provided player.
func Init(player string, properties MediaPlayerProperties) error {
	settings.current = player
	settings.handler = func(e MediaEvent) {}

	return players[player].Init(properties)
}

// EventHandler sends a media event to the preset handler.
func EventHandler(event MediaEvent) {
	settings.mutex.Lock()
	h := settings.handler
	settings.mutex.Unlock()

	h(event)
}

// SetEventHandler sets the media event handler.
func SetEventHandler(handler func(e MediaEvent)) {
	settings.mutex.Lock()
	defer settings.mutex.Unlock()

	settings.handler = handler
}

// Player returns the currently selected player.
func Player() MediaPlayer {
	return players[settings.current]
}

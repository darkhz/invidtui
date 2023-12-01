package mediaplayer

// MediaPlayer describes a media player.
type MediaPlayer interface {
	Init(execpath, ytdlpath, numretries, useragent, socket string) error
	Exit()
	Exited() bool
	SendQuit(socket string)

	LoadFile(title string, duration int64, liveaudio bool, files ...string) error

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

	Volume() int
	VolumeIncrease()
	VolumeDecrease()

	WaitClosed()

	Call(args ...interface{}) (interface{}, error)
	Get(prop string) (interface{}, error)
	Set(prop string, value interface{}) error
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
	current string
	Event   chan MediaEvent

	players = map[string]MediaPlayer{
		"mpv": &mpv,
	}
)

// Init launches the provided player.
func Init(player, execpath, ytdlpath, numretries, useragent, socket string) error {
	current = player

	Event = make(chan MediaEvent, 1000)

	return players[player].Init(
		execpath, ytdlpath,
		numretries, useragent, socket,
	)
}

// Player returns the currently selected player.
func Player() MediaPlayer {
	return players[current]
}

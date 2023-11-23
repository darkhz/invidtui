package mediaplayer

import "context"

// MediaPlayer describes a media player.
type MediaPlayer interface {
	Init(execpath, ytdlpath, numretries, useragent, socket string) error
	Exit()
	Exited() bool
	SendQuit(socket string)

	LoadFile(title string, duration int64, liveaudio bool, files ...string) error
	LoadPlaylist(ctx context.Context, plpath string, replace bool, renewLiveURL func(uri string, audio bool) bool) error

	Title(pos int) string
	MediaType() string

	Play()
	Stop()
	Next()
	Prev()
	SeekForward()
	SeekBackward()
	Position() int64
	Duration() int64

	Paused() bool
	TogglePaused()

	Shuffled() bool
	ToggleShuffled()

	Muted() bool
	ToggleMuted()

	LoopMode() string
	ToggleLoopMode()

	Idle() bool
	Finished() bool
	Buffering() bool

	Volume() int
	VolumeIncrease()
	VolumeDecrease()

	QueueCount() int
	QueuePosition() int
	QueueDelete(number int)
	QueueMove(before, after int)
	QueueSwitchToTrack(number int)
	QueueData() string
	QueuePlayLatest()
	QueueClear()

	WaitClosed()

	Call(args ...interface{}) (interface{}, error)
	Get(prop string) (interface{}, error)
	Set(prop string, value interface{}) error
}

// MediaEvents describes the various media player related events.
type MediaEvents struct {
	FileNumber, ErrorNumber chan int
	ErrorEvent              chan string
	FileLoadedEvent         chan struct{}
	DataEvent               chan []map[string]interface{}
}

var (
	current string
	Events  MediaEvents

	players = map[string]MediaPlayer{
		"mpv": &mpv,
	}
)

// Init launches the provided player.
func Init(player, execpath, ytdlpath, numretries, useragent, socket string) error {
	current = player

	Events.FileNumber, Events.ErrorNumber = make(chan int, 100), make(chan int, 100)
	Events.ErrorEvent = make(chan string, 100)
	Events.FileLoadedEvent = make(chan struct{}, 100)
	Events.DataEvent = make(chan []map[string]interface{}, 10)

	return players[player].Init(
		execpath, ytdlpath,
		numretries, useragent, socket,
	)
}

// Player returns the currently selected player.
func Player() MediaPlayer {
	return players[current]
}

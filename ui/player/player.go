package player

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Player stores the layout for the player.
type Player struct {
	queue Queue

	init, playing bool
	width         int
	states        []string
	history       History

	channel chan bool
	events  chan struct{}

	flex        *tview.Flex
	title, desc *tview.TextView

	lock  *semaphore.Weighted
	mutex sync.Mutex
}

var player Player

// setup sets up the player.
func setup() {
	if player.init {
		return
	}

	player.init = true

	player.channel = make(chan bool, 10)
	player.events = make(chan struct{}, 100)

	player.title, player.desc = tview.NewTextView(), tview.NewTextView()
	player.desc.SetDynamicColors(true)
	player.title.SetDynamicColors(true)
	player.desc.SetTextAlign(tview.AlignCenter)
	player.title.SetTextAlign(tview.AlignCenter)
	player.desc.SetBackgroundColor(tcell.ColorDefault)
	player.title.SetBackgroundColor(tcell.ColorDefault)

	player.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(player.title, 1, 0, false).
		AddItem(player.desc, 1, 0, false)
	player.flex.SetBackgroundColor(tcell.ColorDefault)

	player.lock = semaphore.NewWeighted(10)
}

// Start starts the player and loads its history and states.
func Start() {
	setup()

	go playingStatusCheck()
	go monitorMPVEvents()

	go loadState()
	go loadHistory()

	go player.queue.Start()
}

// Stop stops the player.
func Stop(exit bool) {
	if exit {
		saveState()
		saveHistory()
	}

	sendPlayingStatus(false)

	mp.Player().Stop()
	mp.Player().Exit()
}

// Show shows the player.
func Show() {
	if playingStatus() {
		return
	}

	playingStatus(true)
	sendPlayingStatus(true)

	app.UI.QueueUpdateDraw(func() {
		app.UI.Layout.AddItem(player.flex, 2, 0, false)
		app.ResizeModal()
	})
}

// Hide hides the player.
func Hide() {
	if !playingStatus() {
		return
	}

	playingStatus(false)
	sendPlayingStatus(false)

	app.UI.QueueUpdateDraw(func() {
		app.UI.Layout.RemoveItem(player.flex)
		app.ResizeModal()
	})

	mp.Player().Stop()
	mp.Player().QueueClear()
}

// Resize resizes the player according to the screen width.
func Resize(width int) {
	if width == player.width {
		return
	}

	sendPlayerEvents()

	player.width = width
}

// ParseQuery parses the play-audio or play-video commandline
// parameters, and plays the provided URL.
func ParseQuery() {
	setup()

	mtype, uri, err := cmd.GetQueryParams("play")
	if err != nil {
		return
	}

	playFromURL(uri, mtype == "audio")
}

// Play plays the currently selected audio/video entry.
func Play(audio, current bool, mediaInfo ...inv.SearchData) {
	var err error
	var media string
	var info inv.SearchData

	if mediaInfo != nil {
		info = mediaInfo[0]
	} else {
		info, err = app.FocusedTableReference()
		if err != nil {
			return
		}
	}

	if audio {
		media = "audio"
	} else {
		media = "video"
	}

	if info.Type == "channel" {
		app.ShowError(fmt.Errorf("Player: Cannot play %s for channel type", media))
		return
	}

	go loadSelected(info, audio, current)
}

// IsQueueFocused returns whether the queue is focused.
func IsQueueFocused() bool {
	return player.queue.table != nil && player.queue.table.HasFocus()
}

// IsQueueEmpty returns whether the queue is empty.
func IsQueueEmpty() bool {
	return player.queue.table == nil || len(player.queue.data) == 0
}

// IsHistoryInputFocused returns whether the history search bar is focused.
func IsHistoryInputFocused() bool {
	return player.history.input != nil && player.history.input.HasFocus()
}

// Keybindings define the main player keybindings.
func Keybindings(event *tcell.EventKey) *tcell.EventKey {
	playerKeybindings(event)

	switch cmd.KeyOperation("Player", event) {
	case "Open":
		app.UI.FileBrowser.Show("Open playlist:", openPlaylist)

	case "History":
		showHistory()

	case "QueueAudio", "QueueVideo", "PlayAudio", "PlayVideo":
		playSelected(event.Rune())

	case "Queue":
		player.queue.Show()

	case "AudioURL", "VideoURL":
		playInputURL(event.Rune() == 'b')
		return nil
	}

	return event
}

// playerKeybindings define the playback-related keybindings
// for the player.
func playerKeybindings(event *tcell.EventKey) {
	var nokey bool

	switch cmd.KeyOperation("Player", event) {
	case "Stop":
		sendPlayingStatus(false)

	case "Pause":
		mp.Player().TogglePaused()

	case "SeekForward":
		mp.Player().SeekForward()

	case "SeekBackward":
		mp.Player().SeekBackward()

	case "ToggleLoop":
		mp.Player().ToggleLoopMode()

	case "ToggleShuffle":
		mp.Player().ToggleShuffled()

	case "ToggleMute":
		mp.Player().ToggleMuted()

	case "VolumeIncrease":
		mp.Player().VolumeIncrease()

	case "VolumeDecrease":
		mp.Player().VolumeDecrease()

	case "Prev":
		mp.Player().Prev()

	case "Next":
		mp.Player().Next()

	default:
		nokey = true
	}

	if !nokey {
		sendPlayerEvents()
	}
}

// playSelected determines the media type according
// to the key pressed, and plays the currently selected entry.
func playSelected(r rune) {
	audio := r == 'a' || r == 'A'
	current := r == 'A' || r == 'V'

	Play(audio, current)

	table := app.FocusedTable()
	if table != nil {
		table.InputHandler()(
			tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
			nil,
		)
	}
}

// playInputURL displays an inputbox and plays the entered URL.
func playInputURL(audio bool) {
	media := "video"
	if audio {
		media = "audio"
	}

	dofunc := func(text string) {
		playFromURL(text, audio)
	}

	app.UI.Status.SetInput("Play "+media+" for video/playlist URL or ID:", 0, true, dofunc, nil)
}

// playFromURL plays the given URL.
func playFromURL(text string, audio bool) {
	id, mtype, err := utils.GetVPIDFromURL(text)
	if err != nil {
		app.ShowError(err)
		return
	}

	info := inv.SearchData{
		Title: text,
		Type:  mtype,
	}

	if mtype == "video" {
		info.VideoID = id
	} else {
		info.PlaylistID = id
	}

	Play(audio, false, info)
}

// loadSelected loads the provided entry according to its type (video/playlist).
func loadSelected(info inv.SearchData, audio, current bool) {
	var title string

	err := player.lock.Acquire(context.Background(), 1)
	if err != nil {
		return
	}
	defer player.lock.Release(1)

	app.ShowInfo("Adding "+info.Type+" "+info.Title, true)

	switch info.Type {
	case "playlist":
		title, err = loadPlaylist(info.PlaylistID, audio)

	case "video":
		title, err = loadVideo(info.VideoID, audio)

	default:
		return
	}
	if err != nil {
		if err.Error() != "Rate-limit exceeded" {
			app.ShowError(err)
		}

		return
	}

	info.Title = title
	go addToHistory(info)

	app.ShowInfo("Added "+info.Title, false)

	if current && info.Type == "video" {
		mp.Player().QueuePlayLatest()
	}
}

// loadVideo loads a video into the media player.
func loadVideo(id string, audio bool) (string, error) {
	video, urls, err := inv.VideoLoadParams(id, audio)
	if err != nil {
		return "", err
	}

	mp.Player().LoadFile(
		video.Title,
		video.LengthSeconds,
		audio && video.LiveNow,
		urls...,
	)

	return video.Title, nil
}

// loadPlaylist loads all the entries in the playlist into the media player.
func loadPlaylist(plid string, audio bool) (string, error) {
	var err error

	playlist, err := inv.Playlist(plid, false, 1)
	if err != nil {
		return "", err
	}

	for _, p := range playlist.Videos {
		select {
		case <-client.Ctx().Done():
			return "", client.Ctx().Err()

		default:
		}

		loadVideo(p.VideoID, audio)
	}

	return playlist.Title, nil
}

// renderPlayer renders the media player within the app.
func renderPlayer(cancel context.CancelFunc) {
	var err error
	var width int
	var states []string
	var title, progress string

	app.UI.RLock()
	_, _, width, _ = player.desc.GetRect()
	app.UI.RUnlock()

	title, progress, states, err = getProgress(width)
	if err != nil {
		cancel()
		return
	}

	player.mutex.Lock()
	player.states = states
	player.mutex.Unlock()

	app.UI.QueueUpdateDraw(func() {
		player.desc.SetText(progress)
		player.title.SetText("[::b]" + tview.Escape(title))
	})
}

// playingStatusCheck monitors the playing status.
func playingStatusCheck() {
	var ctx context.Context
	var cancel context.CancelFunc

	for {
		playing, ok := <-player.channel
		if !ok {
			cancel()
			return
		}

		if ctx != nil && !playing {
			cancel()
		}
		if !playing {
			continue
		}

		ctx, cancel = context.WithCancel(context.Background())
		go playerUpdateLoop(ctx, cancel)
	}
}

// playerUpdateLoop updates the player.
func playerUpdateLoop(ctx context.Context, cancel context.CancelFunc) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			Hide()
			player.desc.SetText("")
			player.title.SetText("")
			return

		case <-player.events:
			renderPlayer(cancel)
			t.Reset(1 * time.Second)
			continue

		case <-t.C:
			renderPlayer(cancel)
		}
	}
}

// monitorMPVEvents monitors events sent from MPV.
func monitorMPVEvents() {
	for {
		select {
		case msg, ok := <-mp.Events.ErrorEvent:
			if !ok {
				return
			}

			app.ShowError(fmt.Errorf("Player: Unable to play %s", msg))

		case _, ok := <-mp.Events.FileLoadedEvent:
			if !ok {
				return
			}

			Show()
		}
	}
}

// openPlaylist loads the provided playlist file.
func openPlaylist(file string) {
	app.ShowInfo("Loading "+filepath.Base(file), true)

	err := mp.Player().LoadPlaylist(file, true, checkLiveURL)
	if err != nil {
		app.ShowError(err)
		return
	}

	Show()

	app.UI.QueueUpdateDraw(func() {
		player.queue.Show()
	})

	app.UI.FileBrowser.Hide()

	app.ShowInfo("Loaded "+filepath.Base(file), false)
}

// checkLiveURL checks whether a live video URL is expired or not.
// If it is expired, the video information is renewed.
func checkLiveURL(uri string, audio bool) bool {
	id, expired := inv.CheckLiveURL(uri, audio)

	if expired {
		if _, err := loadVideo(id, audio); err != nil {
			app.ShowError(fmt.Errorf("Player: Unable to renew live URL for video %s", id))
		}
	}

	return expired
}

// getProgress returns the progress bar and information
// of the currently playing track.
//
//gocyclo:ignore
func getProgress(width int) (string, string, []string, error) {
	var lhs, rhs string
	var states []string
	var state, mtype, totaltime, vol string

	ppos := mp.Player().QueuePosition()
	if ppos == -1 {
		return "", "", nil, fmt.Errorf("Player: Empty playlist")
	}

	title := mp.Player().Title(ppos)
	eof := mp.Player().Finished()
	paused := mp.Player().Paused()
	buffering := mp.Player().Buffering()
	shuffle := mp.Player().Shuffled()
	loop := mp.Player().LoopMode()
	mute := mp.Player().Muted()
	volume := mp.Player().Volume()

	duration := mp.Player().Duration()
	timepos := mp.Player().Position()
	currtime := utils.FormatDuration(timepos)

	if volume < 0 {
		vol = "0"
	} else {
		vol = strconv.Itoa(volume)
	}
	states = append(states, "volume "+vol)
	vol += "%"

	if timepos < 0 {
		timepos = 0
	}

	if duration <= 0 {
		duration = 1
	}

	if timepos > duration {
		timepos = duration
	}

	data := utils.GetDataFromURL(title)
	if data != nil {
		if t := data.Get("title"); t != "" {
			title = t
		}

		if l := data.Get("length"); l != "" {
			totaltime = l
		} else {
			totaltime = utils.FormatDuration(duration)
		}

		if m := data.Get("mediatype"); m != "" {
			mtype = m
		} else {
			mtype = mp.Player().MediaType()
		}
	} else {
		totaltime = utils.FormatDuration(duration)
		mtype = mp.Player().MediaType()
	}

	mtype = "(" + mtype + ")"

	width /= 2
	length := width * int(timepos) / int(duration)

	endlength := width - length
	if endlength < 0 {
		endlength = width
	}

	if shuffle {
		lhs += " S"
		states = append(states, "shuffle")
	}

	if mute {
		lhs += " M"
		states = append(states, "mute")
	}

	if loop != "" {
		states = append(states, loop)

		switch loop {
		case "loop-file":
			loop = "R-F"

		case "loop-playlist":
			loop = "R-P"
		}
	}

	if paused {
		if eof {
			state = "[]"
		} else {
			state = "||"
		}
	} else if buffering {
		state = "B"
	} else {
		state = ">"
	}

	rhs = " " + vol + " " + mtype
	lhs = loop + lhs + " " + state + " "
	progress := currtime + " |" + strings.Repeat("█", length) + strings.Repeat(" ", endlength) + "| " + totaltime

	return title, (lhs + progress + rhs), states, nil
}

// sendPlayingStatus sends status events to the player.
// If playing is true, the player is shown and vice-versa.
func sendPlayingStatus(playing bool) {
	select {
	case player.channel <- playing:
		return

	default:
	}
}

// sendPlayerEvents triggers updates for the player.
func sendPlayerEvents() {
	select {
	case player.events <- struct{}{}:
		return

	default:
	}
}

// playingStatus sets the current status of the player.
func playingStatus(set ...bool) bool {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if set != nil {
		player.playing = set[0]
	}

	return player.playing
}

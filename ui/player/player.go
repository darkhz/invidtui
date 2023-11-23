package player

import (
	"context"
	"fmt"
	"image/jpeg"
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

	infoID, thumbURI      string
	init, playing, toggle bool
	width                 int
	states                []string
	history               History

	channel chan bool
	events  chan struct{}

	image        *tview.Image
	flex, region *tview.Flex
	info         *tview.TextView
	quality      *tview.DropDown
	title, desc  *tview.TextView

	lock, render          *semaphore.Weighted
	infoCancel, imgCancel context.CancelFunc
	mutex                 sync.Mutex
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

	player.image = tview.NewImage()
	player.image.SetBackgroundColor(tcell.ColorDefault)
	player.image.SetDithering(tview.DitheringFloydSteinberg)

	player.info = tview.NewTextView()
	player.info.SetDynamicColors(true)
	player.info.SetTextAlign(tview.AlignCenter)
	player.info.SetBackgroundColor(tcell.ColorDefault)

	player.quality = tview.NewDropDown()
	player.quality.SetLabel("[green::b]Quality: ")
	player.quality.SetBackgroundColor(tcell.ColorDefault)
	player.quality.SetFieldTextColor(tcell.ColorOrangeRed)
	player.quality.SetFieldBackgroundColor(tcell.ColorDefault)
	player.quality.List().
		SetMainTextColor(tcell.ColorWhite).
		SetBackgroundColor(tcell.ColorDefault).
		SetBorder(true)

	player.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(player.title, 1, 0, false).
		AddItem(player.desc, 1, 0, false)
	player.flex.SetBackgroundColor(tcell.ColorDefault)

	player.region = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(player.image, 0, 1, false).
		AddItem(player.info, 0, 1, false)
	player.region.SetBackgroundColor(tcell.ColorDefault)

	player.lock = semaphore.NewWeighted(10)
	player.render = semaphore.NewWeighted(1)
}

// Start starts the player and loads its history and states.
func Start() {
	setup()

	loadState()
	loadHistory()

	go playingStatusCheck()
	go monitorMPVEvents()
	go player.queue.Start()
}

// Stop stops the player.
func Stop() {
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

// ToggleInfo toggle the player information view.
func ToggleInfo(hide ...struct{}) {
	if hide != nil || player.toggle {
		player.toggle = false
		player.infoID = ""

		infoContext(true, struct{}{})

		app.UI.Region.Clear().
			AddItem(app.UI.Pages, 0, 1, true)

		if player.region.GetItemCount() > 2 {
			player.region.RemoveItemIndex(1)
		}

		return
	}

	if !player.toggle && playingStatus() {
		player.toggle = true

		box := tview.NewBox()
		box.SetBackgroundColor(tcell.ColorDefault)

		app.UI.Region.Clear().
			AddItem(player.region, 25, 0, false).
			AddItem(box, 1, 0, false).
			AddItem(app.VerticalLine(), 1, 0, false).
			AddItem(box, 1, 0, false).
			AddItem(app.UI.Pages, 0, 1, true)

		Resize(0, struct{}{})
	}
}

// Hide hides the player.
func Hide() {
	if !playingStatus() {
		return
	}

	playingStatus(false)
	sendPlayingStatus(false)
	app.UI.Status.InitializingTag(false)

	app.UI.QueueUpdateDraw(func() {
		app.UI.Layout.RemoveItem(player.flex)
		app.ResizeModal()
	})

	mp.Player().Stop()
	mp.Player().QueueClear()
}

// Resize resizes the player according to the screen width.
func Resize(width int, force ...struct{}) {
	if force != nil {
		_, _, w, _ := app.UI.Area.GetRect()
		width = w

		goto ResizePlayer
	}

	if width == player.width {
		return
	}

ResizePlayer:
	sendPlayerEvents()
	app.UI.Region.ResizeItem(player.region, (width / 4), 0)

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

// IsInfoShown returns whether the player information is shown.
func IsInfoShown() bool {
	return player.region != nil && player.toggle
}

// IsPlayerShown returns whether the player is shown.
func IsPlayerShown() bool {
	return playingStatus()
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

	switch cmd.KeyOperation(event, cmd.KeyContextQueue) {
	case cmd.KeyPlayerOpenPlaylist:
		app.UI.FileBrowser.Show("Open playlist:", openPlaylist)

	case cmd.KeyPlayerHistory:
		showHistory()

	case cmd.KeyPlayerInfo:
		ToggleInfo()

	case cmd.KeyPlayerInfoScrollDown:
		player.info.InputHandler()(tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone), nil)
		return nil

	case cmd.KeyPlayerInfoScrollUp:
		player.info.InputHandler()(tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone), nil)
		return nil

	case cmd.KeyPlayerInfoChangeQuality:
		changeImageQuality()

	case cmd.KeyPlayerQueueAudio, cmd.KeyPlayerQueueVideo, cmd.KeyPlayerPlayAudio, cmd.KeyPlayerPlayVideo:
		playSelected(event.Rune())

	case cmd.KeyQueue:
		player.queue.Show()

	case cmd.KeyAudioURL, cmd.KeyVideoURL:
		playInputURL(event.Rune() == 'b')
		return nil
	}

	return event
}

// playerKeybindings define the playback-related keybindings
// for the player.
func playerKeybindings(event *tcell.EventKey) {
	var nokey bool

	switch cmd.KeyOperation(event, cmd.KeyContextPlayer) {
	case cmd.KeyPlayerStop:
		sendPlayingStatus(false)

	case cmd.KeyPlayerSeekForward:
		mp.Player().SeekForward()

	case cmd.KeyPlayerSeekBackward:
		mp.Player().SeekBackward()

	case cmd.KeyPlayerTogglePlay:
		mp.Player().TogglePaused()

	case cmd.KeyPlayerToggleLoop:
		mp.Player().ToggleLoopMode()

	case cmd.KeyPlayerToggleShuffle:
		mp.Player().ToggleShuffled()

	case cmd.KeyPlayerToggleMute:
		mp.Player().ToggleMuted()

	case cmd.KeyPlayerVolumeIncrease:
		mp.Player().VolumeIncrease()

	case cmd.KeyPlayerVolumeDecrease:
		mp.Player().VolumeDecrease()

	case cmd.KeyPlayerPrev:
		mp.Player().Prev()

	case cmd.KeyPlayerNext:
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
func loadVideo(id string, audio bool, ctx ...context.Context) (string, error) {
	video, urls, err := inv.VideoLoadParams(id, audio, ctx...)
	if err != nil {
		return "", err
	}

	player.queue.currentVideo(id, &video)

	if ctx == nil {
		mp.Player().LoadFile(
			video.Title,
			video.LengthSeconds,
			audio && video.LiveNow,
			urls...,
		)
	}

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
	app.UI.RLock()
	_, _, width, _ := player.desc.GetRect()
	app.UI.RUnlock()

	id, title, progress, states, err := updateProgressAndInfo(width)
	if err != nil {
		cancel()
		return
	}

	player.mutex.Lock()
	cmd.Settings.PlayerStates = states
	player.mutex.Unlock()

	app.UI.QueueUpdateDraw(func() {
		renderInfo(id, title)
		player.desc.SetText(progress)
		player.title.SetText("[::b]" + tview.Escape(title))
	})
}

// changeImageQuality sets or displays options to change the quality of the image
// in the player information area.
//
//gocyclo:ignore
func changeImageQuality(set ...struct{}) {
	var prev string
	var options []string

	video := player.queue.currentVideo(player.infoID)
	if video == nil {
		return
	}

	start, pos := -1, -1
	for i, thumb := range video.Thumbnails {
		if thumb.Quality == "start" {
			start = i
			break
		}

		if thumb.URL == player.thumbURI {
			pos = i
		}

		if set == nil {
			text := fmt.Sprintf("%dx%d", thumb.Width, thumb.Height)
			if prev == text {
				continue
			}

			prev = text
			options = append(options, text)
		}
	}
	if start >= 0 && (pos < 0 || player.thumbURI == "") {
		pos = len(options) - 1
		player.thumbURI = video.Thumbnails[start-1].URL
	}
	if set != nil || !player.toggle || player.quality.HasFocus() {
		return
	}

	thumb := video.Thumbnails[pos]
	for i, option := range options {
		if option == fmt.Sprintf("%dx%d", thumb.Width, thumb.Height) {
			pos = i
			break
		}
	}

	player.region.Clear().
		AddItem(player.image, 0, 1, false).
		AddItem(player.quality, 1, 0, false).
		AddItem(player.info, 0, 1, false)

	player.quality.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			app.SetPrimaryFocus()
			player.region.RemoveItem(player.quality)
		}

		return event
	})
	player.quality.SetDrawFunc(func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
		_, _, w, _ := player.quality.List().GetRect()

		dx := ((width / 2) - w) + 2
		if dx < 0 {
			dx = 0
		}

		return dx, y, width, height
	})

	player.quality.SetOptions(options, func(text string, index int) {
		if index < 0 {
			return
		}

		for i, thumb := range video.Thumbnails {
			if text == fmt.Sprintf("%dx%d", thumb.Width, thumb.Height) {
				index = i
				break
			}
		}

		if uri := video.Thumbnails[index].URL; uri != player.thumbURI {
			player.thumbURI = uri
			go renderInfoImage(infoContext(true), player.infoID, filepath.Base(uri), struct{}{})
		}
	})
	player.quality.SetCurrentOption(pos)
	player.quality.InputHandler()(
		tcell.NewEventKey(tcell.KeyEnter, ' ', tcell.ModNone),
		func(p tview.Primitive) {
			app.UI.SetFocus(p)
		},
	)

	app.UI.SetFocus(player.quality)

	go app.UI.Draw()
}

// renderInfo renders the track information.
func renderInfo(id, title string, force ...struct{}) {
	if !player.render.TryAcquire(1) {
		return
	}
	defer player.render.Release(1)

	if force == nil && (id == "" || id == player.infoID || !player.toggle) {
		return
	}

	player.infoID = id
	player.image.SetImage(nil)
	if player.region.GetItemCount() > 2 {
		player.region.RemoveItemIndex(1)
	}

	player.info.SetText("[::b]Loading information...")

	video := player.queue.currentVideo(id)
	if video == nil {
		go func(ctx context.Context, id, title string) {
			err := player.render.Acquire(ctx, 1)
			if err != nil {
				return
			}

			_, err = loadVideo(id, true, ctx)
			player.render.Release(1)

			app.UI.QueueUpdateDraw(func() {
				if err != nil {
					if ctx.Err() != context.Canceled {
						player.info.SetText("[::b]No information for\n" + title)
					}

					return
				}

				renderInfo(id, title, struct{}{})
			})
		}(infoContext(false), id, title)

		return
	}

	text := "\n"
	if video.Author != "" {
		text += fmt.Sprintf("[::bu]%s[-:-:-]\n\n", video.Author)
	}
	if video.PublishedText != "" {
		text += fmt.Sprintf("[lightpink::b]Uploaded %s[-:-:-]\n", video.PublishedText)
	}
	text += fmt.Sprintf(
		"[aqua::b]%s views[-:-:-] / [red::b]%s likes[-:-:-] / [purple::b]%s subscribers[-:-:-]\n\n",
		utils.FormatNumber(video.ViewCount),
		utils.FormatNumber(video.LikeCount),
		video.SubCountText,
	)
	text += "[::b]" + tview.Escape(video.Description)

	player.info.SetText(text)
	player.info.ScrollToBeginning()

	changeImageQuality(struct{}{})
	go renderInfoImage(infoContext(true), id, filepath.Base(player.thumbURI))
}

// renderInfoImage renders the image for the track information display.
func renderInfoImage(ctx context.Context, id, image string, change ...struct{}) {
	if image == "." {
		return
	}

	app.ShowInfo("Player: Loading image", true, change != nil)

	thumbdata, err := inv.VideoThumbnail(ctx, id, image)
	if err != nil {
		if ctx.Err() != context.Canceled {
			app.ShowError(fmt.Errorf("Player: Unable to download thumbnail"))
		}

		app.ShowInfo("", false, change != nil)

		return
	}

	thumbnail, err := jpeg.Decode(thumbdata.Body)
	if err != nil {
		app.ShowError(fmt.Errorf("Player: Unable to decode thumbnail"))
		return
	}

	app.UI.QueueUpdateDraw(func() {
		player.image.SetImage(thumbnail)
	})

	app.ShowInfo("Player: Image loaded", false, change != nil)
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
			ToggleInfo(struct{}{})
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
		case _, ok := <-mp.Events.StartEvent:
			if !ok {
				return
			}

			if !player.status.Load() && player.setting.Load() {
				app.UI.Status.InitializingTag(true)
			}

		case msg, ok := <-mp.Events.ErrorEvent:
			if !ok {
				return
			}

			app.ShowError(fmt.Errorf("Player: Unable to play %s", msg))

		case _, ok := <-mp.Events.FileLoadedEvent:
			if !ok {
				return
			}

			app.UI.Status.InitializingTag(false)
			Show()
		}
	}
}

// openPlaylist loads the provided playlist file.
func openPlaylist(file string) {
	app.ShowInfo("Loading "+filepath.Base(file), true)

	err := mp.Player().LoadPlaylist(player.queue.Ctx(false), file, true, checkLiveURL)
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

// updateProgressAndInfo returns the progress bar and information
// of the currently playing track, and updates the track information.
//
//gocyclo:ignore
func updateProgressAndInfo(width int) (string, string, string, []string, error) {
	var lhs, rhs string
	var states []string
	var state, mtype, totaltime, vol string

	ppos := mp.Player().QueuePosition()
	if ppos == -1 {
		return "", "", "", nil, fmt.Errorf("Player: Empty playlist")
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

	return data.Get("id"), title, (lhs + progress + rhs), states, nil
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

// infoContext returns a new context for loading the player information.
func infoContext(image bool, all ...struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	if image {
		if player.imgCancel != nil {
			player.imgCancel()
		}

		player.imgCancel = cancel

		if all == nil {
			goto InfoContext
		}
	}

	if player.infoCancel != nil {
		player.infoCancel()
	}

	player.infoCancel = cancel

InfoContext:
	return ctx
}

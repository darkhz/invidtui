package player

import (
	"context"
	"fmt"
	"image/jpeg"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Player stores the layout for the player.
type Player struct {
	queue   Queue
	fetcher Fetcher

	infoID, thumbURI string
	init             bool
	width            int
	history          History
	states           []string

	channel chan bool
	events  chan struct{}

	image        *tview.Image
	flex, region *tview.Flex
	info         *tview.TextView
	quality      *tview.DropDown
	title, desc  *tview.TextView

	property theme.ThemeProperty

	ctx                           context.Context
	cancel, infoCancel, imgCancel context.CancelFunc

	status, setting, toggle atomic.Bool

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

	player.property = theme.ThemeProperty{
		Context: theme.ThemeContextPlayer,
		Item:    theme.ThemeBackground,
	}

	player.channel = make(chan bool, 10)
	player.events = make(chan struct{}, 100)

	player.title, player.desc =
		theme.NewTextView(player.property),
		theme.NewTextView(player.property)
	player.desc.SetTextAlign(tview.AlignCenter)
	player.title.SetTextAlign(tview.AlignCenter)

	player.image = theme.NewImage(
		player.property.SetContext(theme.ThemeContextPlayerInfo),
	)
	player.image.SetDithering(tview.DitheringFloydSteinberg)

	player.info = theme.NewTextView(
		player.property.SetContext(theme.ThemeContextPlayerInfo),
	)
	player.info.SetTextAlign(tview.AlignCenter)

	player.quality = theme.NewDropDown(
		player.property.SetContext(theme.ThemeContextPlayerInfo),
		"Quality:",
	)
	player.quality.List().SetBorder(true)

	player.flex = theme.NewFlex(player.property).
		SetDirection(tview.FlexRow).
		AddItem(player.title, 1, 0, false).
		AddItem(player.desc, 1, 0, false)

	player.region = theme.NewFlex(
		player.property.SetContext(theme.ThemeContextPlayerInfo),
	).
		SetDirection(tview.FlexRow).
		AddItem(player.image, 0, 1, false).
		AddItem(player.info, 0, 1, false)

	player.lock = semaphore.NewWeighted(10)
}

// Start starts the player and loads its history and states.
func Start() {
	setup()
	player.queue.Setup()
	player.fetcher.Setup()

	loadState()
	loadHistory()

	mp.SetEventHandler(mediaEventHandler)

	go playingStatusCheck()
}

// Stop stops the player.
func Stop() {
	sendPlayingStatus(false)

	player.mutex.Lock()
	cmd.Settings.PlayerStates = player.states
	player.mutex.Unlock()

	mp.Player().Stop()
	mp.Player().Exit()
}

// Show shows the player.
func Show() {
	if player.status.Load() || !player.setting.Load() {
		return
	}

	player.status.Store(true)
	sendPlayingStatus(true)

	app.UI.QueueUpdateDraw(func() {
		app.UI.Layout.AddItem(player.flex, 2, 0, false)
		app.ResizeModal()
	})
}

// ToggleInfo toggle the player information view.
func ToggleInfo(hide ...struct{}) {
	if hide != nil || player.toggle.Load() {
		player.toggle.Store(false)
		infoID("")

		infoContext(true, struct{}{})

		app.UI.Region.Clear().
			AddItem(app.UI.Pages, 0, 1, true)

		if player.region.GetItemCount() > 2 {
			player.region.RemoveItemIndex(1)
		}

		return
	}

	if !player.toggle.Load() && player.status.Load() {
		player.toggle.Store(true)

		property := player.property.SetContext(theme.ThemeContextPlayerInfo)
		box := theme.NewBox(property)
		vbox := app.VerticalLine(property.SetItem(theme.ThemeBorder))

		app.UI.Region.Clear().
			AddItem(player.region, 0, 1, false).
			AddItem(box, 1, 0, false).
			AddItem(vbox, 1, 0, false).
			AddItem(box, 1, 0, false).
			AddItem(app.UI.Pages, 0, 2, true)

		Resize(0, struct{}{})

		if data, ok := player.queue.GetCurrent(); ok {
			go renderInfo(data.Reference)
		}
	}
}

// Hide hides the player.
func Hide() {
	if player.setting.Load() {
		return
	}

	Context(true)
	player.queue.Context(true)

	player.status.Store(false)
	sendPlayingStatus(false)

	app.UI.QueueUpdateDraw(func() {
		ToggleInfo(struct{}{})
		app.ResizeModal()
		app.UI.Layout.RemoveItem(player.flex)
	})

	mp.Player().Stop()

	player.queue.Clear()
	player.fetcher.CancelAll(true)
}

// Context cancels and/or returns the player's context.
func Context(cancel bool) context.Context {
	if cancel && player.ctx != nil {
		player.cancel()
	}

	if player.ctx == nil || player.ctx.Err() == context.Canceled {
		player.ctx, player.cancel = context.WithCancel(context.Background())
	}

	return player.ctx
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

	player.setting.Store(true)

	go loadSelected(info, audio, current)
}

// IsInfoShown returns whether the player information is shown.
func IsInfoShown() bool {
	return player.region != nil && player.toggle.Load()
}

// IsPlayerShown returns whether the player is shown.
func IsPlayerShown() bool {
	return player.status.Load()
}

// IsQueueFocused returns whether the queue is focused.
func IsQueueFocused() bool {
	return player.queue.table != nil && player.queue.table.HasFocus()
}

// IsQueueEmpty returns whether the queue is empty.
func IsQueueEmpty() bool {
	return player.queue.table == nil || player.queue.Count() == 0
}

// IsQueueAreaFocused returns whether the queue area is focused.
func IsQueueAreaFocused() bool {
	pages := player.queue.pages
	if pages == nil {
		return false
	}

	_, item := pages.GetFrontPage()

	return item.HasFocus()
}

// IsQueueRecommendsFocused returns whether the queue recommendations page is focused.
func IsQueueRecommendsFocused() bool {
	return player.queue.recommends != nil && player.queue.recommends.HasFocus()
}

// IsHistoryInputFocused returns whether the history search bar is focused.
func IsHistoryInputFocused() bool {
	return player.history.input != nil && player.history.input.HasFocus()
}

// Keybindings define the main player keybindings.
func Keybindings(event *tcell.EventKey) *tcell.EventKey {
	playerKeybindings(event)

	operation := keybinding.KeyOperation(event, keybinding.KeyContextQueue, keybinding.KeyContextFetcher)

	switch operation {
	case keybinding.KeyPlayerOpenPlaylist:
		app.UI.FileBrowser.Show("Open playlist:", openPlaylist)

	case keybinding.KeyPlayerHistory:
		showHistory()

	case keybinding.KeyPlayerInfo:
		ToggleInfo()

	case keybinding.KeyPlayerInfoScrollDown:
		player.info.InputHandler()(tcell.NewEventKey(tcell.KeyDown, ' ', tcell.ModNone), nil)
		return nil

	case keybinding.KeyPlayerInfoScrollUp:
		player.info.InputHandler()(tcell.NewEventKey(tcell.KeyUp, ' ', tcell.ModNone), nil)
		return nil

	case keybinding.KeyPlayerInfoChangeQuality:
		changeImageQuality()

	case keybinding.KeyPlayerQueueAudio, keybinding.KeyPlayerQueueVideo, keybinding.KeyPlayerPlayAudio, keybinding.KeyPlayerPlayVideo:
		playSelected(operation)

	case keybinding.KeyFetcher:
		player.fetcher.Show()

	case keybinding.KeyQueue:
		player.queue.Show()

	case keybinding.KeyQueueCancel:
		player.queue.Context(true)

	case keybinding.KeyAudioURL, keybinding.KeyVideoURL:
		playInputURL(event.Rune() == 'b')
		return nil
	}

	return event
}

// playerKeybindings define the playback-related keybindings
// for the player.
func playerKeybindings(event *tcell.EventKey) {
	var nokey bool

	switch keybinding.KeyOperation(event, keybinding.KeyContextPlayer) {
	case keybinding.KeyPlayerStop:
		player.setting.Store(false)
		go Hide()

	case keybinding.KeyPlayerSeekForward:
		mp.Player().SeekForward()

	case keybinding.KeyPlayerSeekBackward:
		mp.Player().SeekBackward()

	case keybinding.KeyPlayerTogglePlay:
		mp.Player().TogglePaused()

	case keybinding.KeyPlayerToggleLoop:
		player.queue.ToggleRepeatMode()

	case keybinding.KeyPlayerToggleShuffle:
		player.queue.ToggleShuffle()

	case keybinding.KeyPlayerToggleMute:
		mp.Player().ToggleMuted()

	case keybinding.KeyPlayerVolumeIncrease:
		mp.Player().VolumeIncrease()

	case keybinding.KeyPlayerVolumeDecrease:
		mp.Player().VolumeDecrease()

	case keybinding.KeyPlayerPrev:
		player.queue.Previous(struct{}{})

	case keybinding.KeyPlayerNext:
		player.queue.Next(struct{}{})

	default:
		nokey = true
	}

	if !nokey {
		sendPlayerEvents()
	}
}

// playSelected determines the media type according
// to the key pressed, and plays the currently selected entry.
func playSelected(key keybinding.Key) {
	if player.queue.IsQueueShown() {
		player.queue.Keybindings(keybinding.KeyEvent(key))
		return
	}

	audio := key == keybinding.KeyPlayerQueueAudio || key == keybinding.KeyPlayerPlayAudio
	current := key == keybinding.KeyPlayerPlayAudio || key == keybinding.KeyPlayerPlayVideo

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
	var err error

	if info.Type != "video" && info.Type != "playlist" {
		return
	}

	info, err = player.fetcher.Fetch(info, audio)
	if err != nil {
		return
	}

	addToHistory(info)

	if current && info.Type == "video" {
		player.queue.SelectRecentEntry()
	}
}

// renderPlayer renders the media player within the app.
func renderPlayer() {
	var width int
	var marker string

	app.UI.Lock()
	_, _, width, _ = player.desc.GetRect()
	if m := player.queue.marker; m != nil {
		marker = m.Text
	}
	app.UI.Unlock()

	title, desc, states := updateProgressAndInfo(
		player.queue.GetTitle(),
		marker,
		width-10,
	)
	player.title.SetText(title)
	player.desc.SetText(desc)
	app.DrawPrimitives(player.flex)

	player.mutex.Lock()
	player.states = states
	player.mutex.Unlock()
}

// changeImageQuality sets or displays options to change the quality of the image
// in the player information area.
//
//gocyclo:ignore
func changeImageQuality(set ...struct{}) {
	var prev string
	var options []string

	data, ok := player.queue.GetCurrent()
	if !ok {
		return
	}

	video := data.Reference

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
	if set != nil || !player.toggle.Load() || player.quality.HasFocus() {
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
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeyClose:
			app.SetPrimaryFocus()
			player.region.RemoveItem(player.quality)
		}

		return event
	})
	theme.WrapDrawFunc(
		player.quality,
		player.property.SetContext(theme.ThemeContextPlayerInfo),
		func(screen tcell.Screen, x, y, width, height int) (int, int, int, int) {
			_, _, w, _ := player.quality.List().GetRect()

			dx := ((width / 2) - w) + 2
			if dx < 0 {
				dx = 0
			}

			return dx, y, width, height
		},
	)

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
			go renderInfoImage(infoContext(true), infoID(), filepath.Base(uri), struct{}{})
		}
	})
	player.quality.SetCurrentOption(pos)
	player.quality.InputHandler()(
		keybinding.KeyEvent(keybinding.KeySelect),
		func(p tview.Primitive) {
			app.UI.SetFocus(p)
		},
	)

	app.UI.SetFocus(player.quality)

	go app.UI.Draw()
}

// renderInfo renders the track information.
func renderInfo(video inv.VideoData, force ...struct{}) {
	if force == nil && (video.VideoID == infoID() || !player.toggle.Load()) {
		return
	}

	infoContext(true, struct{}{})

	builder := theme.NewTextBuilder(theme.ThemeContextPlayerInfo)

	app.ConditionalDraw(func() bool {
		builder.Append(theme.ThemeDescription, "info", "Loading information...")
		player.info.SetText(builder.Get())

		player.image.SetImage(nil)

		return IsInfoShown()
	})

	if video.Thumbnails == nil {
		go func(ctx context.Context, pos int, v inv.VideoData, b theme.ThemeTextBuilder) {
			v, err := inv.Video(v.VideoID, ctx)
			if err != nil {
				if ctx.Err() != context.Canceled {
					app.ConditionalDraw(func() bool {
						b.Format(theme.ThemeDescription, "info", "No information for\n%s", v.Title)
						player.info.SetText(b.Get())

						return IsInfoShown()
					})
				}

				return
			}

			player.queue.SetReference(pos, v, struct{}{})
			renderInfo(v, struct{}{})
		}(infoContext(false), player.queue.Position(), video, builder)

		return
	}

	app.ConditionalDraw(func() bool {
		infoID(video.VideoID)
		if player.region.GetItemCount() > 2 {
			player.region.RemoveItemIndex(1)
		}

		builder.AppendText("\n")
		if video.Author != "" {
			builder.Append(theme.ThemeAuthor, "author", tview.Escape(video.Author))
			builder.AppendText("\n\n")
		}
		if video.PublishedText != "" {
			builder.Append(theme.ThemePublished, "published", "Uploaded "+video.PublishedText)
			builder.AppendText("\n\n")
		}

		builder.Format(theme.ThemeViews, "views", " %s views / ", utils.FormatNumber(video.ViewCount))
		builder.Format(theme.ThemeLikes, "likes", "%s likes / ", utils.FormatNumber(video.LikeCount))
		builder.Format(theme.ThemeSubscribers, "info", "%s subscribers", video.SubCountText)
		builder.AppendText("\n\n")

		builder.Append(theme.ThemeDescription, "description", tview.Escape(video.Description))

		player.info.SetText(builder.Get())
		player.info.ScrollToBeginning()

		changeImageQuality(struct{}{})

		return IsInfoShown()
	})

	go renderInfoImage(infoContext(true), video.VideoID, filepath.Base(player.thumbURI))
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

	app.ConditionalDraw(func() bool {
		player.image.SetImage(thumbnail)

		return IsInfoShown()
	})

	app.ShowInfo("Player: Image loaded", false, change != nil)
}

// playingStatusCheck monitors the playing status.
func playingStatusCheck() {
	for {
		playing, ok := <-player.channel
		if !ok {
			return
		}
		if !playing {
			continue
		}

		Context(false)
		go playerUpdateLoop(player.ctx, player.cancel)
	}
}

// playerUpdateLoop updates the player.
func playerUpdateLoop(ctx context.Context, cancel context.CancelFunc) {
	evCtx, evCancel := context.WithCancel(context.Background())
	go func(c context.Context) {
		for {
			select {
			case <-c.Done():
				player.desc.Clear()
				player.title.Clear()

				app.UI.Lock()
				app.DrawPrimitives(player.flex, app.UI.Status.Pages)
				app.UI.Unlock()

				return

			case <-player.events:
				renderPlayer()
			}
		}
	}(evCtx)

	for {
		select {
		case <-ctx.Done():
			evCancel()
			return

		case <-time.After(1 * time.Second):
			renderPlayer()
		}
	}
}

// mediaEventHandler monitors events sent from MPV.
func mediaEventHandler(event mp.MediaEvent) {
	switch event {
	case mp.EventInProgress:
		player.queue.MarkPlayingEntry(EntryPlaying)
		player.queue.SetAndClearTimestamp(player.queue.Position())

	case mp.EventLoading:
		player.queue.MarkPlayingEntry(EntryLoading)

	case mp.EventEnd:
		player.queue.MarkPlayingEntry(EntryStopped)
		player.queue.AutoPlay(false)

	case mp.EventError:
		if data, ok := player.queue.GetCurrent(); !ok {
			app.ShowError(fmt.Errorf("Player: Unable to play %q", data.Reference.Title))
		}

		player.queue.AutoPlay(true)
	}
}

// openPlaylist loads the provided playlist file.
func openPlaylist(file string) {
	app.ShowInfo("Loading "+filepath.Base(file), true)

	player.setting.Store(true)

	err := player.queue.LoadPlaylist(player.queue.Context(false), file, true)
	if err != nil {
		app.ShowError(err)
		return
	}

	app.UI.QueueUpdateDraw(func() {
		player.queue.Show()
		app.UI.FileBrowser.Hide()
	})

	app.ShowInfo("Loaded "+filepath.Base(file), false)
}

// updateProgressAndInfo returns the progress bar and information
// of the currently playing track, and updates the track information.
//
//gocyclo:ignore
func updateProgressAndInfo(title, marker string, width int) (string, string, []string) {
	var states []string

	eof := mp.Player().Finished()
	paused := mp.Player().Paused()
	buffering := mp.Player().Buffering()
	shuffle := player.queue.GetShuffleMode()
	mute := mp.Player().Muted()
	volume := mp.Player().Volume()

	duration := mp.Player().Duration()
	timepos := mp.Player().Position()

	builder := theme.NewTextBuilder(theme.ThemeContextPlayer)

	if timepos < 0 {
		timepos = 0
	}
	if duration < 0 {
		duration = 0
	}
	if timepos > duration {
		timepos = duration
	}

	width /= 2
	length := width * int(timepos)
	if duration > 0 {
		length /= int(duration)
	}

	endlength := width - length
	if endlength < 0 {
		endlength = width
	}

	loopsetting := ""
	switch player.queue.GetRepeatMode() {
	case mp.RepeatModeFile:
		builder.Append(theme.ThemeLoop, "loop", "R-F ")
		loopsetting = "loop-file"

	case mp.RepeatModePlaylist:
		builder.Append(theme.ThemeLoop, "loop", "R-P ")
		loopsetting = "loop-playlist"
	}
	if loopsetting != "" {
		states = append(states, loopsetting)
	}

	if shuffle {
		builder.Append(theme.ThemeShuffle, "shuffle", "S ")
		states = append(states, "shuffle")
	}
	if mute {
		builder.Append(theme.ThemeMute, "mute", "M ")
		states = append(states, "mute")
	}

	switch {
	case paused:
		if eof {
			builder.Append(theme.ThemeStop, "stop", "[] ")
		} else {
			builder.Append(theme.ThemePause, "pause", "|| ")
		}

	case buffering:
		builder.Append(theme.ThemeBuffer, "buffer", "B")
		if pct := mp.Player().BufferPercentage(); pct >= 0 {
			builder.Format(theme.ThemeBuffer, "buffer", "(%s%%)", strconv.Itoa(pct))
		}
		builder.AppendText(" ")

	default:
		builder.Append(theme.ThemePlay, "play", "> ")
	}

	builder.Append(theme.ThemeDuration, "duration", utils.FormatDuration(timepos))

	builder.Start(theme.ThemeProgressBar, "progress")
	builder.AppendText(" |")
	for i := 0; i < length; i++ {
		builder.AppendText("█")
	}
	for i := 0; i < endlength; i++ {
		builder.AppendText(" ")
	}
	builder.AppendText("| ")
	builder.Finish()

	builder.Append(theme.ThemeTotalDuration, "totalduration", utils.FormatDuration(duration))

	vol := strconv.Itoa(volume)
	if vol == "" {
		vol = "0"
	}
	states = append(states, "volume "+vol)
	builder.Format(theme.ThemeVolume, "volume", " %s%% ", vol)
	builder.Format(theme.ThemeMediaType, "mediatype", "(%s) ", player.queue.GetMediaType())
	builder.AppendText(marker)

	title = theme.SetTextStyle(
		"title",
		title,
		player.property.Context,
		theme.ThemeTitle,
	)

	return title, builder.Get(), states
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

// infoID returns or sets the video ID for the current player information.
func infoID(set ...string) string {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if set != nil {
		player.infoID = set[0]
	}

	return player.infoID
}

// infoContext returns a new context for loading the player information.
func infoContext(image bool, all ...struct{}) context.Context {
	player.mutex.Lock()
	defer player.mutex.Unlock()

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

package player

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/ui/view"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/etherlabsio/go-m3u8/m3u8"
	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Queue describes the media queue.
type Queue struct {
	init, moveMode bool
	prevrow        int

	status chan struct{}

	modal             *app.Modal
	table, recommends *tview.Table
	tabs              *tview.TextView
	pages             *tview.Pages
	marker            *tview.TableCell

	lock *semaphore.Weighted

	ctx, playctx       context.Context
	cancel, playcancel context.CancelFunc

	position, repeat atomic.Int32
	shuffle, audio   atomic.Bool
	title            atomic.Value

	store      *deque.Deque[*QueueData]
	storeMutex sync.Mutex

	current *QueueData

	tview.TableContentReadOnly
}

// QueueData describes the queue entry data.
type QueueData struct {
	URI                       [2]string
	Reference                 inv.VideoData
	Columns                   [QueueColumnSize]*tview.TableCell
	Audio, Playing, HasPlayed bool
	Timestamp                 *int64
}

// QueueEntryStatus describes the status of a queue entry.
type QueueEntryStatus string

const (
	EntryFetching QueueEntryStatus = "Fetching"
	EntryLoading  QueueEntryStatus = "Loading"
	EntryPlaying  QueueEntryStatus = "Playing"
	EntryStopped  QueueEntryStatus = "Stopped"
)

const (
	QueueColumnSize = 10

	QueuePlayingMarker = QueueColumnSize - 2
	QueueMediaMarker   = QueueColumnSize - 5
)

// Setup sets up the queue.
func (q *Queue) Setup() {
	if q.init {
		return
	}

	property := q.ThemeProperty()
	defer q.tabsHandler()

	q.store = deque.New[*QueueData](100)

	q.status = make(chan struct{}, 100)

	q.pages = theme.NewPages(property)

	q.tabs = theme.NewTextView(property)
	q.tabs.SetTextAlign(tview.AlignCenter)

	q.table = theme.NewTable(property)
	q.table.SetContent(q)
	q.table.SetSelectable(true, false)
	q.table.SetInputCapture(q.Keybindings)
	q.table.SetSelectionChangedFunc(q.selectorHandler)
	q.table.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextQueue, q.table)
	})

	q.recommends = theme.NewTable(property)
	q.recommends.SetSelectable(true, false)
	q.recommends.SetInputCapture(q.Keybindings)
	q.recommends.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextQueue, q.recommends)
	})

	flex := theme.NewFlex(property).
		SetDirection(tview.FlexRow).
		AddItem(q.pages, 0, 1, true).
		AddItem(app.HorizontalLine(property), 1, 0, false).
		AddItem(q.tabs, 1, 0, false)

	q.modal = app.NewModal("queue", "Queue", flex, 40, 0, property)

	q.lock = semaphore.NewWeighted(1)

	q.init = true
}

// Show shows the player queue.
func (q *Queue) Show() {
	if q.IsOpen() || q.Count() == 0 || !player.setting.Load() {
		return
	}

	q.modal.Show(true)
	q.sendStatus()
}

// Hide hides the player queue.
func (q *Queue) Hide() {
	q.modal.Exit(false)
}

// IsOpen returns whether the queue is open.
func (q *Queue) IsOpen() bool {
	return q.modal != nil && q.modal.Open
}

// IsQueueShown returns whether the queue page is shown.
func (q *Queue) IsQueueShown() bool {
	page, _ := q.pages.GetFrontPage()

	return q.IsOpen() && page == "queue"
}

// Add adds an entry to the player queue.
func (q *Queue) Add(video inv.VideoData, audio bool, uri ...[2]string) {
	count := q.Count()
	_, _, w, _ := q.GetRect()

	media := "Audio"
	if !audio {
		media = "Video"
	}

	length := "Live"
	if !video.LiveNow {
		length = utils.FormatDuration(video.LengthSeconds)
	}

	uris := [2]string{}
	if uri != nil {
		uris = uri[0]
	}

	timestamp := video.Timestamp
	video.MediaType = media

	q.SetData(count, QueueData{
		Columns: [QueueColumnSize]*tview.TableCell{
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(1).
				SetSelectable(false).
				SetReference(AttachableReference(video)),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemeVideo,
				tview.Escape(video.Title),
			).
				SetExpansion(1).
				SetMaxWidth(w / 7).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(1).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemeAuthor,
				tview.Escape(video.Author),
			).
				SetExpansion(1).
				SetMaxWidth(w / 7).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(1).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemeMediaType,
				media,
			).
				SetMaxWidth(5).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(1).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemeTotalDuration,
				length,
			).
				SetMaxWidth(10).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(15).
				SetSelectable(true),
			theme.NewTableCell(
				theme.ThemeContextQueue,
				theme.ThemePopupBackground,
				" ",
			).
				SetMaxWidth(1).
				SetSelectable(true),
		},
		Audio:     audio,
		Reference: video,
		URI:       uris,
		Timestamp: timestamp,
	})

	if count == 0 {
		q.SwitchToPosition(count)

		app.ConditionalDraw(func() bool {
			q.SelectCurrentRow()

			return q.IsOpen()
		})
	}
}

// AddRecommendations adds the video-based recommendations to the queue modal.
func (q *Queue) AddRecommendations(video inv.VideoData) {
	q.recommends.Clear()
	_, _, w, _ := q.GetRect()

	for i, v := range video.RecommendedVideos {
		length := "Live"
		if !v.LiveNow {
			length = utils.FormatDuration(v.LengthSeconds)
		}

		q.recommends.SetCell(i, 0, theme.NewTableCell(
			theme.ThemeContextQueue,
			theme.ThemeVideo,
			tview.Escape(v.Title),
		).
			SetExpansion(1).
			SetMaxWidth(w/7).
			SetSelectable(true).
			SetReference(AttachableReference(v)),
		)
		q.recommends.SetCell(i, 1, theme.NewTableCell(
			theme.ThemeContextQueue,
			theme.ThemePopupBackground,
			" ",
		).
			SetMaxWidth(1).
			SetSelectable(true),
		)
		q.recommends.SetCell(i, 2, theme.NewTableCell(
			theme.ThemeContextQueue,
			theme.ThemeAuthor,
			tview.Escape(v.Author),
		).
			SetExpansion(1).
			SetMaxWidth(w/7).
			SetSelectable(true).
			SetAlign(tview.AlignRight),
		)
		q.recommends.SetCell(i, 3, theme.NewTableCell(
			theme.ThemeContextQueue,
			theme.ThemePopupBackground,
			" ",
		).
			SetMaxWidth(1).
			SetSelectable(true),
		)
		q.recommends.SetCell(i, 4, theme.NewTableCell(
			theme.ThemeContextQueue,
			theme.ThemeTotalDuration,
			length,
		).
			SetMaxWidth(10).
			SetSelectable(true),
		)
	}

	q.recommends.Select(0, 0)
	q.recommends.ScrollToBeginning()
}

// AutoPlay automatically selects what to play after
// the current entry has finished playing.
func (q *Queue) AutoPlay(force bool) {
	switch q.GetRepeatMode() {
	case mp.RepeatModeFile:
		return

	case mp.RepeatModePlaylist:
		if !q.shuffle.Load() && q.Position() == q.Count()-1 {
			q.SwitchToPosition(0, struct{}{})
			q.MarkPlayingEntry(EntryPlaying)
			return
		}
	}

	q.Next()
}

// Play plays the entry at the current queue position.
func (q *Queue) Play(norender ...struct{}) {
	go func() {
		if q.playcancel != nil {
			q.playcancel()
		}
		if q.playctx == nil || q.playctx.Err() == context.Canceled {
			q.playctx, q.playcancel = context.WithCancel(context.Background())
		}

		data, ok := q.GetCurrent()
		if !ok {
			app.ShowError(fmt.Errorf("Player: Cannot get media data for %s", data.Reference.Title))
			return
		}

		mp.Player().Stop()

		q.MarkPlayingEntry(EntryFetching)
		q.audio.Store(data.Audio)
		q.title.Store(tview.Escape(data.Reference.Title))

		sendPlayerEvents()
		Show()

		video, uri, err := inv.RenewVideoURI(q.playctx, data.URI, data.Reference, data.Audio)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				q.MarkPlayingEntry(EntryStopped)
				app.ShowError(fmt.Errorf("Player: Cannot get media URI for %s", data.Reference.Title))
			}

			return
		}

		q.SetReference(q.Position(), video, struct{}{})
		q.AddRecommendations(video)
		q.MarkPlayingEntry(EntryLoading)

		if err := mp.Player().LoadFile(
			data.Reference.Title, data.Reference.LengthSeconds,
			data.Audio,
			uri,
		); err != nil {
			app.ShowError(err)
			return
		}

		mp.Player().Play()

		if norender == nil {
			renderInfo(data.Reference, struct{}{})
		}
	}()
}

// Delete removes a entry from the specified position within the queue.
func (q *Queue) Delete(position int) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	q.store.Remove(position)
}

// Move moves the position of the selected queue entry.
func (q *Queue) Move(before, after int) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	length := q.store.Len()
	if (after < 0 || before < 0) ||
		(after >= length || before >= length) {
		return
	}

	if q.Position() == before {
		q.SetPosition(after)
	}

	b := q.store.At(before)

	q.store.Remove(before)
	q.store.Insert(after, b)
}

// Count returns the number of items in the queue.
func (q *Queue) Count() int {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	return q.store.Len()
}

// Position returns the current position within the queue.
func (q *Queue) Position() int {
	return int(q.position.Load())
}

// SetPosition sets the current position within the queue.
func (q *Queue) SetPosition(position int) {
	q.position.Store(int32(position))
}

// SwitchToPosition switches to the specified position within the queue.
func (q *Queue) SwitchToPosition(position int, autoplay ...struct{}) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	data, ok := q.GetEntryPointer(position)
	if !ok {
		return
	}

	skipPlaying := autoplay != nil && position == 0 && data == q.current
	if q.current != nil {
		q.current.Playing = false
	}

	data.Playing = true
	data.HasPlayed = true

	q.current = data

	if skipPlaying {
		return
	}

	q.SetPosition(position)
	q.Play()
}

// SelectRecentEntry selects the recent-most entry in the queue.
func (q *Queue) SelectRecentEntry() {
	q.SwitchToPosition(q.Count() - 1)
}

// Previous selects the previous entry from the current position in the queue.
func (q *Queue) Previous(force ...struct{}) {
	length := q.Count()
	if length == 0 {
		return
	}

	position := q.Position()
	if q.Shuffle(position, length, force...) || position-1 < 0 {
		return
	}

	q.SwitchToPosition(position - 1)
}

// Next selects the next entry from the current position in the queue.
func (q *Queue) Next(force ...struct{}) {
	length := q.Count()
	if length == 0 {
		return
	}

	position := q.Position()
	if q.Shuffle(position, length, force...) || position+1 >= length {
		return
	}

	q.SwitchToPosition(position + 1)
}

// Shuffle chooses and plays a random entry.
func (q *Queue) Shuffle(position, count int, force ...struct{}) bool {
	if !q.shuffle.Load() {
		return false
	}

	skipped := 0
	pos := -1

	q.storeMutex.Lock()
	for skipped < count {
		for {
			pos = rand.Intn(count)
			if pos != position {
				break
			}
		}

		data, ok := q.GetEntryPointer(pos)
		if !ok {
			continue
		}
		if !data.HasPlayed {
			break
		}

		skipped++
	}
	q.storeMutex.Unlock()

	if skipped >= count {
		q.storeMutex.Lock()
		q.store.Index(func(data *QueueData) bool {
			data.HasPlayed = false

			return false
		})
		q.storeMutex.Unlock()

		if mode := q.GetRepeatMode(); mode == mp.RepeatModePlaylist || force != nil {
			q.Shuffle(position, count)
		}
	} else {
		q.SwitchToPosition(pos)
	}

	return true
}

// Get returns the entry data at the specified position from the queue.
func (q *Queue) Get(position int) (QueueData, bool) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	data, ok := q.GetEntryPointer(position)
	if !ok {
		return QueueData{}, false
	}

	return *data, true
}

// GetEntryPointer returns a pointer to the entry data at the specified position from the queue.
func (q *Queue) GetEntryPointer(position int) (*QueueData, bool) {
	length := q.store.Len()
	if position < 0 || position >= length {
		return nil, false
	}

	return q.store.At(position), true
}

// GetPlayingIndex returns the index of the currently playing entry.
func (q *Queue) GetPlayingIndex() int {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	return q.store.Index(func(d *QueueData) bool {
		return d.Playing
	})
}

// GetCurrent returns the entry data at the current position from the queue.
func (q *Queue) GetCurrent() (QueueData, bool) {
	return q.Get(q.Position())
}

// GetTitle returns the title for the currently playing entry.
func (q *Queue) GetTitle() string {
	var title string

	if t, ok := q.title.Load().(string); ok {
		title = t
	}

	return title
}

// GetMediaType returns the media type for the currently playing entry.
func (q *Queue) GetMediaType() string {
	audio := q.audio.Load()
	if audio {
		return "Audio"
	}

	return "Video"
}

// GetRepeatMode returns the current repeat mode.
func (q *Queue) GetRepeatMode() mp.RepeatMode {
	return mp.RepeatMode(int(q.repeat.Load()))
}

// GetShuffleMode returns the current shuffle mode.
func (q *Queue) GetShuffleMode() bool {
	return q.shuffle.Load()
}

// GetCell returns a TableCell from the queue entry data at the specified row and column.
func (q *Queue) GetCell(row, column int) *tview.TableCell {
	data, ok := q.Get(row)
	if !ok {
		return nil
	}

	return data.Columns[column]
}

// GetRowCount returns the number of rows in the table.
func (q *Queue) GetRowCount() int {
	return q.Count()
}

// GetColumnCount returns the number of columns in the table.
func (q *Queue) GetColumnCount() int {
	return QueueColumnSize - 1
}

// SelectCurrentRow selects the specified row within the table.
func (q *Queue) SelectCurrentRow(row ...int) {
	var pos int

	if row != nil {
		pos = row[0]
	} else {
		pos, _ = q.table.GetSelection()
	}

	q.table.Select(pos, 0)
}

// GetRect returns the dimensions of the table.
func (q *Queue) GetRect() (int, int, int, int) {
	var x, y, w, h int

	app.UI.RLock()
	x, y, w, h = q.table.GetRect()
	app.UI.RUnlock()

	return x, y, w, h
}

// MarkPlayingEntry marks the current queue entry as 'playing/loading'.
func (q *Queue) MarkPlayingEntry(status QueueEntryStatus) {
	pos := q.GetPlayingIndex()
	if pos < 0 {
		return
	}

	cell := q.GetCell(pos, QueuePlayingMarker)
	if cell == nil {
		return
	}

	tagMap := map[QueueEntryStatus]theme.ThemeItem{
		EntryFetching: theme.ThemeTagFetching,
		EntryLoading:  theme.ThemeTagLoading,
		EntryPlaying:  theme.ThemeTagPlaying,
		EntryStopped:  theme.ThemeTagStopped,
	}

	app.ConditionalDraw(func() bool {
		if q.marker != nil {
			q.marker.SetText("")
		}

		builder := theme.NewTextBuilder(theme.ThemeContextQueue)
		builder.Format(tagMap[status], "tag", " %s ", status)

		q.marker = cell
		q.marker.SetText(builder.Get())

		return q.IsOpen()
	})
}

// MarkEntryMediaType marks the selected queue entry as 'Audio/Video'.
func (q *Queue) MarkEntryMediaType(key keybinding.Key) {
	var media string

	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	switch key {
	case keybinding.KeyPlayerQueueAudio:
		media = "Audio"

	case keybinding.KeyPlayerQueueVideo:
		media = "Video"

	default:
		return
	}

	audio := media == "Audio"
	pos, _ := q.table.GetSelection()

	data, ok := q.GetEntryPointer(pos)
	if !ok || data.Audio == audio {
		return
	}

	position := mp.Player().Position()

	data.Audio = audio
	data.Timestamp = &position
	data.Columns[QueueMediaMarker].SetText(
		theme.SetTextStyle(
			"mediatype", media, theme.ThemeContextQueue, theme.ThemeMediaType),
	)

	if pos == q.Position() {
		q.Play(struct{}{})
	}
}

// SetData sets/adds entry data in the queue.
func (q *Queue) SetData(row int, data QueueData) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	length := q.store.Len()
	if length == 0 || row >= length {
		q.store.PushBack(&data)
		return
	}

	q.store.Set(row, &data)
}

// SetReference sets the reference for the data at the specified row in the queue.
func (q *Queue) SetReference(row int, video inv.VideoData, checkID ...struct{}) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	data, ok := q.GetEntryPointer(row)
	if !ok || checkID != nil && data.Reference.VideoID != video.VideoID {
		return
	}

	data.Reference = video
	if len(data.Columns) > 0 && data.Columns[0] != nil {
		data.Columns[0].SetReference(AttachableReference(video))
	}
}

// SetState sets the player states (repeat/shuffle).
func (q *Queue) SetState(state string) {
	if state == "shuffle" {
		q.shuffle.Store(true)
		return
	}

	if strings.Contains(state, "loop") {
		repeatMode := statesMap[state]
		q.repeat.Store(int32(repeatMode))
		mp.Player().SetLoopMode(mp.RepeatMode(repeatMode))
	}
}

// SetTimestamp seeks to the available timestamp during playback.
func (q *Queue) SetTimestamp(position int) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	data, ok := q.GetEntryPointer(position)
	if !ok || data.Timestamp == nil {
		return
	}

	timestamp := *data.Timestamp
	if timestamp < 0 {
		return
	}

	mp.Player().SetPosition(timestamp)
	data.Timestamp = nil
}

// Clear clears the queue.
func (q *Queue) Clear() {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	q.store.Clear()
	q.SetPosition(0)
}

// ToggleRepeatMode toggles the repeat mode.
func (q *Queue) ToggleRepeatMode() {
	repeatMode := mp.RepeatMode(int(q.repeat.Load()))

	switch repeatMode {
	case mp.RepeatModeOff:
		repeatMode = mp.RepeatModeFile

	case mp.RepeatModeFile:
		repeatMode = mp.RepeatModePlaylist

	case mp.RepeatModePlaylist:
		repeatMode = mp.RepeatModeOff
	}

	q.repeat.Store(int32(repeatMode))
	mp.Player().SetLoopMode(repeatMode)
}

// ToggleShuffle toggles the shuffle mode.
func (q *Queue) ToggleShuffle() {
	shuffle := q.shuffle.Load()
	q.shuffle.Store(!shuffle)
}

// Context returns/cancels the queue's context.
func (q *Queue) Context(cancel bool) context.Context {
	if cancel && q.ctx != nil {
		q.cancel()
	}

	if q.ctx == nil || q.ctx.Err() == context.Canceled {
		q.ctx, q.cancel = context.WithCancel(context.Background())
	}

	return q.ctx
}

// LoadPlaylist loads the provided playlist into MPV.
// If replace is true, the provided playlist will replace the current playing queue.
// renewLiveURL is a function to check and renew expired liev URLs in the playlist.
//
//gocyclo:ignore
func (q *Queue) LoadPlaylist(ctx context.Context, plpath string, replace bool) error {
	var filesAdded int

	if replace {
		q.Clear()
	}

	pl, err := os.Open(plpath)
	if err != nil {
		return fmt.Errorf("MPV: Unable to open %s", plpath)
	}
	defer pl.Close()

	playlist, err := m3u8.ReadFile(plpath)
	if err != nil {
		return err
	}

	uriMap := make(map[string]struct{}, len(playlist.Items))

ReadPlaylist:
	for _, item := range playlist.Items {
		var mediaURI string
		var audio bool

		video := inv.VideoData{
			Title: "No title",
		}

		switch v := item.(type) {
		case *m3u8.SessionDataItem:
			if v.URI == nil || v.Value == nil {
				continue
			}
			if v.DataID == "" || !strings.HasPrefix(v.DataID, inv.PlaylistEntryPrefix) {
				continue
			}

			uri, err := utils.IsValidURL(*v.URI)
			if err != nil {
				return err
			}

			mediaURI = uri.String()
			if _, ok := uriMap[mediaURI]; ok {
				continue
			}

			uriMap[mediaURI] = struct{}{}

			vmap := make(map[string]string)
			if !utils.DecodeSessionData(*v.Value, func(prop, value string) {
				vmap[prop] = value
			}) {
				continue
			}
			for _, prop := range []string{
				"id",
				"authorId",
				"mediatype",
			} {
				if vmap[prop] == "" {
					continue ReadPlaylist
				}
			}

			audio = vmap["mediatype"] == "Audio"
			title, _ := url.QueryUnescape(vmap["title"])
			author, _ := url.QueryUnescape(vmap["author"])

			length := vmap["length"]

			video.Title = title
			video.Author = author
			video.AuthorID = vmap["authorId"]
			video.VideoID = vmap["id"]
			video.MediaType = vmap["mediatype"]
			video.LiveNow = length == "Live"
			video.LengthSeconds = utils.ConvertDurationToSeconds(vmap["length"])

		case *m3u8.SegmentItem:
			var live bool

			mediaURI = v.Segment
			if strings.HasPrefix(mediaURI, "#") {
				continue
			}
			if _, ok := uriMap[mediaURI]; ok {
				continue
			}

			uri, err := utils.IsValidURL(mediaURI)
			if err != nil {
				return err
			}

			audio = true
			uriMap[mediaURI] = struct{}{}

			data := uri.Query()
			if data.Get("id") == "" {
				id, _ := inv.CheckLiveURL(mediaURI, audio)
				if id == "" {
					continue
				}

				data.Set("id", id)
				live = true
			}

			if v.Comment != nil {
				data.Set("title", *v.Comment)
			}

			for _, d := range []string{"title", "author"} {
				if data.Get(d) == "" {
					data.Set(d, "-")
				}
			}

			video.VideoID = data.Get("id")
			video.Title = data.Get("title")
			video.Author = data.Get("author")
			video.LiveNow = live
			video.MediaType = "Audio"
			video.LengthSeconds = int64(v.Duration)

		default:
			continue
		}
		if video.LiveNow {
			video.HlsURL = mediaURI
		}

		q.Add(video, audio, [2]string{mediaURI})

		filesAdded++
	}

	return nil
}

// Keybindings define the keybindings for the queue.
func (q *Queue) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	operation := keybinding.KeyOperation(event, keybinding.KeyContextQueue, keybinding.KeyContextComments)

	for _, op := range []keybinding.Key{
		keybinding.KeyClose,
		keybinding.KeyQueueSave,
	} {
		if operation == op {
			q.Hide()
			break
		}
	}

	switch operation {
	case keybinding.KeyQueuePlayMove:
		q.play()

	case keybinding.KeyQueueSave:
		app.UI.FileBrowser.Show("Save as:", q.saveAs)

	case keybinding.KeyQueueAppend:
		app.UI.FileBrowser.Show("Append from:", q.appendFrom)

	case keybinding.KeyQueueDelete:
		q.remove()

	case keybinding.KeyQueueMove:
		q.move()

	case keybinding.KeyPlayerQueueAudio, keybinding.KeyPlayerQueueVideo:
		q.MarkEntryMediaType(operation)

	case keybinding.KeyPlayerStop, keybinding.KeyClose:
		q.Hide()

	case keybinding.KeyComments:
		view.Comments.Show()

	case keybinding.KeySwitch:
		q.tabsHandler()
	}

	for _, o := range []keybinding.Key{
		keybinding.KeyQueueMove,
		keybinding.KeyQueueDelete,
	} {
		if operation == o {
			app.ResizeModal()
			break
		}
	}

	return event
}

// ThemeProperty returns the queue's theme property.
func (q *Queue) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextQueue,
		Item:    theme.ThemePopupBackground,
	}
}

// play handles the 'Enter' key event within the queue.
// If the move mode is enabled, the currently moving item
// is set to the position where the selector rests.
// Otherwise, it plays the currently selected queue item.
func (q *Queue) play() {
	row, _ := q.table.GetSelection()

	if q.moveMode {
		selected := row
		q.Move(q.prevrow, row)

		q.moveMode = false
		q.table.Select(selected, 0)

		return
	}

	q.SwitchToPosition(row)
}

// remove handles the 'd' key within the queue.
// It deletes the currently selected queue item.
func (q *Queue) remove() {
	rows := q.table.GetRowCount() - 1
	row, _ := q.table.GetSelection()

	q.Delete(row)

	switch {
	case rows <= 0:
		player.setting.Store(false)

		q.Clear()
		q.Hide()
		go Hide()

		return

	case row >= rows:
		row = rows - 1
	}

	q.SelectCurrentRow(row)

	pos := q.GetPlayingIndex()
	if pos < 0 {
		if row > rows {
			return
		}

		pos = row
		q.SwitchToPosition(row)
	}

	q.SetPosition(pos)
	if pos == row {
		sendPlayerEvents()
	}
}

// move handles the 'M' key within the queue.
// It enables the move mode, and starts moving the selected entry.
func (q *Queue) move() {
	q.prevrow, _ = q.table.GetSelection()
	q.moveMode = true
	q.table.Select(q.prevrow, 0)
}

// selectorHandler checks whether the move mode is enabled or not,
// and displays the appropriate selector indicator within the queue.
func (q *Queue) selectorHandler(row, col int) {
	selector := ">"
	rows := q.table.GetRowCount()
	item := theme.ThemeNormalModeSelector

	if q.moveMode {
		selector = "M"
		item = theme.ThemeMoveModeSelector
	}

	for i := 0; i < rows; i++ {
		cell := q.table.GetCell(i, 0)
		if cell == nil {
			cell = tview.NewTableCell(" ")
			q.table.SetCell(i, 0, cell)
		}

		if i == row {
			cell.SetText(
				theme.SetTextStyle(
					"selector",
					selector,
					theme.ThemeContextQueue,
					item,
				),
			)
			continue
		}

		cell.SetText(" ")
	}
}

// tabsHandler handles tab switching within the queue.
func (q *Queue) tabsHandler() {
	tab := app.Tab{
		Title: "Queue",
		Info: []app.TabInfo{
			{ID: "queue", Title: "Queue"},
			{ID: "recommends", Title: "Recommendations"},
		},

		Context: theme.ThemeContextQueue,

		TabView: q.tabs,
	}

	if q.tabs.GetText(false) == "" {
		builder := theme.NewTextBuilder(theme.ThemeContextQueue)
		items := map[string]tview.Primitive{
			"queue":      q.table,
			"recommends": q.recommends,
		}

		for i, info := range tab.Info {
			q.pages.AddPage(info.ID, items[info.ID], true, info.ID == "queue")

			builder.Format(theme.ThemeTabs, info.ID, " %s ", info.Title)
			if i < len(tab.Info) {
				builder.AppendText(" ")
			}
		}

		q.tabs.SetText(builder.Get())
		q.pages.SwitchToPage("recommends")
	}

	page, _ := q.pages.GetFrontPage()
	tab.Selected = page

	id := app.SwitchTab(false, tab)
	q.pages.SwitchToPage(id)
	for _, info := range tab.Info {
		if id != info.ID {
			continue
		}

		q.modal.Title.SetText(
			theme.SetTextStyle(
				"title", info.Title,
				theme.ThemeContextQueue, theme.ThemeTitle,
			),
		)

		break
	}
}

// saveAs saves the current queue into a playlist M3U8 file.
func (q *Queue) saveAs(file string) {
	if !q.lock.TryAcquire(1) {
		app.ShowInfo("Playlist save in progress", false)
	}
	defer q.lock.Release(1)

	var videos []inv.VideoData

	for i := 0; i < q.Count(); i++ {
		data, ok := q.Get(i)
		if !ok {
			continue
		}

		v := data.Reference
		if v.VideoID != "" {
			videos = append(videos, v)
		}
	}
	if len(videos) == 0 {
		return
	}

	app.UI.FileBrowser.SaveFile(file, func(flags int, appendToFile bool) (string, int, error) {
		return inv.GeneratePlaylist(file, videos, flags, appendToFile)
	})
}

// appendFrom appends the entries from the provided playlist file
// into the currently playing queue.
func (q *Queue) appendFrom(file string) {
	app.ShowInfo("Loading "+filepath.Base(file), true)

	err := q.LoadPlaylist(q.Context(false), file, false)
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

// sendStatus sends status events to the queue.
func (q *Queue) sendStatus() {
	select {
	case q.status <- struct{}{}:

	default:
	}
}

// AttachableReference returns an attachable reference to the video item.
func AttachableReference(v inv.VideoData) inv.SearchData {
	return inv.SearchData{
		Type:     "video",
		Title:    v.Title,
		VideoID:  v.VideoID,
		AuthorID: v.AuthorID,
		Author:   v.Author,
	}
}

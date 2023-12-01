package player

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Queue describes the media queue.
type Queue struct {
	init, moveMode bool
	prevrow        int
	text           string
	videos         map[string]*inv.VideoData

	status chan struct{}

	modal  *app.Modal
	table  *tview.Table
	marker *tview.TableCell

	lock *semaphore.Weighted

	ctx    context.Context
	cancel context.CancelFunc

	position, repeat atomic.Int32
	shuffle, audio   atomic.Bool
	title            atomic.Value

	store      *deque.Deque[QueueData]
	storeMutex sync.Mutex

	tview.TableContentReadOnly
}

// QueueData describes the queue entry data.
type QueueData struct {
	MediaURIs []string
	Audio     bool
	Columns   [QueueColumnSize]*tview.TableCell
	Reference inv.VideoData
}

const (
	QueueColumnSize = 10

	QueuePlayingMarker = QueueColumnSize - 2
	QueueMediaMarker   = QueueColumnSize - 5

	PlayerMarkerFormat = `[%s::b][%s[][-:-:-]`
	MediaMarkerFormat  = `[pink::b]%s[-:-:-]`
)

// Setup sets up the queue.
func (q *Queue) Setup() {
	if q.init {
		return
	}

	q.store = deque.New[QueueData](100)

	q.status = make(chan struct{}, 100)
	q.videos = make(map[string]*inv.VideoData)

	q.table = tview.NewTable()
	q.table.SetContent(q)
	q.table.SetSelectable(true, false)
	q.table.SetInputCapture(q.Keybindings)
	q.table.SetBackgroundColor(tcell.ColorDefault)
	q.table.SetSelectionChangedFunc(q.selectorHandler)
	q.table.SetFocusFunc(func() {
		app.SetContextMenu(cmd.KeyContextQueue, q.table)
	})

	q.modal = app.NewModal("queue", "Queue", q.table, 40, 0)

	q.lock = semaphore.NewWeighted(1)

	q.init = true
}

// Show shows the player queue.
func (q *Queue) Show() {
	if q.Count() == 0 || !player.setting.Load() {
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

// Add adds an entry to the player queue.
func (q *Queue) Add(video inv.VideoData, uris []string, audio bool) {
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

	q.SetData(count-1, QueueData{
		Columns: [QueueColumnSize]*tview.TableCell{
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell("[blue::b]" + tview.Escape(video.Title)).
				SetExpansion(1).
				SetMaxWidth(w / 7).
				SetSelectable(true).
				SetSelectedStyle(app.UI.ColumnStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell("[purple::b]" + tview.Escape(video.Author)).
				SetExpansion(1).
				SetMaxWidth(w / 7).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell(fmt.Sprintf(MediaMarkerFormat, media)).
				SetMaxWidth(5).
				SetSelectable(true).
				SetSelectedStyle(app.UI.ColumnStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell("[pink::b]" + length).
				SetMaxWidth(10).
				SetSelectable(true).
				SetSelectedStyle(app.UI.ColumnStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(11).
				SetSelectable(false),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
		},
		MediaURIs: uris,
		Audio:     audio,
		Reference: video,
	})

	if count == 0 {
		q.Play()

		app.UI.QueueUpdateDraw(func() {
			q.SelectCurrentRow()
		})
	}
}

// AutoPlay automatically selects what to play after
// the current entry has finished playing.
func (q *Queue) AutoPlay(force bool) {
	count := q.Count()
	position := int(q.position.Load())

	if q.shuffle.Load() && count > 0 {
		q.SwitchToPosition(rand.Intn(count))
		return
	}

	if q.GetRepeatMode() == mp.RepeatModePlaylist || force {
		if position == count-1 && !force {
			q.SwitchToPosition(0)
		} else {
			q.Next()
		}
	}
}

// Play plays the entry at the current queue position.
func (q *Queue) Play() {
	go func() {
		data, ok := q.Get(q.Position())
		if !ok {
			return
		}

		if err := mp.Player().LoadFile(
			data.Reference.Title, data.Reference.LengthSeconds,
			data.Audio,
			data.MediaURIs...,
		); err != nil {
			app.ShowError(err)
		}

		mp.Player().Play()

		q.title.Store(data.Reference.Title)
		q.audio.Store(data.Audio)

		app.UI.QueueUpdateDraw(func() {
			renderInfo(data.Reference, struct{}{})
		})
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

func (q *Queue) SetPosition(position int) {
	q.position.Store(int32(position))
}

// SwitchToPosition switches to the specified position within the queue.
func (q *Queue) SwitchToPosition(position int) {
	length := q.Count()
	if position < 0 || position >= length {
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
func (q *Queue) Previous() {
	length := q.Count()
	if length == 0 {
		return
	}

	position := q.Position()
	if position-1 < 0 {
		return
	}

	q.SetPosition(position - 1)
	q.Play()
}

// Next selects the next entry from the current position in the queue.
func (q *Queue) Next() {
	length := q.Count()
	if length == 0 {
		return
	}

	position := q.Position()
	if position+1 >= length {
		return
	}

	q.SetPosition(position + 1)
	q.Play()
}

// Get returns the entry data at the specified position from the queue.
func (q *Queue) Get(position int, nolock ...struct{}) (QueueData, bool) {
	if nolock == nil {
		q.storeMutex.Lock()
		defer q.storeMutex.Unlock()
	}

	length := q.store.Len()
	if position < 0 || position >= length {
		return QueueData{}, false
	}

	return q.store.At(position), true
}

// GetCurrent returns the entry data at the current position from the queue.
func (q *Queue) GetCurrent() (QueueData, bool) {
	return q.Get(q.Position())
}

// GetTitle returns the title for the currently playing entry.
func (q *Queue) GetTitle(set ...string) string {
	if set != nil {
		q.title.Store(set[0])
	}

	return q.title.Load().(string)
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

	app.UI.QueueUpdate(func() {
		x, y, w, h = q.table.GetRect()
	})

	return x, y, w, h
}

// MarkPlayingEntry marks the current queue entry as 'playing/loading'.
func (q *Queue) MarkPlayingEntry(playing bool) {
	app.UI.QueueUpdateDraw(func() {
		if q.marker != nil && q.text != "" {
			q.marker.SetText(q.text)
		}

		q.marker = q.GetCell(q.Position(), QueuePlayingMarker)
		if q.marker == nil {
			return
		}

		color := "white"
		marker := "PLAYING"
		if !playing {
			color = "yellow"
			marker = "LOADING"
		}

		q.text = q.marker.Text
		q.marker.SetText(q.text + fmt.Sprintf(PlayerMarkerFormat, color, marker))
	})
}

// MarkEntryMediaType marks the selected queue entry as 'Audio/Video'.
func (q *Queue) MarkEntryMediaType(key cmd.Key) {
	var media string

	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	switch key {
	case cmd.KeyPlayerQueueAudio:
		media = "Audio"

	case cmd.KeyPlayerQueueVideo:
		media = "Video"

	default:
		return
	}

	audio := media == "Audio"
	pos, _ := q.table.GetSelection()

	data, ok := q.Get(pos, struct{}{})
	if !ok || data.Audio == audio {
		return
	}

	data.Audio = audio
	data.Columns[QueueMediaMarker].SetText(
		fmt.Sprintf(MediaMarkerFormat, media),
	)

	q.SetData(pos, data, struct{}{})
	if pos == q.Position() {
		q.Play()
	}
}

// SetData sets/adds entry data in the queue.
func (q *Queue) SetData(row int, data QueueData, nolock ...struct{}) {
	if nolock == nil {
		q.storeMutex.Lock()
		defer q.storeMutex.Unlock()
	}

	length := q.store.Len()
	if length == 0 || row >= length-1 {
		q.store.PushBack(data)
		return
	}

	q.store.Set(row, data)
}

// SetReference sets the reference for the data at the specified row in the queue.
func (q *Queue) SetReference(row int, video inv.VideoData, checkID ...struct{}) {
	q.storeMutex.Lock()
	defer q.storeMutex.Unlock()

	data, ok := q.Get(row, struct{}{})
	if !ok || checkID != nil && data.Reference.VideoID != video.VideoID {
		return
	}

	data.Reference = video
	q.SetData(row, data, struct{}{})
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

	// We implement a simple playlist parser instead of relying on
	// the m3u8 package here, since that package deals with mainly
	// HLS playlists, and it seems to panic when certain EXTINF fields
	// are blank. With this method, we can parse the URLs from the playlist
	// directly, and pass the relevant options to mpv as well.
	reader := bufio.NewReader(pl)

Reader:
	for {
		select {
		case <-ctx.Done():
			break Reader

		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if err := q.playlistAddEntry(line); err != nil {
			return err
		}

		filesAdded++
	}
	if filesAdded == 0 {
		return fmt.Errorf("MPV: No files were added")
	}

	return nil
}

// Keybindings define the keybindings for the queue.
func (q *Queue) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	operation := cmd.KeyOperation(event, cmd.KeyContextQueue)

	for _, op := range []cmd.Key{
		cmd.KeyClose,
		cmd.KeyQueueSave,
		cmd.KeyQueueAppend,
	} {
		if operation == op {
			q.Hide()
			break
		}
	}

	switch operation {
	case cmd.KeyQueuePlayMove:
		q.play()

	case cmd.KeyQueueSave:
		app.UI.FileBrowser.Show("Save as:", q.saveAs)

	case cmd.KeyQueueAppend:
		app.UI.FileBrowser.Show("Append from:", q.appendFrom)

	case cmd.KeyQueueDelete:
		q.remove()

	case cmd.KeyQueueMove:
		q.move()

	case cmd.KeyQueueCancel:
		q.Context(true)

	case cmd.KeyPlayerQueueAudio, cmd.KeyPlayerQueueVideo:
		q.MarkEntryMediaType(operation)

	case cmd.KeyPlayerStop, cmd.KeyClose:
		q.Hide()
	}

	for _, o := range []cmd.Key{
		cmd.KeyQueueMove,
		cmd.KeyQueueDelete,
	} {
		if operation == o {
			app.ResizeModal()
			break
		}
	}

	return event
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

	sendPlayerEvents()
}

// remove handles the 'd' key within the queue.
// It deletes the currently selected queue item.
func (q *Queue) remove() {
	rows := q.table.GetRowCount()
	row, _ := q.table.GetSelection()

	q.Delete(row)
	rows--

	switch {
	case rows == 0:
		player.setting.Store(false)

		q.Hide()
		go Hide()

		return

	case row >= rows:
		row = rows - 1
	}

	q.SelectCurrentRow(row)
	q.SwitchToPosition(row)

	pos := q.Position()
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

	if q.moveMode {
		selector = "M"
	}

	for i := 0; i < rows; i++ {
		cell := q.table.GetCell(i, 0)
		if cell == nil {
			cell = tview.NewTableCell(" ")
			q.table.SetCell(i, 0, cell)
		}

		if i == row {
			cell.SetText(selector)
			continue
		}

		cell.SetText(" ")
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

	app.UI.FileBrowser.SaveFile(file, func(appendToFile bool) (string, error) {
		return inv.GeneratePlaylist(file, videos, appendToFile)
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

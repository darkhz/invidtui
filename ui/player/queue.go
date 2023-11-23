package player

import (
	"fmt"
	"path/filepath"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/resolver"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Queue describes the layout of the player queue.
type Queue struct {
	init, moveMode bool
	prevrow        int
	data           []map[string]interface{}
	videos         map[string]*inv.VideoData

	status chan struct{}

	modal *app.Modal
	table *tview.Table

	lock *semaphore.Weighted
}

// QueueData stores data for the player queue.
type QueueData struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Playing  bool   `json:"current"`

	inv.SearchData
}

// setup sets up the player queue.
func (q *Queue) setup() {
	if q.init {
		return
	}

	q.status = make(chan struct{}, 100)
	q.videos = make(map[string]*inv.VideoData)

	q.table = tview.NewTable()
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

// Start starts the player queue.
func (q *Queue) Start() {
	q.setup()

	for {
		select {
		case data := <-mp.Events.DataEvent:
			app.UI.QueueUpdateDraw(func() {
				q.render(data)
			})

		case <-q.status:
			app.UI.QueueUpdateDraw(func() {
				q.render(q.data)
			})
		}
	}
}

// Show shows the player queue.
func (q *Queue) Show() {
	if len(q.data) == 0 {
		return
	}

	q.modal.Show(true)
	q.sendStatus()
}

// Hide hides the player queue.
func (q *Queue) Hide() {
	q.modal.Exit(false)
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
		if row > q.prevrow {
			row++
		}

		mp.Player().QueueMove(row, q.prevrow)

		q.moveMode = false
		q.table.Select(selected, 0)

		return
	}

	mp.Player().QueueSwitchToTrack(row)
	mp.Player().Play()

	sendPlayerEvents()
}

// remove handles the 'd' key within the queue.
// It deletes the currently selected queue item.
func (q *Queue) remove() {
	rows := q.table.GetRowCount()
	row, _ := q.table.GetSelection()

	switch {
	case row >= rows-1:
		if mp.Player().LoopMode() != "R-P" {
			mp.Player().Prev()
		}
		q.table.Select(row-1, 0)

	case row < rows && row >= 0:
		q.table.Select(row, 0)
	}

	q.removeVideo(row)

	mp.Player().QueueDelete(row)

	pos := mp.Player().QueuePosition()
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

// render renders the player queue.
func (q *Queue) render(data []map[string]interface{}) {
	if len(data) == 0 {
		q.table.Clear()
		q.data = nil
		q.removeVideo(-1, struct{}{})

		if q.table.HasFocus() {
			q.Hide()
		}

		return
	}

	length := len(q.data)
	_, _, w, _ := q.table.GetRect()
	pos, _ := q.table.GetSelection()

	q.table.SetSelectable(false, false)

	for i, pldata := range data {
		var marker string

		data, skip := q.getData(i, pldata, length)
		if skip {
			continue
		}

		if data.Playing {
			marker = " [white::b](playing)"
		}

		info := inv.SearchData{
			Title:   data.Title,
			Type:    "video",
			Author:  data.Author,
			VideoID: data.VideoID,
		}

		q.table.SetCell(i, 1, tview.NewTableCell("[blue::b]"+tview.Escape(data.Title)+marker).
			SetExpansion(1).
			SetMaxWidth(w/7).
			SetReference(info).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)

		q.table.SetCell(i, 2, tview.NewTableCell(" ").
			SetMaxWidth(1).
			SetSelectable(false),
		)

		q.table.SetCell(i, 3, tview.NewTableCell("[purple::b]"+tview.Escape(data.Author)).
			SetExpansion(1).
			SetMaxWidth(w/7).
			SetSelectable(true).
			SetAlign(tview.AlignRight).
			SetSelectedStyle(app.UI.ColumnStyle),
		)

		q.table.SetCell(i, 4, tview.NewTableCell(" ").
			SetMaxWidth(1).
			SetSelectable(false),
		)

		q.table.SetCell(i, 5, tview.NewTableCell("[pink::b]"+tview.Escape(data.Type)).
			SetMaxWidth(5).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)

		q.table.SetCell(i, 6, tview.NewTableCell(" ").
			SetMaxWidth(1).
			SetSelectable(false),
		)

		q.table.SetCell(i, 7, tview.NewTableCell("[pink::b]"+data.Duration).
			SetMaxWidth(10).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)
	}

	q.table.SetSelectable(true, false)
	q.table.Select(pos, 0)

	app.ResizeModal()

	q.data = data
}

// getData organises and returns the queue data from the provided playlist data map.
//
//gocyclo:ignore
func (q *Queue) getData(row int, pldata map[string]interface{}, length ...int) (QueueData, bool) {
	var data QueueData
	var filename string

	if len(pldata) == 0 {
		return QueueData{}, true
	}

	props := []string{"id", "filename", "current"}

	if length != nil && row < length[0] {
		var count int

		if _, ok := pldata["current"]; !ok {
			pldata["current"] = false
		}

		for _, prop := range props {
			pdata, recent := pldata[prop]
			qdata, existing := q.data[row][prop]
			if (recent && existing) && (pdata == qdata) {
				count++
			}
		}
		if count == len(props) {
			return (QueueData{}), true
		}
	}

	for _, prop := range props {
		value := pldata[prop]

		switch prop {
		case "id":
			if v, ok := value.(int); ok {
				data.ID = v
			}

		case "filename":
			if v, ok := value.(string); ok {
				filename = v
			}

		case "current":
			if v, ok := value.(bool); ok {
				data.Playing = v
			}
		}
	}

	urlData := utils.GetDataFromURL(filename)
	if urlData == nil {
		return (QueueData{}), true
	}

	for udata := range urlData {
		key, value := udata, urlData.Get(udata)
		if value != "" {
			continue
		}

		switch key {
		case "title":
			value = mp.Player().Title(row)

		default:
			if key != "id" {
				value = "-"
			}
		}

		urlData.Set(key, value)
	}

	data.VideoID = urlData.Get("id")
	data.Title = urlData.Get("title")
	data.Author = urlData.Get("author")
	data.Type = urlData.Get("mediatype")
	data.Duration = urlData.Get("length")
	data.LengthSeconds = utils.ConvertDurationToSeconds(data.Duration)

	return data, false
}

// saveAs saves the current queue into a playlist M3U8 file.
func (q *Queue) saveAs(file string) {
	if !q.lock.TryAcquire(1) {
		app.ShowInfo("Playlist save in progress", false)
	}
	defer q.lock.Release(1)

	var videos []inv.VideoData

	for _, v := range q.getQueueData() {
		videos = append(videos, inv.VideoData{
			VideoID:       v.VideoID,
			Title:         v.Title,
			LengthSeconds: v.LengthSeconds,
			Author:        v.Author,
		})
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

	app.UI.QueueUpdateDraw(func() {
		player.queue.Show()
	})

	err := mp.Player().LoadPlaylist(file, false, checkLiveURL)
	if err != nil {
		app.ShowError(err)
		return
	}

	app.ShowInfo("Loaded "+filepath.Base(file), false)

	app.UI.FileBrowser.Hide()
}

// getQueueData returns a list of queue items.
func (q *Queue) getQueueData() []QueueData {
	var data []QueueData

	playlistJSON := mp.Player().QueueData()
	if playlistJSON == "" {
		app.ShowError(fmt.Errorf("Queue: Could not fetch playlist"))
		return []QueueData{}
	}

	err := resolver.DecodeJSONBytes([]byte(playlistJSON), &data)
	if err != nil {
		app.ShowError(fmt.Errorf("Queue: Error while parsing playlist data"))
		return []QueueData{}
	}
	if len(data) == 0 {
		return []QueueData{}
	}

	for i := range data {
		data[i], _ = q.getData(i, map[string]interface{}{
			"id":       data[i].ID,
			"playing":  data[i].Playing,
			"filename": data[i].Filename,
		})
	}

	return data
}

// currentVideo sets or returns the video to/from the store
// according to the provided ID.
func (q *Queue) currentVideo(id string, set ...*inv.VideoData) *inv.VideoData {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if set != nil {
		q.videos[id] = set[0]
	}

	video, ok := q.videos[id]
	if !ok {
		return nil
	}

	return video
}

// removeVideo removes a video from the store.
func (q *Queue) removeVideo(pos int, reset ...struct{}) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if reset != nil && len(q.videos) > 0 {
		q.videos = make(map[string]*inv.VideoData)
		return
	}

	title := mp.Player().Title(pos)
	data := utils.GetDataFromURL(title)

	id := data.Get("id")
	if id == "" {
		return
	}

	delete(q.videos, id)
}

// sendStatus sends status events to the queue.
func (q *Queue) sendStatus() {
	select {
	case q.status <- struct{}{}:

	default:
	}
}

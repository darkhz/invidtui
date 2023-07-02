package player

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	mp "github.com/darkhz/invidtui/mediaplayer"
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
		app.SetContextMenu("Queue", q.table)
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
	operation := cmd.KeyOperation(event, "Queue")

	for _, op := range []string{
		"QueueExit",
		"QueueSave",
		"QueueAppend",
	} {
		if operation == op {
			q.Hide()
			break
		}
	}

	switch operation {
	case "QueuePlayMove":
		q.play()

	case "QueueSave":
		app.UI.FileBrowser.Show("Save as:", q.saveAs)

	case "QueueAppend":
		app.UI.FileBrowser.Show("Append from:", q.appendFrom)

	case "QueueDelete":
		q.remove()

	case "QueueMove":
		q.move()

	case "PlayerStop", "Exit":
		q.Hide()
	}

	for _, o := range []string{
		"QueueMove",
		"QueueDelete",
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
		if row > q.prevrow {
			mp.Player().QueueMove(q.prevrow, row+1)
		} else {
			mp.Player().QueueMove(q.prevrow, row)
		}

		q.moveMode = false
		q.table.Select(row, 0)

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
	q.data = data
	q.table.Clear()

	if len(data) == 0 {
		q.removeVideo(-1, struct{}{})
		if q.table.HasFocus() {
			q.Hide()
		}

		return
	}

	_, _, w, _ := q.table.GetRect()
	pos, _ := q.table.GetSelection()
	q.table.SetSelectable(false, false)

	for i, pldata := range data {
		var marker string

		data := q.getData(i, pldata)
		if data == (QueueData{}) {
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
			SetSelectable(false),
		)

		q.table.SetCell(i, 3, tview.NewTableCell("[purple::b]"+tview.Escape(data.Author)).
			SetMaxWidth(w/5).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)

		q.table.SetCell(i, 4, tview.NewTableCell(" ").
			SetSelectable(false),
		)

		q.table.SetCell(i, 5, tview.NewTableCell("[pink::b]"+tview.Escape(data.Type)).
			SetMaxWidth(w/5).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)

		q.table.SetCell(i, 6, tview.NewTableCell(" ").
			SetSelectable(false),
		)

		q.table.SetCell(i, 7, tview.NewTableCell("[pink::b]"+data.Duration).
			SetSelectable(true).
			SetSelectedStyle(app.UI.ColumnStyle),
		)
	}

	q.table.SetSelectable(true, false)
	q.table.Select(pos, 0)

	app.ResizeModal()
}

// getData organises and returns the queue data from the provided playlist data map.
func (q *Queue) getData(row int, pldata map[string]interface{}) QueueData {
	var id int
	var data QueueData
	var filename string
	var playing bool

	if i, ok := pldata["id"].(float64); ok {
		id = int(i)
	}
	if f, ok := pldata["filename"].(string); ok {
		filename = f
	}
	if p, ok := pldata["current"].(bool); ok {
		playing = p
	}

	urlData := utils.GetDataFromURL(filename)
	if urlData == nil {
		return (QueueData{})
	}

	for _, udata := range []string{
		"id",
		"title",
		"author",
		"length",
		"mediatype",
	} {
		if urlData.Get("id") == "" {
			continue
		}

		if udata == "title" && urlData.Get(udata) == "" {
			urlData.Set(udata, mp.Player().Title(row))
			continue
		}

		if urlData.Get(udata) == "" {
			urlData.Set(udata, "-")
		}
	}

	data.ID = id
	data.Playing = playing
	data.Filename = filename
	data.VideoID = urlData.Get("id")
	data.Title = urlData.Get("title")
	data.Author = urlData.Get("author")
	data.Type = urlData.Get("mediatype")
	data.Duration = urlData.Get("length")

	return data
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

// saveAs saves the current queue into a playlist M3U8 file.
func (q *Queue) saveAs(file string) {
	if !q.lock.TryAcquire(1) {
		app.ShowInfo("Playlist save in progress", false)
	}
	defer q.lock.Release(1)

	list := q.getQueueData()
	if len(list) == 0 {
		return
	}

	flags, appendToFile, confirm, exist := q.confirmOverwrite(file)
	if exist && !confirm {
		return
	}

	entries, err := q.generatePlaylist(file, list, appendToFile)
	if err != nil {
		app.ShowError(err)
		return
	}

	playlistFile, err := os.OpenFile(file, flags, 0664)
	if err != nil {
		app.ShowError(fmt.Errorf("Queue: Unable to open playlist"))
		return
	}

	_, err = playlistFile.WriteString(entries)
	if err != nil {
		app.ShowError(fmt.Errorf("Queue: Unable to save playlist"))
		return
	}

	message := " saved in "
	if appendToFile {
		message = " appended to "
	}

	app.ShowInfo("Playlist"+message+file, false)

	app.UI.FileBrowser.Hide()
}

// confirmOverwrite displays an overwrite confirmation message
// within the file browser. This is triggered if the selected file
// in the file browser already exists and has entries in it.
func (q *Queue) confirmOverwrite(file string) (int, bool, bool, bool) {
	var appendToFile bool

	flags := os.O_CREATE | os.O_WRONLY

	if _, err := os.Stat(file); err != nil {
		return flags, false, false, false
	}

	reply := app.UI.FileBrowser.Query("Overwrite playlist (y/n/a)?", q.validate, 1)
	switch reply {
	case "y":
		flags |= os.O_TRUNC

	case "a":
		flags |= os.O_APPEND
		appendToFile = true

	case "n":
		break

	default:
		reply = ""
	}

	return flags, appendToFile, reply != "", true
}

// validate validates the overwrite confirmation reply.
func (q *Queue) validate(text string, reply chan string) {
	for _, option := range []string{"y", "n", "a"} {
		if text == option {
			select {
			case reply <- text:

			default:
			}

			break
		}
	}
}

// getQueueData returns a list of queue items.
func (q *Queue) getQueueData() []QueueData {
	var data []QueueData

	playlistJSON := mp.Player().QueueData()
	if playlistJSON == "" {
		app.ShowError(fmt.Errorf("Queue: Could not fetch playlist"))
		return []QueueData{}
	}

	err := json.Unmarshal([]byte(playlistJSON), &data)
	if err != nil {
		app.ShowError(fmt.Errorf("Queue: Error while parsing playlist data"))
		return []QueueData{}
	}
	if len(data) == 0 {
		return []QueueData{}
	}

	for i := range data {
		data[i] = q.getData(i, map[string]interface{}{
			"id":       data[i].ID,
			"playing":  data[i].Playing,
			"filename": data[i].Filename,
		})
	}

	return data
}

// generatePlaylist generates a playlist file.
func (q *Queue) generatePlaylist(file string, list []QueueData, appendToFile bool) (string, error) {
	var skipped int
	var entries string
	var fileEntries map[string]struct{}

	if appendToFile {
		fileEntries = make(map[string]struct{})

		existingFile, err := os.Open(file)
		if err != nil {
			return "", fmt.Errorf("Queue: Unable to open playlist")
		}

		scanner := bufio.NewScanner(existingFile)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			fileEntries[line] = struct{}{}
		}
	}

	if !appendToFile {
		entries += "#EXTM3U\n\n"
		entries += "# Autogenerated by invidtui. DO NOT EDIT.\n\n"
	} else {
		entries += "\n"
	}

	for i, data := range list {
		if appendToFile && fileEntries != nil {
			if _, ok := fileEntries[data.Filename]; ok {
				skipped++
				continue
			}
		}

		entries += "#EXTINF:," + data.Title + "\n"
		entries += data.Filename + "\n"

		if i != len(list)-1 {
			entries += "\n"
		}
	}

	if skipped == len(list) {
		return "", fmt.Errorf("Queue: No new items in playlist to append")
	}

	return entries, nil
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

package player

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/tview"
	"github.com/gammazero/deque"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// Fetcher describes the media fetcher.
type Fetcher struct {
	modal  *app.Modal
	table  *tview.Table
	info   *tview.TextView
	marker *tview.TableCell

	items *deque.Deque[*FetcherData]
	mutex sync.Mutex

	lock *semaphore.Weighted

	tview.TableContentReadOnly
}

// FetcherData describes the media fetcher data.
type FetcherData struct {
	Columns [FetchColumnSize]*tview.TableCell
	Info    inv.SearchData
	Error   error
	Audio   bool

	ctx    context.Context
	cancel context.CancelFunc
}

// FetcherStatus describes the status of each media fetcher entry.
type FetcherStatus string

const (
	FetcherStatusAdding FetcherStatus = "Adding"
	FetcherStatusError  FetcherStatus = "Error"
)

const (
	FetchColumnSize = 7

	FetchStatusMarker = FetchColumnSize - 1
)

// Setup sets up the media fetcher.
func (f *Fetcher) Setup() {
	f.items = deque.New[*FetcherData](100)

	f.info = tview.NewTextView()
	f.info.SetDynamicColors(true)
	f.info.SetBackgroundColor(tcell.ColorDefault)

	f.table = tview.NewTable()
	f.table.SetContent(f)
	f.table.SetSelectorWrap(true)
	f.table.SetSelectable(true, false)
	f.table.SetInputCapture(f.Keybindings)
	f.table.SetBackgroundColor(tcell.ColorDefault)
	f.table.SetSelectionChangedFunc(f.selectorHandler)
	f.table.SetFocusFunc(func() {
		app.SetContextMenu(cmd.KeyContextFetcher, f.table)
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f.table, 0, 10, true).
		AddItem(app.HorizontalLine(), 1, 0, false).
		AddItem(f.info, 0, 1, false)

	f.modal = app.NewModal("fetcher", "Media Fetcher", flex, 100, 100)

	f.lock = semaphore.NewWeighted(10)
}

// Show shows the media fetcher.
func (f *Fetcher) Show() {
	if f.IsOpen() {
		return
	}
	if f.Count() == 0 {
		app.ShowInfo("Media Fetcher: No items are being added", false)
		return
	}

	f.modal.Show(false)
}

// Hide hides the media fetcher.
func (f *Fetcher) Hide() {
	f.modal.Exit(false)
}

// IsOpen returns whether the media fetcher is open.
func (f *Fetcher) IsOpen() bool {
	return f.modal != nil && f.modal.Open
}

// Fetch loads media and adds it to the media fetcher.
func (f *Fetcher) Fetch(info inv.SearchData, audio bool, newdata ...*FetcherData) (inv.SearchData, error) {
	data, ctx := f.Add(info, audio, newdata...)

	f.MarkStatus(data, FetcherStatusAdding, nil)
	defer f.UpdateTag(false)

	err := f.lock.Acquire(ctx, 1)
	if err != nil {
		return inv.SearchData{}, err
	}
	defer f.lock.Release(1)

	switch info.Type {
	case "playlist":
		var videos []inv.VideoData

		videos, err = inv.PlaylistVideos(ctx, info.PlaylistID, false, func(stats [3]int64) {
			f.MarkStatus(data, FetcherStatusAdding, nil, fmt.Sprintf("(%d of %d)", stats[1], stats[2]))
		})
		if err == nil {
			for _, v := range videos {
				player.queue.Add(v, audio)
			}
		}

	case "video":
		var video inv.VideoData

		video, err = inv.Video(info.VideoID, ctx)
		if err == nil {
			player.queue.Add(video, audio)
			info.Title = video.Title
		}
	}
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			f.MarkStatus(data, FetcherStatusError, err)
		}

		return inv.SearchData{}, err
	}

	f.Remove(data)

	return info, nil
}

// Add sets/adds entry data in the media fetcher.
func (f *Fetcher) Add(
	info inv.SearchData, audio bool,
	newdata ...*FetcherData,
) (*FetcherData, context.Context) {
	defer f.UpdateTag(false)

	f.mutex.Lock()
	defer f.mutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	media := "Audio"
	if !audio {
		media = "Video"
	}

	data := &FetcherData{
		Columns: [FetchColumnSize]*tview.TableCell{
			tview.NewTableCell("[blue::b]" + tview.Escape(info.Title)).
				SetExpansion(1).
				SetMaxWidth(15).
				SetSelectable(true).
				SetSelectedStyle(app.UI.SelectedStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell("[purple::b]" + tview.Escape(info.Author)).
				SetExpansion(1).
				SetMaxWidth(15).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell(fmt.Sprintf("[pink::b]%s (%s)", info.Type, media)).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			tview.NewTableCell(" ").
				SetMaxWidth(1).
				SetSelectable(false),
			tview.NewTableCell(string(FetcherStatusAdding)).
				SetSelectable(false),
		},
		Info:  info,
		Audio: audio,

		ctx:    ctx,
		cancel: cancel,
	}
	data.Columns[0].SetReference(data)

	if newdata != nil {
		pos := f.items.Index(func(d *FetcherData) bool {
			return d == newdata[0]
		})
		if pos >= 0 {
			newdata[0].cancel()
			f.items.Set(pos, data)

			return data, ctx
		}
	}

	f.items.PushFront(data)

	return data, ctx
}

// Remove removes entry data from the media fetcher.
func (f *Fetcher) Remove(data *FetcherData) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	pos := f.items.Index(func(d *FetcherData) bool {
		return d == data
	})
	if pos >= 0 {
		f.items.Remove(pos)
	}

	if f.items.Len() == 0 {
		f.Hide()
		f.UpdateTag(true)
	}
}

// MarkStatus marks the status of the media fetcher entry.
func (f *Fetcher) MarkStatus(data *FetcherData, status FetcherStatus, err error, text ...string) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	data.Error = err

	color := "yellow"
	if status == FetcherStatusError {
		color = "red"
	}

	cell := data.Columns[FetchStatusMarker]
	if cell != nil {
		extra := ""
		if text != nil {
			extra = text[0]
		}

		go app.UI.QueueUpdateDraw(func() {
			cell.SetText(fmt.Sprintf(`[%s::b][%s[][-:-:-] %s`, color, status, extra))

			pos, _ := f.table.GetSelection()
			f.table.Select(pos, 0)
		})
	}
}

// UpdateTag updates the status bar tag according to the media fetcher status.
func (f *Fetcher) UpdateTag(clear bool) {
	var tag, wrap string
	var info []string
	var queuedCount, errorCount int

	if clear {
		goto Tag
	}

	f.mutex.Lock()
	f.items.Index(func(d *FetcherData) bool {
		if d.Error != nil {
			errorCount++
			return false
		}

		queuedCount++

		return false
	})
	f.mutex.Unlock()

	if queuedCount > 0 {
		info = append(info, fmt.Sprintf("Queuing %d", queuedCount))
	}
	if errorCount > 0 {
		info = append(info, fmt.Sprintf("Errors %d", errorCount))
	}
	if info == nil {
		goto Tag
	}

	wrap = fmt.Sprintf(" (%s)", strings.Join(info, ", "))
	tag = fmt.Sprintf("[black:yellow:b]Media Fetcher%s[-:-:-]", wrap)

Tag:
	go app.UI.Status.Tag(tag)
}

// Count returns the number of items in the media fetcher.
func (f *Fetcher) Count() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	return f.items.Len()
}

// FetchAll fetches all the items in the media fetcher.
func (f *Fetcher) FetchAll() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.items.Index(func(d *FetcherData) bool {
		go f.Fetch(d.Info, d.Audio, d)

		return false
	})
}

// Cancel cancels fetching an item in the media fetcher.
func (f *Fetcher) Cancel(data *FetcherData) {
	f.mutex.Lock()
	data.cancel()
	f.mutex.Unlock()

	f.Remove(data)
}

// Cancel cancels fetching all the items in the media fetcher.
func (f *Fetcher) CancelAll(clear bool) {
	f.mutex.Lock()
	f.items.Index(func(d *FetcherData) bool {
		d.cancel()

		return false
	})
	f.mutex.Unlock()

	if clear {
		f.Clear()
		f.UpdateTag(clear)
	}
}

// Clear clears the media fetcher.
func (f *Fetcher) Clear() {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.items.Clear()
}

// ClearErrors clears all the errors in the media fetcher.
func (f *Fetcher) ClearErrors() {
	f.mutex.Lock()
	for {
		pos := f.items.Index(func(d *FetcherData) bool {
			return d.Error != nil
		})
		if pos < 0 {
			break
		}

		f.items.At(pos).cancel()
		f.items.Remove(pos)
	}
	f.mutex.Unlock()

	if f.Count() == 0 {
		f.Hide()
		f.UpdateTag(true)
	}
}

// Get returns the entry data at the specified position from the media fetcher.
func (f *Fetcher) Get(position int) (FetcherData, bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	length := f.items.Len()
	if position < 0 || position >= length {
		return FetcherData{}, false
	}

	return *f.items.At(position), true
}

// GetCell returns a TableCell from the media fetcher entry data at the specified row and column.
func (f *Fetcher) GetCell(row, column int) *tview.TableCell {
	data, ok := f.Get(row)
	if !ok {
		return nil
	}

	return data.Columns[column]
}

// GetRowCount returns the number of rows in the table.
func (f *Fetcher) GetRowCount() int {
	return f.Count()
}

// GetColumnCount returns the number of columns in the table.
func (f *Fetcher) GetColumnCount() int {
	return FetchColumnSize
}

// GetReference returns the reference of the currently selected column in the table.
func (f *Fetcher) GetReference(do ...func(d *FetcherData)) (*FetcherData, bool) {
	row, _ := f.table.GetSelection()
	ref := f.table.GetCell(row, 0).Reference

	data, ok := ref.(*FetcherData)
	if ok && do != nil {
		do[0](data)
	}

	return data, ok
}

// Keybindings define the keybindings for the media fetcher.
func (f *Fetcher) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	operation := cmd.KeyOperation(event, cmd.KeyContextFetcher)

	switch operation {
	case cmd.KeyFetcherClearCompleted:
		f.ClearErrors()

	case cmd.KeyFetcherCancel:
		f.GetReference(func(d *FetcherData) {
			f.Cancel(d)
		})

	case cmd.KeyFetcherReload:
		f.GetReference(func(d *FetcherData) {
			go f.Fetch(d.Info, d.Audio, d)
		})

	case cmd.KeyFetcherCancelAll:
		f.CancelAll(true)

	case cmd.KeyFetcherReloadAll:
		f.FetchAll()

	case cmd.KeyPlayerStop, cmd.KeyClose:
		f.Hide()
	}

	return event
}

// selectorHandler shows any error messages for any selcted fetcher entry.
func (f *Fetcher) selectorHandler(row, col int) {
	f.info.Clear()

	data, ok := f.Get(row)
	if !ok {
		return
	}

	info := "No errors"
	if data.Error != nil {
		info = fmt.Sprintf("[red::bu]Error:[-:-:-]\n[::b]%s[-:-:-]", data.Error.Error())
	}

	f.info.SetText(info)
}

package ui

import (
	"fmt"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

var (
	// ResultsList is a table to display results.
	ResultsList *tview.Table

	listWidth    int
	searchLock   *semaphore.Weighted
	searchString string
)

const loadingText = "Search still in progress, please wait"

// SetupList sets up a table to display search results.
func SetupList() {
	ResultsList = tview.NewTable()

	ResultsList.SetBorder(false)
	ResultsList.SetSelectorWrap(true)
	ResultsList.SetBackgroundColor(tcell.ColorDefault)

	ResultsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captureListEvents(event)
		capturePlayerEvent(event)

		return event
	})

	ResultsList.SetSelectionChangedFunc(func(row, col int) {
		rows := ResultsList.GetRowCount()

		if row < 0 || row > rows {
			return
		}

		cell := ResultsList.GetCell(row, col)

		if cell == nil {
			return
		}

		ResultsList.SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorBlue).
			Foreground(tcell.ColorWhite).
			Attributes(cell.Attributes | tcell.AttrBold))
	})

	searchLock = semaphore.NewWeighted(1)
}

// SearchAndList displays search results on the screen.
func SearchAndList(text string) {
	var getmore bool

	if !searchLock.TryAcquire(1) {
		InfoMessage(loadingText, false)
		return
	}
	defer searchLock.Release(1)

	if text == "" && searchString == "" {
		return
	}

	msg := "Fetching"
	if text != "" {
		getmore = false
		searchString = text
	} else {
		getmore = true
		msg += " more"
	}

	InfoMessage(msg+" results for '"+tview.Escape(searchString)+"'", true)

	results, err := lib.GetClient().Search(searchString, getmore)
	if err != nil {
		ErrorMessage(err)
		return
	}

	if results == nil {
		ErrorMessage(fmt.Errorf("No more results"))
		return
	}

	App.QueueUpdateDraw(func() {
		pos := -1

		if text != "" {
			searchString = text
			ResultsList.Clear()
			ResultsList.SetSelectable(false, false)
		}

		rows := ResultsList.GetRowCount()
		_, _, width, _ := ResultsList.GetRect()

		for i, result := range results {
			if pos < 0 {
				pos = rows + i
			}

			ResultsList.SetCell(rows+i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
				SetExpansion(1).
				SetMaxWidth((width / 4)).
				SetReference(result.VideoID),
			)

			ResultsList.SetCell(rows+i, 2, tview.NewTableCell(" ").
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)

			ResultsList.SetCell(rows+i, 3, tview.NewTableCell("[purple::b]"+result.Author).
				SetSelectable(false).
				SetMaxWidth((width / 4)).
				SetAlign(tview.AlignLeft),
			)

			ResultsList.SetCell(rows+i, 4, tview.NewTableCell(" ").
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)

			ResultsList.SetCell(rows+i, 5, tview.NewTableCell("[pink]"+lib.FormatDuration(result.LengthSeconds)).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)

			ResultsList.SetCell(rows+i, 6, tview.NewTableCell(" ").
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)

			ResultsList.SetCell(rows+i, 7, tview.NewTableCell("[pink]"+lib.FormatPublished(result.PublishedText)).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)
		}

		ResultsList.Select(pos, 0)
		ResultsList.ScrollToEnd()

		if !inputFocused() && !playlistFocused() {
			App.SetFocus(ResultsList)
		}

		ResultsList.SetSelectable(true, false)
	})

	InfoMessage("Results fetched", false)
}

// captureListEvents binds keys to ResultsList's InputCapture.
func captureListEvents(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEnter:
		loadMoreResults()
	}

	switch event.Rune() {
	case '/':
		searchText()

	case 'q':
		confirmQuit()
	}
}

// resizeListEntries detects if the screen is resized, and resizes
// each TableCell's text to an appropriate width.
func resizeListEntries(width int) {
	if listWidth == width {
		return
	}

	go func() {
		if !searchLock.TryAcquire(1) {
			return
		}
		defer searchLock.Release(1)

		App.QueueUpdateDraw(func() {
			for i := 0; i < ResultsList.GetRowCount(); i++ {
				for j := 0; j < 2; j++ {
					cell := ResultsList.GetCell(i, j)
					if cell == nil {
						continue
					}

					cell.SetMaxWidth((width / 3))
				}
			}

			pos, _ := ResultsList.GetSelection()
			ResultsList.Select(pos, 0)
		})
	}()

	listWidth = width
}

// searchText takes the search string from user input,
// clears ResultsList, and displays new search results.
func searchText() {
	sfunc := func(text string) {
		App.SetFocus(ResultsList)
		Status.SwitchToPage("messages")

		if text != "" {
			lib.AddToHistory(text)
		}

		go SearchAndList(text)
	}

	ifunc := func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:
			sfunc(InputBox.GetText())

		case tcell.KeyEscape:
			App.SetFocus(ResultsList)
			Status.SwitchToPage("messages")

		case tcell.KeyUp:
			t := lib.HistoryReverse()
			if t != "" {
				InputBox.SetText(t)
			}

		case tcell.KeyDown:
			t := lib.HistoryForward()
			if t != "" {
				InputBox.SetText(t)
			}
		}

		return e
	}

	SetInput("Search:", 0, sfunc, ifunc)
}

// loadMoreResults appends more search results to ResultsList
func loadMoreResults() {
	go SearchAndList("")
}

// getListReference gets a saved reference from a selected TableCell.
func getListReference() (string, string, error) {
	if !searchLock.TryAcquire(1) {
		return "", "", fmt.Errorf(loadingText)
	}
	defer searchLock.Release(1)

	row, _ := ResultsList.GetSelection()
	rows := ResultsList.GetRowCount()
	err := fmt.Errorf("Cannot select this entry")

	if row+1 < rows {
		ResultsList.Select(row+1, 0)
	}

	cell := ResultsList.GetCell(row, 0)
	if cell == nil {
		return "", "", err
	}

	ref := cell.GetReference()
	if ref == nil {
		return "", "", err
	}

	return cell.Text, ref.(string), nil
}

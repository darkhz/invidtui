package ui

import (
	"fmt"
	"strconv"

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
	stype        string
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

	toggleSearch()
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

	msg := "Fetching "
	if text != "" {
		getmore = false
		searchString = text
	} else {
		getmore = true
		msg += "more "
	}

	InfoMessage(msg+stype+" results for '"+tview.Escape(searchString)+"'", true)

	results, err := lib.GetClient().Search(stype, searchString, getmore)
	if err != nil {
		ErrorMessage(err)
		return
	}

	if results == nil {
		ErrorMessage(fmt.Errorf("No more results"))
		return
	}

	App.QueueUpdateDraw(func() {
		if text != "" {
			searchString = text
			ResultsList.Clear()
			ResultsList.SetSelectable(false, false)
		}

		searchAndList(results)
	})

	InfoMessage("Results fetched", false)
}

// searchAndList renders the search results list.
func searchAndList(results []lib.SearchResult) {
	pos := -1
	rows := ResultsList.GetRowCount()
	_, _, width, _ := ResultsList.GetRect()

	for i, result := range results {
		var lentext string

		if pos < 0 {
			pos = rows + i
		}

		if result.Title == "" {
			result.Title = result.Author
			result.Author = ""
		}

		if result.LiveNow {
			lentext = "Live"
		} else {
			lentext = lib.FormatDuration(result.LengthSeconds)
		}

		ResultsList.SetCell(rows+i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
			SetExpansion(1).
			SetReference(result).
			SetMaxWidth((width / 4)),
		)

		ResultsList.SetCell(rows+i, 1, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		ResultsList.SetCell(rows+i, 2, tview.NewTableCell("[purple::b]"+result.Author).
			SetSelectable(false).
			SetMaxWidth((width / 4)).
			SetAlign(tview.AlignLeft),
		)

		ResultsList.SetCell(rows+i, 3, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "playlist" || result.Type == "channel" {
			ResultsList.SetCell(rows+i, 4, tview.NewTableCell("[pink]"+strconv.Itoa(result.VideoCount)+" videos").
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)
		} else {
			ResultsList.SetCell(rows+i, 4, tview.NewTableCell("[pink]"+lentext).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)
		}

		ResultsList.SetCell(rows+i, 5, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "channel" {
			ResultsList.SetCell(rows+i, 6, tview.NewTableCell("[pink]"+lib.FormatNumber(result.SubCount)+" subs").
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)
		} else {
			ResultsList.SetCell(rows+i, 6, tview.NewTableCell("[pink]"+lib.FormatPublished(result.PublishedText)).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)
		}
	}

	ResultsList.Select(pos, 0)
	ResultsList.ScrollToEnd()

	ResultsList.SetSelectable(true, false)
}

// captureListEvents binds keys to ResultsList's InputCapture.
func captureListEvents(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEnter:
		loadMoreResults()
	}

	switch event.Rune() {
	case '/':
		searchText(event.Modifiers() == tcell.ModAlt)

	case 'i':
		ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)

	case 'u':
		ViewChannel("video", true, event.Modifiers() == tcell.ModAlt)

	case 'U':
		ViewChannel("playlist", true, event.Modifiers() == tcell.ModAlt)
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
func searchText(channel bool) {
	srchfn := func(text string) {
		if channel {
			SearchChannel(text)
		} else {
			SearchAndList(text)
		}
	}

	srchfocus := func() {
		if channel {
			App.SetFocus(chSearchTable)
		} else {
			App.SetFocus(ResultsList)
		}

		Status.SwitchToPage("messages")
	}

	sfunc := func(text string) {
		srchfocus()

		if text != "" {
			lib.AddToHistory(text)
		} else {
			return
		}

		go srchfn(text)
	}

	ifunc := func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyEnter:
			sfunc(InputBox.GetText())

		case tcell.KeyEscape:
			srchfocus()

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

		case tcell.KeyCtrlE:
			if channel {
				return nil
			}

			InputBox.SetLabel(toggleSearch())
		}

		return e
	}

	label := "Search"

	if channel {
		label += " channel:"
		if ResultsList.HasFocus() {
			err := ViewChannel("search", true, false)
			if err != nil {
				ErrorMessage(err)
				return
			}
		}
	} else {
		label += " (" + stype + "):"
	}

	SetInput(label, 0, sfunc, ifunc)
}

// toggleSearch toggles the search type.
func toggleSearch() string {
	switch stype {
	case "video":
		stype = "playlist"

	case "playlist":
		stype = "channel"

	case "channel":
		fallthrough

	default:
		stype = "video"
	}

	return "[::b]Search (" + stype + "): "
}

// loadMoreResults appends more search results to ResultsList
func loadMoreResults() {
	go SearchAndList("")
}

// getListReference gets a saved reference from a selected TableCell.
func getListReference() (lib.SearchResult, error) {
	var table *tview.Table

	if ResultsList.HasFocus() {
		if !searchLock.TryAcquire(1) {
			return lib.SearchResult{}, fmt.Errorf(loadingText)
		}
		defer searchLock.Release(1)

		table = ResultsList
	} else if plistTable.HasFocus() {
		table = plistTable
	} else {
		_, item := chPages.GetFrontPage()
		table = item.(*tview.Table)
	}

	row, _ := table.GetSelection()
	rows := table.GetRowCount()
	err := fmt.Errorf("Cannot select this entry")

	if row+1 < rows {
		table.Select(row+1, 0)
	}

	cell := table.GetCell(row, 0)
	if cell == nil {
		return lib.SearchResult{}, err
	}

	ref := cell.GetReference()
	if ref == nil {
		return lib.SearchResult{}, err
	}

	return ref.(lib.SearchResult), nil
}

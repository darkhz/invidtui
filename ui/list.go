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
	// ResultsFlex contains the arranged page marker
	// and ResultsList table elements.
	ResultsFlex *tview.Flex

	// ResultsList is a table to display results.
	ResultsList    *tview.Table
	resultPageMark *tview.TextView

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

	resultPageMark = tview.NewTextView()
	resultPageMark.SetWrap(false)
	resultPageMark.SetRegions(true)
	resultPageMark.SetDynamicColors(true)
	resultPageMark.SetBackgroundColor(tcell.ColorDefault)
	resultPageMark.SetText(
		`[::b]Search[-:-:-] ["video"][darkcyan]Videos[""] ["playlist"][darkcyan]Playlists[""] ["channel"][darkcyan]Channels[""]`,
	)

	box := tview.NewBox().
		SetBackgroundColor(tcell.ColorDefault)

	ResultsFlex = tview.NewFlex().
		AddItem(resultPageMark, 1, 0, false).
		AddItem(box, 1, 0, false).
		AddItem(ResultsList, 0, 10, true).
		SetDirection(tview.FlexRow)

	ResultsFlex.SetBackgroundColor(tcell.ColorDefault)

	ResultsList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captureListEvents(event)
		capturePlayerEvent(event)

		return event
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
		searchAndList(results)
	})

	InfoMessage("Results fetched", false)
}

// searchAndList renders the search results list.
func searchAndList(results []lib.SearchResult) {
	var skipped int

	pos := -1
	rows := ResultsList.GetRowCount()
	_, _, width, _ := VPage.GetRect()

	for i, result := range results {
		select {
		case <-lib.GetSearchCtx().Done():
			ResultsList.Clear()
			return

		default:
		}
		var lentext string

		if result.Type == "category" {
			skipped++
			continue
		}

		if pos < 0 {
			pos = (rows + i) - skipped
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

		actualRow := (rows + i) - skipped

		ResultsList.SetCell(actualRow, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
			SetExpansion(1).
			SetReference(result).
			SetMaxWidth((width / 4)).
			SetSelectedStyle(mainStyle),
		)

		ResultsList.SetCell(actualRow, 1, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		ResultsList.SetCell(actualRow, 2, tview.NewTableCell("[purple::b]"+result.Author).
			SetSelectable(true).
			SetMaxWidth((width / 4)).
			SetAlign(tview.AlignLeft).
			SetSelectedStyle(auxStyle),
		)

		ResultsList.SetCell(actualRow, 3, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "playlist" || result.Type == "channel" {
			ResultsList.SetCell(actualRow, 4, tview.NewTableCell("[pink]"+strconv.Itoa(result.VideoCount)+" videos").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)

			if result.Type == "playlist" {
				continue
			}
		} else {
			ResultsList.SetCell(actualRow, 4, tview.NewTableCell("[pink]"+lentext).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}

		ResultsList.SetCell(actualRow, 5, tview.NewTableCell(" ").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)

		if result.Type == "channel" {
			ResultsList.SetCell(actualRow, 6, tview.NewTableCell("[pink]"+lib.FormatNumber(result.SubCount)+" subs").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		} else {
			ResultsList.SetCell(actualRow, 6, tview.NewTableCell("[pink]"+lib.FormatPublished(result.PublishedText)).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}
	}

	ResultsList.Select(pos, 0)
	ResultsList.ScrollToEnd()

	ResultsList.SetSelectable(true, false)

	if bannerShown && len(results) > 0 {
		bannerShown = false
		VPage.SwitchToPage("search")
	}
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

	srchfocus := func() *tview.Table {
		var table *tview.Table

		if channel {
			table = chSearchTable
		} else {
			table = ResultsList
		}

		App.SetFocus(table)
		Status.SwitchToPage("messages")

		return table
	}

	sfunc := func(text string) {
		table := srchfocus()

		if text != "" {
			lib.AddToHistory(text)
			table.Clear()
			table.SetSelectable(false, false)
			resultPageMark.Highlight(stype)
			lib.SearchCancel()
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

// getListTable gets the Table in current focus.
func getListTable() *tview.Table {
	item := App.GetFocus()

	if item, ok := item.(*tview.Table); ok {
		return item
	}

	return nil
}

// getListReference gets a saved reference from a selected TableCell.
func getListReference() (lib.SearchResult, error) {
	var table *tview.Table

	err := fmt.Errorf("Cannot select this entry")

	table = getListTable()
	if table == nil {
		return lib.SearchResult{}, err
	}

	row, _ := table.GetSelection()

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

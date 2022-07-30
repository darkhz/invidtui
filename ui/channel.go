package ui

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	chPages       *tview.Pages
	chTitle       *tview.TextView
	chDesc        *tview.TextView
	chVbox        *tview.Box
	chViewFlex    *tview.Flex
	chPageMark    *tview.TextView
	chVideoTable  *tview.Table
	chPlistTable  *tview.Table
	chSearchTable *tview.Table
	chPrevItem    tview.Primitive

	chanID           string
	currType         string
	chPrevPage       string
	chSearchString   string
	chExited         bool
	chVideoLoaded    bool
	chPlaylistLoaded bool
	chSearchLoaded   bool
	chLock           sync.Mutex
)

// setupViewChannel sets up the channel view.
func setupViewChannel() {
	var tables []*tview.Table

	for i := 0; i <= 2; i++ {
		table := tview.NewTable()
		table.SetSelectorWrap(true)
		table.SetBackgroundColor(tcell.ColorDefault)

		table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			chTableEvents(event)

			return event
		})

		table.SetSelectionChangedFunc(func(row, col int) {
			chTableSelectionFunc(table, row, col)
		})

		tables = append(tables, table)
	}

	chVideoTable = tables[0]
	chPlistTable = tables[1]
	chSearchTable = tables[2]

	chTitle = tview.NewTextView()
	chTitle.SetDynamicColors(true)
	chTitle.SetTextAlign(tview.AlignCenter)
	chTitle.SetBackgroundColor(tcell.ColorDefault)

	chDesc = tview.NewTextView()
	chDesc.SetDynamicColors(true)
	chDesc.SetTextAlign(tview.AlignCenter)
	chDesc.SetBackgroundColor(tcell.ColorDefault)

	chPageMark = tview.NewTextView()
	chPageMark.SetWrap(false)
	chPageMark.SetRegions(true)
	chPageMark.SetDynamicColors(true)
	chPageMark.SetBackgroundColor(tcell.ColorDefault)
	chPageMark.SetText(
		`[::b]Channel[-:-:-] ["video"][darkcyan]Videos[""] ["playlist"][darkcyan]Playlists[""] ["search"][darkcyan]Search[""]`,
	)

	chVbox = getVbox()

	chPages = tview.NewPages().
		AddPage("video", chVideoTable, true, false).
		AddPage("playlist", chPlistTable, true, false).
		AddPage("search", chSearchTable, true, false)

	chViewFlex = tview.NewFlex().
		AddItem(chPageMark, 1, 0, false).
		AddItem(chTitle, 1, 0, false).
		AddItem(chVbox, 1, 0, false).
		AddItem(chDesc, 2, 0, false).
		AddItem(chVbox, 1, 0, false).
		AddItem(chPages, 10, 100, true).
		SetDirection(tview.FlexRow)
}

// ViewChannel shows the playlist contents after loading the playlist URL.
func ViewChannel(vtype string, newlist, noload bool) error {
	var err error
	var info lib.SearchResult

	if noload {
		_, item := chPages.GetFrontPage()
		if !VPage.HasPage("channelview") || item == nil {
			err = fmt.Errorf("Channel not loaded")
			InfoMessage(err.Error(), false)
			return err
		}

		VPage.SwitchToPage("channelview")
		App.SetFocus(item)

		return nil
	}

	if newlist {
		info, err = getListReference()

		if err != nil {
			ErrorMessage(err)
			return err
		}

		setChExited(false)
		setCurrType(vtype)

		chVideoTable.Clear()
		chPlistTable.Clear()
		chSearchTable.Clear()

		for _, v := range []string{
			"video",
			"playlist",
			"search",
		} {
			setChPageLoaded(v, false)
		}
	}

	if info.AuthorID != "" {
		chanID = info.AuthorID
	}

	chPrevPage, chPrevItem = VPage.GetFrontPage()

	chPageMark.Highlight(vtype)
	chPages.SwitchToPage(vtype)

	ResultsList.SetSelectable(false, false)
	go viewChannel(info, vtype, newlist)

	return nil
}

// viewChannel loads the playlist URL and shows the channel contents.
func viewChannel(info lib.SearchResult, vtype string, newlist bool) {
	var err error
	var qsrch, cancel bool
	var result lib.ChannelResult
	var resfunc func(pos, rows, width int) int

	if vtype != "search" {
		InfoMessage("Loading channel "+vtype+" entries", true)
		defer InfoMessage("Loaded channel "+vtype+" entries", false)
	}

	switch vtype {
	case "video":
		result, err = lib.GetClient().ChannelVideos(info.AuthorID)
		resfunc = func(pos, rows, width int) int {
			return listChannelVideos(info, pos, rows, width, result)
		}

	case "playlist":
		result, err = lib.GetClient().ChannelPlaylists(info.AuthorID)
		resfunc = func(pos, rows, width int) int {
			return listChannelPlaylists(info, pos, rows, width, result)
		}

	case "search":
		qsrch = true
		result.Author = info.Author
		result.ChannelID = info.AuthorID
		result.Description = info.Description
		resfunc = func(pos, rows, width int) int {
			return 0
		}

	default:
		return
	}
	if err != nil {
		ErrorMessage(err)
		cancel = true
	}

	rmdesc := func() {
		chViewFlex.RemoveItem(chPages)
		chViewFlex.RemoveItem(chDesc)
		chViewFlex.RemoveItem(chVbox)

		chViewFlex.AddItem(chPages, 0, 10, true)
	}

	insdesc := func(s int) {
		chViewFlex.RemoveItem(chPages)

		chViewFlex.AddItem(chVbox, 1, 0, false)
		chViewFlex.AddItem(chDesc, s, 0, false)
		chViewFlex.AddItem(chVbox, 1, 0, false)
		chViewFlex.AddItem(chPages, 0, 10, true)
	}

	App.QueueUpdateDraw(func() {
		if cancel {
			ResultsList.SetSelectable(true, false)
			return
		}

		_, item := chPages.GetFrontPage()
		chTable := item.(*tview.Table)

		_, _, width, _ := ResultsList.GetRect()

		if newlist {
			desc := strings.ReplaceAll(result.Description, "\n", " ")
			desclen := len(desc)

			rmdesc()

			if desclen > 0 {
				s := 2
				if desclen >= width {
					s++
				} else {
					s--
				}

				insdesc(s)
			}

			chDesc.SetText(desc)
			chTitle.SetText("[::bu]" + result.Author)

			if !VPage.HasPage("channelview") {
				VPage.AddPage("channelview", chViewFlex, true, true)
			}
		}

		chTable.SetSelectable(false, false)

		if vtype == getCurrType() {
			pos := resfunc(-1, chTable.GetRowCount(), width)

			if pos >= 0 {
				chTable.Select(pos, 0)

				if pos == 0 {
					chTable.ScrollToBeginning()
				} else {
					chTable.ScrollToEnd()
				}
			}

			if !getChExited() {
				VPage.SwitchToPage("channelview")

				focusChTable(!qsrch, chTable)
			}

			setChPageLoaded(vtype, true)
		}

		chTable.SetSelectable(true, false)
		ResultsList.SetSelectable(true, false)
	})
}

// listChannelVideos loads and displays videos from a channel.
func listChannelVideos(info lib.SearchResult, pos, rows, width int, result lib.ChannelResult) int {
	var skipped int

	if len(result.Videos) == 0 {
		InfoMessage("No more video results", false)
		return pos
	}

	for i, v := range result.Videos {
		select {
		case <-lib.ChannelCtx().Done():
			return pos

		default:
		}

		if pos < 0 {
			pos = (rows + i) - skipped
		}

		if v.LengthSeconds == 0 {
			skipped++
			continue
		}

		sref := lib.SearchResult{
			Type:     "video",
			Title:    v.Title,
			VideoID:  v.VideoID,
			AuthorID: result.ChannelID,
			Author:   result.Author,
		}

		chVideoTable.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
			SetExpansion(1).
			SetReference(sref).
			SetMaxWidth((width / 4)).
			SetSelectedStyle(mainStyle),
		)

		chVideoTable.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+lib.FormatDuration(v.LengthSeconds)).
			SetSelectable(true).
			SetAlign(tview.AlignRight).
			SetSelectedStyle(auxStyle),
		)
	}

	InfoMessage("Video entries loaded", false)

	return pos
}

// listChannelPlaylists loads and displays playlists from a channel.
func listChannelPlaylists(info lib.SearchResult, pos, rows, width int, result lib.ChannelResult) int {
	if len(result.Playlists) == 0 {
		InfoMessage("No more playlist results", false)
		return pos
	}

	for i, p := range result.Playlists {
		select {
		case <-lib.ChannelCtx().Done():
			return pos

		default:
		}

		if pos < 0 {
			pos = (rows + i)
		}

		sref := lib.SearchResult{
			Type:       "playlist",
			Title:      p.Title,
			PlaylistID: p.PlaylistID,
			AuthorID:   result.ChannelID,
			Author:     result.Author,
		}

		chPlistTable.SetCell((rows + i), 0, tview.NewTableCell("[blue::b]"+tview.Escape(p.Title)).
			SetExpansion(1).
			SetReference(sref).
			SetMaxWidth((width / 4)).
			SetSelectedStyle(mainStyle),
		)

		chPlistTable.SetCell((rows + i), 1, tview.NewTableCell("[pink]"+strconv.Itoa(p.VideoCount)+" videos").
			SetSelectable(true).
			SetAlign(tview.AlignRight).
			SetSelectedStyle(auxStyle),
		)
	}

	InfoMessage("Playlist entries loaded", false)

	return pos
}

// SearchChannel displays search results from the channel to the screen.
func SearchChannel(text string) {
	var getmore bool

	if text == "" && chSearchString == "" {
		return
	}

	msg := "Fetching "
	if text != "" {
		getmore = false
		chSearchString = text
	} else {
		getmore = true
		msg += "more "
	}

	InfoMessage(msg+"search results for '"+tview.Escape(chSearchString)+"'", true)

	results, err := lib.GetClient().Search(stype, chSearchString, getmore, chanID)
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

		rows := chSearchTable.GetRowCount()
		_, _, width, _ := chSearchTable.GetRect()

		for i, result := range results {
			if pos < 0 {
				pos = rows + i
			}

			if result.Title == "" {
				result.Title = result.Author
				result.Author = ""
			}

			chSearchTable.SetCell(rows+i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
				SetExpansion(1).
				SetReference(result).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(mainStyle),
			)

			chSearchTable.SetCell(rows+i, 1, tview.NewTableCell(" ").
				SetSelectable(false).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)

			chSearchTable.SetCell(rows+i, 2, tview.NewTableCell("[pink]"+result.Type).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)
		}

		chSearchTable.Select(pos, 0)
		chSearchTable.ScrollToEnd()

		chSearchTable.SetSelectable(true, false)
	})

	InfoMessage("Results fetched", false)
}

// loadMoreChannelResults appends more playlist results to the playlist
// view table.
func loadMoreChannelResults() {
	ctype := getCurrType()

	if ctype == "playlist" {
		return
	}

	if ctype == "search" {
		go SearchChannel("")
		return
	}

	ViewChannel(ctype, false, false)
}

// chTableEvents handles the input events for the
// video, playlist and search tables.
func chTableEvents(event *tcell.EventKey) {
	capturePlayerEvent(event)

	switch event.Key() {
	case tcell.KeyEnter:
		loadMoreChannelResults()

	case tcell.KeyTab:
		switchChannelTabs()

	case tcell.KeyEscape:
		setChExited(true)
		VPage.SwitchToPage(chPrevPage)
		App.SetFocus(chPrevItem)
		ResultsList.SetSelectable(true, false)
	}

	switch event.Rune() {
	case 'i':
		ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)

	case '/':
		setCurrType("search")
		chPageMark.Highlight("search")
		chPages.SwitchToPage("search")
		searchText(true)
	}
}

// chTableSelectionFunc handles the selection method for the
// video, playlist and search tables.
func chTableSelectionFunc(table *tview.Table, row, col int) {
	rows := table.GetRowCount()

	if row < 0 || row > rows {
		return
	}

	cell := table.GetCell(row, col)

	if cell == nil {
		return
	}

	table.SetSelectedStyle(tcell.Style{}.
		Background(tcell.ColorBlue).
		Foreground(tcell.ColorWhite).
		Attributes(cell.Attributes | tcell.AttrBold))
}

// switchChannelTabs switches the channel pages.
func switchChannelTabs() {
	ctype := getCurrType()

	switch ctype {
	case "video":
		ctype = "playlist"

	case "playlist":
		ctype = "search"

	case "search":
		ctype = "video"
	}

	setCurrType(ctype)

	chPageMark.Highlight(ctype)
	chPages.SwitchToPage(ctype)

	_, item := chPages.GetFrontPage()
	table := item.(*tview.Table)

	App.SetFocus(table)
	table.SetSelectable(true, false)

	if table.GetRowCount() == 0 && !isChPageLoaded(ctype) {
		if ctype == "search" {
			searchText(true)
			return
		}

		info := lib.SearchResult{
			Type:     currType,
			AuthorID: chanID,
		}

		go viewChannel(info, ctype, true)
	}
}

func focusChTable(focus bool, chTable *tview.Table) {
	if !focus {
		return
	}

	if pg, _ := MPage.GetFrontPage(); pg == "ui" {
		App.SetFocus(chTable)
	}
}

func isChPageLoaded(vtype string) bool {
	chLock.Lock()
	defer chLock.Unlock()

	switch vtype {
	case "video":
		return chVideoLoaded

	case "playlist":
		return chPlaylistLoaded

	case "search":
		return chSearchLoaded
	}

	return false
}

func setChPageLoaded(vtype string, loaded bool) {
	chLock.Lock()
	defer chLock.Unlock()

	switch vtype {
	case "video":
		chVideoLoaded = loaded

	case "playlist":
		chPlaylistLoaded = loaded

	case "search":
		chSearchLoaded = loaded
	}
}

func getCurrType() string {
	chLock.Lock()
	defer chLock.Unlock()

	return currType
}

func setCurrType(vtype string) {
	chLock.Lock()
	defer chLock.Unlock()

	currType = vtype
}

func getChExited() bool {
	chLock.Lock()
	defer chLock.Unlock()

	return chExited
}

func setChExited(exit bool) {
	chLock.Lock()
	defer chLock.Unlock()

	chExited = exit
}

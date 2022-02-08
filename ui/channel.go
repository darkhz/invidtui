package ui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

var (
	chViewFlex   *tview.Flex
	chanTable    *tview.Table
	chTableTitle *tview.TextView
	chTableDesc  *tview.TextView
	chTableVBox  *tview.Box

	currType    string
	chRateLimit *semaphore.Weighted
)

func setupViewChannel() {
	chanTable = tview.NewTable()
	chanTable.SetSelectorWrap(true)
	chanTable.SetBackgroundColor(tcell.ColorDefault)

	chTableTitle = tview.NewTextView()
	chTableTitle.SetDynamicColors(true)
	chTableTitle.SetTextAlign(tview.AlignCenter)
	chTableTitle.SetBackgroundColor(tcell.ColorDefault)

	chTableDesc = tview.NewTextView()
	chTableDesc.SetDynamicColors(true)
	chTableDesc.SetTextAlign(tview.AlignCenter)
	chTableDesc.SetBackgroundColor(tcell.ColorDefault)

	chTableVBox = getVbox()

	chViewFlex = tview.NewFlex().
		SetDirection(tview.FlexRow)

	chanTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		capturePlayerEvent(event)

		switch event.Key() {
		case tcell.KeyEnter:
			loadMoreChannelResults()

		case tcell.KeyEscape:
			VPage.SwitchToPage("main")
			App.SetFocus(ResultsList)
			ResultsList.SetSelectable(true, false)

		case tcell.KeyCtrlX:
			lib.GetClient().Playlist("", true)
		}

		switch event.Rune() {
		case 'i':
			ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)
		}

		return event
	})

	chanTable.SetSelectionChangedFunc(func(row, col int) {
		rows := chanTable.GetRowCount()

		if row < 0 || row > rows {
			return
		}

		cell := chanTable.GetCell(row, col)

		if cell == nil {
			return
		}

		chanTable.SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorBlue).
			Foreground(tcell.ColorWhite).
			Attributes(cell.Attributes | tcell.AttrBold))
	})

	chRateLimit = semaphore.NewWeighted(1)
}

// loadMoreChannelResults appends more playlist results to the playlist
// view table.
func loadMoreChannelResults() {
	ViewChannel(currType, false, false)
}

// ViewChannel shows the playlist contents after loading the playlist URL.
func ViewChannel(vtype string, newlist, noload bool) {
	var err error
	var info lib.SearchResult

	if noload {
		if chanTable.GetRowCount() == 0 {
			InfoMessage("No channel entries", false)
			return
		}

		VPage.SwitchToPage("channelview")
		App.SetFocus(chanTable)

		return
	}

	if newlist {
		info, err = getListReference()

		if err != nil {
			ErrorMessage(err)
			return
		}

		if info.Type != "channel" {
			ErrorMessage(fmt.Errorf("Cannot load channel from " + info.Type + " type"))
			return
		}

		currType = vtype
	}

	go viewChannel(info, vtype, newlist)
}

// viewChannel loads the playlist URL and shows the channel contents.
func viewChannel(info lib.SearchResult, vtype string, newlist bool) {
	var err error
	var result lib.ChannelResult
	var resfunc func(pos, rows int) int

	InfoMessage("Loading channel "+vtype+" entries", false)
	ResultsList.SetSelectable(false, false)
	defer ResultsList.SetSelectable(true, false)

	if !chRateLimit.TryAcquire(1) {
		InfoMessage("Channel fetch in progress, please wait", false)
		return
	}
	defer chRateLimit.Release(1)

	_, _, width, _ := ResultsList.GetRect()

	switch vtype {
	case "video":
		result, err = lib.GetClient().ChannelVideos(info.AuthorID, false)
		resfunc = func(pos, rows int) int {
			return loadChannelVideos(info, pos, rows, width, result)
		}

	case "playlist":
		result, err = lib.GetClient().ChannelPlaylists(info.AuthorID, false)
		resfunc = func(pos, rows int) int {
			return loadChannelPlaylists(info, pos, rows, width, result)
		}

	default:
		return
	}
	if err != nil {
		if errors.Is(err, context.Canceled) {
			InfoMessage("Loading cancelled", false)
		}

		return
	}

	App.QueueUpdateDraw(func() {
		if newlist {
			chViewFlex.Clear()
			chanTable.Clear()

			desc := strings.ReplaceAll(result.Description, "\n", " ")
			desclen := len(desc)

			chViewFlex.AddItem(chTableTitle, 1, 0, false)

			if desclen > 0 {
				s := 2
				if desclen >= width {
					s++
				} else {
					s--
				}

				chViewFlex.AddItem(chTableVBox, 1, 0, false)
				chViewFlex.AddItem(chTableDesc, s, 0, false)
				chViewFlex.AddItem(chTableVBox, 1, 0, false)
			}

			chViewFlex.AddItem(chanTable, 0, 10, true)

			chTableDesc.SetText(desc)
			chTableTitle.SetText("[::bu]Channel: " + result.Author)

			VPage.AddAndSwitchToPage("channelview", chViewFlex, true)
		}

		chanTable.SetSelectable(false, false)
		defer chanTable.SetSelectable(true, false)

		pos := -1
		rows := chanTable.GetRowCount()

		pos = resfunc(pos, rows)
		if pos >= 0 {
			chanTable.Select(pos, 0)
		}

		chanTable.ScrollToEnd()

		name, _ := VPage.GetFrontPage()
		if name == "channelview" {
			App.SetFocus(chanTable)
		}
	})
}

// loadChannelVideos loads and displays videos from a channel.
func loadChannelVideos(info lib.SearchResult, pos, rows, width int, result lib.ChannelResult) int {
	if len(result.Videos) == 0 {
		InfoMessage("No more results", false)
		return pos
	}

	for i, v := range result.Videos {
		select {
		case <-lib.PlistCtx.Done():
			return pos

		default:
		}

		if pos < 0 {
			pos = (rows + i)
		}

		sref := lib.SearchResult{
			Type:    "video",
			Title:   v.Title,
			VideoID: v.VideoID,
		}

		chanTable.SetCell((rows + i), 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
			SetExpansion(1).
			SetReference(sref).
			SetMaxWidth((width / 4)),
		)

		chanTable.SetCell((rows + i), 1, tview.NewTableCell("[pink]"+lib.FormatDuration(v.LengthSeconds)).
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)
	}

	InfoMessage("Video entries loaded", false)

	return pos
}

// loadChannelPlaylists loads and displays playlists from a channel.
func loadChannelPlaylists(info lib.SearchResult, pos, rows, width int, result lib.ChannelResult) int {
	if len(result.Playlists) == 0 {
		InfoMessage("No more results", false)
		return pos
	}

	for i, p := range result.Playlists {
		select {
		case <-lib.PlistCtx.Done():
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
		}

		chanTable.SetCell((rows + i), 0, tview.NewTableCell("[blue::b]"+tview.Escape(p.Title)).
			SetExpansion(1).
			SetReference(sref).
			SetMaxWidth((width / 4)),
		)

		chanTable.SetCell((rows + i), 1, tview.NewTableCell("[pink]"+strconv.Itoa(p.VideoCount)+" videos").
			SetSelectable(false).
			SetAlign(tview.AlignRight),
		)
	}

	InfoMessage("Playlist entries loaded", false)

	return pos
}

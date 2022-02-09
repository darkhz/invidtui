package ui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// EntryData stores playlist entry data.
type EntryData struct {
	ID       int    `json:"id"`
	Filename string `json:filename`
	Playing  bool   `json:"playing"`
	Title    string
	Author   string
	Duration string
}

var (
	// Playlist shows the playlist popup
	Playlist   *tview.Flex
	plistPopup *tview.Table
	plistTitle *tview.TextView

	plViewFlex   *tview.Flex
	plistTable   *tview.Table
	plTableTitle *tview.TextView
	plTableDesc  *tview.TextView
	plTableVBox  *tview.Box

	prevrow       int
	prevpage      string
	previtem      tview.Primitive
	moving        bool
	pctx          context.Context
	pcancel       context.CancelFunc
	playlistEvent chan struct{}

	plistIdMap  map[string]struct{}
	plRateLimit *semaphore.Weighted
)

// SetupPlaylist sets up the playlist popup.
func SetupPlaylist() {
	setupViewPlaylist()
	setupViewChannel()
	setupPlaylistPopup()

	playlistEvent = make(chan struct{})
	plistIdMap = make(map[string]struct{})

	plRateLimit = semaphore.NewWeighted(1)
}

// setupViewPlaylist sets up the playlist view page.
func setupViewPlaylist() {
	plistTable = tview.NewTable()
	plistTable.SetSelectorWrap(true)
	plistTable.SetBackgroundColor(tcell.ColorDefault)

	plTableTitle = tview.NewTextView()
	plTableTitle.SetDynamicColors(true)
	plTableTitle.SetTextAlign(tview.AlignCenter)
	plTableTitle.SetBackgroundColor(tcell.ColorDefault)

	plTableDesc = tview.NewTextView()
	plTableDesc.SetDynamicColors(true)
	plTableDesc.SetTextAlign(tview.AlignCenter)
	plTableDesc.SetBackgroundColor(tcell.ColorDefault)

	plTableVBox = getVbox()

	plViewFlex = tview.NewFlex().
		SetDirection(tview.FlexRow)

	plistTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		capturePlayerEvent(event)

		switch event.Key() {
		case tcell.KeyEnter:
			loadMorePlistResults()

		case tcell.KeyEscape:
			VPage.SwitchToPage(prevpage)
			App.SetFocus(previtem)

		case tcell.KeyCtrlX:
			lib.GetClient().Playlist("", true)
		}

		return event
	})

	plistTable.SetSelectionChangedFunc(func(row, col int) {
		rows := plistTable.GetRowCount()

		if row < 0 || row > rows {
			return
		}

		cell := plistTable.GetCell(row, col)

		if cell == nil {
			return
		}

		plistTable.SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorBlue).
			Foreground(tcell.ColorWhite).
			Attributes(cell.Attributes | tcell.AttrBold))
	})
}

// setupPlaylistPopup sets up the playlist popup.
func setupPlaylistPopup() {
	plistTitle := tview.NewTextView()
	plistTitle.SetDynamicColors(true)
	plistTitle.SetTextColor(tcell.ColorBlue)
	plistTitle.SetText("[white::bu]Queue")
	plistTitle.SetTextAlign(tview.AlignCenter)
	plistTitle.SetBackgroundColor(tcell.ColorDefault)

	plistPopup = tview.NewTable()
	plistPopup.SetBorders(false)
	plistPopup.SetBackgroundColor(tcell.ColorDefault)

	Playlist = tview.NewFlex().
		AddItem(plistTitle, 1, 0, false).
		AddItem(plistPopup, 10, 10, false).
		SetDirection(tview.FlexRow)

	plistPopup.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			plEnter()

		case tcell.KeyEscape:
			plExit()

		case tcell.KeyLeft, tcell.KeyRight:
			ResultsList.InputHandler()(event, nil)

		case tcell.KeyCtrlS:
			plExit()
			ShowFileBrowser("Save as:", plSaveAs, plFbExit)

		case tcell.KeyCtrlA:
			plExit()
			ShowFileBrowser("Append from:", plOpenAppend, plFbExit)
		}

		switch event.Rune() {
		case 'd':
			plDelete()

		case 'M':
			plMove()

		case 'S':
			plExit()
			fallthrough

		case '<', '>':
			ResultsList.InputHandler()(event, nil)
			sendPlaylistEvent()

		case ' ', 'l', 's':
			ResultsList.InputHandler()(event, nil)
		}

		return event
	})

	plistPopup.SetSelectionChangedFunc(func(row, col int) {
		selector := ">"
		rows := plistPopup.GetRowCount()

		if moving {
			selector = "M"
		}

		for i := 0; i < rows; i++ {
			cell := plistPopup.GetCell(i, col)
			if cell == nil {
				cell = tview.NewTableCell("")
				plistPopup.SetCell(i, col, cell)
			}

			if i == row {
				cell.SetText(selector)
				continue
			}

			cell.SetText("")
		}

		plistPopup.SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorDefault).
			Foreground(tcell.ColorDefault))
	})
}

// playlistPopup loads the playlist, and displays a popup
// with the playlist items.
func playlistPopup() {
	if lib.GetMPV().PlaylistCount() == 0 {
		InfoMessage("Playlist empty", false)
		return
	}

	pctx, pcancel = context.WithCancel(context.Background())

	if plistPopup.GetRowCount() == 0 {
		plistPopup.SetCell(0, 1, tview.NewTableCell("[::b]Loading...").
			SetSelectable(false))
	}

	MPage.AddAndSwitchToPage(
		"playlist",
		statusmodal(Playlist, plistPopup),
		true,
	).ShowPage("ui")

	App.SetFocus(plistPopup)

	go startPlaylist()
}

// startPlaylist is the playlist update loop.
func startPlaylist() {
	var pos int
	var focused bool

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	update := func() {
		pldata := updatePlaylist()
		if len(pldata) == 0 {
			App.QueueUpdateDraw(func() {
				exitFocus()
				plistPopup.Clear()
			})

			pcancel()

			return
		}

		App.QueueUpdateDraw(func() {
			_, _, w, _ := plistPopup.GetRect()
			plistPopup.SetSelectable(false, false)

			for i, data := range pldata {
				var marker string

				if data.Playing {
					pos = i
					marker = " [white::b](playing)"
				}

				plistPopup.SetCell(i, 1, tview.NewTableCell("[blue::b]"+tview.Escape(data.Title)+marker).
					SetExpansion(1).
					SetMaxWidth(w/5).
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 2, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 3, tview.NewTableCell("[purple::b]"+tview.Escape(data.Author)).
					SetMaxWidth(w/5).
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 4, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 5, tview.NewTableCell("[pink::b]"+data.Duration).
					SetSelectable(false),
				)
			}

			plistPopup.SetSelectable(true, false)

			if !focused {
				plistPopup.Select(pos, 0)
				focused = true
			}

			resizemodal()
		})
	}

	for {
		select {
		case <-pctx.Done():
			return

		case <-playlistEvent:
			update()
			t.Reset(1 * time.Second)
			continue

		case <-t.C:
			update()
		}
	}
}

// loadMorePlistResults appends more playlist results to the playlist
// view table.
func loadMorePlistResults() {
	ViewPlaylist(false, false)
}

// ViewPlaylist shows the playlist contents after loading the playlist URL.
func ViewPlaylist(newlist, noload bool) {
	var err error
	var info lib.SearchResult

	if noload {
		if plistTable.GetRowCount() == 0 {
			InfoMessage("No playlist entries", false)
			return
		}

		VPage.SwitchToPage("playlistview")
		App.SetFocus(plistTable)

		return
	}

	if newlist {
		info, err = getListReference()

		if err != nil {
			ErrorMessage(err)
			return
		}

		if info.Type != "playlist" {
			ErrorMessage(fmt.Errorf("Cannot load playlist from %s type", info.Type))
			return
		}
	}

	prevpage, previtem = VPage.GetFrontPage()

	go viewPlaylist(info, newlist)
}

// viewPlaylist loads the playlist URL and shows the playlist contents.
func viewPlaylist(info lib.SearchResult, newlist bool) {
	var err error

	InfoMessage("Loading playlist entries", false)
	ResultsList.SetSelectable(false, false)
	defer ResultsList.SetSelectable(true, false)

	if !plRateLimit.TryAcquire(1) {
		InfoMessage("Playlist fetch in progress, please wait", false)
		return
	}
	defer plRateLimit.Release(1)

	result, err := lib.GetClient().Playlist(info.PlaylistID, false)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			InfoMessage("Loading cancelled", false)
		}

		return
	}

	App.QueueUpdateDraw(func() {
		var skipped int

		pos := -1

		_, _, width, _ := ResultsList.GetRect()

		if newlist {
			plViewFlex.Clear()
			plistTable.Clear()
			plistIdMap = make(map[string]struct{})

			desc := strings.ReplaceAll(result.Description, "\n", " ")
			desclen := len(desc)

			plViewFlex.AddItem(plTableTitle, 1, 0, false)

			if desclen > 0 {
				s := 2
				if desclen >= width {
					s++
				} else {
					s--
				}

				plViewFlex.AddItem(plTableVBox, 1, 0, false)
				plViewFlex.AddItem(plTableDesc, s, 0, false)
				plViewFlex.AddItem(plTableVBox, 1, 0, false)
			}

			plViewFlex.AddItem(plistTable, 0, 10, true)

			plTableDesc.SetText(desc)
			plTableTitle.SetText("[::bu]Playlist: " + result.Title)

			VPage.AddAndSwitchToPage("playlistview", plViewFlex, true)
		}

		rows := plistTable.GetRowCount()
		plistTable.SetSelectable(false, false)

		for i, v := range result.Videos {
			select {
			case <-lib.PlistCtx.Done():
				return

			default:
			}

			if pos < 0 {
				pos = (rows + i) - skipped
			}

			if v.LengthSeconds == 0 {
				skipped++
				continue
			}

			_, ok := plistIdMap[v.VideoID]
			if ok {
				skipped++
				continue
			}

			sref := lib.SearchResult{
				Type:    "video",
				Title:   v.Title,
				VideoID: v.VideoID,
			}

			plistTable.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)),
			)

			plistTable.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+lib.FormatDuration(v.LengthSeconds)).
				SetSelectable(false).
				SetAlign(tview.AlignRight),
			)

			plistIdMap[v.VideoID] = struct{}{}
		}

		if skipped == len(result.Videos) {
			InfoMessage("No more results", false)
			plistTable.SetSelectable(true, false)
			return
		}

		InfoMessage("Playlist entries loaded", false)

		if pos > 0 {
			plistTable.Select(pos, 0)
		}

		plistTable.ScrollToEnd()
		plistTable.SetSelectable(true, false)

		name, _ := VPage.GetFrontPage()
		if name == "playlistview" {
			App.SetFocus(plistTable)
		}
	})
}

// updatePlaylist returns updated playlist data from mpv.
func updatePlaylist() []EntryData {
	var data []EntryData

	liststr := lib.GetMPV().PlaylistData()
	if liststr == "" {
		ErrorMessage(fmt.Errorf("Could not fetch playlist"))
		return []EntryData{}
	}

	err := json.Unmarshal([]byte(liststr), &data)
	if err != nil {
		ErrorMessage(fmt.Errorf("Error while parsing playlist data"))
		return []EntryData{}
	}
	if len(data) == 0 {
		return []EntryData{}
	}

	for i := range data {
		urlData := lib.GetDataFromURL(data[i].Filename)
		if urlData != nil && urlData[0] == "" {
			data[i].Title = lib.GetMPV().PlaylistTitle(i)
			data[i].Author = "-"
			data[i].Duration = "-"
			continue
		}

		data[i].Title = urlData[0]
		data[i].Author = urlData[1]
		data[i].Duration = urlData[2]
	}

	return data
}

// plEnter either plays a file or, if a playlist entry has begun
// to move, selects the new position of the moving entry.
func plEnter() {
	row, _ := plistPopup.GetSelection()

	if moving {
		if row > prevrow {
			lib.GetMPV().PlaylistMove(prevrow, row+1)
		} else {
			lib.GetMPV().PlaylistMove(prevrow, row)
		}

		moving = false
		plistPopup.Select(row, 0)

		sendPlaylistEvent()
		return
	}

	lib.GetMPV().SetPlaylistPos(row)

	lib.GetMPV().Play()

	sendPlayerEvent()
	sendPlaylistEvent()
}

// plExit exits the playlist popup.
func plExit() {
	pcancel()
	exitFocus()
	plistPopup.Clear()
	popupStatus(false)
	ResultsList.SetSelectable(true, false)
}

// plDelete deletes an entry from the playlist
func plDelete() {
	rows := plistPopup.GetRowCount()
	row, _ := plistPopup.GetSelection()
	lib.GetMPV().PlaylistDelete(row)
	plistPopup.RemoveRow(row)

	switch {
	case row >= rows:
		plistPopup.Select(rows-1, 0)

	case row < rows && row > 0:
		plistPopup.Select(row-1, 0)

	case row == 0:
		plistPopup.Select(row, 0)
	}

	pos := lib.GetMPV().PlaylistPos()
	if pos == row {
		sendPlayerEvent()
	}

	sendPlaylistEvent()
}

// plMove begins to move the position of a playlist entry.
func plMove() {
	prevrow, _ = plistPopup.GetSelection()
	moving = true
	plistPopup.Select(prevrow, 0)
}

// plOpenReplace opens a playlist file, and replaces the current playlist.
func plOpenReplace(openpath string) {
	err := lib.GetMPV().LoadPlaylist(openpath, true)
	if err != nil {
		return
	}

	AddPlayer()

	App.QueueUpdateDraw(func() {
		playlistPopup()
	})
}

// plOpenAppend opens a playlist file, and appends to the current playlist.
func plOpenAppend(openpath string) {
	App.QueueUpdateDraw(func() {
		playlistPopup()
	})

	err := lib.GetMPV().LoadPlaylist(openpath, false)
	if err != nil {
		ErrorMessage(err)
		return
	}
}

// plSaveAs saves a playlist to a file.
func plSaveAs(savepath string) {
	var entries string

	savepath += ".m3u8"

	list := updatePlaylist()
	if len(list) == 0 {
		return
	}

	entries += "#EXTM3U\n\n"
	for i, data := range list {
		entries += "#EXTINF:," + data.Title + "\n"
		entries += data.Filename + "\n"

		if i != len(list)-1 {
			entries += "\n"
		}
	}

	err := ioutil.WriteFile(savepath, []byte(entries), 0664)
	if err != nil {
		ErrorMessage(fmt.Errorf("Unable to save playlist"))
		return
	}

	InfoMessage("Playlist saved in "+savepath, false)
}

// plFbExit exits the filebrowser.
func plFbExit() {
	exitFocus()
	popupStatus(false)
	Status.SwitchToPage("messages")
}

func exitFocus() {
	name, list := VPage.GetFrontPage()

	MPage.SwitchToPage("ui")
	VPage.SwitchToPage(name)

	App.SetFocus(list)
}

// sendPlaylistEvent sends a playlist event.
func sendPlaylistEvent() {
	select {
	case playlistEvent <- struct{}{}:
		return

	default:
	}
}

func playlistFocused() bool {
	return plistPopup.HasFocus()
}

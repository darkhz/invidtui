package ui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	Filename string `json:"filename"`
	Playing  bool   `json:"playing"`
	VideoID  string
	Title    string
	Author   string
	Duration string
	Type     string
}

var (
	// Playlist shows the playlist popup
	Playlist   *tview.Flex
	plistPopup *tview.Table

	plViewFlex   *tview.Flex
	plistTable   *tview.Table
	plTableTitle *tview.TextView
	plTableDesc  *tview.TextView
	plTableVBox  *tview.Box
	plPrevItem   tview.Primitive

	prevrow       int
	moving        bool
	plPrevPage    string
	playlistExit  chan struct{}
	playlistEvent chan struct{}

	plistIdMap    map[string]struct{}
	plistSaveLock *semaphore.Weighted
)

// SetupPlaylist sets up the playlist popup.
func SetupPlaylist() {
	setupViewPlaylist()
	setupViewChannel()
	setupPlaylistPopup()

	playlistExit = make(chan struct{}, 1)
	playlistEvent = make(chan struct{}, 100)

	plistIdMap = make(map[string]struct{})
	plistSaveLock = semaphore.NewWeighted(1)
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
			VPage.SwitchToPage(plPrevPage)
			App.SetFocus(plPrevItem)
		}

		key := event.Rune()
		switch {
		case key == '+' && plPrevPage != "dashboard":
			go Modify(true)

		case key == '_' && plPrevPage == "dashboard":
			go Modify(false)

		case key == ';':
			showLinkPopup()

		case key == 'C':
			ShowComments()
		}

		return event
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
		captureSendPlayerEvent(event)

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
			resizemodal()

		case 'M':
			plMove()
			resizemodal()

		case 'S':
			plExit()
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
			cell := plistPopup.GetCell(i, 0)
			if cell == nil {
				cell = tview.NewTableCell("")
				plistPopup.SetCell(i, 0, cell)
			}

			if i == row {
				cell.SetText(selector)
				continue
			}

			cell.SetText("")
		}
	})
}

// playlistPopup loads the playlist, and displays a popup
// with the playlist items.
func playlistPopup() {
	if lib.GetMPV().PlaylistCount() == 0 {
		InfoMessage("Playlist empty", false)
		return
	}

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

//gocyclo: ignore
// startPlaylist is the playlist update loop.
func startPlaylist() {
	var pos int
	var focused bool

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	tableData := func() []EntryData {
		var rows int

		App.QueueUpdateDraw(func() {
			rows = plistPopup.GetRowCount()
		})

		data := make([]EntryData, rows)

		for row := 0; row < rows; row++ {
			var videoId string
			var isPlaying bool

			var cell *tview.TableCell

			for col := 1; col <= 2; col++ {
				App.QueueUpdateDraw(func() {
					cell = plistPopup.GetCell(row, col)
				})
				if cell == nil {
					continue
				}

				ref := cell.GetReference()
				if ref == nil {
					continue
				}

				if info, ok := ref.(lib.SearchResult); ok {
					videoId = info.VideoID
				}

				if play, ok := ref.(bool); ok {
					isPlaying = play
				}
			}

			data[row] = EntryData{
				VideoID: videoId,
				Playing: isPlaying,
			}
		}

		return data
	}

	clearTableData := func() {
		App.QueueUpdateDraw(func() {
			plistPopup.Clear()
		})
	}

	// Taken from:
	// https://yourbasic.org/golang/compare-slices/
	checkDataChanged := func(a, b []EntryData) bool {
		if len(a) != len(b) {
			return false
		}

		for i, v := range a {
			if v.VideoID != b[i].VideoID || v.Playing != b[i].Playing {
				return false
			}
		}

		return true
	}

	update := func() {
		pldata := updatePlaylist()
		if len(pldata) == 0 && plistPopup.HasFocus() {
			App.QueueUpdateDraw(func() {
				exitFocus()
				plistPopup.Clear()
			})

			sendPlaylistExit()

			return
		}
		if checkDataChanged(pldata, tableData()) {
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

				info := lib.SearchResult{
					Title:   data.Title,
					Type:    "video",
					Author:  data.Author,
					VideoID: data.VideoID,
				}

				plistPopup.SetCell(i, 1, tview.NewTableCell("[blue::b]"+tview.Escape(data.Title)+marker).
					SetExpansion(1).
					SetMaxWidth(w/7).
					SetReference(info).
					SetSelectable(true).
					SetSelectedStyle(auxStyle),
				)

				plistPopup.SetCell(i, 2, tview.NewTableCell(" ").
					SetSelectable(false).
					SetReference(data.Playing),
				)

				plistPopup.SetCell(i, 3, tview.NewTableCell("[purple::b]"+tview.Escape(data.Author)).
					SetMaxWidth(w/5).
					SetSelectable(true).
					SetSelectedStyle(auxStyle),
				)

				plistPopup.SetCell(i, 4, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 5, tview.NewTableCell("[pink::b]"+tview.Escape(data.Type)).
					SetMaxWidth(w/5).
					SetSelectable(true).
					SetSelectedStyle(auxStyle),
				)

				plistPopup.SetCell(i, 6, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				plistPopup.SetCell(i, 7, tview.NewTableCell("[pink::b]"+data.Duration).
					SetSelectable(true).
					SetSelectedStyle(auxStyle),
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
		case <-playlistExit:
			clearTableData()
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
	go viewPlaylist(lib.SearchResult{}, false)
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

	ResultsList.SetSelectable(false, false)
	plPrevPage, plPrevItem = VPage.GetFrontPage()

	go viewPlaylist(info, newlist)
}

// viewPlaylist loads the playlist URL and shows the playlist contents.
func viewPlaylist(info lib.SearchResult, newlist bool) {
	var err error
	var cancel bool

	InfoMessage("Loading playlist entries", true)
	defer InfoMessage("Loaded playlist entries", false)

	result, err := lib.GetClient().Playlist(info.PlaylistID, plPrevPage == "dashboard")
	if err != nil {
		cancel = true
	}

	App.QueueUpdateDraw(func() {
		if cancel {
			ResultsList.SetSelectable(true, false)
			return
		}

		var skipped int

		pos := -1

		_, _, width, _ := ResultsList.GetRect()

		if newlist {
			plViewFlex.Clear()
			plistTable.Clear()
			plistIdMap = make(map[string]struct{})

			desc := strings.ReplaceAll(result.Description, "\n", " ")
			desclen := len(desc)

			header := tview.NewTextView()
			header.SetRegions(true)
			header.SetDynamicColors(true)
			header.SetBackgroundColor(tcell.ColorDefault)
			header.SetText(
				`[::b]Playlist[-:-:-] ["video"][darkcyan]Videos[""]`,
			)
			header.Highlight("video")

			plViewFlex.AddItem(header, 1, 0, false)
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
			plTableTitle.SetText("[::bu]" + result.Title)

			VPage.AddAndSwitchToPage("playlistview", plViewFlex, true)
		}

		rows := plistTable.GetRowCount()
		plistTable.SetSelectable(false, false)

		for i, v := range result.Videos {
			select {
			case <-lib.PlaylistCtx().Done():
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
				Type:       "video",
				Title:      v.Title,
				VideoID:    v.VideoID,
				AuthorID:   v.AuthorID,
				IndexID:    v.IndexID,
				PlaylistID: info.PlaylistID,
				Author:     result.Author,
			}

			plistTable.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(mainStyle),
			)

			plistTable.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+lib.FormatDuration(v.LengthSeconds)).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(auxStyle),
			)

			plistIdMap[v.VideoID] = struct{}{}
		}

		if skipped == len(result.Videos) {
			InfoMessage("No more results", false)
			plistTable.SetSelectable(true, false)
			return
		}

		InfoMessage("Playlist entries loaded", false)

		if pos >= 0 {
			plistTable.Select(pos, 0)

			if pos == 0 {
				plistTable.ScrollToBeginning()
			} else {
				plistTable.ScrollToEnd()
			}
		}

		plistTable.ScrollToEnd()
		plistTable.SetSelectable(true, false)
		ResultsList.SetSelectable(true, false)

		if pg, _ := MPage.GetFrontPage(); pg == "ui" {
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
		if urlData == nil {
			continue
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
				urlData.Set(udata, lib.GetMPV().PlaylistTitle(i))
				continue
			}

			if urlData.Get(udata) == "" {
				urlData.Set(udata, "-")
			}
		}

		data[i].Title = urlData.Get("title")
		data[i].Author = urlData.Get("author")
		data[i].Duration = urlData.Get("length")
		data[i].Type = urlData.Get("mediatype")
		data[i].VideoID = urlData.Get("id")
	}

	return data
}

// createPlaylistForm shows a form for playlist creation.
func createPlaylistForm() {
	formTitle := tview.NewTextView()
	formTitle.SetDynamicColors(true)
	formTitle.SetTextAlign(tview.AlignCenter)
	formTitle.SetText("[::bu]Create Playlist")
	formTitle.SetBackgroundColor(tcell.ColorDefault)

	createForm := tview.NewForm()
	createForm.SetBackgroundColor(tcell.ColorDefault)
	createForm.AddInputField("Name: ", "", 0, nil, nil)
	createForm.AddDropDown("Privacy: ", []string{"public", "unlisted", "private"}, -1, nil)
	createForm.AddButton("Create", func() {
		go func() {
			title := createForm.GetFormItem(0).(*tview.InputField).GetText()
			_, privacy := createForm.GetFormItem(1).(*tview.DropDown).GetCurrentOption()

			if title == "" || privacy == "" {
				ErrorMessage(fmt.Errorf("Cannot submit empty data"))
				return
			}

			App.QueueUpdateDraw(func() {
				exitFormPage("createplaylist")
			})

			InfoMessage("Creating playlist "+title, true)

			if err := lib.GetClient().CreatePlaylist(title, privacy); err != nil {
				ErrorMessage(err)
				return
			}

			InfoMessage("Created playlist "+title, false)

			loadPlaylists(false)
		}()
	})
	createForm.AddButton("Cancel", func() {
		exitFormPage("createplaylist")
	})
	createForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exitFormPage("createplaylist")
		}

		return event
	})

	createFormFlex := tview.NewFlex().
		AddItem(formTitle, 1, 0, false).
		AddItem(createForm, 0, 1, true).
		SetDirection(tview.FlexRow)

	dashPages.AddAndSwitchToPage("createplaylist", createFormFlex, true)

	App.SetFocus(createForm)
}

// editPlaylistForm shows a form for editing the playlist properties.
func editPlaylistForm() {
	var err error
	var info lib.SearchResult

	formTitle := tview.NewTextView()
	formTitle.SetDynamicColors(true)
	formTitle.SetTextAlign(tview.AlignCenter)
	formTitle.SetText("[::bu]Edit Playlist")
	formTitle.SetBackgroundColor(tcell.ColorDefault)

	editForm := tview.NewForm()
	editForm.SetBackgroundColor(tcell.ColorDefault)
	editForm.AddInputField("Name: ", info.Title, 0, nil, nil)
	editForm.AddInputField("Description: ", info.Description, 0, nil, nil)
	editForm.AddDropDown("Privacy: ", []string{"public", "unlisted", "private"}, -1, nil)
	editForm.AddButton("Edit", func() {
		go func() {
			title := editForm.GetFormItem(0).(*tview.InputField).GetText()
			description := editForm.GetFormItem(1).(*tview.InputField).GetText()
			_, privacy := editForm.GetFormItem(2).(*tview.DropDown).GetCurrentOption()

			if title == "" || privacy == "" {
				ErrorMessage(fmt.Errorf("Cannot submit empty data"))
				return
			}

			App.QueueUpdateDraw(func() {
				exitFormPage("editplaylist")
				info, err = getListReference()
			})
			if err != nil {
				ErrorMessage(err)
				return
			}

			InfoMessage("Editing playlist "+info.Title, true)

			err := lib.GetClient().EditPlaylist(info.PlaylistID, title, description, privacy)
			if err != nil {
				ErrorMessage(err)
				return
			}

			InfoMessage("Edited playlist", false)

			App.QueueUpdateDraw(func() {
				newInfo := info
				newInfo.Title = title
				newInfo.Description = description

				title = "[blue::b]" + tview.Escape(title)

				if err := modifyListReference(title, true, info, newInfo); err != nil {
					ErrorMessage(err)
				}
			})
		}()
	})
	editForm.AddButton("Cancel", func() {
		exitFormPage("editplaylist")
	})
	editForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			exitFormPage("editplaylist")
		}

		return event
	})

	editFormFlex := tview.NewFlex().
		AddItem(formTitle, 1, 0, false).
		AddItem(editForm, 0, 1, true).
		SetDirection(tview.FlexRow)

	dashPages.AddAndSwitchToPage("editplaylist", editFormFlex, true)

	App.SetFocus(editForm)
}

// modifyPlaylist modifies the availability of a playlist.
func modifyPlaylist(info lib.SearchResult, add bool) {
	var pg string

	if !lib.IsAuthInstance() {
		InfoMessage("Cannot modify playlist", false)
		return
	}

	App.QueueUpdateDraw(func() {
		pg, _ = VPage.GetFrontPage()
	})

	if !add && pg == "dashboard" {
		InfoMessage("Removing playlist "+info.Title, true)

		if err := lib.GetClient().RemovePlaylist(info.PlaylistID); err != nil {
			ErrorMessage(err)
			return
		}

		App.QueueUpdateDraw(func() {
			if err := modifyListReference("", false, info); err != nil {
				ErrorMessage(err)
			}
		})

		InfoMessage("Removed playlist "+info.Title, false)
	}
}

// modifyPlaylistVideo modifies the availability of the video in a playlist.
func modifyPlaylistVideo(info lib.SearchResult, add bool) {
	if !lib.IsAuthInstance() {
		InfoMessage("Cannot add video to playlist", false)
		return
	}

	if !add {
		InfoMessage("Removing video from "+info.Title, true)

		if err := lib.GetClient().RemovePlaylistVideo(info.PlaylistID, info.IndexID); err != nil {
			ErrorMessage(err)
			return
		}

		App.QueueUpdateDraw(func() {
			if err := modifyListReference("", false, info); err != nil {
				ErrorMessage(err)
			}
		})

		InfoMessage("Removed video from "+info.Title, false)

		return
	}

	InfoMessage("Retrieving playlists", true)

	playlists, err := lib.GetClient().AuthPlaylists()
	if err != nil {
		ErrorMessage(err)
		return
	}

	InfoMessage("Retrieved playlists", false)

	plistSelectTitle := tview.NewTextView()
	plistSelectTitle.SetDynamicColors(true)
	plistSelectTitle.SetTextAlign(tview.AlignCenter)
	plistSelectTitle.SetText("[white::bu]Select playlist")
	plistSelectTitle.SetBackgroundColor(tcell.ColorDefault)

	plistSelectPopup := tview.NewTable()
	plistSelectPopup.SetBorders(false)
	plistSelectPopup.SetSelectorWrap(true)
	plistSelectPopup.SetSelectable(true, false)
	plistSelectPopup.SetBackgroundColor(tcell.ColorDefault)
	plistSelectPopup.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		captureSendPlayerEvent(event)

		switch event.Key() {
		case tcell.KeyEscape:
			exitFocus()

		case tcell.KeyEnter:
			playlist, err := getListReference()
			exitFocus()
			if err != nil {
				ErrorMessage(err)
				return event
			}

			go func() {
				InfoMessage("Adding "+info.Title+" to "+playlist.Title, true)

				err := lib.GetClient().AddPlaylistVideo(playlist.PlaylistID, info.VideoID)
				if err != nil {
					ErrorMessage(err)
					return
				}

				InfoMessage("Added "+info.Title+" to "+playlist.Title, true)
			}()
		}

		return event
	})

	for i, p := range playlists {
		ref := lib.SearchResult{
			Type:       "playlist",
			Title:      p.Title,
			PlaylistID: p.PlaylistID,
			Author:     p.Author,
		}

		plistSelectPopup.SetCell(i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(p.Title)).
			SetExpansion(1).
			SetReference(ref).
			SetSelectedStyle(mainStyle),
		)

		plistSelectPopup.SetCell(i, 1, tview.NewTableCell("[pink]"+strconv.Itoa(p.VideoCount)+" videos").
			SetSelectable(true).
			SetAlign(tview.AlignRight).
			SetSelectedStyle(auxStyle),
		)
	}

	plistSelectFlex := tview.NewFlex().
		AddItem(plistSelectTitle, 1, 0, false).
		AddItem(plistSelectPopup, 10, 10, false).
		SetDirection(tview.FlexRow)

	MPage.AddAndSwitchToPage(
		"selectplaylist",
		statusmodal(plistSelectFlex, plistSelectPopup),
		true,
	).ShowPage("ui")

	App.SetFocus(plistSelectPopup)
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
	sendPlaylistExit()

	exitFocus()
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
	InfoMessage("Loading "+filepath.Base(openpath), true)

	err := lib.GetMPV().LoadPlaylist(openpath, true)
	if err != nil {
		return
	}

	AddPlayer()

	App.QueueUpdateDraw(func() {
		playlistPopup()
	})

	InfoMessage("Loaded "+filepath.Base(openpath), false)
}

// plOpenAppend opens a playlist file, and appends to the current playlist.
func plOpenAppend(openpath string) {
	InfoMessage("Loading "+filepath.Base(openpath), true)

	App.QueueUpdateDraw(func() {
		playlistPopup()
	})

	err := lib.GetMPV().LoadPlaylist(openpath, false)
	if err != nil {
		ErrorMessage(err)
		return
	}

	InfoMessage("Loaded "+filepath.Base(openpath), false)
}

// plSaveAs saves a playlist to a file.
func plSaveAs(savepath string) {
	var entries string
	var appendfile bool

	if !plistSaveLock.TryAcquire(1) {
		InfoMessage("Playlist save in progress", false)
		return
	}
	defer plistSaveLock.Release(1)

	savedstr := " saved in "
	cancelled := make(chan bool, 1)
	flags := os.O_CREATE | os.O_WRONLY

	if filepath.Ext(savepath) != ".m3u8" {
		savepath += ".m3u8"
	}

	list := updatePlaylist()
	if len(list) == 0 {
		return
	}

	if _, err := os.Stat(savepath); err == nil {
		go func() {
			App.QueueUpdateDraw(func() {
				SetInput(
					"Overwrite? [y/n/a]", 1,
					func(text string) {
						var exit, cancel bool

						switch text {
						case "y", "n", "a":
							if text == "a" {
								appendfile = true
								flags |= os.O_APPEND
							}

							if text == "y" {
								flags |= os.O_TRUNC
							}

							if text == "n" {
								cancel = true
							}

							exit = true
						}

						if exit {
							_, item := VPage.GetFrontPage()
							App.SetFocus(item)
							Status.SwitchToPage("messages")
							cancelled <- cancel
						}
					}, nil,
				)
			})
		}()

		if c := <-cancelled; c {
			return
		}
	}

	entries, err := plGetEntries(savepath, list, appendfile)
	if err != nil {
		ErrorMessage(err)
		return
	}

	file, err := os.OpenFile(savepath, flags, 0664)
	if err != nil {
		ErrorMessage(fmt.Errorf("Unable to open playlist"))
		return
	}

	_, err = file.WriteString(entries)
	if err != nil {
		ErrorMessage(fmt.Errorf("Unable to save playlist"))
		return
	}

	if appendfile {
		savedstr = " appended to "
	}

	InfoMessage("Playlist"+savedstr+savepath, false)
}

// plGetEntries generates playlist entries with a m3u8 header if entries are being
// overwritten to a playlist file. If appendfile is set, it reads the playlist
// file, filters out the duplicates from the playlist entry list, and appends entries
// to the already existing playlist entries from the playlist file.
func plGetEntries(savepath string, list []EntryData, appendfile bool) (string, error) {
	var skipped int
	var entries string
	var fileEntries map[string]struct{}

	if appendfile {
		fileEntries = make(map[string]struct{})

		plfile, err := os.Open(savepath)
		if err != nil {
			return "", fmt.Errorf("Unable to open playlist")
		}

		scanner := bufio.NewScanner(plfile)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := scanner.Text()

			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			fileEntries[line] = struct{}{}
		}
	}

	if !appendfile {
		entries += "#EXTM3U\n\n"
		entries += "# Autogenerated by invidtui. DO NOT EDIT.\n\n"
	} else {
		entries += "\n"
	}
	for i, data := range list {
		if appendfile && fileEntries != nil {
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
		return "", fmt.Errorf("No new items in playlist to append")
	}

	return entries, nil
}

// plFbExit exits the filebrowser.
func plFbExit() {
	exitFocus()
	popupStatus(false)
	Status.SwitchToPage("messages")
}

// exitFocus closes the popup.
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

// sendPlaylistExit sends a playlist exit event.
func sendPlaylistExit() {
	select {
	case playlistExit <- struct{}{}:
		return

	default:
	}
}

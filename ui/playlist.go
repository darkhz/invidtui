package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
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

	prevrow       int
	moving        bool
	pctx          context.Context
	pcancel       context.CancelFunc
	playlistEvent chan struct{}
)

// SetupPlaylist sets up the playlist popup.
func SetupPlaylist() {
	plistTitle := tview.NewTextView()
	plistTitle.SetDynamicColors(true)
	plistTitle.SetTextColor(tcell.ColorBlue)
	plistTitle.SetText("[white::bu]Playlist")
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

		case 'm':
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

	playlistEvent = make(chan struct{})
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

	Pages.AddAndSwitchToPage(
		"playlist",
		statusmodal(Playlist, plistPopup),
		true,
	).ShowPage("main")

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
				plistPopup.Clear()
				Pages.SwitchToPage("main")
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
	plistPopup.Clear()
	popupStatus(false)
	Pages.SwitchToPage("main")
	App.SetFocus(ResultsList)
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

	if !playing {
		AddPlayer()
	}

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
	popupStatus(false)
	Pages.SwitchToPage("main")
	Status.SwitchToPage("messages")
	App.SetFocus(ResultsList)
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

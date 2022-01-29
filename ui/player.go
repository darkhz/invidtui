package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
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
	// Player displays the media player.
	Player *tview.Flex

	playPopup     *tview.Table
	playerTitle   *tview.TextView
	playerDesc    *tview.TextView
	playerChan    chan bool
	playing       bool
	playerEvent   chan struct{}
	playlistEvent chan struct{}

	monitorId    int
	monitorMutex sync.RWMutex
	monitorMap   map[int]string

	addRateLimit *semaphore.Weighted
)

// SetupPlayer sets up a player view.
func SetupPlayer() {
	playerDesc = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	playerTitle = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	playPopup = tview.NewTable().
		SetBorders(false)

	Player = tview.NewFlex().
		AddItem(playerTitle, 1, 0, false).
		AddItem(playerDesc, 1, 0, false).
		SetDirection(tview.FlexRow)

	Player.SetBackgroundColor(tcell.ColorDefault)
	playPopup.SetBackgroundColor(tcell.ColorWhite)
	playerTitle.SetBackgroundColor(tcell.ColorDefault)
	playerDesc.SetBackgroundColor(tcell.ColorDefault)

	playerChan = make(chan bool)
	monitorMap = make(map[int]string)
	playerEvent = make(chan struct{})
	playlistEvent = make(chan struct{})

	addRateLimit = semaphore.NewWeighted(2)

	go StartPlayer()
	go monitorErrors()
}

// AddPlayer unhides the player view.
func AddPlayer() {
	if playing {
		return
	}

	playing = true
	SetPlayer(true)

	App.QueueUpdateDraw(func() {
		UIFlex.AddItem(Player, 2, 0, false)
	})
}

// RemovePlayer hides the player view and clears the playlist.
func RemovePlayer() {
	if !playing {
		return
	}

	playing = false
	SetPlayer(false)

	App.QueueUpdateDraw(func() {
		UIFlex.RemoveItem(Player)
	})

	lib.GetMPV().Stop()
	lib.GetMPV().PlaylistClear()

	monitorMutex.Lock()
	monitorMap = make(map[int]string)
	monitorMutex.Unlock()
}

// StartPlayer starts the player loop, which gets the information
// on the currently playing file from mpv, sets the media title and
// displays the relevant information along with a progress bar.
func StartPlayer() {
	var pctx context.Context
	var pcancel context.CancelFunc

	for {
		play, ok := <-playerChan
		if !ok {
			return
		}

		if pctx != nil && !play {
			pcancel()
		}

		if !play {
			continue
		}

		pctx, pcancel = context.WithCancel(context.Background())

		go startPlayer(pctx, pcancel)
	}
}

// startPlayer is the player update loop.
func startPlayer(ctx context.Context, cancel context.CancelFunc) {
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	update := func() {
		App.QueueUpdateDraw(func() {
			_, _, width, _ := playerDesc.GetRect()

			title, progressText, err := lib.GetProgress(width)
			if err != nil {
				cancel()
				return
			}

			playerDesc.SetText(progressText)
			playerTitle.SetText("[::b]" + tview.Escape(title))
		})
	}

	for {
		select {
		case <-ctx.Done():
			RemovePlayer()
			return

		case <-playerEvent:
			update()
			t.Reset(1 * time.Second)
			continue

		case <-t.C:
			update()
		}

	}
}

// StopPlayer finalizes the player before exit.
func StopPlayer() {
	SetPlayer(false)
	lib.GetMPV().MPVStop(true)
}

// SetPlayer sends a signal to StartPlayer on whether to
// start or stop the playback loop.
func SetPlayer(play bool) {
	playerChan <- play
}

// PlaySelected plays the current selection.
func PlaySelected(audio, current bool) {
	var media string

	title, id, err := getListReference()
	if err != nil {
		return
	}

	if audio {
		media = "audio"
	} else {
		media = "video"
	}

	monitorMutex.Lock()
	monitorId++
	monitorMap[monitorId] = title
	monitorMutex.Unlock()

	InfoMessage("Loading "+media+" for "+title, true)

	go func() {
		err := addRateLimit.Acquire(context.Background(), 1)
		if err != nil {
			return
		}
		defer addRateLimit.Release(1)

		video, err := lib.GetClient().Video(id)
		if err != nil {
			ErrorMessage(err)
			return
		}

		err = lib.LoadVideo(video, audio)
		if err != nil {
			ErrorMessage(err)
			return
		}

		InfoMessage("Added "+title, false)

		AddPlayer()

		if current {
			lib.GetMPV().PlaylistPlayLatest()
		}
	}()
}

// monitorErrors monitors for errors related to loading media
// from MPV.
func monitorErrors() {
	for {
		select {
		case val, ok := <-lib.MPVErrChan:
			if !ok {
				return
			}

			monitorMutex.Lock()

			title := monitorMap[val]
			delete(monitorMap, val)

			monitorMutex.Unlock()

			ErrorMessage(fmt.Errorf("Unable to play %s", title))
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
		if urlData == nil {
			continue
		}

		data[i].Title = urlData[0]
		data[i].Author = urlData[1]
		data[i].Duration = urlData[2]
	}

	return data
}

// playlistPopup loads the playlist, and displays a popup
// with the playlist items.
func playlistPopup() {
	var moving bool
	var prevrow int

	if lib.GetMPV().PlaylistCount() == 0 {
		InfoMessage("Playlist empty", false)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	enter := func() {
		row, _ := playPopup.GetSelection()

		if moving {
			if row > prevrow {
				lib.GetMPV().PlaylistMove(prevrow, row+1)
			} else {
				lib.GetMPV().PlaylistMove(prevrow, row)
			}

			moving = false
			playPopup.Select(row, 0)

			sendPlaylistEvent()
			return
		}

		lib.GetMPV().SetPlaylistPos(row)

		lib.GetMPV().Play()

		sendPlayerEvent()
		sendPlaylistEvent()
	}

	exit := func() {
		cancel()
		playPopup.Clear()
		popupStatus(false)
		Pages.SwitchToPage("main")
		App.SetFocus(ResultsList)
	}

	del := func() {
		rows := playPopup.GetRowCount()
		row, _ := playPopup.GetSelection()
		lib.GetMPV().PlaylistDelete(row)
		playPopup.RemoveRow(row)

		switch {
		case row >= rows:
			playPopup.Select(rows-1, 0)

		case row < rows && row > 0:
			playPopup.Select(row-1, 0)

		case row == 0:
			playPopup.Select(row, 0)
		}

		pos := lib.GetMPV().PlaylistPos()
		if pos == row {
			sendPlayerEvent()
		}

		sendPlaylistEvent()
	}

	move := func() {
		prevrow, _ = playPopup.GetSelection()
		moving = true
		playPopup.Select(prevrow, 0)
	}

	title := tview.NewTextView()
	title.SetDynamicColors(true)
	title.SetText("[::bu]Playlist")
	title.SetTextColor(tcell.ColorBlue)
	title.SetTextAlign(tview.AlignCenter)

	flex := tview.NewFlex().
		AddItem(title, 1, 0, false).
		AddItem(playPopup, 10, 10, false).
		SetDirection(tview.FlexRow)

	flex.SetBackgroundColor(tcell.ColorDefault)
	title.SetBackgroundColor(tcell.ColorDefault)
	playPopup.SetBackgroundColor(tcell.ColorDefault)

	playPopup.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			enter()

		case tcell.KeyEscape:
			exit()

		case tcell.KeyLeft, tcell.KeyRight:
			ResultsList.InputHandler()(event, nil)
		}

		switch event.Rune() {
		case 'd':
			del()

		case 'm':
			move()

		case '<', '>':
			ResultsList.InputHandler()(event, nil)
			sendPlaylistEvent()

		case ' ', 'l', 'S', 's':
			ResultsList.InputHandler()(event, nil)
		}

		return event
	})

	playPopup.SetSelectionChangedFunc(func(row, col int) {
		selector := ">"
		rows := playPopup.GetRowCount()

		if moving {
			selector = "M"
		}

		for i := 0; i < rows; i++ {
			cell := playPopup.GetCell(i, col)
			if cell == nil {
				cell = tview.NewTableCell("")
				playPopup.SetCell(i, col, cell)
			}

			if i == row {
				cell.SetText(selector)
				continue
			}

			cell.SetText("")
		}

		playPopup.SetSelectedStyle(tcell.Style{}.
			Background(tcell.ColorDefault).
			Foreground(tcell.ColorDefault))
	})

	go startPlaylist(ctx, flex)
}

// startPlaylist is the playlist update loop.
func startPlaylist(ctx context.Context, flex *tview.Flex) {
	var pos int
	var focused bool

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	update := func() {
		pldata := updatePlaylist()
		if len(pldata) == 0 {
			App.QueueUpdateDraw(func() {
				playPopup.Clear()
				Pages.SwitchToPage("main")
			})

			return
		}

		App.QueueUpdateDraw(func() {
			_, _, w, _ := playPopup.GetRect()
			playPopup.SetSelectable(false, false)

			for i, data := range pldata {
				var marker string

				if data.Playing {
					pos = i
					marker = " [white::b](playing)"
				}

				playPopup.SetCell(i, 1, tview.NewTableCell("[blue::b]"+tview.Escape(data.Title)+marker).
					SetExpansion(1).
					SetMaxWidth(w/5).
					SetSelectable(false),
				)

				playPopup.SetCell(i, 2, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				playPopup.SetCell(i, 3, tview.NewTableCell("[purple::b]"+tview.Escape(data.Author)).
					SetMaxWidth(w/5).
					SetSelectable(false),
				)

				playPopup.SetCell(i, 4, tview.NewTableCell(" ").
					SetSelectable(false),
				)

				playPopup.SetCell(i, 5, tview.NewTableCell("[pink::b]"+data.Duration).
					SetSelectable(false),
				)
			}

			playPopup.SetSelectable(true, false)

			if !focused {
				Pages.AddAndSwitchToPage(
					"playlist",
					statusmodal(flex, playPopup),
					true,
				).ShowPage("main")

				App.SetFocus(playPopup)
				playPopup.Select(pos, 0)
				focused = true
			}

			resizemodal()
		})
	}

	for {
		select {
		case <-ctx.Done():
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

// capturePlayerEvent maps custom keybindings to the relevant
// mpv commands. This function is attached to ResultsList's InputCapture.
func capturePlayerEvent(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyRight:
		lib.GetMPV().SeekForward()
		sendPlayerEvent()

	case tcell.KeyLeft:
		lib.GetMPV().SeekBackward()
		sendPlayerEvent()
	}

	switch event.Rune() {
	case 'a':
		PlaySelected(true, false)

	case 'v':
		PlaySelected(false, false)

	case 'A':
		PlaySelected(true, true)

	case 'V':
		PlaySelected(false, true)

	case 'S':
		SetPlayer(false)
		sendPlayerEvent()

	case 'p':
		playlistPopup()

	case 'l':
		lib.GetMPV().CycleLoop()
		sendPlayerEvent()

	case 's':
		lib.GetMPV().CycleShuffle()
		sendPlayerEvent()

	case '<':
		lib.GetMPV().Prev()
		sendPlayerEvent()

	case '>':
		lib.GetMPV().Next()
		sendPlayerEvent()

	case ' ':
		lib.GetMPV().CyclePaused()
		sendPlayerEvent()
	}
}

// sendPlayerEvent sends a player event.
func sendPlayerEvent() {
	select {
	case playerEvent <- struct{}{}:
		return

	default:
	}
}

// sendPlaylistEvent sends a playlist event.
func sendPlaylistEvent() {
	select {
	case playlistEvent <- struct{}{}:
		return

	default:
	}
}

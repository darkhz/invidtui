package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

var (
	// Player displays the media player.
	Player *tview.Flex

	playerTitle     *tview.TextView
	playerDesc      *tview.TextView
	playerChan      chan bool
	playing         bool
	playingLock     sync.Mutex
	playStateLock   sync.Mutex
	playHistoryLock sync.Mutex
	playerEvent     chan struct{}
	playerWidth     int
	playerStates    []string
	playHistory     []lib.SearchResult

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

	Player = tview.NewFlex().
		AddItem(playerTitle, 1, 0, false).
		AddItem(playerDesc, 1, 0, false).
		SetDirection(tview.FlexRow)

	Player.SetBackgroundColor(tcell.ColorDefault)
	playerTitle.SetBackgroundColor(tcell.ColorDefault)
	playerDesc.SetBackgroundColor(tcell.ColorDefault)

	playerChan = make(chan bool, 10)
	playerEvent = make(chan struct{}, 100)

	addRateLimit = semaphore.NewWeighted(2)

	go StartPlayer()
	go monitorErrors()
	go loadPlayerState()
	go loadPlayHistory()
}

// AddPlayer unhides the player view.
func AddPlayer() {
	if isPlaying() {
		return
	}

	SetPlayer(true)
	setPlaying(true)

	App.QueueUpdateDraw(func() {
		UIFlex.AddItem(Player, 2, 0, false)
		resizemodal()
	})
}

// RemovePlayer hides the player view and clears the playlist.
func RemovePlayer() {
	if !isPlaying() {
		return
	}

	SetPlayer(false)
	setPlaying(false)

	App.QueueUpdateDraw(func() {
		UIFlex.RemoveItem(Player)
		resizemodal()
	})

	lib.VideoCancel()
	lib.GetMPV().Stop()
	lib.GetMPV().PlaylistClear()
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
		var err error
		var width int
		var states []string
		var title, progressText string

		App.QueueUpdate(func() {
			_, _, width, _ = playerDesc.GetRect()
		})

		title, progressText, states, err = lib.GetProgress(width)
		if err != nil {
			cancel()
			return
		}

		playStateLock.Lock()
		playerStates = states
		playStateLock.Unlock()

		App.QueueUpdateDraw(func() {
			playerDesc.SetText(progressText)
			playerTitle.SetText("[::b]" + tview.Escape(title))
		})
	}

	for {
		select {
		case <-ctx.Done():
			RemovePlayer()
			playerDesc.SetText("")
			playerTitle.SetText("")
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
func StopPlayer(closeInstances bool) {
	SetPlayer(false)
	if !closeInstances {
		savePlayerState()
		savePlayHistory()
	}
	lib.GetMPV().MPVStop(true)
}

// SetPlayer sends a signal to StartPlayer on whether to
// start or stop the playback loop.
func SetPlayer(play bool) {
	select {
	case playerChan <- play:
		return

	default:
	}
}

// PlaySelected plays the current selection.
func PlaySelected(audio, current bool) {
	var media string

	info, err := getListReference()
	if err != nil {
		return
	}

	if audio {
		media = "audio"
	} else {
		media = "video"
	}

	if info.Type == "channel" {
		ErrorMessage(fmt.Errorf("Cannot play %s for channel type", media))
		return
	}

	InfoMessage("Loading "+media+" for "+info.Type+" "+info.Title, true)

	go func() {
		err := addRateLimit.Acquire(context.Background(), 1)
		if err != nil {
			return
		}
		defer addRateLimit.Release(1)

		lib.VideoNewCtx()

		switch info.Type {
		case "playlist":
			err = lib.LoadPlaylist(info.PlaylistID, audio)

		case "video":
			err = lib.LoadVideo(info.VideoID, audio)

		default:
			return
		}
		if err != nil {
			if err.Error() != "Rate-limit exceeded" {
				ErrorMessage(err)
			}

			if info.Type == "playlist" && err.Error() != "context canceled" {
				return
			}
		}

		go addToPlayHistory(info)

		InfoMessage("Added "+info.Title, false)

		if current && info.Type == "video" {
			lib.GetMPV().PlaylistPlayLatest()
		}
	}()
}

// isPlaying returns the currently playing status.
func isPlaying() bool {
	playingLock.Lock()
	defer playingLock.Unlock()

	return playing
}

// setPlaying sets the new playing status.
func setPlaying(status bool) {
	playingLock.Lock()
	defer playingLock.Unlock()

	playing = status
}

// loadPlayerState sets player volume, loop, mute and shuffle
// settings to its last known state.
func loadPlayerState() {
	var states []string

	state, err := lib.ConfigPath("state")
	if err != nil {
		return
	}

	stfile, err := os.Open(state)
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(stfile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		states = append(states, strings.Split(line, ",")...)
		break
	}

	if len(states) == 0 {
		return
	}

	for _, s := range states {
		if strings.Contains(s, "volume") {
			vol := strings.Split(s, " ")[1]
			lib.GetMPV().Set("volume", vol)
		}

		if strings.Contains(s, "loop") {
			lib.GetMPV().Set(s, "yes")
			continue
		}

		lib.GetMPV().Call("cycle", s)
	}
}

// savePlayerState saves the player volume, loop, mute and
// shuffle settings.
func savePlayerState() {
	playStateLock.Lock()
	defer playStateLock.Unlock()

	if len(playerStates) == 0 {
		return
	}

	statefile, err := lib.ConfigPath("state")
	if err != nil {
		return
	}

	states := strings.Join(playerStates, ",")

	err = ioutil.WriteFile(statefile, []byte(states+"\n"), 0664)
	if err != nil {
		return
	}
}

// loadPlayHistory loads the play history.
func loadPlayHistory() {
	playHistoryLock.Lock()
	defer playHistoryLock.Unlock()

	var hist []lib.SearchResult

	playhistory, err := lib.ConfigPath("playhistory.json")
	if err != nil {
		return
	}

	phfile, err := os.Open(playhistory)
	if err != nil {
		return
	}

	err = json.NewDecoder(phfile).Decode(&hist)
	if err != nil {
		return
	}

	playHistory = hist
}

// addToPlayHistory adds a loaded media item into the history.
func addToPlayHistory(info lib.SearchResult) {
	playHistoryLock.Lock()
	defer playHistoryLock.Unlock()

	// Taken from:
	// https://github.com/golang/go/wiki/SliceTricks#move-to-front-or-prepend-if-not-present-in-place-if-possible
	if len(playHistory) != 0 && playHistory[0] == info {
		return
	}

	prevInfo := info

	for i, phInfo := range playHistory {
		switch {
		case i == 0:
			playHistory[0] = info
			prevInfo = phInfo

		case phInfo == info:
			playHistory[i] = prevInfo
			return

		default:
			playHistory[i] = prevInfo
			prevInfo = phInfo
		}
	}

	playHistory = append(playHistory, prevInfo)
}

// showPlayHistory displays a popup with the play history.
func showPlayHistory() {
	playHistoryLock.Lock()
	defer playHistoryLock.Unlock()

	if len(playHistory) == 0 {
		return
	}

	if pg, _ := MPage.GetFrontPage(); pg == "playhistory" {
		return
	}

	App.QueueUpdateDraw(func() {
		var histTable *tview.Table

		histTable = tview.NewTable()
		histTable.SetSelectorWrap(true)
		histTable.SetSelectable(true, false)
		histTable.SetBackgroundColor(tcell.ColorDefault)
		histTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			var exit bool

			exitFunc := func() {
				exitFocus()
				Status.SwitchToPage("messages")
			}

			capturePlayerEvent(event)

			switch event.Key() {
			case tcell.KeyEscape:
				exitFunc()
			}

			switch event.Rune() {
			case '/':
				App.SetFocus(InputBox)
				Status.SwitchToPage("input")

			case 'i':
				exit = true
				ViewPlaylist(true, event.Modifiers() == tcell.ModAlt)

			case 'u':
				exit = true
				ViewChannel("video", true, event.Modifiers() == tcell.ModAlt)

			case 'U':
				exit = true
				ViewChannel("playlist", true, event.Modifiers() == tcell.ModAlt)
			}

			if exit {
				exitFunc()
			}

			return event
		})

		histTitle := tview.NewTextView()
		histTitle.SetDynamicColors(true)
		histTitle.SetText("[::bu]Play History")
		histTitle.SetTextAlign(tview.AlignCenter)
		histTitle.SetBackgroundColor(tcell.ColorDefault)

		histFlex := tview.NewFlex().
			AddItem(histTitle, 1, 0, false).
			AddItem(histTable, 10, 10, true).
			SetDirection(tview.FlexRow)

		fillTable := func(text string) {
			var row int
			text = strings.ToLower(text)

			histTable.Clear()

			for _, ph := range playHistory {
				if text != "" && strings.Index(strings.ToLower(ph.Title), text) < 0 {
					continue
				}

				histTable.SetCell(row, 0, tview.NewTableCell("[blue::b]"+ph.Title).
					SetExpansion(1).
					SetReference(ph).
					SetSelectedStyle(mainStyle),
				)

				histTable.SetCell(row, 1, tview.NewTableCell("").
					SetSelectable(false),
				)

				histTable.SetCell(row, 2, tview.NewTableCell("[purple::b]"+ph.Author).
					SetSelectedStyle(auxStyle),
				)

				histTable.SetCell(row, 3, tview.NewTableCell("").
					SetSelectable(false),
				)

				histTable.SetCell(row, 4, tview.NewTableCell("[pink]"+ph.Type).
					SetSelectedStyle(auxStyle),
				)

				row++
			}

			histTable.ScrollToBeginning()

			resizemodal()
		}

		chgfunc := func(text string) {
			fillTable(text)
		}
		ifunc := func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEscape, tcell.KeyEnter:
				App.SetFocus(histFlex)
			}

			return event
		}
		SetInput("Filter:", 0, nil, ifunc, chgfunc)

		fillTable("")

		MPage.AddAndSwitchToPage(
			"playhistory",
			statusmodal(histFlex, histTable),
			true,
		).ShowPage("ui")

		App.SetFocus(histFlex)
	})
}

// savePlayHistory saves the play history.
func savePlayHistory() {
	playHistoryLock.Lock()
	defer playHistoryLock.Unlock()

	phfile, err := lib.ConfigPath("playhistory.json")
	if err != nil {
		return
	}

	data, err := json.MarshalIndent(playHistory, "", " ")
	if err != nil {
		return
	}

	err = ioutil.WriteFile(phfile, data, 0664)
	if err != nil {
		return
	}
}

// monitorErrors monitors for errors related to loading media
// from MPV.
func monitorErrors() {
	for {
		select {
		case msg, ok := <-lib.MPVErrors:
			if !ok {
				return
			}

			ErrorMessage(fmt.Errorf("Unable to play %s", msg))

		case _, ok := <-lib.MPVFileLoaded:
			if !ok {
				return
			}

			AddPlayer()
		}
	}
}

// capturePlayerEvent maps custom keybindings to the relevant
// mpv commands. This function is attached to ResultsList's InputCapture.
func capturePlayerEvent(event *tcell.EventKey) {
	captureSendPlayerEvent(event)

	switch event.Key() {
	case tcell.KeyCtrlO:
		ShowFileBrowser("Open playlist:", plOpenReplace, plFbExit)

	case tcell.KeyCtrlH:
		go showPlayHistory()
	}

	switch event.Rune() {
	case 'a', 'A', 'v', 'V':
		playSelected(event.Rune())

	case 'p':
		playlistPopup()
	}
}

// captureSendPlayerEvent maps custom keybindings to
// the relevant mpv commands and sends a player event.
func captureSendPlayerEvent(event *tcell.EventKey) {
	var nokey, norune bool

	switch event.Key() {
	case tcell.KeyRight:
		lib.GetMPV().SeekForward()

	case tcell.KeyLeft:
		lib.GetMPV().SeekBackward()

	default:
		nokey = true
	}

	switch event.Rune() {
	case 'S':
		SetPlayer(false)

	case 'l':
		lib.GetMPV().CycleLoop()

	case 's':
		lib.GetMPV().CycleShuffle()

	case 'm':
		lib.GetMPV().CycleMute()

	case '=':
		lib.GetMPV().VolumeIncrease()

	case '-':
		lib.GetMPV().VolumeDecrease()

	case '<':
		lib.GetMPV().Prev()

	case '>':
		lib.GetMPV().Next()

	case ' ':
		lib.GetMPV().CyclePaused()

	default:
		norune = true
	}

	if !nokey || !norune {
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

func playSelected(r rune) {
	audio := r == 'a' || r == 'A'
	current := r == 'A' || r == 'V'

	PlaySelected(audio, current)

	table := getListTable()
	if table != nil {
		table.InputHandler()(
			tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone),
			nil,
		)
	}
}

func resizePlayer(width int) {
	if width == playerWidth {
		return
	}

	sendPlayerEvent()

	playerWidth = width
}

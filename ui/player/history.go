package player

import (
	"strings"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/ui/view"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// History describes the layout of the history popup
// and stores the entries.
type History struct {
	entries []cmd.PlayHistorySettings

	modal *app.Modal
	flex  *tview.Flex
	table *tview.Table
	input *tview.InputField
}

// loadHistory loads the saved play history.
func loadHistory() {
	player.history.entries = cmd.Settings.PlayHistory
}

// addToHistory adds a currently playing item to the history.
func addToHistory(data inv.SearchData) {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	info := cmd.PlayHistorySettings{
		Type:       data.Type,
		Title:      data.Title,
		Author:     data.Author,
		VideoID:    data.VideoID,
		PlaylistID: data.PlaylistID,
		AuthorID:   data.AuthorID,
	}

	if len(player.history.entries) != 0 && player.history.entries[0] == info {
		return
	}

	prevInfo := info

	for i, phInfo := range player.history.entries {
		switch {
		case i == 0:
			player.history.entries[0] = info
			prevInfo = phInfo

		case phInfo == info:
			player.history.entries[i] = prevInfo
			return

		default:
			player.history.entries[i] = prevInfo
			prevInfo = phInfo
		}
	}

	player.history.entries = append(player.history.entries, prevInfo)
	cmd.Settings.PlayHistory = player.history.entries
}

// showHistory shows a popup with the history entries.
func showHistory() {
	var history []cmd.PlayHistorySettings
	var property theme.ThemeProperty

	player.mutex.Lock()
	history = player.history.entries
	player.mutex.Unlock()

	if len(history) == 0 {
		return
	}

	if player.history.modal != nil {
		if player.history.modal.Open {
			return
		}

		goto Render
	}

	property = theme.ThemeProperty{
		Context: theme.ThemeContextHistory,
		Item:    theme.ThemePopupBackground,
	}

	player.history.table = theme.NewTable(property)
	player.history.table.SetSelectable(true, false)
	player.history.table.SetInputCapture(historyTableKeybindings)
	player.history.table.SetFocusFunc(func() {
		app.SetContextMenu(cmd.KeyContextHistory, player.history.table)
	})

	player.history.input = theme.NewInputField(property, "Filter:")
	player.history.input.SetChangedFunc(historyFilter)
	player.history.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			app.UI.SetFocus(player.history.table)
		}

		return event
	})

	player.history.flex = theme.NewFlex(property).
		SetDirection(tview.FlexRow).
		AddItem(player.history.table, 0, 10, true).
		AddItem(app.HorizontalLine(property), 1, 0, false).
		AddItem(player.history.input, 1, 0, false)

	player.history.modal = app.NewModal("player_history", "Previously played", player.history.flex, 40, 0, property)

Render:
	player.history.modal.Show(true)

	historyFilter("")
}

// historyTableKeybindings defines the keybindings for the history popup.
func historyTableKeybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event) {
	case cmd.KeyQuery:
		app.UI.SetFocus(player.history.input)

	case cmd.KeyChannelVideos:
		view.Channel.EventHandler("video", event.Modifiers() == tcell.ModAlt)

	case cmd.KeyChannelPlaylists:
		view.Channel.EventHandler("playlist", event.Modifiers() == tcell.ModAlt)

	case cmd.KeyClose:
		player.history.modal.Exit(false)
	}

	for _, k := range []cmd.Key{cmd.KeyChannelVideos, cmd.KeyChannelPlaylists} {
		if cmd.KeyOperation(event) == k {
			player.history.modal.Exit(false)
			app.UI.Status.SwitchToPage("messages")

			break
		}
	}

	return event
}

// historyFilter filters the history entries according to the provided text.
// This handler is attached to the history popup's input.
func historyFilter(text string) {
	var row int
	text = strings.ToLower(text)

	player.history.table.Clear()

	for _, ph := range player.history.entries {
		if text != "" && !strings.Contains(strings.ToLower(ph.Title), text) {
			continue
		}

		info := inv.SearchData{
			Type:       ph.Type,
			Title:      ph.Title,
			Author:     ph.Author,
			VideoID:    ph.VideoID,
			PlaylistID: ph.PlaylistID,
			AuthorID:   ph.AuthorID,
		}

		player.history.table.SetCell(row, 0, theme.NewTableCell(
			theme.ThemeContextHistory,
			theme.ThemeVideo,
			tview.Escape(ph.Title),
		).
			SetExpansion(1).
			SetReference(info),
		)

		player.history.table.SetCell(row, 1, theme.NewTableCell(
			theme.ThemeContextHistory,
			theme.ThemePopupBackground,
			"",
		).
			SetSelectable(true),
		)

		player.history.table.SetCell(row, 2, theme.NewTableCell(
			theme.ThemeContextHistory,
			theme.ThemeAuthor,
			tview.Escape(ph.Author),
		),
		)

		player.history.table.SetCell(row, 3, theme.NewTableCell(
			theme.ThemeContextHistory,
			theme.ThemePopupBackground,
			"",
		).
			SetSelectable(true),
		)

		player.history.table.SetCell(row, 4, theme.NewTableCell(
			theme.ThemeContextHistory,
			theme.ThemeMediaType,
			ph.Type,
		),
		)

		row++
	}

	player.history.table.ScrollToBeginning()

	app.ResizeModal()
}

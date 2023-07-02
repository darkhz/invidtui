package view

import (
	"fmt"

	"github.com/darkhz/invidtui/client"
	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/popup"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"golang.org/x/sync/semaphore"
)

// PlaylistView describes the layout of a playlist view.
type PlaylistView struct {
	init, removed bool
	page          int
	currentID     string

	table    *tview.Table
	infoView InfoView

	lock *semaphore.Weighted
}

// Playlist stores the playlist view properties.
var Playlist PlaylistView

// Name returns the name of the playlist view.
func (p *PlaylistView) Name() string {
	return "Playlist"
}

// Init initializes the playlist view.
func (p *PlaylistView) Init() bool {
	if p.init {
		return true
	}

	p.table = tview.NewTable()
	p.table.SetSelectorWrap(true)
	p.table.SetInputCapture(p.Keybindings)
	p.table.SetBackgroundColor(tcell.ColorDefault)
	p.table.SetFocusFunc(func() {
		app.SetContextMenu("Playlist", p.table)
	})

	p.infoView.Init(p.table)

	p.lock = semaphore.NewWeighted(1)

	p.init = true

	return true
}

// Exit closes the playlist view.
func (p *PlaylistView) Exit() bool {
	if p.removed {
		if v := PreviousView(); v != nil && v.Name() == Dashboard.Name() {
			Dashboard.Load(Dashboard.CurrentPage(), struct{}{})
		}
	}

	return true
}

// Tabs returns the tab layout for the playlist view.
func (p *PlaylistView) Tabs() app.Tab {
	return app.Tab{
		Title: "Playlist",
		Info: []app.TabInfo{
			{ID: "video", Title: "Videos"},
		},

		Selected: "video",
	}
}

// Primitive returns the primitive for the playlist view.
func (p *PlaylistView) Primitive() tview.Primitive {
	return p.infoView.flex
}

// View shows the playlist view.
func (p *PlaylistView) View() {
	if p.infoView.flex == nil || p.infoView.flex.GetItemCount() == 0 {
		return
	}

	SetView(&Playlist)
}

// EventHandler shows the playlist view for the currently selected playlist.
func (p *PlaylistView) EventHandler(justView bool, loadMore ...struct{}) {
	if justView {
		p.View()
		return
	}

	p.Init()

	p.removed = false

	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}
	if info.Type != "playlist" {
		app.ShowError(fmt.Errorf("View: Playlist: Cannot load from %s type", info.Type))
		return
	}

	go p.Load(info.PlaylistID, loadMore...)
}

// Load loads the playlist.
func (p *PlaylistView) Load(id string, loadMore ...struct{}) {
	var auth bool

	if !p.lock.TryAcquire(1) {
		app.ShowError(fmt.Errorf("View: Playlist: Still loading data"))
		return
	}
	defer p.lock.Release(1)

	if loadMore != nil {
		p.page++
	} else {
		p.page = 1
		p.currentID = id
	}

	app.ShowInfo("Loading Playlist results", true)

	result, err := inv.Playlist(p.currentID, auth, p.page)
	if err != nil {
		app.ShowError(err)
		return
	}
	if len(result.Videos) == 0 {
		app.ShowError(fmt.Errorf("View: Playlist: No more results"))
		return
	}

	app.UI.QueueUpdateDraw(func() {
		if loadMore == nil {
			p.infoView.Set(result.Title, result.Description)
			p.View()

			p.table.Clear()
		}

		p.renderPlaylist(result, p.currentID)
	})

	app.ShowInfo("Playlist loaded", false)
}

// Keybindings describes the keybindings for the playlist view.
func (p *PlaylistView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event) {
	case "LoadMore":
		go p.Load(p.currentID, struct{}{})

	case "Exit":
		CloseView()

	case "Add":
		if !Dashboard.IsFocused() {
			Dashboard.ModifyHandler(true)
		}

	case "Remove":
		if v := PreviousView(); v != nil && v.Name() == Dashboard.Name() {
			Dashboard.ModifyHandler(false)
		}

	case "Link":
		popup.ShowLink()

	case "Comments":
		Comments.Show()
	}

	return event
}

// renderPlaylist renders the playlist view.
func (p *PlaylistView) renderPlaylist(result inv.PlaylistData, id string) {
	var skipped int

	pos := -1
	rows := p.table.GetRowCount()
	plistIDMap := make(map[string]struct{})
	_, _, pageWidth, _ := app.UI.Pages.GetRect()

	previousView := PreviousView()
	prevDashboard := previousView != nil && previousView.Name() == Dashboard.Name()

	p.table.SetSelectable(false, false)

	for i, v := range result.Videos {
		select {
		case <-client.Ctx().Done():
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

		if !prevDashboard {
			_, ok := plistIDMap[v.VideoID]
			if ok {
				skipped++
				continue
			}

			plistIDMap[v.VideoID] = struct{}{}
		}

		sref := inv.SearchData{
			Type:       "video",
			Title:      v.Title,
			VideoID:    v.VideoID,
			AuthorID:   v.AuthorID,
			IndexID:    v.IndexID,
			PlaylistID: id,
			Author:     result.Author,
		}

		p.table.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
			SetExpansion(1).
			SetReference(sref).
			SetMaxWidth((pageWidth / 4)).
			SetSelectedStyle(app.UI.SelectedStyle),
		)

		p.table.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+utils.FormatDuration(v.LengthSeconds)).
			SetSelectable(true).
			SetAlign(tview.AlignRight).
			SetSelectedStyle(app.UI.ColumnStyle),
		)
	}

	if skipped == len(result.Videos) {
		app.ShowInfo("No more results", false)
		p.table.SetSelectable(true, false)

		return
	}

	app.ShowInfo("Playlist entries loaded", false)

	if pos >= 0 {
		p.table.Select(pos, 0)

		if pos == 0 {
			p.table.ScrollToBeginning()
		} else {
			p.table.ScrollToEnd()
		}
	}

	p.table.ScrollToEnd()
	p.table.SetSelectable(true, false)

	if pg, _ := app.UI.Pages.GetFrontPage(); pg == "ui" {
		app.UI.SetFocus(p.table)
	}
}

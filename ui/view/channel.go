package view

import (
	"fmt"
	"strconv"
	"sync"

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

// ChannelView describes the layout of a channel view.
type ChannelView struct {
	init                               bool
	searchText, currentID, currentType string
	continuation                       map[string]*ChannelContinuation

	infoView InfoView
	views    *tview.Pages
	tableMap map[string]*ChannelTable

	lock  *semaphore.Weighted
	mutex sync.Mutex
}

// ChannelTable describes the properties of a channel table.
type ChannelTable struct {
	loaded bool

	table *tview.Table
}

// ChannelContinuation describes the page/continuation data
// for the channel table.
type ChannelContinuation struct {
	loaded bool

	page         int
	continuation string
}

// Channel stores the channel view properties.
var Channel ChannelView

// Name returns the name of the channel view.
func (c *ChannelView) Name() string {
	return "Channel"
}

// Init initializes the channel view.
func (c *ChannelView) Init() bool {
	if c.init {
		return true
	}

	c.continuation = make(map[string]*ChannelContinuation)
	for _, i := range c.Tabs().Info {
		c.continuation[i.ID] = &ChannelContinuation{}
	}

	c.views = tview.NewPages()
	c.views.SetBackgroundColor(tcell.ColorDefault)

	c.queueWrite(func() {
		c.tableMap = make(map[string]*ChannelTable)

		for _, info := range c.Tabs().Info {
			table := tview.NewTable()
			table.SetSelectorWrap(true)
			table.SetSelectable(true, false)
			table.SetInputCapture(c.Keybindings)
			table.SetBackgroundColor(tcell.ColorDefault)
			table.SetSelectionChangedFunc(func(row, col int) {
				c.selectorHandler(table, row, col)
			})
			table.SetFocusFunc(func() {
				app.SetContextMenu("Channel", c.views)
			})

			c.tableMap[info.Title] = &ChannelTable{
				table: table,
			}

			c.views.AddPage(info.Title, table, true, false)
		}
	})

	c.infoView.Init(c.views)

	c.lock = semaphore.NewWeighted(1)

	c.init = true

	return true
}

// Exit closes the channel view
func (c *ChannelView) Exit() bool {
	return true
}

// Tabs describes the tab layout for the channel view.
func (c *ChannelView) Tabs() app.Tab {
	return app.Tab{
		Title: "Channel",
		Info: []app.TabInfo{
			{ID: "video", Title: "Videos"},
			{ID: "playlist", Title: "Playlists"},
			{ID: "search", Title: "Search"},
		},

		Selected: c.currentType,
	}
}

// Primitive returns the primitive for the channel view.
func (c *ChannelView) Primitive() tview.Primitive {
	return c.infoView.flex
}

// View shows the channel view.
func (c *ChannelView) View(pageType string) {
	if c.infoView.flex == nil || c.infoView.flex.GetItemCount() == 0 {
		return
	}

	SetView(&Channel)
	app.SelectTab(pageType)
	c.currentType = pageType

	for _, i := range c.Tabs().Info {
		if i.ID == pageType {
			c.views.SwitchToPage(i.Title)
			app.UI.SetFocus(c.getTableMap()[i.Title].table)

			break
		}
	}
}

// EventHandler shows the channel view according to the provided page type.
func (c *ChannelView) EventHandler(pageType string, justView bool) {
	if justView {
		c.View(pageType)
		return
	}

	c.Init()

	info, err := app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}

	c.queueWrite(func() {
		c.currentID = info.AuthorID
		for _, i := range c.Tabs().Info {
			ct := c.tableMap[i.Title]
			ct.table.Clear()
			ct.loaded = false
		}
	})

	go c.Load(pageType)
}

// Load loads the channel view according to the page type.
//
//gocyclo:ignore
func (c *ChannelView) Load(pageType string, loadMore ...struct{}) {
	var err error
	var author, description string

	if !c.lock.TryAcquire(1) {
		app.ShowError(fmt.Errorf("View: Channel: Still loading data"))
		return
	}
	defer c.lock.Release(1)

	if loadMore == nil {
		for _, i := range c.Tabs().Info {
			if i.ID != pageType {
				continue
			}

			if c.getTableMap()[i.Title].loaded {
				goto RenderView
			}

			break
		}
	}

	switch pageType {
	case "video":
		author, description, err = c.Videos(c.currentID, loadMore...)

	case "playlist":
		author, description, err = c.Playlists(c.currentID, loadMore...)

	case "search":
		err = nil
	}
	if err != nil {
		app.ShowError(err)
		return
	}

RenderView:
	app.UI.QueueUpdateDraw(func() {
		if GetCurrentView() != &Channel && author != "" {
			c.infoView.Set(author, description)
		}
		if GetCurrentView() != &Channel || app.GetCurrentTab() != pageType {
			c.View(pageType)
		}

		if pageType == "search" {
			if loadMore == nil {
				c.Query()
			} else {
				go c.Search("")
			}
		}
	})
}

// Videos loads the channel videos.
func (c *ChannelView) Videos(id string, loadMore ...struct{}) (string, string, error) {
	emptyVideoErr := fmt.Errorf("View: Channel: No more video results in channel")

	videoContinuation := c.continuation["video"]
	if loadMore == nil {
		videoContinuation.loaded = false
		videoContinuation.continuation = ""
	}
	if videoContinuation.loaded {
		app.ShowError(emptyVideoErr)

		return "", "", emptyVideoErr
	}

	app.ShowInfo("Loading Channel videos", true)

	result, err := inv.ChannelVideos(id, videoContinuation.continuation)
	if err != nil {
		app.ShowError(err)

		return "", "", err
	}
	if len(result.Videos) == 0 {
		app.ShowError(emptyVideoErr)

		return "", "", emptyVideoErr
	}
	if result.Continuation == "" {
		videoContinuation.loaded = true
	}

	videoContinuation.continuation = result.Continuation

	app.UI.QueueUpdateDraw(func() {
		var skipped int

		pos := -1
		_, _, pageWidth, _ := app.UI.Pages.GetRect()

		videoMap := c.getTableMap()["Videos"]
		videoTable := videoMap.table
		rows := videoTable.GetRowCount()

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

			sref := inv.SearchData{
				Type:     "video",
				Title:    v.Title,
				VideoID:  v.VideoID,
				AuthorID: result.ChannelID,
				Author:   result.Author,
			}

			videoTable.SetCell((rows+i)-skipped, 0, tview.NewTableCell("[blue::b]"+tview.Escape(v.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((pageWidth / 4)).
				SetSelectedStyle(app.UI.SelectedStyle),
			)

			videoTable.SetCell((rows+i)-skipped, 1, tview.NewTableCell("[pink]"+utils.FormatDuration(v.LengthSeconds)).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			)
		}

		c.queueWrite(func() {
			videoMap.loaded = true
		})
	})

	app.ShowInfo("Video entries loaded", false)

	return result.Author, result.Description, nil
}

// Playlists loads the channel playlists.
func (c *ChannelView) Playlists(id string, loadMore ...struct{}) (string, string, error) {
	emptyPlaylistErr := fmt.Errorf("View: Channel: No more playlist results in channel")

	playlistContinuation := c.continuation["playlist"]
	if loadMore == nil {
		playlistContinuation.loaded = false
		playlistContinuation.continuation = ""
	}
	if playlistContinuation.loaded {
		app.ShowError(emptyPlaylistErr)

		return "", "", emptyPlaylistErr
	}

	app.ShowInfo("Loading Channel playlists", true)

	result, err := inv.ChannelPlaylists(id, playlistContinuation.continuation)
	if err != nil {
		return "", "", err
	}
	if len(result.Playlists) == 0 {
		app.ShowError(emptyPlaylistErr)

		return "", "", emptyPlaylistErr
	}
	if result.Continuation == "" {
		playlistContinuation.loaded = true
	}

	playlistContinuation.continuation = result.Continuation

	app.UI.QueueUpdateDraw(func() {
		pos := -1
		_, _, pageWidth, _ := app.UI.Pages.GetRect()

		playlistMap := c.getTableMap()["Playlists"]
		playlistTable := playlistMap.table
		rows := playlistTable.GetRowCount()

		for i, p := range result.Playlists {
			select {
			case <-client.Ctx().Done():
				return

			default:
			}

			if pos < 0 {
				pos = (rows + i)
			}

			sref := inv.SearchData{
				Type:       "playlist",
				Title:      p.Title,
				PlaylistID: p.PlaylistID,
				AuthorID:   result.ChannelID,
				Author:     result.Author,
			}

			playlistTable.SetCell((rows + i), 0, tview.NewTableCell("[blue::b]"+tview.Escape(p.Title)).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((pageWidth / 4)).
				SetSelectedStyle(app.UI.SelectedStyle),
			)

			playlistTable.SetCell((rows + i), 1, tview.NewTableCell("[pink]"+strconv.Itoa(p.VideoCount)+" videos").
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			)
		}

		c.queueWrite(func() {
			playlistMap.loaded = true
		})
	})

	app.ShowInfo("Playlist entries loaded", false)

	return result.Author, result.Description, nil
}

// Search searches for the provided query within the channel.
func (c *ChannelView) Search(text string) {
	searchContinuation := c.continuation["search"]
	if text == "" {
		if c.searchText == "" {
			return
		}

		text = c.searchText
	} else {
		c.searchText = text
		searchContinuation.page = 0
	}

	app.ShowInfo("Fetching search results for "+tview.Escape(c.searchText), true)

	results, page, err := inv.ChannelSearch(
		c.currentID, c.searchText, searchContinuation.page,
	)
	if err != nil {
		app.ShowError(err)
		return
	}
	if results == nil {
		err := fmt.Errorf("View: Channel: No more search results")
		app.ShowError(err)

		return
	}

	searchContinuation.page = page

	app.UI.QueueUpdateDraw(func() {
		pos := -1

		searchMap := c.getTableMap()["Search"]

		searchTable := searchMap.table
		if text != "" {
			searchTable.Clear()
		}

		rows := searchTable.GetRowCount()
		_, _, width, _ := searchTable.GetRect()

		for i, result := range results {
			if pos < 0 {
				pos = rows + i
			}

			if result.Title == "" {
				result.Title = result.Author
				result.Author = ""
			}

			searchTable.SetCell(rows+i, 0, tview.NewTableCell("[blue::b]"+tview.Escape(result.Title)).
				SetExpansion(1).
				SetReference(result).
				SetMaxWidth((width / 4)).
				SetSelectedStyle(app.UI.SelectedStyle),
			)

			searchTable.SetCell(rows+i, 1, tview.NewTableCell(" ").
				SetSelectable(false).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			)

			searchTable.SetCell(rows+i, 2, tview.NewTableCell("[pink]"+result.Type).
				SetSelectable(true).
				SetAlign(tview.AlignRight).
				SetSelectedStyle(app.UI.ColumnStyle),
			)
		}

		searchTable.Select(pos, 0)
		searchTable.ScrollToEnd()

		searchTable.SetSelectable(true, false)

		c.queueWrite(func() {
			searchMap.loaded = true
		})
	})

	app.ShowInfo("Fetched search results", false)
}

// Query prompts for a query and searches the channel.
func (c *ChannelView) Query() {
	c.Init()

	label := "[::b]Search channel:"
	app.UI.Status.SetInput(label, 0, false, c.Search, c.inputFunc)
}

// Keybindings describes the keybindings for the channel view.
func (c *ChannelView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation("Channel", event) {
	case "Switch":
		tab := c.Tabs()
		tab.Selected = c.currentType
		c.currentType = app.SwitchTab(false, tab)

		c.View(c.currentType)
		go c.Load(c.currentType)

	case "LoadMore":
		go c.Load(c.currentType, struct{}{})

	case "Exit":
		CloseView()

	case "Query":
		c.currentType = "search"
		go c.Load(c.currentType)

	case "Playlist":
		go Playlist.EventHandler(event.Modifiers() == tcell.ModAlt)

	case "AddTo":
		Dashboard.ModifyHandler(true)

	case "Comments":
		Comments.Show()

	case "Link":
		popup.ShowVideoLink()
	}

	return event
}

// inputFunc describes the keybindings for the search input area.
func (c *ChannelView) inputFunc(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEnter:
		go c.Search(app.UI.Status.GetText())
		fallthrough

	case tcell.KeyEscape:
		app.UI.Status.Pages.SwitchToPage("messages")
		app.SetPrimaryFocus()
	}

	return e
}

// selectorHandler sets the attributes for the currently selected entry.
func (c *ChannelView) selectorHandler(table *tview.Table, row, col int) {
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

// getTableMap returns a map of tables within the channel view.
func (c *ChannelView) getTableMap() map[string]*ChannelTable {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.tableMap
}

// queueWrite executes the given function thread-safely.
func (c *ChannelView) queueWrite(write func()) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	write()
}

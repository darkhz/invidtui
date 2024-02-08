package view

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/darkhz/invidtui/client"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/popup"
	"github.com/darkhz/invidtui/ui/theme"
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

	property theme.ThemeProperty

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

	c.property = c.ThemeProperty()

	c.continuation = make(map[string]*ChannelContinuation)
	for _, i := range c.Tabs().Info {
		c.continuation[i.ID] = &ChannelContinuation{}
	}

	c.views = theme.NewPages(c.property)

	c.queueWrite(func() {
		c.tableMap = make(map[string]*ChannelTable)

		for _, info := range c.Tabs().Info {
			table := theme.NewTable(c.property)
			table.SetSelectable(true, false)
			table.SetInputCapture(c.Keybindings)
			table.SetFocusFunc(func() {
				app.SetContextMenu(keybinding.KeyContextChannel, c.views)
			})

			c.tableMap[info.Title] = &ChannelTable{
				table: table,
			}

			c.views.AddPage(info.Title, table, true, false)
		}
	})

	c.infoView.Init(c.views, c.property)

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
			{ID: "releases", Title: "Releases"},
			{ID: "search", Title: "Search"},
		},

		Selected: c.currentType,
	}
}

// Primitive returns the primitive for the channel view.
func (c *ChannelView) Primitive() tview.Primitive {
	return c.infoView.flex
}

// ThemeProperty returns the channel view's theme property.
func (c *ChannelView) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextChannel,
		Item:    theme.ThemeBackground,
	}
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
			app.SetPrimaryFocus()

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

	if err := c.lock.Acquire(context.Background(), 1); err != nil {
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

	case "releases":
		author, description, err = c.Releases(c.currentID, loadMore...)

	case "search":
		err = nil
	}
	if err != nil {
		app.ShowError(err)
		return
	}

RenderView:
	app.UI.QueueUpdateDraw(func() {
		if author != "" {
			c.infoView.Set(tview.Escape(author), tview.Escape(description))
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

			videoTable.SetCell((rows+i)-skipped, 0, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeVideo,
				tview.Escape(v.Title),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((pageWidth / 4)),
			)

			videoTable.SetCell((rows+i)-skipped, 1, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeTotalDuration,
				utils.FormatDuration(v.LengthSeconds),
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		app.SetTableSelector(videoTable, rows)

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

			sref := inv.SearchData{
				Type:       "playlist",
				Title:      p.Title,
				PlaylistID: p.PlaylistID,
				AuthorID:   result.ChannelID,
				Author:     result.Author,
			}

			playlistTable.SetCell((rows + i), 0, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemePlaylist,
				tview.Escape(p.Title),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((pageWidth / 4)),
			)

			playlistTable.SetCell((rows + i), 1, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeTotalVideos,
				strconv.FormatInt(p.VideoCount, 10)+" videos",
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		app.SetTableSelector(playlistTable, rows)

		c.queueWrite(func() {
			playlistMap.loaded = true
		})
	})

	app.ShowInfo("Playlist entries loaded", false)

	return result.Author, result.Description, nil
}

// Releases loads the channel releases.
func (c *ChannelView) Releases(id string, loadMore ...struct{}) (string, string, error) {
	noReleasesErr := fmt.Errorf("View: Channel: No more releases in channel")

	releaseContinuation := c.continuation["releases"]
	if loadMore == nil {
		releaseContinuation.loaded = false
		releaseContinuation.continuation = ""
	}
	if releaseContinuation.loaded {
		app.ShowError(noReleasesErr)
		return "", "", noReleasesErr
	}

	app.ShowInfo("Loading Channel releases", true)

	result, err := inv.ChannelReleases(id, releaseContinuation.continuation)
	if err != nil {
		return "", "", err
	}

	if len(result.Playlists) == 0 {
		app.ShowError(noReleasesErr)
		return "", "", noReleasesErr
	}

	if result.Continuation == "" {
		releaseContinuation.loaded = true
	}

	releaseContinuation.continuation = result.Continuation

	app.UI.QueueUpdateDraw(func() {
		_, _, pageWidth, _ := app.UI.Pages.GetRect()

		releaseMap := c.getTableMap()["Releases"]
		releaseTable := releaseMap.table
		rows := releaseTable.GetRowCount()

		for i, r := range result.Playlists {
			select {
			case <-client.Ctx().Done():
				return
			default:
			}

			sref := inv.SearchData{
				Type:       "playlist",
				Title:      r.Title,
				PlaylistID: r.PlaylistID,
				AuthorID:   result.ChannelID,
				Author:     result.Author,
			}

			releaseTable.SetCell(rows+i, 0, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemePlaylist,
				tview.Escape(r.Title),
			).
				SetExpansion(1).
				SetReference(sref).
				SetMaxWidth((pageWidth / 4)),
			)
		}

		app.SetTableSelector(releaseTable, rows)

		c.queueWrite(func() {
			releaseMap.loaded = true
		})
	})

	app.ShowInfo("Releases entries loaded", false)

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
		searchMap := c.getTableMap()["Search"]

		searchTable := searchMap.table
		if text != "" {
			searchTable.Clear()
		}

		rows := searchTable.GetRowCount()
		_, _, width, _ := searchTable.GetRect()

		for i, result := range results {
			if result.Title == "" {
				result.Title = result.Author
				result.Author = ""
			}

			searchTable.SetCell(rows+i, 0, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeVideo,
				tview.Escape(result.Title),
			).
				SetExpansion(1).
				SetReference(result).
				SetMaxWidth((width / 4)),
			)

			searchTable.SetCell(rows+i, 1, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeBackground,
				" ",
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)

			searchTable.SetCell(rows+i, 2, theme.NewTableCell(
				theme.ThemeContextChannel,
				theme.ThemeMediaType,
				result.Type,
			).
				SetSelectable(true).
				SetAlign(tview.AlignRight),
			)
		}

		app.SetTableSelector(searchTable, rows)
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

	label := "Search channel:"
	app.UI.Status.SetInput(label, 0, false, c.Search, c.inputFunc)
}

// Keybindings describes the keybindings for the channel view.
func (c *ChannelView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(event, keybinding.KeyContextComments) {
	case keybinding.KeySwitchTab:
		c.currentType = app.SwitchTab(false)

		client.Cancel()
		c.View(c.currentType)
		go c.Load(c.currentType)

	case keybinding.KeyLoadMore:
		go c.Load(c.currentType, struct{}{})

	case keybinding.KeyClose:
		CloseView()

	case keybinding.KeyQuery:
		c.currentType = "search"
		go c.Load(c.currentType)

	case keybinding.KeyPlaylist:
		go Playlist.EventHandler(event.Modifiers() == tcell.ModAlt, false)

	case keybinding.KeyAdd:
		Dashboard.ModifyHandler(true)

	case keybinding.KeyComments:
		Comments.Show()

	case keybinding.KeyLink:
		popup.ShowLink()
	}

	return event
}

// inputFunc describes the keybindings for the search input area.
func (c *ChannelView) inputFunc(e *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(e, keybinding.KeyContextCommon) {
	case keybinding.KeySelect:
		go c.Search(app.UI.Status.GetText())
		fallthrough

	case keybinding.KeyClose:
		app.UI.Status.Pages.SwitchToPage("messages")
		app.SetPrimaryFocus()
	}

	return e
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

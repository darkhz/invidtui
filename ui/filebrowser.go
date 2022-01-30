package ui

import (
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/sync/semaphore"
)

var (
	// FileBrowser shows the file browser.
	FileBrowser  *tview.Flex
	browserList  *tview.Table
	browserTitle *tview.TextView

	isHidden    bool
	hideLock    sync.Mutex
	prevDir     string
	currentPath string
	listLock    *semaphore.Weighted
)

// SetupFileBrowser sets up the file browser popup.
func SetupFileBrowser() {
	browserList = tview.NewTable()
	browserList.SetSelectorWrap(true)
	browserList.SetBackgroundColor(tcell.ColorDefault)

	browserTitle = tview.NewTextView()
	browserTitle.SetDynamicColors(true)
	browserTitle.SetTextAlign(tview.AlignCenter)
	browserTitle.SetBackgroundColor(tcell.ColorDefault)

	FileBrowser = tview.NewFlex().
		AddItem(browserTitle, 1, 0, false).
		AddItem(browserList, 10, 10, false).
		SetDirection(tview.FlexRow)

	browserList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			go changeDir("", false, true)

		case tcell.KeyRight:
			sel, _ := browserList.GetSelection()
			cell := browserList.GetCell(sel, 0)
			go changeDir(filepath.Clean(cell.Text), true, false)

		case tcell.KeyCtrlH:
			toggleHidden()
			go changeDir("", false, false)
		}

		return event
	})

	isHidden = true
	listLock = semaphore.NewWeighted(1)
}

// ShowFileBrowser shows the filebrowser popup and the input area.
func ShowFileBrowser(
	inputText string,
	dofunc func(text string), exitfunc func(),
) {
	ifunc := func(e *tcell.EventKey) *tcell.EventKey {
		switch e.Key() {
		case tcell.KeyUp, tcell.KeyDown:
			fallthrough

		case tcell.KeyRight, tcell.KeyLeft:
			fallthrough

		case tcell.KeyCtrlH:
			browserList.InputHandler()(e, nil)

		case tcell.KeyEnter:
			dofunc(InputBox.GetText())

		case tcell.KeyEscape:
			exitfunc()
		}

		return e
	}

	Pages.AddAndSwitchToPage(
		"filebrowser",
		statusmodal(FileBrowser, browserList),
		true,
	).ShowPage("main")

	SetInput(inputText, 0, dofunc, ifunc)

	go changeDir("", false, false)
}

// changeDir changes to a directory and lists its contents.
func changeDir(entry string, cdFwd bool, cdBack bool) {
	var testPath string

	if !listLock.TryAcquire(1) {
		return
	}
	defer listLock.Release(1)

	if currentPath == "" {
		var err error

		currentPath, err = homedir.Dir()
		if err != nil {
			ErrorMessage(err)
			return
		}
	}

	testPath = currentPath

	switch {
	case cdFwd:
		testPath = trimPath(testPath, false)
		testPath = filepath.Join(testPath, entry)

	case cdBack:
		prevDir = filepath.Base(testPath)
		testPath = trimPath(testPath, cdBack)
	}

	dlist, listed := dirList(filepath.FromSlash(testPath))
	if !listed {
		return
	}

	sort.Slice(dlist, func(i, j int) bool {
		if dlist[i].IsDir() != dlist[j].IsDir() {
			return dlist[i].IsDir()
		}

		return dlist[i].Name() < dlist[j].Name()
	})

	currentPath = testPath

	createDirList(dlist, cdBack)
}

// dirList lists a directory's contents.
func dirList(testPath string) ([]fs.FileInfo, bool) {
	var dlist []fs.FileInfo

	_, err := os.Lstat(testPath)
	if err != nil {
		return nil, false
	}

	file, err := os.Open(testPath)
	if err != nil {
		ErrorMessage(err)
		return nil, false
	}
	defer file.Close()

	list, err := ioutil.ReadDir(testPath)
	if err != nil {
		return nil, false
	}

	for _, entry := range list {
		if getHidden() && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		dlist = append(dlist, entry)
	}

	return dlist, true
}

// createDirList displays the contents of the directory on
// the filebrowser popup.
func createDirList(dlist []fs.FileInfo, cdBack bool) {
	App.QueueUpdateDraw(func() {
		var pos int

		browserList.SetSelectable(false, false)
		browserList.Clear()

		for row, entry := range dlist {
			name := entry.Name()
			if entry.IsDir() {
				if cdBack && name == prevDir {
					pos = row
				}

				name += "/"
			}

			browserList.SetCell(row, 0, tview.NewTableCell(name).
				SetTextColor(tcell.ColorBlue).
				SetAttributes(tcell.AttrBold))
		}

		browserTitle.SetText("[::bu]" + currentPath)

		browserList.ScrollToBeginning()
		browserList.SetSelectable(true, false)
		browserList.Select(pos, 0)
		resizemodal()
	})
}

// trimPath trims a given path and appends a path separator
// where appropriate.
func trimPath(testPath string, cdBack bool) string {
	testPath = filepath.Clean(testPath)

	if cdBack {
		testPath = filepath.Dir(testPath)
	}

	if testPath != "/" {
		testPath = testPath + "/"
	}

	return testPath
}

// getHidden checks if hidden files can be shown or not.
func getHidden() bool {
	hideLock.Lock()
	defer hideLock.Unlock()

	return isHidden
}

// toggleHidden toggles the hidden files mode.
func toggleHidden() {
	hideLock.Lock()
	defer hideLock.Unlock()

	isHidden = !isHidden
}

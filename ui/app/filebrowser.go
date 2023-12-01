package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/darkhz/invidtui/cmd"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/mitchellh/go-homedir"
	"golang.org/x/sync/semaphore"
)

// FileBrowser describes the layout of a file browser.
type FileBrowser struct {
	init, hidden, dironly        bool
	prevDir, currentPath, prompt string

	dofunc func(text string)

	modal *Modal
	flex  *tview.Flex
	table *tview.Table
	title *tview.TextView
	input *tview.InputField

	lock  *semaphore.Weighted
	mutex sync.Mutex
}

// FileBrowserOptions describes the file browser options.
type FileBrowserOptions struct {
	ShowDirOnly bool
	SetDir      string
}

// setup sets up the file browser.
func (f *FileBrowser) setup() {
	if f.init {
		return
	}

	f.title = tview.NewTextView()
	f.title.SetDynamicColors(true)
	f.title.SetTextAlign(tview.AlignCenter)
	f.title.SetBackgroundColor(tcell.ColorDefault)

	f.table = tview.NewTable()
	f.table.SetSelectorWrap(true)
	f.table.SetInputCapture(f.Keybindings)
	f.table.SetBackgroundColor(tcell.ColorDefault)
	f.table.SetSelectionChangedFunc(f.selectorHandler)

	f.input = tview.NewInputField()
	f.input.SetLabel("[::b]File: ")
	f.input.SetInputCapture(f.inputFunc)
	f.input.SetLabelColor(tcell.ColorWhite)
	f.input.SetBackgroundColor(tcell.ColorDefault)
	f.input.SetFieldBackgroundColor(tcell.ColorDefault)
	f.input.SetFocusFunc(func() {
		SetContextMenu("Files", f.input)
	})

	f.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f.title, 1, 0, false).
		AddItem(f.table, 0, 1, false).
		AddItem(HorizontalLine(), 1, 0, false).
		AddItem(f.input, 1, 0, true)

	f.modal = NewModal("Files", "Browse", f.flex, 60, 100)

	f.lock = semaphore.NewWeighted(1)

	f.hidden = true
	f.init = true
}

// Show displays the file browser.
func (f *FileBrowser) Show(prompt string, dofunc func(text string), options ...FileBrowserOptions) {
	f.setup()

	f.dofunc = dofunc
	f.dironly = false

	f.prompt = "[::b]" + prompt + " "
	f.input.SetLabel(f.prompt)

	if options != nil {
		f.dironly = options[0].ShowDirOnly
		if dir := options[0].SetDir; dir != "" {
			f.currentPath = dir
		}
	}

	f.modal.Show(false)
	go f.cd("", false, false)
}

// Hide hides the file browser.
func (f *FileBrowser) Hide() {
	f.modal.Exit(false)
}

// Query displays a confirmation message within the file browser.
func (f *FileBrowser) Query(
	prompt string,
	validate func(text string, reply chan string),
	max ...int,
) string {
	reply := make(chan string)

	var acceptFunc func(text string, ch rune) bool
	if max != nil {
		acceptFunc = tview.InputFieldMaxLength(max[0])
	}

	UI.QueueUpdateDraw(func() {
		f.input.SetText("")
		f.input.SetLabel(prompt + " ")
		f.input.SetAcceptanceFunc(acceptFunc)
		f.input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEnter:
				go validate(f.input.GetText(), reply)

			case tcell.KeyEscape:
				select {
				case reply <- "":

				default:
				}
			}

			return event
		})
	})

	response := <-reply

	UI.QueueUpdateDraw(func() {
		row, _ := f.table.GetSelection()
		f.table.Select(row, 0)

		f.input.SetLabel(f.prompt)
		f.input.SetInputCapture(f.inputFunc)
	})

	return response
}

// SaveFile saves the generated entries into a file.
func (f *FileBrowser) SaveFile(
	file string,
	entriesFunc func(flags int, appendToFile bool) (string, int, error),
) {
	flags, appendToFile, confirm, exist := f.confirmOverwrite(file)
	if exist && !confirm {
		return
	}

	f.Hide()

	entries, newflags, err := entriesFunc(flags, appendToFile)
	if err != nil {
		ShowError(err)
		return
	}

	saveFile, err := os.OpenFile(file, newflags, 0664)
	if err != nil {
		ShowError(fmt.Errorf("FileBrowser: Unable to open file"))
		return
	}

	_, err = saveFile.WriteString(entries)
	if err != nil {
		ShowError(fmt.Errorf("FileBrowser: Unable to save file"))
		return
	}

	message := " saved in "
	if appendToFile {
		message = " appended to "
	}

	ShowInfo("Contents"+message+file, false)
}

// Keybindings define the keybindings for the file browser.
func (f *FileBrowser) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event, "Files") {
	case cmd.KeyFilebrowserDirForward:
		sel, _ := f.table.GetSelection()
		cell := f.table.GetCell(sel, 0)
		go f.cd(filepath.Clean(cell.Text), true, false)

	case cmd.KeyFilebrowserDirBack:
		go f.cd("", false, true)

	case cmd.KeyFilebrowserToggleHidden:
		f.hiddenStatus(struct{}{})
		go f.cd("", false, false)

	case cmd.KeyFilebrowserNewFolder:
		go f.newFolder()

	case cmd.KeyFilebrowserRename:
		go f.renameItem()
		return nil
	}

	return event
}

// inputFunc defines the keybindings for the file browser's inputbox.
func (f *FileBrowser) inputFunc(e *tcell.EventKey) *tcell.EventKey {
	var toggle bool

	switch cmd.KeyOperation(e, "Files") {
	case cmd.KeyFilebrowserToggleHidden:
		toggle = true

	case cmd.KeyFilebrowserSelect:
		text := f.input.GetText()
		if text == "" {
			goto Event
		}

		go f.dofunc(filepath.Join(f.currentPath, text))

	case cmd.KeyClose:
		f.modal.Exit(false)
		goto Event
	}

	f.table.InputHandler()(tcell.NewEventKey(e.Key(), ' ', e.Modifiers()), nil)

Event:
	if toggle {
		e = nil
	}

	return e
}

// selectorHandler checks whether the selected item is a file,
// and automatically appends the filename to the input box.
func (f *FileBrowser) selectorHandler(row, col int) {
	sel, _ := f.table.GetSelection()
	cell := f.table.GetCell(sel, 0)

	if !f.dironly && strings.Contains(cell.Text, string(os.PathSeparator)) {
		f.input.SetText("")
		return
	}

	f.input.SetText(cell.Text)
}

// cd changes the directory.
func (f *FileBrowser) cd(entry string, cdFwd bool, cdBack bool) {
	var testPath string

	if !f.lock.TryAcquire(1) {
		return
	}
	defer f.lock.Release(1)

	if f.currentPath == "" {
		var err error

		f.currentPath, err = homedir.Dir()
		if err != nil {
			ShowError(err)
			return
		}
	}

	testPath = f.currentPath

	switch {
	case cdFwd:
		testPath = utils.TrimPath(testPath, false)
		testPath = filepath.Join(testPath, entry)

	case cdBack:
		f.prevDir = filepath.Base(testPath)
		testPath = utils.TrimPath(testPath, cdBack)
	}

	dlist, listed := f.list(filepath.FromSlash(testPath))
	if !listed {
		return
	}

	sort.Slice(dlist, func(i, j int) bool {
		if dlist[i].IsDir() != dlist[j].IsDir() {
			return dlist[i].IsDir()
		}

		return dlist[i].Name() < dlist[j].Name()
	})

	f.currentPath = testPath

	f.render(dlist, cdBack)
}

// list lists a directory's contents.
func (f *FileBrowser) list(testPath string) ([]fs.DirEntry, bool) {
	var dlist []fs.DirEntry

	stat, err := os.Lstat(testPath)
	if err != nil {
		return nil, false
	}
	if !stat.IsDir() {
		return nil, false
	}

	file, err := os.Open(testPath)
	if err != nil {
		ShowError(err)
		return nil, false
	}
	defer file.Close()

	list, err := os.ReadDir(testPath)
	if err != nil {
		if err.Error() != "EOF" {
			ShowError(err)
		}

		return nil, false
	}

	for _, entry := range list {
		if f.hiddenStatus() && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if f.dironly && !entry.IsDir() {
			continue
		}

		dlist = append(dlist, entry)
	}

	return dlist, true
}

// render displays the contents of the directory on
// the filebrowser popup.
func (f *FileBrowser) render(dlist []fs.DirEntry, cdBack bool) {
	UI.QueueUpdateDraw(func() {
		var pos int

		f.table.Clear()
		f.table.SetSelectable(false, false)

		for row, entry := range dlist {
			var color tcell.Color

			name := entry.Name()
			if entry.IsDir() {
				if cdBack && name == f.prevDir {
					pos = row
				}

				name += string(os.PathSeparator)
				color = tcell.ColorBlue
			} else {
				color = tcell.ColorWhite
			}

			f.table.SetCell(row, 0, tview.NewTableCell(name).
				SetTextColor(color))
		}

		f.table.ScrollToBeginning()
		f.table.SetSelectable(true, false)
		f.table.Select(pos, 0)

		f.title.SetText("[::bu]" + f.currentPath)

		ResizeModal()
	})
}

// confirmOverwrite displays an overwrite confirmation message
// within the file browser. This is triggered if the selected file
// in the file browser already exists and has entries in it.
func (f *FileBrowser) confirmOverwrite(file string) (int, bool, bool, bool) {
	var appendToFile bool

	flags := os.O_CREATE | os.O_WRONLY

	if _, err := os.Stat(file); err != nil {
		return flags, false, false, false
	}

	reply := f.Query("Overwrite file (y/n/a)?", f.validate, 1)
	switch reply {
	case "y":
		flags |= os.O_TRUNC

	case "a":
		flags |= os.O_APPEND
		appendToFile = true

	case "n":
		break

	default:
		reply = ""
	}

	return flags, appendToFile, reply != "", true
}

// newFolder prompts for a name and creates a directory.
func (f *FileBrowser) newFolder() {
	name := f.Query("[::b]Folder name:", f.validate)
	if name == "" {
		return
	}

	if err := os.Mkdir(filepath.Join(f.currentPath, name), os.ModePerm); err != nil {
		ShowError(fmt.Errorf("Filebrowser: Could not create directory %s", name))
		return
	}

	go f.cd("", false, false)
}

// renameItem prompts for a name and renames the currently selected entry.
func (f *FileBrowser) renameItem() {
	name := f.Query("[::b]Rename to:", f.validate)
	if name == "" {
		return
	}

	row, _ := f.table.GetSelection()
	oldname := f.table.GetCell(row, 0).Text

	if err := os.Rename(filepath.Join(f.currentPath, oldname), filepath.Join(f.currentPath, name)); err != nil {
		ShowError(fmt.Errorf("Filebrowser: Could not rename %s to %s", oldname, name))
		return
	}

	go f.cd("", false, false)
}

// validate validates the overwrite confirmation reply.
func (f *FileBrowser) validate(text string, reply chan string) {
	for _, option := range []string{"y", "n", "a"} {
		if text == option {
			select {
			case reply <- text:

			default:
			}

			break
		}
	}
}

// hiddenStatus returns whether hidden files are displayed or not.
func (f *FileBrowser) hiddenStatus(toggle ...struct{}) bool {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if toggle != nil {
		f.hidden = !f.hidden
	}

	return f.hidden
}

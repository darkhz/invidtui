package view

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/darkhz/invidtui/cmd"
	inv "github.com/darkhz/invidtui/invidious"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/schollz/progressbar/v3"
)

// DownloadsView describes the layout of a downloads view.
type DownloadsView struct {
	init          bool
	modal         *app.Modal
	options, view *tview.Table
}

// DownloadProgress describes the layout of a progress indicator.
type DownloadProgress struct {
	desc, progress *tview.TableCell
	bar            *progressbar.ProgressBar

	cancelFunc context.CancelFunc
}

// DownloadData describes the information for the downloading item.
type DownloadData struct {
	id, title, dtype string

	format inv.VideoFormat
}

// Downloads stores the downloads view properties.
var Downloads DownloadsView

// Name returns the name of the downloads view.
func (d *DownloadsView) Name() string {
	return "Downloads"
}

// Init initializes the downloads view.
func (d *DownloadsView) Init() bool {
	if d.init {
		return true
	}

	d.options = tview.NewTable()
	d.options.SetSelectorWrap(true)
	d.options.SetSelectable(true, false)
	d.options.SetBackgroundColor(tcell.ColorDefault)
	d.options.SetInputCapture(d.OptionKeybindings)
	d.options.SetFocusFunc(func() {
		app.SetContextMenu(cmd.KeyContextDownloads, d.options)
	})

	d.view = tview.NewTable()
	d.view.SetBorder(true)
	d.view.SetSelectorWrap(true)
	d.view.SetTitle("Download List")
	d.view.SetSelectable(true, false)
	d.view.SetTitleAlign(tview.AlignLeft)
	d.view.SetBackgroundColor(tcell.ColorDefault)
	d.view.SetInputCapture(d.Keybindings)
	d.view.SetFocusFunc(func() {
		app.SetContextMenu(cmd.KeyContextDownloads, d.view)
	})

	d.modal = app.NewModal("downloads", "Select Download Option", d.options, 40, 60)

	d.init = true

	return true
}

// Exit closes the downloads view.
func (d *DownloadsView) Exit() bool {
	return true
}

// Tabs describes the tab layout for the downloads view.
func (d *DownloadsView) Tabs() app.Tab {
	return app.Tab{}
}

// Primitive returns the primitive for the downloads view.
func (d *DownloadsView) Primitive() tview.Primitive {
	return d.view
}

// View shows the download view.
func (d *DownloadsView) View() {
	if d.view == nil {
		return
	}

	SetView(&Downloads)
}

// ShowOptions shows a list of download options for the selected video.
func (d *DownloadsView) ShowOptions(data ...inv.SearchData) {
	var err error
	var info inv.SearchData

	if data != nil {
		info = data[0]
		goto Options
	}

	info, err = app.FocusedTableReference()
	if err != nil {
		app.ShowError(err)
		return
	}
	if info.Type != "video" {
		return
	}

	if cmd.GetOptionValue("download-dir") == "" {
		d.SetDir(info)
		return
	}

Options:
	d.Init()

	go d.LoadOptions(info.VideoID, info.Title)
}

// SetDir sets the download directory.
func (d *DownloadsView) SetDir(info ...inv.SearchData) {
	app.UI.FileBrowser.Show("Download file to:", func(name string) {
		if stat, err := os.Stat(name); err != nil || !stat.IsDir() {
			if err == nil {
				err = fmt.Errorf("View: Downloads: Selected item is not a directory")
			}

			app.ShowError(err)
			return
		}

		cmd.SetOptionValue("download-dir", name)

		app.UI.QueueUpdateDraw(func() {
			app.UI.FileBrowser.Hide()

			if info != nil {
				d.ShowOptions(info[0])
			}
		})
	}, app.FileBrowserOptions{
		ShowDirOnly: true,
		SetDir:      cmd.GetOptionValue("download-dir"),
	})
}

// LoadOptions loads the download options for the selected video.
func (d *DownloadsView) LoadOptions(id, title string) {
	app.ShowInfo("Getting download options", true)

	video, err := inv.Video(id)
	if err != nil {
		app.ShowError(err)
		return
	}
	if video.LiveNow {
		app.ShowError(fmt.Errorf("View: Downloads: Cannot download live video"))
		return
	}

	app.ShowInfo("Showing download options", false)

	go app.UI.QueueUpdateDraw(func() {
		d.renderOptions(video)
		d.modal.Show(false)
	})
}

// TransferVideo starts the download for the selected video.
func (d *DownloadsView) TransferVideo(id, itag, filename string) {
	var progress DownloadProgress

	app.ShowInfo("Starting download for video "+tview.Escape(filename), false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, file, err := inv.DownloadParams(ctx, id, itag, filename)
	if err != nil {
		app.ShowError(err)
		return
	}
	defer res.Body.Close()
	defer file.Close()

	progress.renderBar(filename, res.ContentLength, cancel, true)
	defer app.UI.QueueUpdateDraw(func() {
		progress.remove()
	})

	_, err = io.Copy(io.MultiWriter(file, progress.bar), res.Body)
	if err != nil {
		app.ShowError(err)
	}
}

// TransferPlaylist starts the download for the selected playlist.
func (d *DownloadsView) TransferPlaylist(id, file string, flags int, auth, appendToFile bool) (string, int, error) {
	var progress DownloadProgress

	filename := filepath.Base(file)

	d.Init()

	app.ShowInfo("Starting download for playlist '"+tview.Escape(filename)+"'", false)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	progress.renderBar(filename, 0, cancel, false)
	defer app.UI.QueueUpdateDraw(func() {
		progress.remove()
	})

	videos, err := inv.PlaylistVideos(ctx, id, auth, func(stats [3]int64) {
		if progress.bar.GetMax() <= 0 {
			progress.bar.ChangeMax64(stats[2])
			progress.bar.Reset()
		}

		progress.bar.Set64(stats[1])
	})
	if err != nil {
		return "", flags, err
	}

	return inv.GeneratePlaylist(file, videos, flags, appendToFile)
}

// OptionKeybindings describes the keybindings for the download options popup.
func (d *DownloadsView) OptionKeybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event, cmd.KeyContextDownloads) {
	case cmd.KeyDownloadChangeDir:
		d.SetDir()

	case cmd.KeyDownloadOptionSelect:
		row, _ := d.options.GetSelection()
		cell := d.options.GetCell(row, 0)

		if data, ok := cell.GetReference().(DownloadData); ok {
			filename := data.title + "." + data.format.Container
			go d.TransferVideo(data.id, data.format.Itag, filename)
		}

		fallthrough

	case cmd.KeyClose:
		d.modal.Exit(false)
	}

	return event
}

// Keybindings describes the keybindings for the downloads view.
func (d *DownloadsView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch cmd.KeyOperation(event, cmd.KeyContextDownloads) {
	case cmd.KeyDownloadCancel:
		row, _ := Downloads.view.GetSelection()

		cell := Downloads.view.GetCell(row, 0)
		if progress, ok := cell.GetReference().(*DownloadProgress); ok {
			progress.cancelFunc()
		}

	case cmd.KeyClose:
		CloseView()
	}

	return event
}

// renderOptions render the download options popup.
func (d *DownloadsView) renderOptions(video inv.VideoData) {
	var skipped, width int

	d.options.Clear()

	for i, formatData := range [][]inv.VideoFormat{
		video.FormatStreams,
		video.AdaptiveFormats,
	} {
		rows := d.options.GetRowCount()

		for row, format := range formatData {
			var err error
			var minfo, size string
			var optionInfo []string

			if i != 0 {
				minfo = " only"
			} else {
				minfo = " + audio"
				clen := utils.GetDataFromURL(format.URL).Get("clen")
				format.ContentLength, err = strconv.ParseInt(clen, 10, 64)
				if err != nil {
					format.ContentLength = 0
				}
			}

			mtype := strings.Split(strings.Split(format.Type, ";")[0], "/")
			if (mtype[0] == "audio" && (format.Container == "" || format.Encoding == "")) ||
				(mtype[0] == "video" && format.FPS == 0) {
				skipped++
				continue
			}

			if format.ContentLength == 0 {
				size = "-"
			} else {
				size = strconv.FormatFloat(float64(format.ContentLength)/1024/1024, 'f', 2, 64)
			}

			optionInfo = []string{
				"[red::b]" + mtype[0] + minfo + "[-:-:-]",
				"[blue::b]" + size + " MB[-:-:-]",
				"[purple::b]" + format.Container + "/" + format.Encoding + "[-:-:-]",
			}
			if mtype[0] != "audio" {
				optionInfo = append(optionInfo, []string{
					"[green::b]" + format.Resolution + "[-:-:-]",
					"[yellow::b]" + strconv.Itoa(format.FPS) + "fps[-:-:-]",
				}...)
			} else {
				optionInfo = append(optionInfo, []string{
					"[lightpink::b]" + strconv.Itoa(format.AudioSampleRate) + "kHz[-:-:-]",
					"[grey::b]" + strconv.Itoa(format.AudioChannels) + "ch[-:-:-]",
				}...)
			}

			data := DownloadData{
				id:    video.VideoID,
				title: video.Title,

				dtype:  "video",
				format: format,
			}

			option := strings.Join(optionInfo, ", ")
			optionLength := tview.TaggedStringWidth(option) + 6
			if optionLength > width {
				width = optionLength
			}

			d.options.SetCell((rows+row)-skipped, 0, tview.NewTableCell(option).
				SetExpansion(1).
				SetReference(data).
				SetSelectedStyle(app.UI.ColumnStyle),
			)
		}
	}

	d.modal.Width = width
	if d.options.GetRowCount() < d.modal.Height {
		d.modal.Height = d.options.GetRowCount() + 4
	}
}

// remove removes the currently downloading item from the downloads view.
func (p *DownloadProgress) remove() {
	if Downloads.view == nil {
		return
	}

	for row := 0; row < Downloads.view.GetRowCount(); row++ {
		cell := Downloads.view.GetCell(row, 0)

		progress, ok := cell.GetReference().(*DownloadProgress)
		if !ok {
			continue
		}

		if p == progress {
			Downloads.view.RemoveRow(row)
			Downloads.view.RemoveRow(row - 1)

			break
		}
	}

	if Downloads.view.HasFocus() && Downloads.view.GetRowCount() == 0 {
		Downloads.view.InputHandler()(tcell.NewEventKey(tcell.KeyEscape, ' ', tcell.ModNone), nil)
	}
}

// renderBar renders the progress bar within the downloads view.
func (p *DownloadProgress) renderBar(filename string, clen int64, cancel func(), video bool) {
	options := []progressbar.Option{
		progressbar.OptionSpinnerType(34),
		progressbar.OptionSetWriter(p),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionThrottle(200 * time.Millisecond),
	}
	if video {
		options = append(options, progressbar.OptionShowBytes(true))
	}

	p.desc = tview.NewTableCell("[::b]" + tview.Escape(filename)).
		SetExpansion(1).
		SetSelectable(true).
		SetAlign(tview.AlignLeft)

	p.progress = tview.NewTableCell("").
		SetExpansion(1).
		SetSelectable(false).
		SetAlign(tview.AlignRight)

	p.bar = progressbar.NewOptions64(clen, options...)

	p.cancelFunc = cancel

	app.UI.QueueUpdateDraw(func() {
		rows := Downloads.view.GetRowCount()

		Downloads.view.SetCell(rows+1, 0, p.desc.SetReference(p))
		Downloads.view.SetCell(rows+1, 1, p.progress)
		Downloads.view.Select(rows+1, 0)
	})
}

// Write generates the progress bar.
func (p *DownloadProgress) Write(b []byte) (int, error) {
	app.UI.QueueUpdateDraw(func() {
		p.progress.SetText(string(b))
	})

	return 0, nil
}

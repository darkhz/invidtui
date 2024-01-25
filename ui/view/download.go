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
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
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

	property theme.ThemeProperty
}

// DownloadProgress describes the layout of a progress indicator.
type DownloadProgress struct {
	desc, progress *tview.TableCell
	bar            *progressbar.ProgressBar
	builder        theme.ThemeTextBuilder

	cancelFunc context.CancelFunc
}

// DownloadData describes the information for the downloading item.
type DownloadData struct {
	id, title, dtype string

	format inv.VideoFormat
}

type DownloadItem struct {
	region, text string
	item         theme.ThemeItem
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

	d.property = d.ThemeProperty()

	d.options = theme.NewTable(d.property)
	d.options.SetSelectable(true, false)
	d.options.SetInputCapture(d.OptionKeybindings)
	d.options.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextDownloads, d.options)
	})

	d.view = theme.NewTable(d.property.SetItem(theme.ThemeBackground))
	d.view.SetBorder(true)
	d.view.SetTitle(
		theme.SetTextStyle(
			"title",
			"Download List",
			d.property.Context,
			theme.ThemeTitle,
		),
	)
	d.view.SetSelectable(true, false)
	d.view.SetTitleAlign(tview.AlignLeft)
	d.view.SetInputCapture(d.Keybindings)
	d.view.SetFocusFunc(func() {
		app.SetContextMenu(keybinding.KeyContextDownloads, d.view)
	})

	d.modal = app.NewModal("downloads", "Select Download Option", d.options, 40, 60, d.property)

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

// ThemeProperty returns the download view's theme property.
func (d *DownloadsView) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextDownloads,
		Item:    theme.ThemePopupBackground,
	}
}

// IsInitialized returns whether the downloads view is initialized.
func (d *DownloadsView) IsInitialized() bool {
	return d.init
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
	switch keybinding.KeyOperation(event, keybinding.KeyContextDownloads) {
	case keybinding.KeyDownloadChangeDir:
		d.SetDir()

	case keybinding.KeySelect:
		row, _ := d.options.GetSelection()
		cell := d.options.GetCell(row, 0)

		if data, ok := cell.GetReference().(DownloadData); ok {
			filename := data.title + "." + data.format.Container
			go d.TransferVideo(data.id, data.format.Itag, filename)
		}

		fallthrough

	case keybinding.KeyClose:
		d.modal.Exit(false)
	}

	return event
}

// Keybindings describes the keybindings for the downloads view.
func (d *DownloadsView) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	switch keybinding.KeyOperation(event, keybinding.KeyContextDownloads) {
	case keybinding.KeyDownloadCancel:
		row, _ := Downloads.view.GetSelection()

		cell := Downloads.view.GetCell(row, 0)
		if progress, ok := cell.GetReference().(*DownloadProgress); ok {
			progress.cancelFunc()
		}

	case keybinding.KeyClose:
		CloseView()
	}

	return event
}

// renderOptions render the download options popup.
func (d *DownloadsView) renderOptions(video inv.VideoData) {
	var skipped, width int

	d.options.Clear()

	builder := theme.NewTextBuilder(theme.ThemeContextDownloads)

	for i, formatData := range [][]inv.VideoFormat{
		video.FormatStreams,
		video.AdaptiveFormats,
	} {
		rows := d.options.GetRowCount()

		for row, format := range formatData {
			mtype := strings.Split(strings.Split(format.Type, ";")[0], "/")
			if (mtype[0] == "audio" && (format.Container == "" || format.Encoding == "")) ||
				(mtype[0] == "video" && format.FPS == 0) {
				skipped++
				continue
			}

			builder.Start(theme.ThemeMediaInfo, "minfo")
			fmt.Fprintf(&builder, "%s", mtype[0])
			if i != 0 {
				builder.AppendText(" only")
			} else {
				var err error

				builder.AppendText(" + audio")
				clen := utils.GetDataFromURL(format.URL).Get("clen")
				format.ContentLength, err = strconv.ParseInt(clen, 10, 64)
				if err != nil {
					format.ContentLength = 0
				}
			}
			builder.Finish()
			builder.AppendText(", ")

			builder.Start(theme.ThemeMediaSize, "msize")
			if format.ContentLength == 0 {
				builder.AppendText("-")
			} else {
				fmt.Fprintf(&builder, "%.2f MB", float64(format.ContentLength)/1024/1024)
			}
			builder.Finish()
			builder.AppendText(", ")

			builder.Format(theme.ThemeMediaType, "mtype", "%s / %s, ", format.Container, format.Encoding)
			if mtype[0] != "audio" {
				builder.Format(theme.ThemeVideoResolution, "vres", "%s, ", format.Resolution)
				builder.Format(theme.ThemeVideoFPS, "vfps", "%d fps", format.FPS)
			} else {
				builder.Format(theme.ThemeAudioSampleRate, "akhz", "%d kHz, ", format.AudioSampleRate)
				builder.Format(theme.ThemeAudioChannels, "auch", "%d ch", format.AudioChannels)
			}

			data := DownloadData{
				id:    video.VideoID,
				title: video.Title,

				dtype:  "video",
				format: format,
			}

			text := builder.Get()
			optionLength := tview.TaggedStringWidth(text) + 6
			if optionLength > width {
				width = optionLength
			}

			d.options.SetCell((rows+row)-skipped, 0, tview.NewTableCell(text).
				SetExpansion(1).
				SetReference(data),
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
		Downloads.view.InputHandler()(keybinding.KeyEvent(keybinding.KeyClose), nil)
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

	p.desc = theme.NewTableCell(
		theme.ThemeContextDownloads,
		theme.ThemeProgressText,
		tview.Escape(filename),
	).
		SetExpansion(1).
		SetSelectable(true).
		SetAlign(tview.AlignLeft)

	p.progress = theme.NewTableCell(
		theme.ThemeContextDownloads,
		theme.ThemeProgressBar,
		"",
	).
		SetExpansion(1).
		SetSelectable(false).
		SetAlign(tview.AlignRight)

	p.bar = progressbar.NewOptions64(clen, options...)

	p.cancelFunc = cancel

	p.builder = theme.NewTextBuilder(theme.ThemeContextDownloads)

	app.UI.QueueUpdateDraw(func() {
		rows := Downloads.view.GetRowCount()

		Downloads.view.SetCell(rows+1, 0, p.desc.SetReference(p))
		Downloads.view.SetCell(rows+1, 1, p.progress)
		Downloads.view.Select(rows+1, 0)
	})
}

// Write generates the progress bar.
func (p *DownloadProgress) Write(b []byte) (int, error) {
	app.UI.Lock()
	open := Downloads.view.HasFocus() && Downloads.view.GetRowCount() != 0
	app.UI.Unlock()
	if !open {
		return len(b), nil
	}

	p.builder.Start(theme.ThemeProgressBar, "progress")
	p.builder.Write(b)
	p.builder.Finish()

	app.UI.Lock()
	p.progress.SetText(p.builder.Get())
	app.UI.Unlock()

	app.DrawPrimitives(Downloads.view)

	return len(b), nil
}

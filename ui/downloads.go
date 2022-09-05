package ui

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/darkhz/invidtui/lib"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/schollz/progressbar/v3"
)

// DownloadProgress stores the progress data.
type DownloadProgress struct {
	desc     *tview.TableCell
	progress *tview.TableCell

	progressBar *progressbar.ProgressBar

	cancelFunc context.CancelFunc
}

var (
	downloadView *tview.Table

	prevPage string
	prevItem tview.Primitive
)

// ShowDownloadView opens the download view.
func ShowDownloadView() {
	if downloadView == nil || downloadView.GetRowCount() == 0 {
		InfoMessage("No downloads in progress", false)
		return
	}

	MPage.SwitchToPage("ui")

	prevPage, prevItem = VPage.GetFrontPage()

	title := tview.NewTextView()
	title.SetDynamicColors(true)
	title.SetText("[::bu]Downloads")
	title.SetTextAlign(tview.AlignLeft)
	title.SetBackgroundColor(tcell.ColorDefault)

	downloadFlex := tview.NewFlex().
		AddItem(title, 1, 0, false).
		AddItem(downloadView, 0, 10, false).
		SetDirection(tview.FlexRow)

	VPage.AddAndSwitchToPage("downloadview", downloadFlex, true)

	App.SetFocus(downloadView)
}

//gocyclo:ignore
// ShowDownloadOptions shows the download options for the video.
func ShowDownloadOptions() {
	var err error
	var vpg, mpg string
	var skipped, length int
	var info lib.SearchResult
	var vtable tview.Primitive

	App.QueueUpdateDraw(func() {
		info, err = getListReference()

		vpg, vtable = VPage.GetFrontPage()
		mpg, _ = MPage.GetFrontPage()
	})
	if err != nil {
		ErrorMessage(err)
		return
	}

	if lib.DownloadFolder() == "" {
		ErrorMessage(fmt.Errorf("No download folder specified"))
		return
	}

	if info.Type != "video" {
		return
	}

	InfoMessage("Getting download options", true)

	lib.VideoNewCtx()

	video, err := lib.GetClient().Video(info.VideoID)
	if err != nil {
		ErrorMessage(err)
		return
	}

	if video.LiveNow {
		ErrorMessage(fmt.Errorf("Cannot download live video"))
		return
	}

	optionsPopup := tview.NewTable()
	optionsPopup.SetBorder(true)
	optionsPopup.SetSelectorWrap(true)
	optionsPopup.SetSelectable(true, false)
	optionsPopup.SetTitle(" [::b]Select download option ")
	optionsPopup.SetBackgroundColor(tcell.ColorDefault)
	optionsPopup.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			row, _ := optionsPopup.GetSelection()
			cell := optionsPopup.GetCell(row, 0)

			if format, ok := cell.GetReference().(lib.FormatData); ok {
				filename := info.Title + "." + format.Container
				go startDownload(info.VideoID, format.Itag, filename)
			}

			fallthrough

		case tcell.KeyEscape:
			VPage.RemovePage("dloption")

			if mpg != "ui" {
				App.SetFocus(popup.primitive)
			} else {
				App.SetFocus(vtable)
			}
		}

		return event
	})

	App.QueueUpdateDraw(func() {
		for i, formatData := range [][]lib.FormatData{
			video.FormatStreams,
			video.AdaptiveFormats,
		} {
			rows := optionsPopup.GetRowCount()

			for row, format := range formatData {
				var minfo, size string
				var optionInfo []string

				if i != 0 {
					minfo = " only"
				} else {
					minfo = " + audio"

					clen := lib.GetDataFromURL(format.URL).Get("clen")
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

				option := strings.Join(optionInfo, ", ")
				optionLength := tview.TaggedStringWidth(option) + 6

				if optionLength > length {
					length = optionLength
				}

				optionsPopup.SetCell((rows+row)-skipped, 0, tview.NewTableCell(option).
					SetExpansion(1).
					SetReference(format).
					SetSelectedStyle(auxStyle),
				)
			}
		}

		height, tpad, bpad := 40, 20, 20
		if mpg != "ui" {
			height, tpad, bpad = 20, 0, 20
		}

		wrapOptions := tview.NewFlex().
			AddItem(nil, 0, tpad, false).
			AddItem(optionsPopup, 0, height, false).
			AddItem(nil, 0, bpad, false).
			SetDirection(tview.FlexRow)
		wrapOptions.SetBackgroundColor(tcell.ColorDefault)

		optionsFlex := tview.NewFlex().
			AddItem(nil, 0, 10, false).
			AddItem(wrapOptions, length, 0, false).
			AddItem(nil, 0, 10, false).
			SetDirection(tview.FlexColumn)
		optionsFlex.SetBackgroundColor(tcell.ColorDefault)

		VPage.AddAndSwitchToPage("dloption", optionsFlex, true).ShowPage(vpg)

		App.SetFocus(optionsPopup)
	})

	InfoMessage("Download options loaded", false)
}

// startDownload starts the download and tracks its progress.
func startDownload(id, itag, filename string) {
	var download DownloadProgress

	InfoMessage("Starting download for "+tview.Escape(filename), true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res, file, err := lib.GetDownload(id, itag, filename, ctx)
	if err != nil {
		ErrorMessage(err)
		return
	}
	defer res.Body.Close()
	defer file.Close()

	download.desc = tview.NewTableCell("[::b]" + tview.Escape(filename)).
		SetExpansion(1).
		SetSelectable(true).
		SetAlign(tview.AlignLeft)

	download.progress = tview.NewTableCell("").
		SetExpansion(1).
		SetSelectable(false).
		SetAlign(tview.AlignRight)

	download.progressBar = progressbar.NewOptions64(
		res.ContentLength,
		progressbar.OptionSpinnerType(34),
		progressbar.OptionSetWriter(&download),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetElapsedTime(false),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionThrottle(200*time.Millisecond),
	)

	download.cancelFunc = cancel

	App.QueueUpdateDraw(func() {
		if downloadView == nil {
			downloadView = tview.NewTable()
			downloadView.SetSelectorWrap(true)
			downloadView.SetSelectable(true, false)
			downloadView.SetBackgroundColor(tcell.ColorDefault)
			downloadView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				switch event.Key() {
				case tcell.KeyEscape:
					VPage.SwitchToPage(prevPage)
					App.SetFocus(prevItem)
				}

				switch event.Rune() {
				case 'x':
					row, _ := downloadView.GetSelection()

					cell := downloadView.GetCell(row, 0)
					if download, ok := cell.GetReference().(*DownloadProgress); ok {
						download.cancelFunc()
					}
				}

				return event
			})
		}

		rows := downloadView.GetRowCount()

		downloadView.SetCell(rows+1, 0, download.desc.SetReference(&download))
		downloadView.SetCell(rows+1, 1, download.progress)

		downloadView.Select(rows+1, 0)
	})
	defer download.removeDownload()

	InfoMessage("Download started for "+tview.Escape(filename), false)

	_, err = io.Copy(io.MultiWriter(file, download.progressBar), res.Body)
	if err != nil {
		ErrorMessage(err)
	}
}

// removeDownload removes the download from the download view.
func (d *DownloadProgress) removeDownload() {
	App.QueueUpdateDraw(func() {
		if downloadView == nil {
			return
		}

		for row := 0; row < downloadView.GetRowCount(); row++ {
			cell := downloadView.GetCell(row, 0)

			download, ok := cell.GetReference().(*DownloadProgress)
			if !ok {
				continue
			}

			if d == download {
				downloadView.RemoveRow(row)
				downloadView.RemoveRow(row - 1)

				break
			}
		}

		if downloadView.GetRowCount() == 0 {
			downloadView.InputHandler()(tcell.NewEventKey(tcell.KeyEscape, ' ', tcell.ModNone), nil)
		}
	})
}

// Write displays the progressbar on the screen.
func (d *DownloadProgress) Write(b []byte) (int, error) {
	App.QueueUpdateDraw(func() {
		d.progress.SetText(string(b))
	})

	return 0, nil
}

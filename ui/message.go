package ui

import (
	"context"
	"errors"
	"time"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

type message struct {
	text    string
	persist bool
}

var (
	// Status enables switching between
	// MessageBox and InputBox.
	Status *tview.Pages

	//MessageBox is an area to display messages.
	MessageBox *tview.TextView

	sctx    context.Context
	scancel context.CancelFunc
	msgchan chan message
)

// SetupStatus sets up the statusbar.
func SetupStatus() {
	Status = tview.NewPages()

	Status.AddPage("input", InputBox, true, true)
	Status.AddPage("messages", MessageBox, true, true)

	msgchan = make(chan message, 10)
	sctx, scancel = context.WithCancel(context.Background())

	go startStatus()
}

// StopStatus stops the message event loop.
func StopStatus() {
	scancel()
	close(msgchan)
}

// SetupMessageBox sets up a text area to receive messages.
func SetupMessageBox() {
	MessageBox = tview.NewTextView().
		SetDynamicColors(true)

	MessageBox.SetBackgroundColor(tcell.ColorDefault)
}

// InfoMessage sends an info message to the status bar.
func InfoMessage(text string, persist bool) {
	select {
	case msgchan <- message{"[white::b]" + text, persist}:
		return

	default:
	}
}

// ErrorMessage sends an error message to the status bar.
func ErrorMessage(err error) {
	if errors.Is(err, context.Canceled) {
		return
	}

	select {
	case msgchan <- message{"[red::b]" + err.Error(), false}:
		return

	default:
	}
}

// startStatus starts the message event loop
func startStatus() {
	var text string
	var cleared bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-sctx.Done():
			return

		case msg, ok := <-msgchan:
			if !ok {
				return
			}

			t.Reset(2 * time.Second)

			cleared = false

			if msg.persist {
				text = msg.text

			}

			if !msg.persist && text != "" {
				text = ""
			}

			App.QueueUpdateDraw(func() {
				MessageBox.SetText(msg.text)
			})

		case <-t.C:
			if cleared {
				continue
			}

			cleared = true

			App.QueueUpdateDraw(func() {
				MessageBox.SetText(text)
			})
		}
	}
}

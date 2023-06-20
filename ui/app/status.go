package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// Status describes the layout for a status bar
type Status struct {
	Message *tview.TextView

	acceptMax    int
	inputLabel   string
	inputBoxFunc func(text string)
	inputChgFunc func(text string)
	defaultIFunc func(event *tcell.EventKey) *tcell.EventKey

	ctx     context.Context
	Cancel  context.CancelFunc
	msgchan chan message

	*tview.Pages
	*tview.InputField
}

// message includes the text to be shown within the status bar,
// and determines whether the message is to be shown persistently.
type message struct {
	text    string
	persist bool
}

// Setup sets up the status bar.
func (s *Status) Setup() {
	s.Pages = tview.NewPages()

	s.Message = tview.NewTextView()
	s.Message.SetDynamicColors(true)
	s.Message.SetBackgroundColor(tcell.ColorDefault)

	s.InputField = tview.NewInputField()
	s.InputField.SetLabelColor(tcell.ColorWhite)
	s.InputField.SetBackgroundColor(tcell.ColorDefault)
	s.InputField.SetFieldBackgroundColor(tcell.ColorDefault)
	s.InputField.SetFocusFunc(s.inputFocus)
	s.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			text := s.InputField.GetText()

			if text == "" {
				return event
			}

			s.inputBoxFunc(text)

			fallthrough

		case tcell.KeyEscape:
			_, item := UI.Pages.GetFrontPage()
			UI.SetFocus(item)

			s.Pages.SwitchToPage("messages")
		}

		return event
	})

	s.Pages.AddPage("input", s.InputField, true, true)
	s.Pages.AddPage("messages", s.Message, true, true)

	s.msgchan = make(chan message, 10)
	s.defaultIFunc = s.InputField.GetInputCapture()
	s.ctx, s.Cancel = context.WithCancel(context.Background())

	go s.startStatus()
}

// InfoMessage sends an info message to the status bar.
func (s *Status) InfoMessage(text string, persist bool) {
	select {
	case s.msgchan <- message{"[white::b]" + text, persist}:
		return

	default:
	}
}

// ErrorMessage sends an error message to the status bar.
func (s *Status) ErrorMessage(err error) {
	if errors.Is(err, context.Canceled) {
		return
	}

	select {
	case s.msgchan <- message{"[red::b]" + err.Error(), false}:
		return

	default:
	}
}

// SetInput sets up the prompt and appropriate handlers
// for the input area within the status bar.
func (s *Status) SetInput(label string,
	max int,
	clearInput bool,
	dofunc func(text string),
	ifunc func(event *tcell.EventKey) *tcell.EventKey,
	chgfunc ...func(text string),
) {
	s.inputBoxFunc = dofunc

	if max > 0 {
		s.InputField.SetAcceptanceFunc(tview.InputFieldMaxLength(max))
	} else {
		s.InputField.SetAcceptanceFunc(nil)
	}

	s.acceptMax = max

	if chgfunc != nil {
		s.inputChgFunc = chgfunc[0]
	} else {
		s.inputChgFunc = nil
	}
	s.InputField.SetChangedFunc(s.inputChgFunc)

	if clearInput {
		s.InputField.SetText("")
	}
	s.InputField.SetLabel("[::b]" + label + " ")

	if ifunc != nil {
		s.InputField.SetInputCapture(ifunc)
	} else {
		s.InputField.SetInputCapture(s.defaultIFunc)
	}

	UI.Status.Pages.SwitchToPage("input")
	UI.SetFocus(s.InputField)
}

// SetFocusFunc sets the function to be executed when the input is focused.
func (s *Status) SetFocusFunc(focus ...func()) {
	if focus == nil {
		s.InputField.SetFocusFunc(s.inputFocus)
		return
	}

	s.InputField.SetFocusFunc(focus[0])
}

// startStatus starts the message event loop
func (s *Status) startStatus() {
	var text string
	var cleared bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return

		case msg, ok := <-s.msgchan:
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

			UI.QueueUpdateDraw(func() {
				s.Message.SetText(msg.text)
			})

		case <-t.C:
			if cleared {
				continue
			}

			cleared = true

			UI.QueueUpdateDraw(func() {
				s.Message.SetText(text)
			})
		}
	}
}

func (s *Status) inputFocus() {
	label := s.InputField.GetLabel()
	if label != s.inputLabel {
		s.inputLabel = strings.TrimSpace(label)
	}
}

// ShowInfo shows an information message.
func ShowInfo(text string, persist bool) {
	UI.Status.InfoMessage(text, persist)
}

// ShowError shows an error message.
func ShowError(err error) {
	UI.Status.ErrorMessage(err)
}

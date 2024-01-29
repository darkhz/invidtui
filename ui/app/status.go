package app

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
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

	tag chan string

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
	property := s.ThemeProperty()

	s.Pages = theme.NewPages(property)

	s.Message = theme.NewTextView(property)

	s.InputField = theme.NewInputField(property, "")
	s.InputField.SetFocusFunc(s.inputFocus)
	s.InputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch keybinding.KeyOperation(event, keybinding.KeyContextCommon) {
		case keybinding.KeySelect:
			text := s.InputField.GetText()

			if text == "" {
				return event
			}

			s.inputBoxFunc(text)

			fallthrough

		case keybinding.KeyClose:
			_, item := UI.Pages.GetFrontPage()
			UI.SetFocus(item)

			s.Pages.SwitchToPage("messages")
		}

		return event
	})

	s.Pages.AddPage("input", s.InputField, true, true)
	s.Pages.AddPage("messages", s.Message, true, true)

	s.tag = make(chan string, 1)
	s.msgchan = make(chan message, 10)
	s.defaultIFunc = s.InputField.GetInputCapture()
	s.ctx, s.Cancel = context.WithCancel(context.Background())

	go s.startStatus()
}

// InfoMessage sends an info message to the status bar.
func (s *Status) InfoMessage(text string, persist bool) {
	text = theme.SetTextStyle(
		"message",
		text,
		theme.ThemeContextStatusBar,
		theme.ThemeInfoMessage,
	)

	select {
	case s.msgchan <- message{text, persist}:
		return

	default:
	}
}

// ErrorMessage sends an error message to the status bar.
func (s *Status) ErrorMessage(err error) {
	if errors.Is(err, context.Canceled) {
		ShowInfo("", false)
		return
	}

	text := theme.SetTextStyle(
		"message",
		err.Error(),
		theme.ThemeContextStatusBar,
		theme.ThemeInfoMessage,
	)

	select {
	case s.msgchan <- message{text, false}:
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
	s.InputField.SetLabel(theme.GetLabel(
		s.ThemeProperty().SetItem(theme.ThemeInputLabel), label, true),
	)

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

// Tag sets a tag to the status bar.
func (s *Status) Tag(tag string) {
	select {
	case s.tag <- tag:
		return

	default:
	}
}

func (s *Status) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextStatusBar,
		Item:    theme.ThemeBackground,
	}
}

// startStatus starts the message event loop
func (s *Status) startStatus() {
	var tag, text, message string
	var cleared bool

	t := time.NewTicker(2 * time.Second)
	defer t.Stop()

	for {
		msgtext := ""

		select {
		case <-s.ctx.Done():
			return

		case msg, ok := <-s.msgchan:
			if !ok {
				return
			}

			t.Reset(2 * time.Second)

			cleared = false

			message = msg.text

			if msg.persist {
				text = msg.text
			}

			if !msg.persist && text != "" {
				text = ""
			}

			msgtext = tag + message

		case t, ok := <-s.tag:
			if !ok {
				return
			}

			tag = t
			if tag != "" {
				tag += " "
			}

			m := message
			if text != "" {
				m = text
			}

			msgtext = tag + m

		case <-t.C:
			message = ""

			if cleared {
				continue
			}

			cleared = true

			msgtext = tag + text
		}

		go UI.QueueUpdateDraw(func() {
			s.Message.SetText(msgtext)
		})
	}
}

func (s *Status) inputFocus() {
	label := s.InputField.GetLabel()
	if label != s.inputLabel {
		s.inputLabel = strings.TrimSpace(label)
	}
}

// ShowInfo shows an information message.
func ShowInfo(text string, persist bool, print ...bool) {
	if print != nil && !print[0] {
		return
	}

	UI.Status.InfoMessage(text, persist)
}

// ShowError shows an error message.
func ShowError(err error) {
	UI.Status.ErrorMessage(err)
}

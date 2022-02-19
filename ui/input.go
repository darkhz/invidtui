package ui

import (
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	// InputBox is an input area.
	InputBox *tview.InputField

	inputLabel   string
	inputBoxFunc func(text string)
	defaultIFunc func(event *tcell.EventKey) *tcell.EventKey
)

// SetupInputBox sets up an inputbox to enter text.
func SetupInputBox() {
	InputBox = tview.NewInputField()

	InputBox.SetFocusFunc(func() {
		label := InputBox.GetLabel()
		if label != inputLabel {
			inputLabel = label
			InputBox.SetText("")
		}
	})

	InputBox.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			text := InputBox.GetText()

			if text == "" {
				return event
			}

			inputBoxFunc(text)

		case tcell.KeyEscape:
			App.SetFocus(ResultsList)
			Status.SwitchToPage("messages")
		}

		return event
	})

	defaultIFunc = InputBox.GetInputCapture()

	InputBox.SetLabel("[::b]Search: ")
	InputBox.SetLabelColor(tcell.ColorWhite)
	InputBox.SetBackgroundColor(tcell.ColorDefault)
	InputBox.SetFieldBackgroundColor(tcell.ColorDefault)
}

// SetInput sets up a custom label and function to be executed
// on pressing the Enter key in the inputbox.
func SetInput(label string,
	max int,
	dofunc func(text string),
	ifunc func(event *tcell.EventKey) *tcell.EventKey,
) {
	inputBoxFunc = dofunc

	if max > 0 {
		InputBox.SetAcceptanceFunc(tview.InputFieldMaxLength(max))
	} else {
		InputBox.SetAcceptanceFunc(nil)
	}

	InputBox.SetText("")
	InputBox.SetLabel("[::b]" + label + " ")
	if ifunc != nil {
		InputBox.SetInputCapture(ifunc)
	} else {
		InputBox.SetInputCapture(defaultIFunc)
	}

	App.SetFocus(InputBox)
	Status.SwitchToPage("input")
}

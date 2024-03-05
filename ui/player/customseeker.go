package player

import (
	"fmt"
	"strconv"
	"strings"

	mp "github.com/darkhz/invidtui/mediaplayer"
	"github.com/darkhz/invidtui/ui/app"
	"github.com/darkhz/invidtui/ui/keybinding"
	"github.com/darkhz/invidtui/ui/theme"
	"github.com/darkhz/invidtui/utils"
	"github.com/darkhz/tview"
	"github.com/gdamore/tcell/v2"
)

// CustomSeeker describes the custom playback seeker.
type CustomSeeker struct {
	modal *app.Modal
	table *tview.Table
	info  *tview.TextView
	flex  *tview.Flex

	inputs   []*tview.InputField
	data     []string
	focuspos int
}

const (
	SeekMaxHours             = 600
	SeekMaxMinutesAndSeconds = 59

	SeekInputHoursLength             = 3
	SeekInputMinutesAndSecondsLength = 2

	SeekInputByPosition      = 0
	SeekInputHoursPosition   = 1
	SeekInputMinutesPosition = 2
	SeekInputSecondsPosition = 3
)

// Setup sets up the custom seeker.
func (c *CustomSeeker) Setup() {
	property := c.ThemeProperty()

	labels := []string{"Seek By (+/-)", "hh", "mm", "ss"}
	c.data = make([]string, len(labels))

	c.info = theme.NewTextView(property)
	c.info.SetTextAlign(tview.AlignCenter)

	box := theme.NewBox(property)

	seekByFlex := theme.NewFlex(property).
		AddItem(box, 2, 0, false)

	posFlex := theme.NewFlex(property)

	for i, label := range labels {
		input := theme.NewInputField(property, label+":")
		input.SetInputCapture(c.Keybindings)
		input.SetAcceptanceFunc(c.inputContentHandler)
		input.SetChangedFunc(c.inputChangedHandler)

		c.inputs = append(c.inputs, input)
		c.focuspos = i

		if i == 0 {
			input.SetText("+1")
			input.SetFieldWidth(10)

			seekByFlex.AddItem(input, 0, 1, true)
			seekByFlex.AddItem(box, 1, 0, false)

			continue
		}

		input.SetText("00")
		input.SetFieldWidth(4)
		posFlex.AddItem(input, 0, 1, true)
		if i <= 2 {
			posFlex.AddItem(box, 2, 0, false)
		}
	}

	c.flex = theme.NewFlex(property).
		SetDirection(tview.FlexRow).
		AddItem(seekByFlex, 0, 1, false).
		AddItem(posFlex, 0, 1, true).
		AddItem(app.HorizontalLine(property), 1, 0, false).
		AddItem(c.info, 1, 0, false)
	c.flex.SetFocusFunc(func() {
		app.UI.SetFocus(c.inputs[c.focuspos])
	})

	c.modal = app.NewModal("customseek", "Custom Seek", c.flex, 10, (len(c.inputs)-1)*10, property)

	c.focuspos = 0
	c.inputChangedHandler(c.inputs[0].GetText())
}

// Show shows the custom seeker.
func (c *CustomSeeker) Show() {
	if c.IsOpen() || !IsPlayerShown() || IsQueueEmpty() {
		return
	}

	app.SetContextMenu(keybinding.KeyContextSeek, c.flex)
	c.modal.Show(false)
}

// Hide hides the custom seeker.
func (c *CustomSeeker) Hide() {
	if !c.IsOpen() {
		return
	}

	c.modal.Exit(false)
}

// IsOpen returns whether the custom seeker is open.
func (c *CustomSeeker) IsOpen() bool {
	return c.modal != nil && c.modal.Open
}

// SeekToPosition seeks to the specified position.
func (c *CustomSeeker) SeekToPosition() {
	if c.focuspos == 0 {
		go c.seekBy(c.data[0])
		return
	}

	go c.seekTo(c.data[1:])
}

// seekBy seeks relative to the current playback position.
func (c *CustomSeeker) seekBy(by string) {
	if by == "" {
		app.ShowError(fmt.Errorf("Custom Seek: No 'seek-by' value specified"))
		return
	}

	mp.Player().SeekToPosition(by)
}

// seekTo seeks to the absolute playback position.
func (c *CustomSeeker) seekTo(hhmmss []string) {
	for i, value := range []string{"hour", "minute", "seconds"} {
		if hhmmss[i] == "" {
			app.ShowError(fmt.Errorf("Custom Seek: No '%s' value specified", value))
			return
		}
	}

	mp.Player().SetPosition(utils.ConvertDurationToSeconds(strings.Join(hhmmss, ":")))
}

// inputChangedHandler shows messages according to which input field's contents has changed.
func (c *CustomSeeker) inputChangedHandler(text string) {
	c.data[c.focuspos] = text

	builder := theme.NewTextBuilder(theme.ThemeContextCustomSeeker)
	builder.Start(theme.ThemeText, "info")
	builder.AppendText("Seek")

	if c.focuspos == 0 {
		if len(text) < 2 {
			text = "0"
		}

		fmt.Fprintf(&builder, " by %ss", text)
	} else {
		builder.AppendText(" to ")
		for i, m := range []string{"h", "m", "s"} {
			t := c.data[i+1]
			if t == "" {
				t = "00"
			}

			builder.AppendText(t)
			builder.AppendText(m)
			if i <= 2 {
				builder.AppendText(" ")
			}
		}
	}

	builder.Finish()
	c.info.SetText(builder.Get())
}

// inputContentHandler handles entering content into the input field.
func (c *CustomSeeker) inputContentHandler(text string, ch rune) bool {
	pos := c.focuspos

	length := SeekInputMinutesAndSecondsLength
	if pos == SeekInputHoursPosition {
		length = SeekInputHoursLength
	}

	runes := []rune(text)
	runeslen := len(runes)

	if pos == 0 && runeslen > 0 && (runes[0] != '+' && runes[0] != '-') {
		return false
	}

	if pos != 0 && (runeslen > length || ch == '.' || strings.Count(text, ".") > 0) {
		return false
	}

	switch text {
	case "+", "-":
		if pos == 0 {
			return true
		}

	default:
		n, err := strconv.ParseFloat(text, 10)
		if err != nil {
			return false
		}

		switch pos {
		case SeekInputByPosition:
			return true

		case SeekInputHoursPosition:
			if n < SeekMaxHours {
				return true
			}

		case SeekInputMinutesPosition, SeekInputSecondsPosition:
			if n <= SeekMaxMinutesAndSeconds {
				return true
			}
		}
	}

	return false
}

// Keybindings define the keybindings for the custom seeker.
func (c *CustomSeeker) Keybindings(event *tcell.EventKey) *tcell.EventKey {
	operation := keybinding.KeyOperation(event, keybinding.KeyContextCommon)

	switch operation {
	case keybinding.KeySelect:
		c.SeekToPosition()

	case keybinding.KeySwitch:
		c.focuspos++
		if c.focuspos == len(c.inputs) {
			c.focuspos = 0
		}

		input := c.inputs[c.focuspos]
		app.UI.SetFocus(input)
		c.inputChangedHandler(input.GetText())

	case keybinding.KeyClose:
		c.Hide()
	}

	return event
}

// ThemeProperty returns the custom seeker's theme property.
func (c *CustomSeeker) ThemeProperty() theme.ThemeProperty {
	return theme.ThemeProperty{
		Context: theme.ThemeContextCustomSeeker,
		Item:    theme.ThemePopupBackground,
		IsForm:  true,
	}
}

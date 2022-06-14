package lib

import (
	"io/ioutil"
	"os"
	"strings"
)

var (
	history     []string
	historyFile string
	histpos     int
)

// SetupHistory reads the history file and loads the search history.
// Taken from https://github.com/abs-lang/abs/repl/history.go (Start)
func SetupHistory() {
	var err error

	historyFile, err = ConfigPath("history")
	if err != nil {
		return
	}

	file, err := os.OpenFile(historyFile, os.O_RDONLY|os.O_CREATE, 0664)
	if err != nil {
		return
	}
	file.Close()

	bytes, err := ioutil.ReadFile(historyFile)
	if err != nil {
		return
	}

	if len(bytes) <= 0 {
		return
	}

	history = strings.Split(string(bytes), "\n")
	histpos = len(history)
}

// AddToHistory adds text to the history buffer.
// Taken from https://github.com/abs-lang/abs/repl/history.go (addToHistory)
func AddToHistory(text string) {
	if text == "" {
		return
	}

	if len(history) == 0 {
		history = append(history, text)
	} else if text != history[len(history)-1] {
		history = append(history, text)
	}

	histpos = len(history)
}

// SaveHistory saves the contents of the history buffer to the history file.
// Taken from https://github.com/abs-lang/abs/repl/history.go (saveHistory)
func SaveHistory() {
	historyStr := strings.Join(history, "\n")
	err := ioutil.WriteFile(historyFile, []byte(historyStr), 0664)
	if err != nil {
		return
	}
}

// HistoryForward moves a step forward in the history buffer, and returns a text.
func HistoryForward() string {
	if histpos+1 >= len(history) {
		return ""
	}

	histpos++

	return history[histpos]
}

// HistoryReverse moves a step back in the history buffer, and returns a text.
func HistoryReverse() string {
	if histpos-1 < 0 || histpos-1 >= len(history) {
		return ""
	}

	histpos--

	return history[histpos]
}

// HistoryReset resets the position in the history buffer.
func HistoryReset() {
	histpos = len(history)
}

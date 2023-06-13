package player

import (
	"bufio"
	"os"
	"strings"

	"github.com/darkhz/invidtui/cmd"
	mp "github.com/darkhz/invidtui/mediaplayer"
)

// loadState loads the saved player states.
func loadState() {
	var states []string

	state, err := cmd.GetPath("state")
	if err != nil {
		return
	}

	stfile, err := os.Open(state)
	if err != nil {
		return
	}
	defer stfile.Close()

	scanner := bufio.NewScanner(stfile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		states = append(states, strings.Split(line, ",")...)
		break
	}

	if len(states) == 0 {
		return
	}

	for _, s := range states {
		if strings.Contains(s, "volume") {
			vol := strings.Split(s, " ")[1]
			mp.Player().Set("volume", vol)
		}

		if strings.Contains(s, "loop") {
			mp.Player().Set(s, "yes")
			continue
		}

		mp.Player().Call("cycle", s)
	}
}

// saveStates saves the current player states.
func saveState() {
	player.mutex.Lock()
	defer player.mutex.Unlock()

	if len(player.states) == 0 {
		return
	}

	statefile, err := cmd.GetPath("state")
	if err != nil {
		return
	}

	states := strings.Join(player.states, ",")

	file, err := os.OpenFile(statefile, os.O_WRONLY, os.ModePerm)
	if err != nil {
		cmd.PrintError("Player: Could not open states file", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(states)
	if err != nil {
		cmd.PrintError("Player: Could not save states", err)
		return
	}

	if err := file.Sync(); err != nil {
		cmd.PrintError("Player: Error syncing states file", err)
	}
}

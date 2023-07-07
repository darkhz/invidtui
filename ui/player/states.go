package player

import (
	"strings"

	"github.com/darkhz/invidtui/cmd"
	mp "github.com/darkhz/invidtui/mediaplayer"
)

// loadState loads the saved player states.
func loadState() {
	states := cmd.Settings.PlayerStates
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

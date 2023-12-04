package player

import (
	"strings"

	"github.com/darkhz/invidtui/cmd"
	mp "github.com/darkhz/invidtui/mediaplayer"
)

var statesMap = map[string]int{
	"loop-file":     int(mp.RepeatModeFile),
	"loop-playlist": int(mp.RepeatModePlaylist),
}

// loadState loads the saved player states.
func loadState() {
	states := cmd.Settings.PlayerStates
	if len(states) == 0 {
		return
	}

	for _, s := range states {
		for _, state := range []string{
			"volume",
			"mute",
			"loop",
			"shuffle",
		} {

			if !strings.Contains(s, state) {
				continue
			}

			switch state {
			case "volume":
				vol := strings.Split(s, " ")[1]
				mp.Player().Set("volume", vol)

			case "mute":
				mp.Player().ToggleMuted()

			case "loop", "shuffle":
				player.queue.SetState(s)
			}

		}
	}
}

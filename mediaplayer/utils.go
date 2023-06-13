package mediaplayer

import "strings"

// replaceOptions replaces the run and subprocess options from the options parameter.
func replaceOptions(options string) string {
	opts := strings.Split(options, ",")
	newopts := opts[:0]

	for _, o := range opts {
		arg := strings.Split(o, "=")[0]

		if arg != "run" && arg != "subprocess" {
			newopts = append(newopts, o)
		}
	}

	if len(newopts) == 0 {
		newopts = opts
	}

	return strings.Join(newopts, ",")
}

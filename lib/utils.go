package lib

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FormatDuration takes a duration as seconds and returns a hh:mm:ss string.
func FormatDuration(duration int) string {
	var durationtext string

	input, err := time.ParseDuration(strconv.Itoa(duration) + "s")
	if err != nil {
		return "00:00"
	}

	d := input.Round(time.Second)

	h := d / time.Hour
	d -= h * time.Hour

	m := d / time.Minute
	d -= m * time.Minute

	s := d / time.Second

	if h > 0 {
		if h < 10 {
			durationtext += "0"
		}

		durationtext += strconv.Itoa(int(h))
		durationtext += ":"
	}

	if m > 0 {
		if m < 10 {
			durationtext += "0"
		}

		durationtext += strconv.Itoa(int(m))
	} else {
		durationtext += "00"
	}

	durationtext += ":"

	if s < 10 {
		durationtext += "0"
	}

	durationtext += strconv.Itoa(int(s))

	return durationtext
}

// FormatPublished takes a duration in the format: "1 day ago",
// and returns it in the format: "1d".
func FormatPublished(published string) string {
	ptext := strings.Split(published, " ")

	if len(ptext) > 1 {
		return ptext[0] + string(ptext[1][0])
	}

	return ptext[0]
}

// FormatNumber takes a number and represents it in the
// billions(B), millions(M), or thousands(K) format, with
// one decimal place. If there is a zero after the decimal,
// it is removed.
func FormatNumber(num int) string {
	for i, n := range []int{
		1000000000,
		1000000,
		1000,
	} {
		if num >= n {
			str := fmt.Sprintf("%.1f%c", float64(num)/float64(n), "BMK"[i])

			split := strings.Split(str, ".")
			if strings.Contains(split[1], "0") {
				str = split[0]
			}

			return str
		}
	}

	return strconv.Itoa(num)
}

//gocyclo:ignore
// GetProgress renders a progress bar and media data.
func GetProgress(width int) (string, string, error) {
	var lhs, rhs string
	var state, mtype, totaltime, vol string

	ppos := GetMPV().PlaylistPos()
	if ppos == -1 {
		return "", "", fmt.Errorf("Empty playlist")
	}

	title := GetMPV().PlaylistTitle(ppos)
	eof := GetMPV().IsEOF()
	paused := GetMPV().IsPaused()
	buffering := GetMPV().IsBuffering()
	shuffle := GetMPV().IsShuffle()
	loop := GetMPV().LoopType(true)
	mute := GetMPV().IsMuted()
	volume := GetMPV().Volume()

	duration := GetMPV().Duration()
	timepos := GetMPV().TimePosition()
	currtime := FormatDuration(timepos)

	if volume < 0 {
		vol = "0"
	} else {
		vol = strconv.Itoa(volume)
	}
	vol += "%"

	if timepos < 0 {
		timepos = 0
	}

	if duration <= 0 {
		duration = 1
	}

	if timepos > duration {
		timepos = duration
	}

	data := GetDataFromURL(title)
	if data != nil {
		if data[0] != "" {
			title = data[0]
		}

		if data[2] != "" {
			totaltime = data[2]
		} else {
			totaltime = FormatDuration(duration)
		}

		if data[3] != "" {
			mtype = data[3]
		} else {
			mtype = GetMPV().MediaType()
		}
	} else {
		totaltime = FormatDuration(duration)
		mtype = GetMPV().MediaType()
	}

	mtype = "(" + mtype + ")"

	width /= 2
	length := width * timepos / duration

	endlength := width - length
	if endlength < 0 {
		endlength = width
	}

	if shuffle {
		lhs += " S"
	}

	if mute {
		lhs += " M"
	}

	if paused {
		if eof {
			state = "[]"
		} else {
			state = "||"
		}
	} else if buffering {
		state = "B"
	} else {
		state = ">"
	}

	rhs = " " + vol + " " + mtype
	lhs = loop + lhs + " " + state + " "
	progress := currtime + " |" + strings.Repeat("█", length) + strings.Repeat(" ", endlength) + "| " + totaltime

	strings.TrimPrefix(lhs, " ")
	strings.TrimPrefix(rhs, " ")

	return title, (lhs + progress + rhs), nil
}

// IsValidURL checks if a URL is valid.
func IsValidURL(uri string) (*url.URL, error) {
	u, err := url.ParseRequestURI(uri)

	return u, err
}

// GetDataFromURL parses specific url fields and returns their values.
func GetDataFromURL(uri string) []string {
	var data []string
	u, err := IsValidURL(uri)
	if err != nil {
		return nil
	}

	for _, query := range []string{
		"title",
		"author",
		"length",
		"mediatype",
	} {
		data = append(data, strings.Join(u.Query()[query], " "))
	}

	return data
}

package lib

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// FormatDuration takes a duration as seconds and returns a hh:mm:ss string.
func FormatDuration(duration int64) string {
	var durationtext string

	input, err := time.ParseDuration(strconv.FormatInt(duration, 10) + "s")
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

// GetProgress renders a progress bar and media data.
//
//gocyclo:ignore
func GetProgress(width int) (string, string, []string, error) {
	var lhs, rhs string
	var states []string
	var state, mtype, totaltime, vol string

	ppos := GetMPV().PlaylistPos()
	if ppos == -1 {
		return "", "", nil, fmt.Errorf("Empty playlist")
	}

	title := GetMPV().PlaylistTitle(ppos)
	eof := GetMPV().IsEOF()
	paused := GetMPV().IsPaused()
	buffering := GetMPV().IsBuffering()
	shuffle := GetMPV().IsShuffle()
	loop := GetMPV().LoopType()
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
	states = append(states, "volume "+vol)
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
		if t := data.Get("title"); t != "" {
			title = t
		}

		if l := data.Get("length"); l != "" {
			totaltime = l
		} else {
			totaltime = FormatDuration(duration)
		}

		if m := data.Get("mediatype"); m != "" {
			mtype = m
		} else {
			mtype = GetMPV().MediaType()
		}
	} else {
		totaltime = FormatDuration(duration)
		mtype = GetMPV().MediaType()
	}

	mtype = "(" + mtype + ")"

	width /= 2
	length := width * int(timepos) / int(duration)

	endlength := width - length
	if endlength < 0 {
		endlength = width
	}

	if shuffle {
		lhs += " S"
		states = append(states, "shuffle")
	}

	if mute {
		lhs += " M"
		states = append(states, "mute")
	}

	if loop != "" {
		states = append(states, loop)

		switch loop {
		case "loop-file":
			loop = "R-F"

		case "loop-playlist":
			loop = "R-P"
		}
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

	return title, (lhs + progress + rhs), states, nil
}

// IsValidURL checks if a URL is valid.
func IsValidURL(uri string) (*url.URL, error) {
	u, err := url.ParseRequestURI(uri)

	return u, err
}

// IsValidJSON checks if the text is valid JSON.
func IsValidJSON(text string) bool {
	var msg json.RawMessage

	return json.Unmarshal([]byte(text), &msg) == nil
}

// GetDataFromURL parses specific url fields and returns their values.
func GetDataFromURL(uri string) url.Values {
	u, err := IsValidURL(uri)
	if err != nil {
		return nil
	}

	return u.Query()
}

// GetLink returns invidious and youtube links.
func GetLinks(info SearchResult) (string, string) {
	var linkparam string

	invlink := "https://" + GetClient().SelectedInstance()
	ytlink := "https://youtube.com"

	switch info.Type {
	case "video":
		linkparam = "/watch?v=" + info.VideoID

	case "playlist":
		linkparam = "/playlist?list=" + info.PlaylistID

	case "channel":
		linkparam = "/channel/" + info.AuthorID
	}

	invlink += linkparam
	ytlink += linkparam

	return invlink, ytlink
}

// GetVPIDFromURL gets the video/playlist ID from a URL.
func GetVPIDFromURL(uri string) (string, string, error) {
	mediaURL := uri

	if !strings.HasPrefix(uri, "https://") {
		mediaURL = "https://" + uri
	}

	u, err := IsValidURL(mediaURL)
	if err != nil {
		return "", "", err
	}

	if strings.Contains(uri, "youtu.be") {
		return strings.TrimLeft(u.Path, "/"), "video", nil
	} else if strings.Contains(uri, "watch?v=") {
		return u.Query().Get("v"), "video", nil
	} else if strings.Contains(uri, "playlist?list=") {
		return u.Query().Get("list"), "playlist", nil
	}

	if strings.Index(uri, "/channel") != -1 ||
		(strings.HasPrefix(uri, "UC") && len(uri) >= 24) {
		return "", "", fmt.Errorf("The URL or ID is a channel")
	}

	if strings.HasPrefix(uri, "PL") && len(uri) >= 34 {
		return uri, "playlist", nil
	}

	return uri, "video", nil
}

// GetHostname gets the hostname of the given URL.
func GetHostname(hostURL string) string {
	uri, _ := url.Parse(hostURL)

	return uri.Hostname()
}

// GetUnixTimeAfter returns the Unix time after the
// given number of years.
func GetUnixTimeAfter(years int) int64 {
	return time.Now().AddDate(years, 0, 0).Unix()
}

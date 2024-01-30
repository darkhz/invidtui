package utils

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/darkhz/invidtui/resolver"
	urlverify "github.com/davidmytton/url-verifier"
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
	var sb strings.Builder

	for i, n := range []int{
		1000000000,
		1000000,
		1000,
	} {
		if num >= n {
			fmt.Fprintf(&sb, "%.0f%c", float64(num)/float64(n), "BMK"[i])
			break
		}
	}

	if sb.Len() == 0 {
		fmt.Fprintf(&sb, "%d", num)
	}

	return sb.String()
}

// ConvertDurationToSeconds converts a "hh:mm:ss" string to seconds.
func ConvertDurationToSeconds(duration string) int64 {
	if duration == "" {
		return 0
	}

	dursplit := strings.Split(duration, ":")
	length := len(dursplit)
	switch {
	case length <= 1:
		return 0

	case length == 2:
		dursplit = append([]string{"00"}, dursplit...)
	}

	for i, v := range []string{"h", "m", "s"} {
		dursplit[i] = dursplit[i] + v
	}

	d, _ := time.ParseDuration(strings.Join(dursplit, ""))

	return int64(d.Seconds())
}

// SanitizeCookie sanitizes and returns the provided cookie.
// This is used to avoid the logging present in the net/http package.
// https://cs.opensource.google/go/go/+/refs/tags/go1.20.5:src/net/http/cookie.go;l=428
func SanitizeCookie(cookie string) string {
	valid := func(b byte) bool {
		return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
	}

	ok := true
	for i := 0; i < len(cookie); i++ {
		if valid(cookie[i]) {
			continue
		}

		ok = false
		break
	}
	if ok {
		return cookie
	}

	buf := make([]byte, 0, len(cookie))
	for i := 0; i < len(cookie); i++ {
		if b := cookie[i]; valid(b) {
			buf = append(buf, b)
		}
	}

	return string(buf)
}

// Deduplicate removes duplicate values from the slice.
func Deduplicate(values []string) []string {
	encountered := make(map[string]int, len(values))
	for v := range values {
		encountered[values[v]] = v
	}

	i := 0
	keys := make([]int, len(encountered))
	for _, pos := range encountered {
		keys[i] = pos
		i++
	}
	sort.Ints(keys)

	dedup := make([]string, len(keys))
	for key, pos := range keys {
		dedup[key] = values[pos]
	}

	return dedup
}

// DecodeSessionData decodes session data from a playlist item.
func DecodeSessionData(data string, apply func(prop, value string)) bool {
	values := strings.Split(data, ",")
	if len(values) == 0 {
		return false
	}

	for _, value := range values {
		prop := strings.Split(value, "=")
		if len(prop) != 2 {
			continue
		}

		apply(prop[0], prop[1])
	}

	return true
}

// TrimPath cleans and returns a directory path.
func TrimPath(testPath string, cdBack bool) string {
	testPath = filepath.Clean(testPath)

	if cdBack {
		testPath = filepath.Dir(testPath)
	}

	return filepath.FromSlash(testPath)
}

// IsValidURL checks if a URL is valid.
func IsValidURL(uri string) (*url.URL, error) {
	v, err := urlverify.NewVerifier().Verify(uri)
	if err != nil {
		return nil, err
	}
	if !v.IsURL {
		return nil, fmt.Errorf("invalid URL")
	}

	return url.Parse(uri)
}

// IsValidJSON checks if the text is valid JSON.
func IsValidJSON(text string) bool {
	var msg struct{}

	return resolver.DecodeJSONBytes([]byte(text), &msg) == nil
}

// GetDataFromURL parses specific url fields and returns their values.
func GetDataFromURL(uri string) url.Values {
	u, err := IsValidURL(uri)
	if err != nil {
		return nil
	}

	return u.Query()
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

	if strings.Contains(uri, "/channel") ||
		(strings.HasPrefix(uri, "UC") && len(uri) >= 24) {
		return "", "", fmt.Errorf("the URL or ID is a channel")
	}

	if strings.HasPrefix(uri, "PL") && len(uri) >= 34 {
		return uri, "playlist", nil
	}

	return uri, "video", nil
}

// GetHostname gets the hostname of the given URL.
func GetHostname(hostURL string) string {
	uri, _ := url.Parse(hostURL)

	hostname := uri.Hostname()
	if hostname == "" {
		return hostURL
	}

	return hostname
}

// GetUnixTimeAfter returns the Unix time after the
// given number of years.
func GetUnixTimeAfter(years int) int64 {
	return time.Now().AddDate(years, 0, 0).Unix()
}

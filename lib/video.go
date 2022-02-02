package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// VideoResult stores the video data.
type VideoResult struct {
	Title           string       `json:"title"`
	Author          string       `json:"author"`
	VideoID         string       `json:"videoId"`
	LengthSeconds   int          `json:"lengthSeconds"`
	AdaptiveFormats []FormatData `json:"adaptiveFormats"`
}

// FormatData stores the media format data.
type FormatData struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	Itag       string `json:"itag"`
	Resolution string `json:"resolution,omitempty"`
}

const videoFields = "?fields=title,videoId,author,publishedText,lengthSeconds,adaptiveFormats"

// Video gets the video with the given ID and returns a VideoResult.
func (c *Client) Video(id string) (VideoResult, error) {
	var result VideoResult

	res, err := c.ClientRequest(context.Background(), "videos/"+id+videoFields)
	if err != nil {
		return VideoResult{}, err
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		return VideoResult{}, err
	}

	return result, nil
}

// LoadVideo takes a VideoResult, determines whether to play
// video or just audio (according to the audio parameter), and
// appropriately loads the URLs into mpv.
func LoadVideo(video VideoResult, audio bool) error {
	var err error
	var audioUrl, videoUrl string

	videoUrl, audioUrl = getVideoByItag(video, audio)

	if audio && audioUrl == "" {
		return fmt.Errorf("Could not find an audio stream")
	}

	if !audio && videoUrl == "" {
		return fmt.Errorf("Could not find a video stream")
	}

	// A data parameter is appended to audioUrl/videoUrl so that
	// updatePlaylist() can display media data.
	// MPV does not return certain track data like author and duration.
	titleparam := "&title=" + url.QueryEscape(video.Title)
	titleparam += "&author=" + url.QueryEscape(video.Author)
	titleparam += "&length=" + url.QueryEscape(FormatDuration(video.LengthSeconds))

	if audio {
		_, err = IsValidURL(audioUrl + titleparam)
		if err != nil {
			return fmt.Errorf("Could not find an audio stream")
		}

		audioUrl += titleparam

		err = GetMPV().LoadFile(
			video.Title,
			video.LengthSeconds,
			audioUrl)

	} else {
		_, err = IsValidURL(videoUrl + titleparam)
		if err != nil {
			return fmt.Errorf("Could not find a video stream")
		}

		videoUrl += titleparam

		err = GetMPV().LoadFile(
			video.Title,
			video.LengthSeconds,
			videoUrl, audioUrl)
	}
	if err != nil {
		return err
	}

	return nil
}

// getVideoByItag gets the appropriate itag of the video format, and
// returns a video and audio url using getLatestURL().
func getVideoByItag(video VideoResult, audio bool) (string, string) {
	var videoUrl, audioUrl string

	// For video streams, itag 22 is 720p and itag 18 is 360p
	// as of now in most invidious instances, may change.
	if !audio && (*videoResolution == "720p" || *videoResolution == "360p") {
		var itag22, itag18 bool

		for _, format := range video.AdaptiveFormats {
			if itag22 || itag18 {
				break
			}

			switch format.Itag {
			case "22":
				itag22 = true

			case "18":
				itag18 = true
			}
		}

		switch {
		case itag22:
			videoUrl = getLatestURL(video.VideoID, "22")
		case itag18:
			videoUrl = getLatestURL(video.VideoID, "18")
		}

		// audioUrl is blank since the audio stream is
		// is already merged along with the video in
		// videoUrl.
		if videoUrl != "" {
			return videoUrl, audioUrl
		}
	}

	videoUrl, audioUrl = loopFormats(
		audio, video,

		func(v VideoResult, f FormatData) string {
			return getLatestURL(v.VideoID, f.Itag)
		},

		func(v VideoResult, f FormatData) string {
			return videoWithResolution(v, "itag")
		},
	)

	return videoUrl, audioUrl
}

// getVideoByFormatURL returns a URL from a VideoResult's AdaptiveFormats.
func getVideoByFormatURL(video VideoResult, audio bool) (string, string) {
	var videoUrl, audioUrl string

	videoUrl, audioUrl = loopFormats(
		audio, video,

		func(v VideoResult, f FormatData) string {
			return f.URL
		},

		func(v VideoResult, f FormatData) string {
			return videoWithResolution(v, "url")
		},
	)

	return videoUrl, audioUrl
}

// videoWithResolution returns a video URL that corresponds to the
// videoResolution setting (passed via command line option --video-res=).
func videoWithResolution(video VideoResult, vtype string) string {
	var prevData string

	vq := *videoResolution

	for _, format := range video.AdaptiveFormats {
		q := format.Resolution
		if len(q) <= 0 {
			continue
		}

		switch vtype {
		case "url":
			if q == vq {
				return format.URL
			}

			prevData = format.URL

		case "itag":
			if q == vq {
				return getLatestURL(video.VideoID, format.Itag)
			}

			prevData = getLatestURL(video.VideoID, format.Itag)
		}
	}

	return prevData
}

// loopFormats loops over a video's AdaptiveFormats data and gets the
// audio/video URL according to the values returned by afunc/vfunc.
func loopFormats(
	audio bool, video VideoResult,
	afunc, vfunc func(video VideoResult, format FormatData) string,
) (string, string) {
	var ftype, videoUrl, audioUrl string

	for _, format := range video.AdaptiveFormats {
		v := strings.Split(format.Type, ";")
		p := strings.Split(v[0], "/")

		if (audio && audioUrl != "") || (!audio && videoUrl != "") {
			break
		}

		if ftype == "" {
			ftype = p[1]
		}

		if p[1] == ftype {
			if p[0] == "audio" {
				audioUrl = afunc(video, format)
			} else if p[0] == "video" {
				videoUrl = vfunc(video, format)
			}
		}
	}

	return videoUrl, audioUrl
}

// getLatestURL appends the latest_version query to the current client's host URL.
// For example: https://invidious.snopyta.org/latest_version?id=mWDOxRWcoPE&itag=22&local=true
func getLatestURL(id, itag string) string {
	host := GetClient().host

	idstr := "id=" + id
	itagstr := "&itag=" + itag

	return host + "/latest_version?" + idstr + itagstr + "&local=true"
}

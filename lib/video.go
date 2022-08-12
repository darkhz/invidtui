package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/etherlabsio/go-m3u8/m3u8"
)

// VideoResult stores the video data.
type VideoResult struct {
	Title           string       `json:"title"`
	Author          string       `json:"author"`
	VideoID         string       `json:"videoId"`
	HlsURL          string       `json:"hlsUrl"`
	LengthSeconds   int          `json:"lengthSeconds"`
	LiveNow         bool         `json:"liveNow"`
	FormatStreams   []FormatData `json:"formatStreams"`
	AdaptiveFormats []FormatData `json:"adaptiveFormats"`
}

// FormatData stores the media format data.
type FormatData struct {
	Type       string `json:"type"`
	URL        string `json:"url"`
	Itag       string `json:"itag"`
	Resolution string `json:"resolution,omitempty"`
}

var (
	videoCtx     context.Context
	videoCancel  context.CancelFunc
	videoCtxLock sync.Mutex
)

const videoFields = "?fields=title,videoId,author,hlsUrl,publishedText,lengthSeconds,adaptiveFormats,liveNow"

// Video gets the video with the given ID and returns a VideoResult.
func (c *Client) Video(id string) (VideoResult, error) {
	var result VideoResult

	if videoCtx == nil {
		return VideoResult{}, fmt.Errorf("No video context found")
	}

	res, err := c.ClientRequest(videoCtx, "videos/"+id+videoFields)
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

// LoadVideo takes a video ID, determines whether to play
// video or just audio (according to the audio parameter), and
// appropriately loads the URLs into mpv.
func LoadVideo(id string, audio bool) error {
	var err error
	var liveaudio bool
	var mtype, lentext, audioUrl, videoUrl string

	video, err := GetClient().Video(id)
	if err != nil {
		return err
	}

	if audio {
		mtype = "Audio"
	} else {
		mtype = "Video"
	}

	if video.LiveNow {
		liveaudio = audio && video.LiveNow
		audio = false
		lentext = "Live"
		videoUrl, audioUrl = getLiveVideo(video, audio)
	} else {
		lentext = FormatDuration(video.LengthSeconds)
		videoUrl, audioUrl = getVideoByItag(video, audio)
	}

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
	titleparam += "&mediatype=" + url.QueryEscape(mtype)
	titleparam += "&length=" + url.QueryEscape(lentext)

	if audio {
		_, err = IsValidURL(audioUrl + titleparam)
		if err != nil {
			return fmt.Errorf("Could not find an audio stream")
		}

		audioUrl += titleparam

		err = GetMPV().LoadFile(
			video.Title,
			video.LengthSeconds,
			liveaudio,
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
			liveaudio,
			videoUrl, audioUrl)
	}
	if err != nil {
		return err
	}

	return nil
}

// VideoNewCtx renews the video's context.
func VideoNewCtx() {
	videoCtxLock.Lock()
	defer videoCtxLock.Unlock()

	videoCtx, videoCancel = context.WithCancel(context.Background())
}

// VideoCancel cancels the video's context.
func VideoCancel() {
	videoCtxLock.Lock()
	defer videoCtxLock.Unlock()

	if videoCtx != nil {
		videoCancel()
	}
}

// refreshLiveURL gets the video ID from an expired live video URL,
// and loads the latest URL for the live video.
func refreshLiveURL(uri string, audio bool) bool {
	var id string

	// Split the uri parameters.
	uriSplit := strings.Split(uri, "/")
	for i, v := range uriSplit {
		if v == "expire" {
			// Return if the uri is not expired.
			exptime, err := strconv.ParseInt(uriSplit[i+1], 10, 64)
			if err == nil && time.Now().Unix() < exptime {
				return false
			}
		}

		if v == "id" {
			// Get the id value from the uri path.
			id = strings.Split(uriSplit[i+1], ".")[0]
			break
		}
	}

	VideoNewCtx()

	LoadVideo(id, audio)

	return true
}

// getLiveVideo gets the hls playlist, parses and finds the appropriate
// live video stream.
func getLiveVideo(video VideoResult, audio bool) (string, string) {
	var videoUrl, audioUrl string

	if video.HlsURL == "" {
		return "", ""
	}

	url, _ := IsValidURL(video.HlsURL)
	res, err := GetClient().GetRequest(context.Background(), url.RequestURI())
	if err != nil {
		return "", ""
	}
	defer res.Body.Close()

	pl, err := m3u8.Read(res.Body)
	if err != nil {
		return "", ""
	}

	for _, p := range pl.Playlists() {
		height := strconv.Itoa(p.Resolution.Height) + "p"

		// Since the retrieved HLS playlist is sorted in ascending order of resolutions,
		// for the audio stream, we grab the first stream (with the lowest quality),
		// and instruct mpv not to play video for the audio stream. For the video stream,
		// we grab the stream where the playlist entry's resolution and the required
		// resolution are equal.
		if audio || (!audio && height == videoResolution) {
			url, _ := IsValidURL(p.URI)
			videoUrl = "https://manifest.googlevideo.com" + url.RequestURI() + "/?"

			break
		}
	}

	return videoUrl, audioUrl
}

// getVideoByItag gets the appropriate itag of the video format, and
// returns a video and audio url using getLatestURL().
func getVideoByItag(video VideoResult, audio bool) (string, string) {
	var videoUrl, audioUrl string

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
// videoResolution setting (passed via command line option --video-res=), and
// the resolutions listed in a video's AdaptiveFormats.
func videoWithResolution(video VideoResult, vtype string) string {
	var prevData string

	vq := videoResolution

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

	// For videos, we loop through FormatStreams first and get the videoUrl.
	// This works mainly for 720p, 360p and 144p video streams.
	if !audio {
		for _, format := range video.FormatStreams {
			if format.Resolution == videoResolution {
				videoUrl = getLatestURL(video.VideoID, format.Itag)
				return videoUrl, audioUrl
			}
		}
	}

	// If the required resolution wasn't found in FormatStreams, we loop through
	// AdaptiveFormats and get a video of the required resolution, along with the
	// audio stream so that MPV can merge them and play. Or if only audio is required,
	// return a blank videoUrl and a non-empty audioUrl.
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

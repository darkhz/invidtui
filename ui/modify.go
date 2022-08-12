package ui

import (
	"sync"

	"github.com/darkhz/invidtui/lib"
	"golang.org/x/sync/semaphore"
)

var (
	modifyMap     map[string]*semaphore.Weighted
	modifyMapLock sync.Mutex
)

// Modify retrieves the reference from the table in focus, determines
// its type and runs the appropriate modification handler.
func Modify(add bool) {
	var err error
	var info lib.SearchResult

	App.QueueUpdateDraw(func() {
		info, err = getListReference()
	})
	if err != nil {
		return
	}

	modifyMapLock.Lock()
	if modifyMap == nil {
		modifyMap = make(map[string]*semaphore.Weighted)
		for _, mtype := range []string{
			"video",
			"playlist",
			"channel",
		} {
			modifyMap[mtype] = semaphore.NewWeighted(1)
		}
	}

	lock := modifyMap[info.Type]
	modifyMapLock.Unlock()

	if !lock.TryAcquire(1) {
		InfoMessage("Add/remove in progress for "+info.Type, false)
		return
	}
	defer lock.Release(1)

	switch info.Type {
	case "video":
		modifyPlaylistVideo(info, add)

	case "playlist":
		modifyPlaylist(info, add)

	case "channel":
		modifyChannelSubscription(info, add)
	}
}

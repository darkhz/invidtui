package cmd

import (
	"github.com/gdamore/tcell/v2"
)

// Keybinding describes a keybinding.
type Keybinding struct {
	Key  tcell.Key
	Rune rune
	Mod  tcell.ModMask
}

var (
	// OperationKeys matches the operation name (or the menu ID)
	// with the keybinding.
	OperationKeys = map[string]map[string]Keybinding{
		"App": {
			"Menu":            {tcell.KeyRune, 'm', tcell.ModAlt},
			"Dashboard":       {tcell.KeyCtrlD, ' ', tcell.ModCtrl},
			"Suspend":         {tcell.KeyCtrlZ, ' ', tcell.ModCtrl},
			"Cancel":          {tcell.KeyCtrlX, ' ', tcell.ModCtrl},
			"DownloadView":    {tcell.KeyRune, 'Y', tcell.ModNone},
			"DownloadOptions": {tcell.KeyRune, 'y', tcell.ModNone},
			"InstancesList":   {tcell.KeyRune, 'o', tcell.ModNone},
			"Quit":            {tcell.KeyRune, 'Q', tcell.ModNone},
			"Ctrl-C":          {tcell.KeyCtrlC, ' ', tcell.ModCtrl},
		},
		"Start": {
			"Search": {tcell.KeyRune, '/', tcell.ModNone},
		},
		"Files": {
			"CDFwd":        {tcell.KeyRight, ' ', tcell.ModNone},
			"CDBack":       {tcell.KeyLeft, ' ', tcell.ModNone},
			"ToggleHidden": {tcell.KeyCtrlG, ' ', tcell.ModCtrl},
		},
		"Comments": {
			"Replies": {tcell.KeyRune, ' ', tcell.ModNone},
			"Exit":    {tcell.KeyEscape, ' ', tcell.ModNone},
		},
		"Downloads": {
			"Select": {tcell.KeyEnter, ' ', tcell.ModNone},
			"Cancel": {tcell.KeyRune, 'x', tcell.ModNone},
			"Exit":   {tcell.KeyEscape, ' ', tcell.ModNone},
		},
		"Playlist": {
			"Comments":           {tcell.KeyRune, 'C', tcell.ModNone},
			"Link":               {tcell.KeyRune, ';', tcell.ModNone},
			"AddToPlaylist":      {tcell.KeyRune, '+', tcell.ModNone},
			"RemoveFromPlaylist": {tcell.KeyRune, '_', tcell.ModNone},
			"LoadMore":           {tcell.KeyEnter, ' ', tcell.ModNone},
			"Exit":               {tcell.KeyEscape, ' ', tcell.ModNone},
			"DownloadOptions":    {tcell.KeyRune, 'y', tcell.ModNone},
		},
		"Channel": {
			"Switch":          {tcell.KeyTab, ' ', tcell.ModNone},
			"LoadMore":        {tcell.KeyEnter, ' ', tcell.ModNone},
			"Exit":            {tcell.KeyEscape, ' ', tcell.ModNone},
			"Query":           {tcell.KeyRune, '/', tcell.ModNone},
			"Playlist":        {tcell.KeyRune, 'i', tcell.ModNone},
			"AddTo":           {tcell.KeyRune, '+', tcell.ModNone},
			"Comments":        {tcell.KeyRune, 'C', tcell.ModNone},
			"Link":            {tcell.KeyRune, ';', tcell.ModNone},
			"DownloadOptions": {tcell.KeyRune, 'y', tcell.ModNone},
		},
		"Dashboard": {
			"Switch":           {tcell.KeyTab, ' ', tcell.ModNone},
			"Exit":             {tcell.KeyEscape, ' ', tcell.ModNone},
			"Reload":           {tcell.KeyCtrlD, ' ', tcell.ModNone},
			"LoadMore":         {tcell.KeyEnter, ' ', tcell.ModNone},
			"AddVideo":         {tcell.KeyRune, '+', tcell.ModNone},
			"Comments":         {tcell.KeyRune, 'C', tcell.ModNone},
			"Link":             {tcell.KeyRune, ';', tcell.ModNone},
			"Playlist":         {tcell.KeyRune, 'i', tcell.ModNone},
			"Create":           {tcell.KeyRune, 'c', tcell.ModNone},
			"Edit":             {tcell.KeyRune, 'e', tcell.ModNone},
			"Remove":           {tcell.KeyRune, '_', tcell.ModNone},
			"ChannelVideos":    {tcell.KeyRune, 'u', tcell.ModNone},
			"ChannelPlaylists": {tcell.KeyRune, 'U', tcell.ModNone},
		},
		"Search": {
			"Start":             {tcell.KeyEnter, ' ', tcell.ModNone},
			"Exit":              {tcell.KeyEscape, ' ', tcell.ModNone},
			"Suggestions":       {tcell.KeyTab, ' ', tcell.ModNone},
			"SwitchMode":        {tcell.KeyCtrlE, ' ', tcell.ModCtrl},
			"HistoryReverse":    {tcell.KeyUp, ' ', tcell.ModNone},
			"HistoryForward":    {tcell.KeyDown, ' ', tcell.ModNone},
			"SuggestionReverse": {tcell.KeyUp, ' ', tcell.ModCtrl},
			"SuggestionForward": {tcell.KeyDown, ' ', tcell.ModCtrl},
			"Parameters":        {tcell.KeyRune, 'e', tcell.ModAlt},
			"Query":             {tcell.KeyRune, '/', tcell.ModNone},
			"Playlist":          {tcell.KeyRune, 'i', tcell.ModNone},
			"ChannelVideos":     {tcell.KeyRune, 'u', tcell.ModNone},
			"ChannelPlaylists":  {tcell.KeyRune, 'U', tcell.ModNone},
			"Comments":          {tcell.KeyRune, 'C', tcell.ModNone},
			"Link":              {tcell.KeyRune, ';', tcell.ModNone},
			"AddVideo":          {tcell.KeyRune, '+', tcell.ModNone},
			"DownloadOptions":   {tcell.KeyRune, 'y', tcell.ModNone},
		},
		"Player": {
			"Open":           {tcell.KeyCtrlO, ' ', tcell.ModCtrl},
			"History":        {tcell.KeyRune, 'h', tcell.ModAlt},
			"SeekForward":    {tcell.KeyRight, ' ', tcell.ModNone},
			"SeekBackward":   {tcell.KeyLeft, ' ', tcell.ModNone},
			"QueueAudio":     {tcell.KeyRune, 'a', tcell.ModNone},
			"QueueVideo":     {tcell.KeyRune, 'v', tcell.ModNone},
			"PlayAudio":      {tcell.KeyRune, 'A', tcell.ModNone},
			"PlayVideo":      {tcell.KeyRune, 'V', tcell.ModNone},
			"Queue":          {tcell.KeyRune, 'q', tcell.ModNone},
			"AudioURL":       {tcell.KeyRune, 'b', tcell.ModNone},
			"VideoURL":       {tcell.KeyRune, 'B', tcell.ModNone},
			"Stop":           {tcell.KeyRune, 'S', tcell.ModNone},
			"ToggleLoop":     {tcell.KeyRune, 'l', tcell.ModNone},
			"ToggleShuffle":  {tcell.KeyRune, 's', tcell.ModNone},
			"ToggleMute":     {tcell.KeyRune, 'm', tcell.ModNone},
			"Prev":           {tcell.KeyRune, '<', tcell.ModNone},
			"Next":           {tcell.KeyRune, '>', tcell.ModNone},
			"VolumeIncrease": {tcell.KeyRune, '=', tcell.ModNone},
			"VolumeDecrease": {tcell.KeyRune, '-', tcell.ModNone},
			"Pause":          {tcell.KeyRune, ' ', tcell.ModNone},
		},
		"Queue": {
			"Play":   {tcell.KeyEnter, ' ', tcell.ModNone},
			"Save":   {tcell.KeyCtrlS, ' ', tcell.ModCtrl},
			"Append": {tcell.KeyCtrlA, ' ', tcell.ModCtrl},
			"Delete": {tcell.KeyRune, 'd', tcell.ModNone},
			"Move":   {tcell.KeyRune, 'M', tcell.ModNone},
			"Stop":   {tcell.KeyRune, 'S', tcell.ModNone},
			"Exit":   {tcell.KeyEscape, ' ', tcell.ModNone},
		},
		"History": {
			"Query":            {tcell.KeyRune, '/', tcell.ModNone},
			"ChannelVideos":    {tcell.KeyRune, 'u', tcell.ModNone},
			"ChannelPlaylists": {tcell.KeyRune, 'U', tcell.ModNone},
			"Exit":             {tcell.KeyEscape, ' ', tcell.ModNone},
		},
	}

	// Keys match the keybinding to the operation name.
	Keys map[string]map[Keybinding]string
)

// OperationKey returns the keybinding associated with
// the provided keyID and operation name.
func OperationKey(keyID, operation string) Keybinding {
	return OperationKeys[keyID][operation]
}

// KeyOperation returns the operation name for the provided keyID
// and the keyboard event.
func KeyOperation(keyID string, event *tcell.EventKey) string {
	if Keys == nil {
		Keys = make(map[string]map[Keybinding]string)
		for keyType, keys := range OperationKeys {
			Keys[keyType] = make(map[Keybinding]string)

			for keyName, key := range keys {
				Keys[keyType][key] = keyName
			}
		}
	}

	ch := event.Rune()
	if event.Key() != tcell.KeyRune {
		ch = ' '
	}

	operation, ok := Keys[keyID][Keybinding{event.Key(), ch, event.Modifiers()}]
	if !ok {
		return ""
	}

	return operation
}

package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
)

func yt(s slack.SlashCommand, w http.ResponseWriter) {
	// TODO: would be nice to handle https://www.youtube.com/c/STLChessClub/videos
	// style links too
	var msg string

	if strings.Contains(s.Text, "playlist") {
		msg = fmt.Sprintf(
			"/feed add %s",
			"https://www.youtube.com/feeds/videos.xml?playlist_id="+s.Text[38:],
		)
	} else if strings.Contains(s.Text, "channel") {
		splitText := strings.Split(s.Text, "/")
		ytChannelID := splitText[len(splitText)-1]
		msg = fmt.Sprintf(
			"/feed add %s",
			"https://www.youtube.com/feeds/videos.xml?channel_id="+ytChannelID,
		)

	} else {
		msg = fmt.Sprintf("url format not recognised for %s", s.Text)
	}
	logErrMsgSlack(w, msg)
	return
}

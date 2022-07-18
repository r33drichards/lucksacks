package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/slack-go/slack"
)

func ttv(s slack.SlashCommand, w http.ResponseWriter) {
	splitText := strings.Split(s.Text, "/")
	twitchChannelID := splitText[len(splitText)-1]
	msg := fmt.Sprintf("/feed add https://twitchrss.appspot.com/vod/%s", twitchChannelID)
	logErrMsgSlack(w, msg)
}

package main

import (
	"encoding/base64"

	"github.com/slack-go/slack"
)

func b64(s slack.SlashCommand) string {
	return base64.StdEncoding.EncodeToString([]byte(s.Text))
}

package main

import (
	"crypto/sha256"
	"encoding/hex"

	"net/http"

	"github.com/slack-go/slack"
)

func mysha256(s slack.SlashCommand, w http.ResponseWriter) {
	h := sha256.New()
	h.Write([]byte(s.Text))
	msg := hex.EncodeToString(h.Sum(nil))
	logErrMsgSlack(w, msg)
}

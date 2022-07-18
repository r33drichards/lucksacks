package main

import (
	"io"
	"net/http"
	"strconv"

	"github.com/slack-go/slack"
)

func qrngSlackCommand(w http.ResponseWriter, s slack.SlashCommand, url string) {
	// fetch random 2 digit hex from https://qrng.anu.edu.au/
	var randNumSourceUrl = url
	var numRequests int
	var err error

	if s.Text == "" {
		numRequests = 1
	} else {
		numRequests, err = strconv.Atoi(s.Text)
		if err != nil {
			msg := "invalid input: " + s.Text
			logErrMsgSlack(w, msg)
		}

	}

	i := 0
	msg := ""
	for i < numRequests {
		resp, err := http.Get(randNumSourceUrl)
		if err != nil {
			msg := "error fetching " + randNumSourceUrl
			logErrMsgSlack(w, msg)
			return
		} else {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				logErrMsgSlack(w, "error reading body from "+randNumSourceUrl)
				return
			} else {
				msg = msg + string(body)
			}
		}
		i = i + 1
	}
	logErrMsgSlack(w, msg)

}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/slack-go/slack"
)

func rcolor(s slack.SlashCommand, w http.ResponseWriter) {
	// fetch random color from https://qrng.anu.edu.au/
	var randNumSourceUrl = "https://qrng.anu.edu.au/wp-content/plugins/colours-plugin/get_one_colour.php"
	var slackParams *slack.Msg

	resp, err := http.Get(randNumSourceUrl)
	if err != nil {
		slackParams = &slack.Msg{Text: "error fetching " + randNumSourceUrl}
		b, err := json.Marshal(slackParams)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			log.Println(err)
		}
		return
	} else {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			slackParams = &slack.Msg{Text: "error reading body from " + randNumSourceUrl}
			b, err := json.Marshal(slackParams)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}
			return
		} else {
			colorString := string(body)
			msg := fmt.Sprintf("%s\nhttps://coolors.co/%s", colorString, colorString)
			slackParams = &slack.Msg{Text: msg}

			b, err := json.Marshal(slackParams)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

package main

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

func choose(s slack.SlashCommand, w http.ResponseWriter) {
	var vals []string
	if strings.Contains(s.Text, "\"") {

		ss := strings.Split(s.Text, "\"")
		for _, s := range ss {
			if s != "" && s != " " {
				vals = append(vals, s)
			}
		}
	} else {
		vals = strings.Split(s.Text, " ")
	}
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	source := rand.NewSource(time.Now().Unix())
	r := rand.New(source) // initialize local pseudorandom generator

	var msg string
	if len(vals) > 0 {
		msg = vals[r.Intn(len(vals))]
	} else {
		msg = "nothing to choose from"
	}
	logErrMsgSlack(w, msg)

}

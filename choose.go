package main

import (
	"math/rand"
	"net/http"
	"strconv"
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

type weightedChoice struct {
	choice string
	weight int
}

func atoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 1
	}
	return i
}

func wchoose(s slack.SlashCommand, w http.ResponseWriter) {
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

	var choices []weightedChoice
	for _, v := range vals {
		if strings.Contains(v, ":") {
			ss := strings.Split(v, ":")
			choices = append(choices, weightedChoice{choice: ss[0], weight: atoi(ss[1])})
		} else {
			choices = append(choices, weightedChoice{choice: v, weight: 1})
		}
	}

	choicesStr := []string{}

	for _, c := range choices {
		for i := 0; i < c.weight; i++ {
			choicesStr = append(choicesStr, c.choice)
		}
	}

	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	source := rand.NewSource(time.Now().Unix())
	r := rand.New(source) // initialize local pseudorandom generator

	var msg string
	if len(vals) > 0 {
		msg = choicesStr[r.Intn(len(choicesStr))]
	} else {
		msg = "nothing to choose from"
	}
	logErrMsgSlack(w, msg)

}

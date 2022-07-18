package main

import (
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/slack-go/slack"
)

func roll(s slack.SlashCommand, w http.ResponseWriter) {
	if s.Text == "" || s.Text == "help" {
		msg := `returns a random number between 1 and N

Example:

/roll 6

-> 1
`
		logErrMsgSlack(w, msg)
		return
	}
	i, err := strconv.Atoi(s.Text)
	if err != nil {
		logErrMsgSlack(w, "Invalid input: "+s.Text)
		return
	}
	if i <= 0 {
		logErrMsgSlack(w, "provide integer greater than 0")
		return
	}
	rand.Seed(time.Now().UnixNano())
	randInt := rand.Intn(i) + 1
	err = msgSlack(strconv.Itoa(randInt), w)
	if err != nil {
		log.Println(err)
	}
}

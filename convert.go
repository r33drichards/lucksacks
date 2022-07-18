package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	u "github.com/bcicen/go-units"

	"github.com/slack-go/slack"
)

func convert(s slack.SlashCommand, w http.ResponseWriter) {
	vals := strings.Split(s.Text, " ")
	from, err := u.Find(vals[1])
	if err != nil {
		logErrMsgSlack(w, vals[1]+" not valid unit")
		return
	}
	to, err := u.Find(vals[2])
	if err != nil {
		logErrMsgSlack(w, vals[2]+" not valid unit")
		return
	}

	val, err := strconv.ParseFloat(vals[0], 64)
	if err != nil {
		logErrMsgSlack(w, vals[0]+" failed to parse")
		return
	}

	message, err := u.ConvertFloat(val, from, to)

	if err != nil {
		logErrMsgSlack(w, "failed to preform conversion for: "+vals[0]+" "+vals[1]+" "+" "+vals[2])
		return

	}
	logErrMsgSlack(w, fmt.Sprintf("%s %ss is %s", vals[0], from.Name, message.String()))
}

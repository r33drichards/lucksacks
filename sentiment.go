package main

import (
	"net/http"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	"github.com/slack-go/slack"
)

func unquoteCodePoint(s string) (string, error) {
	n, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return "", err
	}
	r := rune(n)
	return string(r), nil

}

func sentiment(s slack.SlashCommand, w http.ResponseWriter) {
	// s.Text
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-2"),
	}))

	// Create a Comprehend client from just a session.
	client := comprehend.New(sess)

	params := comprehend.DetectSentimentInput{}
	params.SetLanguageCode("en")
	params.SetText(s.Text)

	req, resp := client.DetectSentimentRequest(&params)

	err := req.Send()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// https://stackoverflow.com/questions/55700149/print-emoji-from-unicode-literal-loaded-from-file
	sentEmoji := make(map[string]string)
	frown, _ := unquoteCodePoint("\\U00002639")
	sentEmoji["NEGATIVE"] = frown
	grin, _ := unquoteCodePoint("\\U0001f600")
	sentEmoji["POSITIVE"] = grin
	upsideDownFace, _ := unquoteCodePoint("\\U0001f643")
	sentEmoji["MIXED"] = upsideDownFace
	expressionless, _ := unquoteCodePoint("\\U0001f611")
	sentEmoji["NEUTRAL"] = expressionless

	msg := sentEmoji[*resp.Sentiment] + " " + *resp.Sentiment
	logErrMsgSlack(w, msg)
}

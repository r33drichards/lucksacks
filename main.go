package main

import (
	_ "embed"
	"encoding/json"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"

	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/getsentry/sentry-go"

	log "github.com/sirupsen/logrus"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

//go:embed docs/tz.md
var TZ_HELP string

func msgSlack(msg string, w http.ResponseWriter) error {
	params := &slack.Msg{Text: msg}
	b, err := json.Marshal(params)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(b)
	return err

}

func logErrMsgSlack(w http.ResponseWriter, msg string) {
	err := msgSlack(msg, w)
	sentry.CaptureException(err)
	if err != nil {
		log.Println(err)
	}
}

func main() {
	anthropicClient := anthropic.NewClient()

	messageStore := NewSlackMessageStore(NewLLM(anthropicClient))

	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://7a6c1d7fa62d70dffc54d0d4d8a92efb@o4507134751408128.ingest.us.sentry.io/4509460668809216",
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	log.SetFormatter(&log.JSONFormatter{})

	log.WithFields(log.Fields{"string": "foo", "int": 1, "float": 1.1}).Info("My first event from golang to stdout")

	api := slack.New(os.Getenv("SLACK_BOT_TOKEN"))
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")

	http.HandleFunc("/slash", func(w http.ResponseWriter, r *http.Request) {

		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Body = ioutil.NopCloser(io.TeeReader(r.Body, &verifier))
		s, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err = verifier.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch s.Command {

		case "/anagram":
			anagram(s, api, w)
			return
		case "/convert":
			convert(s, w)
			return
		case "/tz":
			tz(s, w)
			return
		case "/yt":
			yt(s, w)
			return
		case "/ttv":
			ttv(s, w)
			return
		case "/roll":
			roll(s, w)
			return
		case "/wchoose":
			wchoose(s, w)
			return
		case "/choose":
			choose(s, w)
			return
		case "/sha256":
			mysha256(s, w)
			return
		case "/sentiment":
			sentiment(s, w)
			return
		case "/hex":
			// fetch random 2 digit hex
			qrngSlackCommand(w, s, "https://qrng.anu.edu.au/wp-content/plugins/colours-plugin/get_one_hex.php")
			return
		case "/binary":
			// fetch random 8 bit binary number
			qrngSlackCommand(w, s, "https://qrng.anu.edu.au/wp-content/plugins/colours-plugin/get_one_binary.php")
			return
		case "/rcolor":
			rcolor(s, w)
		case "/ralpha":
			// fetch 1024 random char block from https://qrng.anu.edu.au/
			qrngSlackCommand(w, s, "https://qrng.anu.edu.au/wp-content/plugins/colours-plugin/get_block_alpha.php")
			return
		case "/jwtdecode":
			msg, err := jwtdecode(s.Text)
			if err != nil {
				logErrMsgSlack(w, err.Error())
			}
			msgSlack(msg, w)
			return
		case "/gpt3":
			msg, err := gpt3(s.Text)
			if err != nil {
				logErrMsgSlack(w, err.Error())
			}
			msgSlack(msg, w)
			return
		case "/b64":
			msgSlack(b64(s), w)
			return
		case "/date":
			msgSlack(date(), w)
			return
		case "/streak":
			msg, err := streak(s)
			if err != nil {
				sentry.CaptureException(err)
				logErrMsgSlack(w, err.Error())
			}
			msgSlack(msg, w)
			return

		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			log.Println(err)
		}
	})
	http.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		log.WithFields(log.Fields{"reqID": reqID}).Info("events")
		// handle slack events and verify ownership
		// https://api.slack.com/events/url_verification
		// https://api.slack.com/events

		// parse event from slack
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		log.WithFields(log.Fields{"reqID": reqID, "body": string(body)}).Info("body")
		sv, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if _, err := sv.Write(body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := sv.Ensure(); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionNoVerifyToken())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if eventsAPIEvent.Type == slackevents.URLVerification {
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		go func(reqID string) {
			if eventsAPIEvent.Type == slackevents.CallbackEvent {
				// write 200 ok

				innerEvent := eventsAPIEvent.InnerEvent
				switch ev := innerEvent.Data.(type) {
				case *slackevents.AppMentionEvent:
					// Reply in thread if possible
					threadTS := ev.ThreadTimeStamp
					if threadTS == "" {
						threadTS = ev.TimeStamp
					}
					message, err := messageStore.CallLLM(threadTS, ev.Text)
					if err != nil {
						sentry.CaptureException(err)
						log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to call LLM")
					}
					_, _, err = api.PostMessage(
						ev.Channel,
						slack.MsgOptionText(message, false),
						slack.MsgOptionTS(threadTS),
					)
					if err != nil {
						sentry.CaptureException(err)
						log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to reply in thread")
					}
				case *slackevents.MessageEvent:
					log.Println(ev.Channel, ev.Text)
					if ev.ThreadTimeStamp != "" && ev.User != "U090FSXLJ9Y" {
						log.WithFields(log.Fields{
							"reqID":   reqID,
							"channel": ev.Channel,
							"text":    ev.Text,
							"thread":  ev.ThreadTimeStamp,
							"user":    ev.User,
						}).Info("message event")
						message, err := messageStore.CallLLM(ev.ThreadTimeStamp, ev.Text)
						if err != nil {
							sentry.CaptureException(err)
							log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to call LLM")
						}
						_, _, err = api.PostMessage(
							ev.Channel,
							slack.MsgOptionText(message, false),
							slack.MsgOptionTS(ev.ThreadTimeStamp),
						)
						if err != nil {
							sentry.CaptureException(err)
							log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to reply in thread (message event)")
						}
					}
				}
			}
		}(reqID)

	})
	// server hello world on /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqID := uuid.New().String()
		log.WithFields(log.Fields{"reqID": reqID}).Info("hello world")
		_, err := w.Write([]byte("Hello, World!"))
		if err != nil {
			log.Println(err)
		}
	})
	log.Println("server listening")
	// TODO: port should be env var
	http.ListenAndServe(":3000", nil)
}

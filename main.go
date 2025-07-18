package main

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/rosbit/go-quickjs"

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
	var tools = []ToolHandler{
		CreateToolHandler("jwtdecode", func(input struct {
			Token string `json:"token"`
		}) (*string, error) {
			response, err := jwtdecode(input.Token)
			if err != nil {
				return nil, errors.Wrap(err, "failed to decode JWT")
			}
			return &response, nil
		}),
		CreateToolHandler("quickjs", func(input struct {
			Code string `json:"code"`
		}) (*string, error) {
			ctx, err := quickjs.NewContext()
			if err != nil {
				return nil, errors.Wrap(err, "failed to create context")
			}
			res, err := ctx.Eval(input.Code, nil)
			if err != nil {
				// js errors come back as err
				response := fmt.Sprintf("Error: %v", err)
				return &response, nil
			}
			response := fmt.Sprintf("%v", res)
			return &response, nil
		}),
		CreateToolHandler("postgres_query", func(input struct {
			Query string `json:"query"`
		}) (*string, error) {
			dbURL := os.Getenv("DATABASE_URL")
			if dbURL == "" {
				dbURL = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
			}
			// TODO: use a proper connection string from config
			db, err := sql.Open("postgres", dbURL)
			if err != nil {
				return nil, errors.Wrap(err, "failed to connect to database")
			}
			defer db.Close()

			rows, err := db.Query(input.Query)
			if err != nil {
				// To be friendlier to the LLM, we'll return db errors as part of the response string
				response := fmt.Sprintf("Error: %v", err)
				return &response, nil
			}
			defer rows.Close()

			columns, err := rows.Columns()
			if err != nil {
				return nil, errors.Wrap(err, "failed to get columns")
			}

			var results []map[string]interface{}
			for rows.Next() {
				values := make([]interface{}, len(columns))
				valuePtrs := make([]interface{}, len(columns))
				for i := range columns {
					valuePtrs[i] = &values[i]
				}

				if err := rows.Scan(valuePtrs...); err != nil {
					return nil, errors.Wrap(err, "failed to scan row")
				}

				rowMap := make(map[string]interface{})
				for i, col := range columns {
					val := values[i]

					if b, ok := val.([]byte); ok {
						rowMap[col] = string(b)
					} else {
						rowMap[col] = val
					}
				}
				results = append(results, rowMap)
			}

			if err := rows.Err(); err != nil {
				return nil, errors.Wrap(err, "error during rows iteration")
			}

			jsonResult, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal result to JSON")
			}

			response := string(jsonResult)
			return &response, nil
		}),
	}
	messageStore := NewSlackMessageStore(
		NewLLM(
			anthropicClient,
			NewAnthropicMessageHandler(
				tools,
			),
		),
	)

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
		go func(reqID string, eventsAPIEvent slackevents.EventsAPIEvent) {
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
					callLLm(threadTS, ev.Text, messageStore, ev.Channel, threadTS, api, reqID)
				case *slackevents.AssistantThreadStartedEvent:
					log.WithFields(log.Fields{"reqID": reqID, "thread": ev.EventTimestamp}).Info("assistant thread started")
					// Let's set some suggested prompts
					err := api.SetAssistantThreadsSuggestedPrompts(
						slack.AssistantThreadsSetSuggestedPromptsParameters{
							ThreadTS: eventsAPIEvent.InnerEvent.Data.(*slackevents.AssistantThreadStartedEvent).AssistantThread.ThreadTimeStamp,
							Prompts: []slack.AssistantThreadsPrompt{
								{
									Title:   "How many seconds are in a month? use js to calculate",
									Message: "how many seconds are in a month? use js to calculate",
								},
								{
									// TODO: add a prompt to get the database schema
									Title:   "Get the database schema",
									Message: "Get the database schema",
								},
							},
							ChannelID: eventsAPIEvent.InnerEvent.Data.(*slackevents.AssistantThreadStartedEvent).AssistantThread.ChannelID,
							Title:     "TODO :)",
						})
					if err != nil {
						log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to set assistant thread suggested prompts")
						sentry.CaptureException(err)
					}
				case *slackevents.MessageEvent:
					log.Println(ev.Channel, ev.Text)
					log.WithFields(log.Fields{"reqID": reqID, "channel": ev.Channel, "text": ev.Text, "thread": ev.ThreadTimeStamp, "user": ev.User, "channelType": ev.ChannelType}).Info("message event")
					text := ev.Text
					// if text starts with 34F1C711-9E95-4B6E-B898-0CD940057B0E event type
					if strings.HasPrefix(text, "34F1C711-9E95-4B6E-B898-0CD940057B0E") {
						// 34F1C711-9E95-4B6E-B898-0CD940057B0E event type
						// {{{{inputs.Ft090RD6LWPL__list_id}}}}
						// {{{{inputs.Ft090RD6LWPL__user_id}}}} set {{{{inputs.Ft090RD6LWPL__fields_name}}}} to {{{{inputs.Ft090RD6LWPL__fields_Col08SE07CK5X}}}}
						// extract list id
						// extract message
						// look at $channel history and find most recent message containing the list id in a link like https://saphira-hq.slack.com/lists/T063KRC3CN8/F090H9ZV1PG
						// if found, reply to the message in thread with the message
						split := strings.Split(text, "\n")

						index := map[int]string{}
						for i, v := range split {
							index[i] = v
						}
						// now I can safely check with ok
						var listID string
						var userID string
						var taskName string
						var taskstatus string
						if v, ok := index[1]; ok {
							listID = v
						}
						if v, ok := index[2]; ok {
							userID = v
						}
						if v, ok := index[3]; ok {
							taskName = v
						}
						if v, ok := index[4]; ok {
							taskstatus = v
						}
						if taskName == "" {
							// filter
							return
						}

						msg := fmt.Sprintf("%s set %s to %s", userID, taskName, taskstatus)

						channel := "C07T9KYKUJU" // tasks
						log.WithFields(log.Fields{"reqID": reqID, "listID": listID, "msg": msg}).Info("listID and msg")
						channelHistory, err := api.GetConversationHistory(&slack.GetConversationHistoryParameters{
							ChannelID: channel,
							Limit:     100,
						})
						if err != nil {
							log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to get channel history")
						}
						for _, message := range channelHistory.Messages {
							if strings.Contains(message.Text, listID) {
								log.WithFields(log.Fields{"reqID": reqID, "message": message.Text}).Info("message found")
								// reply to the message in thread with the message
								_, _, err = api.PostMessage(
									channel,
									slack.MsgOptionText(msg, false),
									slack.MsgOptionTS(message.Timestamp),
								)
								if err != nil {
									log.WithFields(log.Fields{"reqID": reqID, "error": err}).Error("Failed to reply in thread")
									sentry.CaptureException(err)
								}
								return
							}

						}
					}
					// handle AI app messages (message.im) and threaded messages
					if (ev.ChannelType == "im") && ev.User != "U090FSXLJ9Y" {
						log.WithFields(log.Fields{
							"reqID":   reqID,
							"channel": ev.Channel,
							"text":    ev.Text,
							"thread":  ev.ThreadTimeStamp,
							"user":    ev.User,
						}).Info("message event")
						threadTS := ev.ThreadTimeStamp
						if threadTS == "" {
							threadTS = ev.TimeStamp
						}
						if ev.SubType == "message_changed" {
							log.WithFields(log.Fields{"reqID": reqID, "channel": ev.Channel, "text": ev.Text, "thread": ev.ThreadTimeStamp, "user": ev.User}).Info("message changed")
							return
						}
						callLLm(threadTS, ev.Text, messageStore, ev.Channel, threadTS, api, reqID)
					}
				}
			}
		}(reqID, eventsAPIEvent)

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	http.ListenAndServe(":"+port, nil)
}

func callLLm(
	timestamp string,
	message string,
	messageStore MessageStore,
	channel string,
	thread string,
	api *slack.Client,
	reqID string,

) {
	resp, err := messageStore.CallLLM(thread, message)
	if err != nil {
		sentry.CaptureException(err)
		log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to call LLM")
		_, _, err = api.PostMessage(
			channel,
			slack.MsgOptionText(
				"Error: "+err.Error()+fmt.Sprintf(
					"\n\n%+v",
					err,
				),
				false,
			),
			slack.MsgOptionTS(thread),
		)
		if err != nil {
			sentry.CaptureException(err)
			log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to reply in thread (message event)")
		}
		return
	}
	_, _, err = api.PostMessage(
		channel,
		slack.MsgOptionText(resp.Message, false),
		slack.MsgOptionTS(thread),
	)
	if err != nil {
		sentry.CaptureException(err)
		log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to reply in thread (message event)")
	}
	counter := 0
	maxLoops := 10
	log.WithFields(log.Fields{"reqID": reqID, "thread": thread, "message": message, "counter": counter, "loop": resp.Loop}).Info("should loop")
	if resp.Loop {
		log.WithFields(log.Fields{"reqID": reqID, "thread": thread, "message": message, "counter": counter, "loop": resp.Loop}).Info("looping")
		counter++
		if counter > maxLoops {
			log.WithFields(log.Fields{"reqID": reqID, "thread": thread, "message": message, "counter": counter}).Info("max loops reached")
			_, _, err = api.PostMessage(
				channel,
				slack.MsgOptionText("Max loops reached", false),
				slack.MsgOptionTS(thread),
			)
			if err != nil {
				sentry.CaptureException(err)
				log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to reply in thread (max loops reached)")
			}
			return
		}
		log.WithFields(log.Fields{"reqID": reqID, "thread": thread, "message": message, "counter": counter}).Info("looping")
		resp, err = messageStore.Loop(thread, api, reqID)
		if err != nil {
			sentry.CaptureException(err)
			log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to loop")
			_, _, err = api.PostMessage(
				channel,
				slack.MsgOptionText(
					"Error: "+err.Error()+fmt.Sprintf(
						"\n\n%+v",
						err,
					),
					false,
				),
				slack.MsgOptionTS(thread),
			)
			if err != nil {
				sentry.CaptureException(err)
				log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to reply in thread (loop)")
			}
			return
		}
		if resp == nil {
			log.WithFields(log.Fields{"reqID": reqID, "thread": thread, "message": message, "counter": counter}).Info("resp is nil")
			return
		}
		_, _, err = api.PostMessage(
			channel,
			slack.MsgOptionText(resp.Message, false),
			slack.MsgOptionTS(thread),
		)
		if err != nil {
			sentry.CaptureException(err)
			log.WithFields(log.Fields{"reqID": reqID, "error": err, "stack": fmt.Sprintf("%+v", err)}).Error("Failed to reply in thread (loop)")
		}
	}
}

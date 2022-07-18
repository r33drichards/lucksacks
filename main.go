package main

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	_ "github.com/joho/godotenv/autoload"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
)

//go:embed docs/tz.md
var TZ_HELP string

func unquoteCodePoint(s string) (string, error) {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	return string(r), err
}

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
	if err != nil {
		log.Println(err)
	}
}

func main() {

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
		case "/echo":
			logErrMsgSlack(w, s.Text)
			return
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
			return

		case "/choose":
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
			s := rand.NewSource(time.Now().Unix())
			r := rand.New(s) // initialize local pseudorandom generator

			var msg string
			if len(vals) > 0 {
				msg = vals[r.Intn(len(vals))]
			} else {
				msg = "nothing to choose from"
			}
			logErrMsgSlack(w, msg)
			return
		case "/sha256":
			h := sha256.New()
			h.Write([]byte(s.Text))
			msg := hex.EncodeToString(h.Sum(nil))
			logErrMsgSlack(w, msg)
			return
		case "/sentiment":
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

			err = req.Send()
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
			return
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
	log.Println("server listening")
	// TODO: port should be env var
	http.ListenAndServe(":3000", nil)
}

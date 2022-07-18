package main

import (
	_ "embed"
	"encoding/json"

	_ "github.com/joho/godotenv/autoload"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/slack-go/slack"
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
			roll(s, w)
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

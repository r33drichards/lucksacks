package main

import (
	"bufio"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/comprehend"
	u "github.com/bcicen/go-units"
	_ "github.com/joho/godotenv/autoload"

	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/slack-go/slack"
)

type Constructions [][]string
type Anagrams [][]string
type Memo map[string]Constructions
type Dictionary map[string]bool

// file is ~200 kb, binary is ~7.5mb
//go:embed popular.txt
var content embed.FS

// We assume they are unique
func permutations(word string) []string {
	if word == "" {
		return []string{""}
	}
	perms := []string{}
	for i, rn := range word {
		rest := word[:i] + word[i+1:]
		//fmt.Println(rest)
		for _, result := range permutations(rest) {
			perms = append(perms, fmt.Sprintf("%c", rn)+result)
		}
		//perms = append(perms, fmt.Sprintf("%c\n", rn))
	}
	return perms
}

func allConstructions(target string, dictionary Dictionary, memo Memo) Constructions {
	if target == "" {
		return [][]string{{}}
	}
	if val, ok := memo[target]; ok {
		return val
	}
	var constructions [][]string
	for word := range dictionary {
		wordLength := len(word)
		targetLength := len(target)
		if wordLength > targetLength {
			continue
		}
		if word == target[:wordLength] {
			guess := allConstructions(target[wordLength:], dictionary, memo)
			for _, g := range guess {
				g = append(g, word)
				constructions = append(constructions, g)
			}
		}
	}
	memo[target] = constructions
	return memo[target]
}

func allAnagrams(word string) Anagrams {
	// populate initial dictionary
	file, err := content.Open("popular.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	dictionary := make(Dictionary)
	for scanner.Scan() {
		dictionary[scanner.Text()] = true
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	allPermutations := permutations(word)
	uniquePermutations := make(map[string]bool)

	for _, perm := range allPermutations {
		uniquePermutations[perm] = true
	}

	var ans Anagrams
	for uperm := range uniquePermutations {
		guess := allConstructions(uperm, dictionary, make(Memo))
		for _, g := range guess {
			ans = append(ans, g)
		}
	}
	return ans
}

//go:embed docs/tz.md
var TZ_HELP string

func unquoteCodePoint(s string) (string, error) {
	r, err := strconv.ParseInt(strings.TrimPrefix(s, "\\U"), 16, 32)
	return string(r), err
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
			params := &slack.Msg{Text: s.Text}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}
		case "/anagram":
			if len(s.Text) > 8 {
				params := &slack.Msg{Text: "too long, max 8 letters"}
				b, err := json.Marshal(params)
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
			}

			// slack wants a fast response, anagrams can take a while to find,
			// dm user anagrams after found and respond right away
			params := &slack.Msg{Text: "searching...\nwill dm when finished"}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}

			go func() {
				anagrams := allAnagrams(s.Text)
				var items []string
				for _, a := range anagrams {
					items = append(items, strings.Join(a, " "))
				}
				_, _, err = api.PostMessage(
					s.UserID,
					slack.MsgOptionText("```\n"+strings.Join(items, "\n")+"\n```", false),
					slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
				)
				if err != nil {
					log.Println(err)
				}
			}()

		case "/convert":
			vals := strings.Split(s.Text, " ")
			from, err := u.Find(vals[1])
			if err != nil {
				params := &slack.Msg{Text: vals[1] + " not valid unit"}
				b, err := json.Marshal(params)
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
			}
			to, err := u.Find(vals[2])
			if err != nil {
				params := &slack.Msg{Text: vals[2] + " not valid unit"}
				b, err := json.Marshal(params)
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
			}

			val, err := strconv.ParseFloat(vals[0], 64)
			if err != nil {
				params := &slack.Msg{Text: vals[0] + " failed to parse"}
				b, err := json.Marshal(params)
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
			}

			message, err := u.ConvertFloat(val, from, to)

			if err != nil {
				params := &slack.Msg{Text: "failed to preform conversion for: " + vals[0] + " " + vals[1] + " " + " " + vals[2]}
				b, err := json.Marshal(params)
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

			}

			params := &slack.Msg{Text: fmt.Sprintf("%s %ss is %s", vals[0], from.Name, message.String())}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}
		case "/tz":
			// TODO: set default timezone
			// TODO: set default conversion
			// TODO: only works on military time

			vals := strings.Split(s.Text, " ")

			if vals[0] == "help" {

				params := &slack.Msg{Text: TZ_HELP}
				b, err := json.Marshal(params)
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
			} else if vals[0] == "now" {

				location := vals[1]

				locationOlsenTime := map[string][]string{
					"usa": []string{
						"America/New_York",
						"America/Chicago",
						"America/Denver",
						"America/Phoenix",
						"America/Los_Angeles",
					},
				}
				names := locationOlsenTime[location]
				var olsenTimes = make([]*time.Location, len(names))

				for i, name := range names {
					timeLocation, err := time.LoadLocation(name)
					if err != nil {
						log.Println(err)
					}
					olsenTimes[i] = timeLocation

				}
				message := "```"
				now := time.Now()
				for _, olsenTime := range olsenTimes {

					message = message + now.In(olsenTime).Format("15:04 MST ") + olsenTime.String() + "\n"

				}
				message = message + "```"
				params := &slack.Msg{Text: message}
				b, err := json.Marshal(params)
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

			}

			timeString := vals[0]

			var layout = "15:04"

			zoneFrom := vals[1]
			if len(zoneFrom) == 3 {
				// probably EST as est or something like that
				zoneFrom = strings.ToUpper(zoneFrom)

				// Saturday, June 12, 2021,
				//
				// 11:11 PM  Eastern Daylight Time          Washington, DC (GMT-4)  EDT
				// 10:11 PM  Central Daylight Time          Chicago        (GMT-5)	CDT
				//  9:11 PM  Mountain Daylight Time         Denver         (GMT-6)	MDT
				//  8:11 PM  Mountain Standard Time         Phoenix        (GMT-7)	MST
				//  8:11 PM  Pacific Daylight Time          Los Angeles    (GMT-7)	PDT
				//  7:11 PM  Alaska Daylight Time           Anchorage      (GMT-8)	ADT
				//  5:11 PM  Hawaii-Aleutian Standard Time  Honolulu       (GMT-10)	HAST
				// https://en.wikipedia.org/wiki/List_of_tz_database_time_zones
				// https://www.timeanddate.com/time/zone/usa
				// https://stackoverflow.com/questions/48942916/difference-between-utc-and-gmt/48960297
				abbrevOlsenLocation := map[string]string{
					"EST": "America/New_York",
					"EDT": "America/New_York",
					"CDT": "America/Chicago",
					"CST": "America/Chicago",
					"MDT": "America/Denver",
					"MST": "America/Denver",
					"PDT": "America/Los_Angeles",
					"PST": "America/Los_Angeles",
				}
				zoneFrom = abbrevOlsenLocation[zoneFrom]
			}

			locationFrom, err := time.LoadLocation(zoneFrom)
			if err != nil {
				log.Println(err)
			}

			t, err := time.ParseInLocation(layout, timeString, locationFrom)
			if err != nil {
				log.Println(err)
			}

			names := []string{
				"America/New_York",
				"America/Chicago",
				"America/Denver",
				"America/Phoenix",
				"America/Los_Angeles",
			}
			var olsenTimes = make([]*time.Location, len(names))

			for i, name := range names {
				timeLocation, err := time.LoadLocation(name)
				if err != nil {
					log.Println(err)
				}
				olsenTimes[i] = timeLocation

			}
			olsenLocationAbbrev := map[string]string{
				"America/New_York":    "EDT/EST",
				"America/Chicago":     "CDT/CST",
				"America/Denver":      "MDT/MST",
				"America/Phoenix":     "MST",
				"America/Los_Angeles": "PDT/PST",
			}
			message := "```\n"
			for _, olsenTime := range olsenTimes {
				message = message + strconv.Itoa(t.In(olsenTime).Hour()) + ":" + strconv.Itoa(t.Minute()) + " " + olsenLocationAbbrev[olsenTime.String()] + " " + olsenTime.String() + "\n"
			}

			message = message + "```"
			params := &slack.Msg{Text: message}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}
		case "/yt":
			// TODO: would be nice to handle https://www.youtube.com/c/STLChessClub/videos
			// style links too
			var msg string

			if strings.Contains(s.Text, "playlist") {
				msg = fmt.Sprintf(
					"/feed add %s",
					"https://www.youtube.com/feeds/videos.xml?playlist_id="+s.Text[38:],
				)
			} else if strings.Contains(s.Text, "channel") {
				splitText := strings.Split(s.Text, "/")
				ytChannelID := splitText[len(splitText)-1]
				msg = fmt.Sprintf(
					"/feed add %s",
					"https://www.youtube.com/feeds/videos.xml?channel_id="+ytChannelID,
				)

			} else {
				msg = fmt.Sprintf("url format not recognised for %s", s.Text)
			}
			params := &slack.Msg{Text: msg}
			b, err := json.Marshal(params)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(b)
			if err != nil {
				log.Println(err)
			}

		case "/roll":
			if s.Text == "" || s.Text == "help" {
				params := &slack.Msg{Text: `returns a random number between 1 and N

Example:

/roll 6

-> 1
`,
				}
				b, err := json.Marshal(params)
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
			}
			i, err := strconv.Atoi(s.Text)
			if err != nil {
				params := &slack.Msg{Text: "Invalid input: " + s.Text}
				b, err := json.Marshal(params)
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
			}
			if i <= 0 {
				params := &slack.Msg{Text: "provide integer greater than 0"}
				b, err := json.Marshal(params)
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
			}
			rand.Seed(time.Now().UnixNano())
			randInt := rand.Intn(i) + 1
			params := &slack.Msg{Text: strconv.Itoa(randInt)}
			b, err := json.Marshal(params)
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

			var params *slack.Msg
			if len(vals) > 0 {
				params = &slack.Msg{Text: vals[r.Intn(len(vals))]}
			} else {
				params = &slack.Msg{Text: "nothing to choose from"}
			}
			b, err := json.Marshal(params)
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
		case "/sha256":
			h := sha256.New()
			h.Write([]byte(s.Text))
			params := &slack.Msg{Text: hex.EncodeToString(h.Sum(nil))}
			b, err := json.Marshal(params)
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

			slackParams := &slack.Msg{Text: sentEmoji[*resp.Sentiment] + " " + *resp.Sentiment}
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

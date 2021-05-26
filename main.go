package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	u "github.com/bcicen/go-units"
	_ "github.com/joho/godotenv/autoload"
	"strconv"

	"github.com/slack-go/slack"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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
			if err != nil{
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
			if err != nil{
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

			if err != nil{
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

		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err	:= w.Write([]byte("OK"))
		if err != nil {
			log.Println(err)
		}
	})
	log.Println("server listening")
	// TODO: port should be env var
	http.ListenAndServe(":3000", nil)
}

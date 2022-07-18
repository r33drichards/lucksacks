package main

import (
	"bufio"
	"embed"
	"fmt"
	"log"
	"net/http"
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

func anagram(s slack.SlashCommand, api *slack.Client, w http.ResponseWriter) {
	if len(s.Text) > 8 {
		logErrMsgSlack(w, "message too long, max 8 chars")
		return
	}

	// slack wants a fast response, anagrams can take a while to find,
	// dm user anagrams after found and respond right away
	logErrMsgSlack(w, "searching...\nwill dm when finished")

	go func() {
		anagrams := allAnagrams(s.Text)
		var items []string
		for _, a := range anagrams {
			items = append(items, strings.Join(a, " "))
		}
		_, _, err := api.PostMessage(
			s.UserID,
			slack.MsgOptionText("```\n"+strings.Join(items, "\n")+"\n```", false),
			slack.MsgOptionAsUser(true), // Add this if you want that the bot would post message as a user, otherwise it will send response using the default slackbot
		)
		if err != nil {
			log.Println(err)
		}
	}()
}

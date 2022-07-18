package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

func tz(s slack.SlashCommand, w http.ResponseWriter) {
	// TODO: set default timezone
	// TODO: set default conversion
	// TODO: only works on military time

	vals := strings.Split(s.Text, " ")

	if vals[0] == "help" {
		logErrMsgSlack(w, TZ_HELP)
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
		logErrMsgSlack(w, message)
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
	logErrMsgSlack(w, message)
	return

}

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/html"

	"github.com/antchfx/htmlquery"
)

const SmallsCalendarUrl = "https://www.smallslive.com/events/calendar/"

type Event struct {
	Name string
	Time string
	Url  string
}

func (e Event) String() string {
	return fmt.Sprintf("%s -- %s, %s", e.Name, e.Time, e.Url)
}

var month = time.Now().Month().String()

// TODO schedule only shows 2 weeks at a time. name should be something like 11-10-2018_11-23-2018.html
var CachedFilePath = fmt.Sprintf("%s.html", month)

func isFileCached() bool {
	if _, err := os.Stat(CachedFilePath); !os.IsNotExist(err) {
		return true
	}
	return false
}

func main() {
	if !isFileCached() {
		log.Printf("Schedule is not cached. Loading from: %s", SmallsCalendarUrl)
		resp, err := http.Get(SmallsCalendarUrl)
		if err != nil {
			log.Fatalf("Could not load url: %s", SmallsCalendarUrl)
		}

		bytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Could not read bytes from response")
		}

		err = ioutil.WriteFile(CachedFilePath, bytes, 0644)
		if err != nil {
			log.Fatalf("Could not write HTML file")
		}
	} else {
		log.Println("Schedule is cached. Loading from file...")
	}

	f, err := os.Open(CachedFilePath)
	if err != nil {
		log.Fatalf("Could not open file: %s", CachedFilePath)
	}
	defer f.Close()

	doc, err := htmlquery.Parse(f)
	if err != nil {
		log.Fatalf("Could not load html from file: %s", CachedFilePath)
	}

	// Parse days
	days := htmlquery.Find(doc, `//section[contains(@class, "schedule")]/div[contains(@class, "day")]`)
	log.Printf("Found %d nights with events", len(days))
	allEvents := make(map[string][]Event)
	for _, day := range days {
		dateString := htmlquery.FindOne(day, "//h2").FirstChild.Data
		log.Println(dateString)
		data := htmlquery.Find(day, "//dl/*")
		events := parseDescriptionList(data)
		allEvents[dateString] = events
		log.Println()
	}
}

// parse <dl> description list
func parseDescriptionList(data []*html.Node) []Event {
	var events []Event
	// Split slice into chunks of 2, since format is <dt>Sunday</dt><dd>Event details</dd>
	var divided [][]*html.Node
	chunkSize := 2
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize

		if end > len(data) {
			end = len(data)
		}

		divided = append(divided, data[i:end])
	}

	for _, e := range divided {
		timeString := e[0].FirstChild.Data
		anchor := e[1].FirstChild
		anchorAttrs := anchor.Attr
		var href = ""
		for _, h := range anchorAttrs {
			if h.Key == "href" {
				href = h.Val
			}
		}
		eventName := anchor.FirstChild.Data
		event := Event{
			Name: eventName,
			Time: timeString,
		}
		if href == "" {
			log.Fatalf("No href found for event: %s", event)
		}
		event.Url = href
		log.Println(event)
		events = append(events, event)
	}
	return events
}

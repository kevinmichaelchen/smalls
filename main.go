package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/antchfx/htmlquery"
)

const CacheDirectory = "cache"
const HtmlCacheDirectory = CacheDirectory + "/html"
const JsonCacheDirectory = CacheDirectory + "/json"
const BaseUrl = "https://www.smallslive.com"
const SmallsCalendarUrl = "https://www.smallslive.com/events/calendar/"

var month = time.Now().Month().String()

// TODO schedule only shows 2 weeks at a time. name should be something like 11-10-2018_11-23-2018.html
var SchedulePath = fmt.Sprintf("%s/%s.html", HtmlCacheDirectory, month)

type Musician struct {
	Name       string
	Instrument string
	Bio        string
}

func (m Musician) String() string {
	return fmt.Sprintf("%s - %s - %s", m.Name, m.Instrument, m.Bio)
}

type Event struct {
	Name      string
	Time      string
	Url       string
	Musicians []Musician
}

func (e Event) String() string {
	b, err := json.Marshal(e)
	if err != nil {
		log.Fatalf("Could not marshal event: %s", err.Error())
	}
	return string(b)
}

func main() {
	os.MkdirAll(HtmlCacheDirectory, 0755)
	os.MkdirAll(JsonCacheDirectory, 0755)
	if !FileExists(SchedulePath) {
		log.Printf("Schedule is not cached. Loading from: %s", SmallsCalendarUrl)
		WriteUrlToFile(SmallsCalendarUrl, SchedulePath)
	} else {
		log.Println("Schedule is cached. Loading from file...")
	}

	doc := FileToHtmlNode(SchedulePath)
	allEvents := parseEventsForMonth(doc)
	persistEvents(allEvents)
}

// persistEvents scrapes each event detail page, persists the HTML for it (if not persisted already),
// and writes the JSON for the event to disk.
func persistEvents(allEvents map[string][]Event) {
	for dateString, events := range allEvents {
		for _, event := range events {
			s := fmt.Sprintf("%s %s", dateString, event.Time)
			log.Println("Fetching musicians for", s)
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			eventUrl := fmt.Sprintf("%s%s", BaseUrl, event.Url)
			eventPath := fmt.Sprintf("%s/%s.html", HtmlCacheDirectory, hash)
			if !FileExists(eventPath) {
				WriteUrlToFile(eventUrl, eventPath)
			}

			doc := FileToHtmlNode(eventPath)
			artistInfos := htmlquery.Find(doc, `//div[contains(@class, "mini-artist-info")]`)
			log.Printf("Found info for %d artists", len(artistInfos))
			for _, artistInfo := range artistInfos {
				m := Musician{
					Name:       htmlquery.FindOne(artistInfo, `//h2[contains(@class, "mini-artist-info__title")]`).FirstChild.FirstChild.Data,
					Instrument: htmlquery.FindOne(artistInfo, `//p[contains(@class, "mini-artist-info__instrument")]`).FirstChild.Data,
					Bio:        htmlquery.FindOne(artistInfo, `//p[contains(@class, "mini-artist-info__bio")]`).FirstChild.Data,
				}
				event.Musicians = append(event.Musicians, m)
				log.Println(event)
			}
		}

		dayJsonFilename := getJsonFilename(dateString)
		dayJsonPath := fmt.Sprintf("%s/%s.html", JsonCacheDirectory, dayJsonFilename)
		if !FileExists(dayJsonPath) {
			b, err := json.Marshal(allEvents[dateString])
			if err != nil {
				log.Fatalf("Could not marshal JSON for: %s", dateString)
			}
			err = ioutil.WriteFile(dayJsonPath, b, 0644)
			if err != nil {
				log.Fatalf("Could not write to path: %s", dayJsonPath)
			}
		}
	}
}

// dateString will be "Friday 11/16/2018"
func getJsonFilename(dateString string) string {
	var s = dateString
	s = strings.ToLower(s)
	// replace slashes with hyphens
	s = strings.Replace(s, "/", "-", -1)
	a := strings.Split(s, " ")
	return a[1] + "_" + a[0]
}

// parseEventsForMonth parses up to 2 weeks worth of events for the current month
func parseEventsForMonth(doc *html.Node) map[string][]Event {
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
	return allEvents
}

// parseDescriptionList chunks a list of element children inside the description list (<dl>).
// Chunks consist of 2 consecutive elements, a <dt> and a <dd> element.
// The <dt> element contains the event time, e.g., 10:30 PM - 1:00 AM
// The <dd> element contains the event name and URL.
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

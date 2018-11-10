package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/html"

	"github.com/antchfx/htmlquery"
)

const CacheDirectory = "cache"
const BaseUrl = "https://www.smallslive.com"
const SmallsCalendarUrl = "https://www.smallslive.com/events/calendar/"

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

var month = time.Now().Month().String()

// TODO schedule only shows 2 weeks at a time. name should be something like 11-10-2018_11-23-2018.html
var CachedFilePath = fmt.Sprintf("%s/%s.html", CacheDirectory, month)

func fileToHtmlNode(path string) *html.Node {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("Could not open file: %s", path)
	}
	defer f.Close()

	doc, err := htmlquery.Parse(f)
	if err != nil {
		log.Fatalf("Could not load html from file: %s", path)
	}
	return doc
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func writeUrlToFile(url, path string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Could not load url: %s", url)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Could not read bytes from response")
	}

	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		log.Fatalf("Could not write HTML file")
	}
}

func main() {
	os.Mkdir(CacheDirectory, 0755)
	if !fileExists(CachedFilePath) {
		log.Printf("Schedule is not cached. Loading from: %s", SmallsCalendarUrl)
		writeUrlToFile(SmallsCalendarUrl, CachedFilePath)
	} else {
		log.Println("Schedule is cached. Loading from file...")
	}

	doc := fileToHtmlNode(CachedFilePath)

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

	for dateString, events := range allEvents {
		for _, event := range events {
			s := fmt.Sprintf("%s %s", dateString, event.Time)
			log.Println("Fetching musicians for", s)
			h := sha1.New()
			h.Write([]byte(s))
			hash := hex.EncodeToString(h.Sum(nil))
			eventUrl := fmt.Sprintf("%s%s", BaseUrl, event.Url)
			eventPath := fmt.Sprintf("%s/%s.html", CacheDirectory, hash)
			if !fileExists(eventPath) {
				writeUrlToFile(eventUrl, eventPath)
			}

			doc := fileToHtmlNode(eventPath)
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

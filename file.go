package main

import (
	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func FileToHtmlNode(path string) *html.Node {
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

func FileExists(path string) bool {
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return true
	}
	return false
}

func WriteUrlToFile(url, path string) {
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
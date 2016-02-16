package main

import (
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/http"
	"os"
	"strings"
)

// The structure to hold info about particular item
type Item struct {
	id      string
	imgUrl  string
	pageUrl string
	title   string
}

// Helper function to pull the 'href' attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}
	return
}
// Helper function to pull the 'src' attribute from Token
func getImgSrc(t html.Token) (ok bool, src string) {
	for _, a := range t.Attr {
		if a.Key == "src" {
			src = a.Val
			ok = true
			break
		}
	}
	return
}
// Helper function to pull the 'title' attribute from Token
func getTitle(t html.Token) (ok bool, title string) {
	for _, a := range t.Attr {
		if a.Key == "title" {
			title = a.Val
			ok = true
			break
		}
	}
	return
}

// Extract all items of interest from a given webpage
func crawl(url string, ch chan Item, chFinished chan bool) {
	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
		fmt.Println("ERROR: Failed to crawl \"" + url + "\"")
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function returns

	z := html.NewTokenizer(b)

	item := Item{}
	idFound := false
	imgFound := false

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return

		case tt == html.SelfClosingTagToken:
			t := z.Token()
			// get next IMG
			if idFound && t.DataAtom == atom.Img {
				ok, src := getImgSrc(t)
				if !ok {
					continue
				}
				item.imgUrl = src
				imgFound = true
			}
			break

		case tt == html.StartTagToken:
			t := z.Token()

			// check if at the beginning of interest's area
			if t.DataAtom == atom.Span {
				ok, id := getTitle(t)
				if !ok {
					continue
				}
				item.id = id
				idFound = true
			}

			// get next <a>
			if imgFound && t.DataAtom == atom.A {
				// Extract the href value, if there is one
				ok, url := getHref(t)
				if !ok {
					continue
				}
				item.pageUrl = url

				ok, title := getTitle(t)
				if ok {
					item.title = title
				}

				// Make sure the url begins in http**
				hasProto := strings.Index(url, "http") == 0
				if hasProto {
					ch <- item

					idFound = false
					imgFound = false
				}
			}
		}
	}
}

func main() {
	var foundItems []Item
	seedUrls := os.Args[1:]

	// Channels
	chItems := make(chan Item)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	for _, url := range seedUrls {
		go crawl(url, chItems, chFinished)
	}

	// Subscribe to both channels
	for c := 0; c < len(seedUrls); {
		select {
		case item := <-chItems:
			foundItems = append(foundItems, item)
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...
	fmt.Printf("\nFound %d unique items:\n", len(foundItems))

	for _, item := range foundItems {
		fmt.Printf("%s, %s, %s\n", item.id, item.pageUrl, item.title)
	}

	close(chItems)
}

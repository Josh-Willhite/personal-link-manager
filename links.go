package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type Link struct {
	URL       string    "json:link"
	Tags      []string  "json:tags"
	TagsText  string    "json:-"
	Notes     string    "json:notes"
	Timestamp time.Time "json:timestamp"
}

type linkStore struct {
	links      map[string]Link
	path       string //location on disk
	serviceURL string
	listHTML   string
	addHTML    string
	editHTML   string
}

func main() {
	ls := linkStore{
		path:       "links.json",
		listHTML:   "links.html",
		editHTML:   "edit.html",
		links:      map[string]Link{},
		serviceURL: "http://links.joshwillhite.com",
	}

	ls.readLinks()

	http.HandleFunc("/", ls.listHandler)
	http.HandleFunc("/add", ls.addHandler)
	http.HandleFunc("/delete", ls.deleteHandler)
	http.HandleFunc("/edit", ls.editHandler)
	http.HandleFunc("/search", ls.searchHandler)
	http.ListenAndServe(":8080", nil)
}

func (ls linkStore) searchHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: searchHandler: %+v\n", time.Now(), r)
	var links []Link
	query := r.URL.Query()
	tag := query.Get("tag")
	terms := query.Get("terms")

	tmpl := template.Must(template.ParseFiles(ls.listHTML))

	if terms != "" {
		links = ls.searchTerms(strings.Fields(terms))
	} else if tag != "" {
		links = ls.searchTags(tag)
	}

	data := struct {
		Title       string
		Links       []Link
		ServiceHome string
	}{
		Title:       "the title",
		Links:       links,
		ServiceHome: ls.serviceURL,
	}
	tmpl.Execute(w, data)
}

func (ls linkStore) searchTerms(searchTerms []string) []Link {
	//brute force term searhc
	var linkMatch = false
	var links = []Link{}

	for url, link := range ls.links {
		linkMatch = false
	loop:
		for _, term := range searchTerms {
			if strings.Contains(url, term) {
				linkMatch = true
				break loop
			}
			if strings.Contains(link.Notes, term) {
				linkMatch = true
				break loop
			}
			for _, tag := range link.Tags {
				if strings.Contains(tag, term) {
					linkMatch = true
					break loop
				}
			}
		}
		if linkMatch {
			links = append(links, link)
		}
	}
	return links
}

func (ls linkStore) searchTags(searchTag string) []Link {
	links := []Link{}
	for _, link := range ls.links {
		for _, tag := range link.Tags {
			if searchTag == tag {
				links = append(links, link)
			}
		}
	}
	return links
}

func (ls linkStore) editHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: editHandler: %+v\n", time.Now(), r)
	url := r.FormValue("url")
	tmpl := template.Must(template.ParseFiles(ls.editHTML))

	if r.Method != http.MethodPost {
		tmpl.Execute(w, ls.links[url])
	}

	ls.readLinks()
	ls.addLink(
		Link{
			URL:       r.FormValue("url"),
			Tags:      strings.Split(r.FormValue("tags"), ","),
			Notes:     r.FormValue("notes"),
			Timestamp: time.Now(),
		},
	)
	ls.writeLinks()

	http.Redirect(w, r, ls.serviceURL, http.StatusSeeOther)
}

func (ls linkStore) deleteHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: deleteHandler: %+v\n", time.Now(), r)
	url := r.FormValue("url")
	ls.deleteLink(Link{URL: url})
	ls.writeLinks()
	http.Redirect(w, r, ls.serviceURL, http.StatusSeeOther)
}

func (ls linkStore) addHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: addHandler: %+v\n", time.Now(), r)
	tmpl := template.Must(template.ParseFiles("add.html"))

	if r.Method != http.MethodPost {
		tmpl.Execute(w, nil)
		return
	}

	ls.readLinks()
	ls.addLink(
		Link{
			URL:       r.FormValue("url"),
			Tags:      strings.Split(r.FormValue("tags"), ","),
			Notes:     r.FormValue("notes"),
			Timestamp: time.Now(),
		},
	)
	ls.writeLinks()

	http.Redirect(w, r, ls.serviceURL, http.StatusSeeOther)
}

func (ls linkStore) listHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("%s: listHandler: %+v\n", time.Now(), r)
	tmpl := template.Must(template.ParseFiles(ls.listHTML))
	links := []Link{}
	for _, link := range ls.links {
		links = append(links, link)
	}

	data := struct {
		Title       string
		Links       []Link
		ServiceHome string
	}{
		Title:       "the title",
		Links:       links,
		ServiceHome: ls.serviceURL,
	}
	tmpl.Execute(w, data)
}

func (ls linkStore) readLinks() {
	links := []Link{}
	bytes, _ := ioutil.ReadFile(ls.path)
	err := json.Unmarshal(bytes, &links)
	if err != nil {
		fmt.Printf("Failed to load links with: %s", err)
	}
	for _, link := range links {
		ls.links[link.URL] = link
	}
}

func (ls linkStore) writeLinks() {
	links := []Link{}
	for _, l := range ls.links {
		links = append(links, l)
	}

	marshalled, err := json.Marshal(links)
	if err != nil {
		fmt.Printf("\nFailed to marshall with: %s", err)
	}
	err = ioutil.WriteFile(ls.path, marshalled, os.ModeExclusive)
	if err != nil {
		fmt.Printf("\nFailed to write with: %s", err)
	}
}

func (ls linkStore) addLink(link Link) {
	ls.links[link.URL] = link
}

func (ls linkStore) deleteLink(link Link) {
	delete(ls.links, link.URL)
}

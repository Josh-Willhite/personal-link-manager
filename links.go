package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
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

type basicAuth struct {
	users map[string]string
}

func main() {
	ls := linkStore{
		path:       "links.json",
		listHTML:   "links.html",
		editHTML:   "edit.html",
		links:      map[string]Link{},
		serviceURL: "http://localhost:8080",
		// serviceURL: "http://links.joshwillhite.com",
	}

	ls.readLinks()

	bs := basicAuth{
		users: map[string]string{},
	}

	bs.loadUsers("users.txt")

	mux := http.NewServeMux()

	mux.Handle("/", http.HandlerFunc(ls.listHandler))
	mux.Handle("/search", http.HandlerFunc(ls.searchHandler))
	mux.Handle("/add", bs.basicAuthCheck(http.HandlerFunc(ls.addHandler)))
	mux.Handle("/delete", bs.basicAuthCheck(http.HandlerFunc(ls.deleteHandler)))
	mux.Handle("/edit", bs.basicAuthCheck(http.HandlerFunc(ls.editHandler)))

	http.ListenAndServe(":8080", mux)
}

func (bs basicAuth) loadUsers(usersPath string) {
	file, err := os.Open(usersPath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		// using ':' in a password is going to be problematic, not a general solution
		creds := strings.Split(scanner.Text(), ":")
		bs.users[creds[0]] = creds[1]
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func (bs basicAuth) basicAuthCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encodedAuth, ok := r.Header["Authorization"]
		if !ok || len(encodedAuth) != 2 {
			w.Header().Add("WWW-Authenticate", "Basic")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		decodedAuth, err := base64.StdEncoding.DecodeString(strings.Split(encodedAuth[0], " ")[1])
		if err != nil {
			log.Println("failed to decode with: ", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		creds := strings.Split(string(decodedAuth), ":")

		password, ok := bs.users[creds[0]]
		if !ok {
			log.Println("user not found: ", creds[0])
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if creds[1] != password {
			log.Println("password failed for: ", creds[0])
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (ls linkStore) searchHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s: searchHandler: %+v\n", time.Now(), r)
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
	log.Printf("%s: editHandler: %+v\n", time.Now(), r)
	url := r.FormValue("url")
	tmpl := template.Must(template.ParseFiles(ls.editHTML))

	if r.Method != http.MethodPost {
		type editData struct {
			URL       string
			TagString string
			Notes     string
		}
		link := ls.links[url]
		tmpl.Execute(w, editData{link.URL, strings.Join(link.Tags, ","), link.Notes})
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

	http.Redirect(w, r, ls.serviceURL, http.StatusSeeOther)
}

func (ls linkStore) deleteHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s: deleteHandler: %+v\n", time.Now(), r)
	url := r.FormValue("url")
	ls.deleteLink(Link{URL: url})
	ls.writeLinks()
	http.Redirect(w, r, ls.serviceURL, http.StatusSeeOther)
}

func (ls linkStore) addHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s: addHandler: %+v\n", time.Now(), r)
	tmpl := template.Must(template.ParseFiles(ls.listHTML))

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
	log.Printf("%s: listHandler: %+v\n", time.Now(), r)
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

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

type LinkDetails struct {
	Link      string    "json:link"
	Tags      []string  "json:tags"
	Notes     string    "json:notes"
	Timestamp time.Time "json:timestamp"
}

func main() {
	links := readLinks()
	fmt.Println(links)
	http.HandleFunc("/addLink", addLinkHandler)
	// http.HandleFunc("/searchLink", linkHandler)
	http.HandleFunc("/listLinks", listLinksHandler)
	http.ListenAndServe(":8080", nil)
}

func listLinksHandler(w http.ResponseWriter, r *http.Request) {
	const theList = `
	<!DOCTYPE html>
		<html>
		<head>
		<meta charset="UTF-8">
		<title>{{.Title}}</title>
		</head>
		<body>
        <table style="width:100%" border="solid">
        <th>link</th>
        <th>tags</th>
        <th>notes</th>
		{{range .Links}}
        <tr>
        <td><a href={{ .Link }}>{{ .Link }}</a></td>
        <td>{{ .Tags }}</td>
        <td>{{ .Notes }}</td>
        </tr>
        {{else}}
        <tr>
		<td></td>
		<td></td>
		<td></td>
		<td></td>
        </tr>
        {{end}}
        </table>
	</body>
		</html>`

	tmpl, _ := template.New("list").Parse(theList)

	data := struct {
		Title string
		Links []LinkDetails
	}{
		Title: "the title",
		Links: readLinks(),
	}

	tmpl.Execute(w, data)
}

func addLinkHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("add-link.html"))

	if r.Method != http.MethodPost {
		tmpl.Execute(w, nil)
		return
	}

	link := LinkDetails{
		Link:      r.FormValue("link"),
		Tags:      strings.Split(r.FormValue("tags"), ","),
		Notes:     r.FormValue("notes"),
		Timestamp: time.Now(),
	}

	writeLink(link)
	http.Redirect(w, r, "http://localhost:8080/listLinks", http.StatusSeeOther)
}

func writeLink(link LinkDetails) {
	links := readLinks()
	links = append(links, link)

	marshalled, err := json.Marshal(links)
	if err != nil {
		fmt.Printf("\nERR: %s", err)
	}
	err = ioutil.WriteFile("links.json", marshalled, os.ModeExclusive)
	if err != nil {
		fmt.Printf("\nFailed to write links with: %s", err)
	}
}

func readLinks() []LinkDetails {
	links := []LinkDetails{}
	bytes, _ := ioutil.ReadFile("links.json")

	err := json.Unmarshal(bytes, &links)
	if err != nil {
		fmt.Printf("\nFailed to read links with: %s", err)
	}
	fmt.Println("Reading Links: ", links)
	return links
}

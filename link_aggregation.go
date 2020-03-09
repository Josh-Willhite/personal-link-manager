package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
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
	http.HandleFunc("/link", linkHandler)
	http.ListenAndServe(":8080", nil)
}

func linkHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("link.html"))

	if r.Method != http.MethodPost {
		tmpl.Execute(w, nil)
		return
	}

	details := LinkDetails{
		Link:      r.FormValue("link"),
		Tags:      strings.Split(r.FormValue("tags"), ","),
		Notes:     r.FormValue("notes"),
		Timestamp: time.Now(),
	}
	marshalled, err := json.Marshal(details)
	if err != nil {
		fmt.Printf("\nERR: %s", err)
	}
	writeData(marshalled)
	// commit and push to github
	tmpl.Execute(w, struct{ Success bool }{Success: true})
}

func writeData(newEntry []byte) {
	fmt.Println(string(newEntry))
}

package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

type Emails []string
type Results map[string]bool

var numEmails int
var resultsChan = make(chan Results)
var wg sync.WaitGroup

func SendMail(index int, wg *sync.WaitGroup, emails []string) {
	defer wg.Done()
	email := emails[index]
	fmt.Printf("email: %v\n", email)
	status := true
	result := Results{email: status}
	resultsChan <- result
}

func ReadStatus(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	for result := range resultsChan {
		for email, status := range result {
			fmt.Fprintf(w, "data: {\"email\": \"%s\", \"status\": %v}\n\n", email, status)
			w.(http.Flusher).Flush()
		}
	}
}

func mailer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		tmpl, _ := template.ParseFiles("public/index.html")
		tmpl.Execute(w, nil)
		return
	}
	req.ParseMultipartForm(10 << 20)
	getEmails := req.PostFormValue("email-list")
	addresses := strings.Split(getEmails, "\n")

	var emails Emails
	for _, address := range addresses {
		emails = append(emails, strings.TrimSpace(address))
	}
	numEmails = len(emails)
	wg.Add(numEmails)
	for i := 0; i < numEmails; i++ {
		go SendMail(i, &wg, emails)
	}

	wg.Wait()
	fmt.Println("i am done!")
	w.Write([]byte("all is done!"))
}

func main() {
	http.HandleFunc("/", mailer)
	http.HandleFunc("/status", ReadStatus)
	http.ListenAndServe(":3000", nil)
}

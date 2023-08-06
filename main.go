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
var results = make(Results)
var wg sync.WaitGroup

func SendMail(index int, wg *sync.WaitGroup, emails []string) {
	defer wg.Done()
	email := emails[index]
	fmt.Printf("email: %v\n", email)
	status := true
	result := Results{email: status}
	select {
	case resultsChan <- result:
	default:
		fmt.Println("Warning: WebSocket not available to send update.")
	}
}

func ReadStatus() {
	wg.Done()
	for result := range resultsChan {
		for email, status := range result {
			results[email] = status
		}
	}
	fmt.Printf("results: %v\n", results)
}

func mailer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		tmpl, _ := template.ParseFiles("public/index.html")
		tmpl.Execute(w, nil)
		return
	}
	req.ParseForm()
	getEmails := req.PostFormValue("email-list")
	addresses := strings.Split(getEmails, "\n")

	var emails Emails
	for _, address := range addresses {
		emails = append(emails, strings.TrimSpace(address))
	}
	numEmails = len(emails)
	w.WriteHeader(http.StatusOK)
	wg.Add(numEmails + 1)
	go ReadStatus()

	for i := 0; i < numEmails; i++ {
		go SendMail(i, &wg, emails)
	}

	wg.Wait()
	fmt.Println("i am done!")
	fmt.Printf("all results: %v\n", results)
}

func main() {
	http.HandleFunc("/", mailer)
	http.ListenAndServe(":3000", nil)
}

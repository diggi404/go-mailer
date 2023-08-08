package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/gomail.v2"
)

type Emails []string
type Results map[string]bool
type Body []string

// declare global variables
var numEmails int
var resultsChan = make(chan Results)
var wg sync.WaitGroup
var body Body

// check smtp connection before starting goroutines.
func ConnectSmtp(smtpCreds []string) (*gomail.Dialer, error) {
	username, password, port, host := smtpCreds[0], smtpCreds[1], smtpCreds[2], smtpCreds[3]
	num, _ := strconv.Atoi(port)
	dailer := gomail.NewDialer(host, int(num), username, password)
	_, err := dailer.Dial()
	if err != nil {
		return nil, err
	}
	return dailer, nil
}

// goroutines to send emails concurrently
func SendMail(index int, wg *sync.WaitGroup, emails []string, smtp *gomail.Dialer, body Body) {
	defer wg.Done()
	email := emails[index]
	senderName, subject, message, _, fromAddress := body[0], body[1], body[2], body[3], body[4]
	goMailer := gomail.NewMessage()
	goMailer.SetAddressHeader("From", fromAddress, senderName)
	goMailer.SetAddressHeader("To", email, "")
	goMailer.SetHeader("Subject", subject)
	goMailer.SetBody("text/plain", message)
	err := smtp.DialAndSend(goMailer)
	var status bool
	if err != nil {
		fmt.Printf("err: %v\n", err)
		status = false
		result := Results{email: status}
		resultsChan <- result
	} else {
		status = true
		result := Results{email: status}
		resultsChan <- result
	}
}

// read the data written to the resultChan and send to client using SSE
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

// root server handler
func mailer(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		tmpl, _ := template.ParseFiles("public/index.html")
		tmpl.Execute(w, nil)
		return
	}

	// parse html form data
	req.ParseMultipartForm(10 << 20)
	getEmails := req.PostFormValue("email-list")
	getUsername := req.PostFormValue("username")
	getPassword := req.PostFormValue("password")
	getPort := req.PostFormValue("port")
	getHost := req.PostFormValue("host")
	getSender := req.PostFormValue("sender-name")
	getSubject := req.PostFormValue("subject")
	getMessage := req.PostFormValue("message")
	getMsgType := req.PostFormValue("message-type")

	// append all parsed form data to body slice
	body = append(body, getSender, getSubject, getMessage, getMsgType, getUsername)

	// store parsed smtp credentials for authentication
	smtpCreds := []string{getUsername, getPassword, getPort, getHost}
	smtp, err := ConnectSmtp(smtpCreds)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error connecting to SMTP Server."))
		fmt.Printf("err: %v\n", err)
		return
	}
	addresses := strings.Split(getEmails, "\n")

	var emails Emails
	for _, address := range addresses {
		emails = append(emails, strings.TrimSpace(address))
	}
	numEmails = len(emails)

	//set number of emails to the waitGroup and execute the goroutines
	wg.Add(numEmails)
	for i := 0; i < numEmails; i++ {
		go SendMail(i, &wg, emails, smtp, body)
	}

	wg.Wait()

	// send response to client after all goroutines are done.
	w.Write([]byte("all is done!"))
}

func main() {
	http.HandleFunc("/", mailer)
	http.HandleFunc("/status", ReadStatus)
	http.ListenAndServe(":3000", nil)
}

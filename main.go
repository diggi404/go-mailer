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
var resultsChan = make(chan Results)
var wg sync.WaitGroup

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
func SendMail(index int, wg *sync.WaitGroup, emails Emails, smtp *gomail.Dialer, body Body, fileContent []byte) {
	defer wg.Done()
	email := emails[index]
	senderName, subject, message, fromAddress := body[0], body[1], body[2], body[3]
	goMailer := gomail.NewMessage()
	goMailer.SetAddressHeader("From", fromAddress, senderName)
	goMailer.SetAddressHeader("To", email, "")
	goMailer.SetHeader("Subject", subject)
	if len(fileContent) == 0 {
		goMailer.SetBody("text/plain", message)
	} else {
		goMailer.SetBody("text/html", string(fileContent))
	}
	err := smtp.DialAndSend(goMailer)
	var status bool
	if err != nil {
		fmt.Printf("err: %v\n", err)
		status = false
		result := Results{email: status}
		resultsChan <- result
		fmt.Printf("result: %v\n", result)
	} else {
		status = true
		result := Results{email: status}
		resultsChan <- result
		fmt.Printf("result: %v\n", result)
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
	if req.Method == http.MethodGet {
		tmpl, _ := template.ParseFiles("templates/index.html")
		data := struct {
			CSSURL string
			JSURL  string
		}{
			CSSURL: "/static/styles.css",
			JSURL:  "/static/script.js",
		}
		tmpl.Execute(w, data)
		return
	} else if req.Method == http.MethodPost {
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
		var fileContent []byte
		if getMsgType == "html" {
			file, handler, _ := req.FormFile("html-file")
			defer file.Close()
			fileContent = make([]byte, handler.Size)
			_, err := file.Read(fileContent)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("Error reading file."))
				return
			}
		}
		// append all parsed form data to body slice
		body := Body{getSender, getSubject, getMessage, getUsername}

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
		numEmails := len(emails)
		fmt.Printf("emails: %v\n", emails)
		fmt.Printf("numEmails: %v\n", numEmails)

		//set number of emails to the waitGroup and execute the goroutines
		wg.Add(numEmails)
		for i := 0; i < numEmails; i++ {
			go SendMail(i, &wg, emails, smtp, body, fileContent)
		}

		wg.Wait()
		// send response to client after all goroutines are done.
		w.Write([]byte("Emails sent successfully."))
	}
}
func main() {
	http.HandleFunc("/", mailer)
	http.HandleFunc("/status", ReadStatus)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.ListenAndServe(":8080", nil)
}

package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// reportHandler handles GET and POST requests for the report form
func reportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		// Serve the report form
		tmpl := template.Must(template.New("report").Parse(`
        <html>
            <body>
                <h1>Submit Investigation Report</h1>
                <form method="POST">
                    <label>Time: <input type="text" name="time"></label><br>
                    <label>Title: <input type="text" name="title"></label><br>
                    <label>Severity: <input type="text" name="severity"></label><br>
                    <label>Description: <textarea name="description"></textarea></label><br>
                    <input type="submit" value="Send Report">
                </form>
            </body>
        </html>
        `))
		tmpl.Execute(w, nil)
	} else if r.Method == "POST" {
		// Process the submitted report
		time := r.FormValue("time")
		title := r.FormValue("title")
		severity := r.FormValue("severity")
		description := r.FormValue("description")

		// Create and send the report
		card := createInvestigationCard(time, title, severity, description)
		channelID := getChannelID()
		if channelID == "" {
			http.Error(w, "Channel ID not set", http.StatusInternalServerError)
			return
		}

		err := sendBotMessage(channelID, "", card)
		if err != nil {
			http.Error(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Report sent successfully! <a href='/report'>Send another report</a>")
	}
}

// startReportServer initializes and starts the report server
func startReportServer() {
	r := mux.NewRouter()
	r.HandleFunc("/report", reportHandler)

	srv := &http.Server{
		Handler:      r,
		Addr:         ":3798",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Println("Report server started at http://localhost:3798/report")
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	}()
}

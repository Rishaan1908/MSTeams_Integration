package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

const (
	graphAPIBaseURL = "https://graph.microsoft.com/v1.0"
)

// Global variables
var (
	currentBotToken *BotToken
	botTokenMutex   sync.RWMutex
	channelID       string
	channelIDMutex  sync.RWMutex
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize OAuth configuration
	initOAuthConfig()

	// Create a new router
	r := mux.NewRouter()

	// Define routes
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	r.HandleFunc("/logout", logoutHandler)

	r.HandleFunc("/api/messages", messagesHandler).Methods("POST")

	// Start the report server in a separate goroutine
	go startReportServer()

	// Set the router as the HTTP handler
	http.Handle("/", r)

	// Start the main server
	log.Println("Main server started at http://localhost:3958/login")
	log.Fatal(http.ListenAndServe(":3958", nil))
}

// updateChannelID updates the global channelID
func updateChannelID(id string) {
	channelIDMutex.Lock()
	defer channelIDMutex.Unlock()
	channelID = id
}

// getChannelID retrieves the global channelID
func getChannelID() string {
	channelIDMutex.RLock()
	defer channelIDMutex.RUnlock()
	return channelID
}

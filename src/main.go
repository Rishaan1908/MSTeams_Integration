package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize the OAuth config
	initOAuthConfig()

	// Create a new router
	r := mux.NewRouter()
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/callback", callbackHandler)
	http.Handle("/", r)

	// Start the server
	log.Println("Server started at http://localhost:3958/login")
	log.Fatal(http.ListenAndServe(":3958", nil))
}

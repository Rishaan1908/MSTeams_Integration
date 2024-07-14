package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Login the user by starting a new OAuth session
func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		clearAuthSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}
	startNewAuthSession(session, r, w)
}

// Logout the user by clearing the session and redirect them to new login page
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	clearSession(session, r, w)
	http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
}

// Handle the callback from Microsoft OAuth
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	if handleOAuthError(r, w) {
		return
	}

	// Exchange token
	token, err := exchangeToken(r)
	if err != nil {
		log.Printf("Failed to exchange token: %v", err)
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store the token in the session
	err = storeTokenInSession(session, r, w, token)
	if err != nil {
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Setup the environment (create Teams and Channel and upload app)
	result, err := setupEnvironment(token.AccessToken)
	if err != nil {
		log.Printf("Failed to setup environment: %v", err)
		http.Error(w, "Failed to setup environment: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Display the result to the user
	fmt.Fprint(w, result)
}

func messagesHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var activity Activity
	err = json.Unmarshal(body, &activity)
	if err != nil {
		log.Printf("Error unmarshaling JSON: %v", err)
		http.Error(w, "Failed to parse request body", http.StatusBadRequest)
		return
	}

	if activity.Type == "message" && activity.Value.UserQuestion != "" {
		// This is a card submission
		handleCardResponse(w, activity)
	} else if activity.Type == "message" {
		// This is a new user message
		handleNewUserMessage(activity)
	}

	w.WriteHeader(http.StatusOK)
}

func handleNewUserMessage(activity Activity) {
	conversationID := activity.Conversation.ID
	userName := activity.From.Name

	// Record the user's message
	err := RecordUserMessage(activity)
	if err != nil {
		log.Printf("Failed to record user message: %v", err)
	}

	// Send personalized welcome message
	welcomeMessage := fmt.Sprintf("Hello **%s**, I hope you are having a great day!\n\n I am Culminate Security's virtual assistant and I am here to respond to any questions you have.", userName)
	err = sendBotMessage(conversationID, welcomeMessage)
	if err != nil {
		log.Printf("Failed to send welcome message to user %s: %v", userName, err)
	}

	// Send welcome card
	err = sendWelcomeCardToConversation(conversationID, userName)
	if err != nil {
		log.Printf("Failed to send welcome card to user %s: %v", userName, err)
	}
}

func handleCardResponse(w http.ResponseWriter, activity Activity) {
	if activity.Value.UserQuestion == "" {
		log.Println("Received empty question, ignoring")
		return
	}

	log.Printf("Received question from user %s: %s", activity.From.Name, activity.Value.UserQuestion)

	// Record the user's question
	err := RecordUserMessage(activity)
	if err != nil {
		log.Printf("Failed to record user question: %v", err)
	}

	response := fmt.Sprintf("Thank you for your question, **%s**\n\n**Your Question:** '%s'\n\nOur team will get back to you shortly!", activity.From.Name, activity.Value.UserQuestion)

	// Send and record the bot's response
	err = sendBotMessage(activity.Conversation.ID, response)
	if err != nil {
		log.Printf("Failed to send bot message: %v", err)
		http.Error(w, "Failed to send response", http.StatusInternalServerError)
		return
	}

	// Record the bot's response
	err = SendTeamsMessage(TeamsMessageRequest{
		TeamsUserId: activity.From.ID,
		Message:     response,
		Context:     "Bot response to user question",
	})
	if err != nil {
		log.Printf("Failed to record bot message: %v", err)
	}
}

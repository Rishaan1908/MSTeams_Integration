package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		//clear invalid cookie
		clearAuthSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
		return
	}

	// Check if already authenticated
	if _, ok := session.Values["token"]; ok {
		fmt.Fprintf(w, "Already authenticated. <a href='/logout'>Logout</a> to use a different account.")
		return
	}

	// Clear existing session and start a new auth session
	startNewAuthSession(session, r, w)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Handle OAuth errors
	if handleOAuthError(r, w) {
		return
	}

	state := r.URL.Query().Get("state")
	if state != "state" {
		log.Println("State parameter does not match")
		http.Error(w, "State parameter does not match", http.StatusBadRequest)
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
	profile, err := getUserProfile(token)
	if err != nil {
		log.Printf("Failed to get user profile: %v", err)
		http.Error(w, "Failed to get user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	teams, err := getUserTeams(token)
	if err != nil {
		log.Printf("Failed to get user teams: %v", err)
		http.Error(w, "Failed to get user teams: "+err.Error(), http.StatusInternalServerError)
		return
	}

	renderTeamSelectionPage(w, profile.Email, teams)
}

func selectTeamHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	token, err := getTokenFromSession(session)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	teamID := r.FormValue("team")
	if teamID == "" {
		http.Error(w, "No team selected", http.StatusBadRequest)
		return
	}

	// Check if the "culminate security" channel already exists
	channelName := "TEST_CHANNEL_1"
	channelID, err := getChannelID(token, teamID, channelName)
	if err == nil {
		// Channel already exists, display a message
		fmt.Fprintf(w, "The 'Culminate Security' channel already exists in the selected team. Please look at your teams.")
		//http.Error(w, "Channel EXISTS: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the channel if it doesn't exist
	if err := createChannel(token, teamID, channelName); err != nil {
		log.Printf("Failed to create channel: %v", err)
		http.Error(w, "Failed to create channel: "+err.Error(), http.StatusInternalServerError)
		return
	}

	channelID, err = getChannelID(token, teamID, channelName)
	if err != nil {
		log.Printf("Failed to get channel ID: %v", err)
		http.Error(w, "Failed to get channel ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send a welcome message to the new channel
	welcomeMessage := "Welcome to " + channelName + ": No alert left behind with our AI expert investigator"
	if err := sendWelcomeMessage(token, teamID, channelID, welcomeMessage); err != nil {
		log.Printf("Failed to send welcome message: %v", err)
		http.Error(w, "Failed to send welcome message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Channel '"+channelName+"' created successfully in the selected team.")
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		log.Printf("Failed to get session: %v", err)
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Clear the session
	clearSession(session, r, w)

	// Redirect to login
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// renderTeamSelectionPage renders the HTML page for team selection
func renderTeamSelectionPage(w http.ResponseWriter, email string, teams []map[string]interface{}) {
	teamSelectionHTML := buildTeamSelectionHTML(teams)
	fmt.Fprintf(w, `
	<html>
	<body>
		<p>Your Email: %s</p>
		<p>Select the team you want your report in:</p>
		<form action="/select-team" method="POST">
			%s
			<button type="submit">Select Team</button>
		</form>
	</body>
	</html>`, email, teamSelectionHTML)
}

func buildTeamSelectionHTML(teams []map[string]interface{}) string {
	var sb strings.Builder
	for _, team := range teams {
		id := team["id"].(string)
		name := team["displayName"].(string)
		sb.WriteString(fmt.Sprintf(`<input type="radio" name="team" value="%s">%s<br>`, id, name))
	}
	return sb.String()
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var (
	oauthConfig *oauth2.Config
	store       = sessions.NewCookieStore([]byte("your-secret-key"))
)

func initOAuthConfig() {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	//tenantID := os.Getenv("TENANT_ID")
	redirectURL := os.Getenv("REDIRECT_URL")

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
			"profile",
			"User.Read",
			"Chat.ReadWrite",
			"TeamsAppInstallation.ReadWriteForUser",
			"User.ReadWrite.All",
			"Chat.Create",
			"Chat.ReadWrite.All",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["token"]; ok {
		// Option 1: Use the existing token
		fmt.Fprintf(w, "Already authenticated. <a href='/logout'>Logout</a> to use a different account.")
		return

	}

	// Clear the session data to ensure a fresh OAuth flow
	session.Options.MaxAge = -1
	session.Save(r, w)

	// Generate a new state parameter for CSRF protection
	state := "state"
	session.Values["state"] = state
	session.Save(r, w)

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	log.Printf("Redirecting to URL: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return // <-- Return immediately after setting an error status
	}

	// Verify the state parameter for CSRF protection
	state := r.URL.Query().Get("state")
	if state != "state" {
		http.Error(w, "State parameter does not match", http.StatusBadRequest)
		return // <-- Return immediately
	}

	// Exchange the authorization code for an access token
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return // <-- Return immediately
	}

	// Store the token in the session
	session.Values["token"] = token
	session.Save(r, w)

	// Print the token for debugging purposes
	fmt.Printf("Access Token: %s\n", token.AccessToken)

	// Get the user profile
	profile, err := getUserProfile(token)
	if err != nil {
		http.Error(w, "Failed to get user profile: "+err.Error(), http.StatusInternalServerError)
		return // <-- Return immediately
	}
	/*
		// Send a message to the user
		err = sendMessageToUser(token, "xdnotrish19@gmail.com", "Hello, this is a test")
		if err != nil {
			http.Error(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
			return // <-- Return immediately
		}
	*/
	// If we've made it this far without returning, we can safely write the success response
	fmt.Fprintf(w, "User Profile: %s\nMessage sent successfully!", profile)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Clear the session data
	session.Options.MaxAge = -1
	session.Values = make(map[interface{}]interface{})
	session.Save(r, w)

	// Redirect to the login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

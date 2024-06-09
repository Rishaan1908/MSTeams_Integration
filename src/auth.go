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
	store       *sessions.CookieStore
)

func initOAuthConfig() {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")

	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		// API Permissions
		Scopes: []string{
			"openid",
			"profile",
			"User.Read",
			"Chat.ReadWrite",
			"Chat.Create",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}
	secureKey, err := GenerateSecureKey(32)
	if err != nil {
		log.Fatal("Error generating secure key:", err)
	}
	store = sessions.NewCookieStore([]byte(secureKey))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	if _, ok := session.Values["token"]; ok {
		// Use the existing token if user has already been authenticated
		fmt.Fprintf(w, "Already authenticated. <a href='/logout'>Logout</a> to use a different account.")
		return
	}

	// Clear the session data, for a new login
	session.Options.MaxAge = -1
	session.Save(r, w)

	// Generate a new state parameter
	state := "state"
	session.Values["state"] = state
	session.Save(r, w)

	// Authorization URL
	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	log.Printf("Redirecting to URL: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Get the session
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	// Check for an error parameter in the query string
	if err := r.URL.Query().Get("error"); err != "" {
		if err == "access_denied" {
			// User canceled the login
			log.Println("User canceled the login process.")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		} else {
			http.Error(w, "OAuth error: "+err, http.StatusBadRequest)
			return
		}
	}

	// Verify the state parameter
	state := r.URL.Query().Get("state")
	if state != "state" {
		http.Error(w, "State parameter does not match", http.StatusBadRequest)
		return
	}

	// Exchange the authorization code for an access token
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Store the token in the session
	session.Values["token"] = token
	session.Save(r, w)

	// Print the token (in case debug)
	fmt.Printf("Access Token: %s\n", token.AccessToken)

	profile, err := getUserProfile(token)
	if err != nil {
		http.Error(w, "Failed to get user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	//JSON format user profile
	fmt.Fprintf(w, "Successfully Logged in!!\nUser Profile: %s\n", profile)
}

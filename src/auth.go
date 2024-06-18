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
		Scopes: []string{
			"openid",
			"profile",
			"User.Read",
			"Chat.ReadWrite",
			"Chat.Create",
			"ChannelMessage.Send",
			"User.Read.All",
			"User.ReadWrite.All",
			"Team.ReadBasic.All",
			"ChannelSettings.Read.All",
			"Channel.ReadBasic.All",
			"ChannelSettings.ReadWrite.All", // Add this scope to read and write channel settings
			"Group.Read.All",                // Add this scope to read group info
			"Group.ReadWrite.All",           // Add this scope to read and write group info
			"Directory.Read.All",            // Add this scope to read directory info
			"Directory.ReadWrite.All",
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
		fmt.Fprintf(w, "Already authenticated. <a href='/logout'>Logout</a> to use a different account.")
		return
	}

	session.Options.MaxAge = -1
	session.Save(r, w)

	state := "state"
	session.Values["state"] = state
	session.Save(r, w)

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	log.Printf("Redirecting to URL: %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	if err := r.URL.Query().Get("error"); err != "" {
		if err == "access_denied" {
			log.Println("User canceled the login process.")
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		} else {
			http.Error(w, "OAuth error: "+err, http.StatusBadRequest)
			return
		}
	}

	state := r.URL.Query().Get("state")
	if state != "state" {
		http.Error(w, "State parameter does not match", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	code := r.URL.Query().Get("code")
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session.Values["token"] = token
	session.Save(r, w)

	fmt.Printf("Access Token: %s\n", token.AccessToken)

	profile, err := getUserProfile(token)
	if err != nil {
		http.Error(w, "Failed to get user profile: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("User Profile: %s\n", profile)

	teamID, err := getTeamID(token, "Culminate-Test")
	if err != nil {
		http.Error(w, "Failed to get team ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	channelID, err := getChannelID(token, teamID, "TEST")
	if err != nil {
		http.Error(w, "Failed to get channel ID: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := sendChannelMessage(token, teamID, channelID, "Testing", "successful"); err != nil {
		http.Error(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Message sent successfully to the TEST channel in Culminate-Test team.")
}

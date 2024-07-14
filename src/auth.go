package main

import (
	"log"
	"os"
	"sync"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

// Global variables
var (
	oauthConfig  *oauth2.Config
	store        *sessions.CookieStore
	tokenStore   = make(map[string]string)
	tokenStoreMu sync.Mutex
)

// initOAuthConfig initializes the OAuth2 configuration
func initOAuthConfig() {
	// Retrieve environment variables
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	redirectURL := os.Getenv("REDIRECT_URL")

	// Set up OAuth2 configuration
	oauthConfig = &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid", "profile", "User.Read", "Team.Create", "Channel.Create", "Chat.Create",
			"Chat.ReadWrite", "ChannelMessage.Send", "ChannelSettings.ReadWrite.All",
			"Team.ReadBasic.All", "TeamSettings.ReadWrite.All", "TeamsAppInstallation.ReadWriteForTeam",
			"AppCatalog.ReadWrite.All", "User.Read.All", "ChatMessage.Send",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}

	// Generate a secure key for cookie store
	secureKey, err := GenerateSecureKey(32)
	if err != nil {
		log.Fatal("Error generating secure key:", err)
	}
	store = sessions.NewCookieStore([]byte(secureKey))
}

package main

import (
	"log"
	"os"
	"sync"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var (
	oauthConfig  *oauth2.Config
	store        *sessions.CookieStore
	tokenStore   = make(map[string]string)
	tokenStoreMu sync.Mutex
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
			"openid", "profile", "User.Read", "Chat.ReadWrite", "Chat.Create",
			"ChannelMessage.Send", "Channel.Create", "User.Read.All", "User.ReadWrite.All",
			"Team.ReadBasic.All", "ChannelSettings.Read.All", "Channel.ReadBasic.All",
			"ChannelSettings.ReadWrite.All", "Directory.Read.All", "Directory.ReadWrite.All",
		},
		Endpoint: microsoft.AzureADEndpoint("common"),
	}
	secureKey, err := GenerateSecureKey(32)
	if err != nil {
		log.Fatal("Error generating secure key:", err)
	}
	store = sessions.NewCookieStore([]byte(secureKey))
}

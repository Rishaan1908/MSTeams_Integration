package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

func GenerateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate session ID: %v", err)
	}
	return hex.EncodeToString(bytes)
}

func clearAuthSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "auth-session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

func startNewAuthSession(session *sessions.Session, r *http.Request, w http.ResponseWriter) {
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to save session: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// New session
	state := "state"
	session.Values["state"] = state
	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to save session: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func handleOAuthError(r *http.Request, w http.ResponseWriter) bool {
	if err := r.URL.Query().Get("error"); err != "" {
		if err == "access_denied" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return true
		} else {
			http.Error(w, "OAuth error: "+err, http.StatusBadRequest)
			return true
		}
	}
	return false
}

func exchangeToken(r *http.Request) (*oauth2.Token, error) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	return oauthConfig.Exchange(ctx, code)
}

func storeTokenInSession(session *sessions.Session, r *http.Request, w http.ResponseWriter, token *oauth2.Token) error {
	sessionID := generateSessionID()
	tokenStoreMu.Lock()
	tokenStore[sessionID] = token.AccessToken
	tokenStoreMu.Unlock()

	session.Values["sessionID"] = sessionID
	if err := session.Save(r, w); err != nil {
		return err
	}
	return nil
}

func getTokenFromSession(session *sessions.Session) (*oauth2.Token, error) {
	sessionID, ok := session.Values["sessionID"].(string)
	if !ok {
		return nil, fmt.Errorf("session ID not found")
	}

	tokenStoreMu.Lock()
	accessToken, ok := tokenStore[sessionID]
	tokenStoreMu.Unlock()

	if !ok {
		return nil, fmt.Errorf("token not found for session")
	}

	return &oauth2.Token{AccessToken: accessToken}, nil
}

func clearSession(session *sessions.Session, r *http.Request, w http.ResponseWriter) {
	session.Options.MaxAge = -1
	session.Save(r, w)
}

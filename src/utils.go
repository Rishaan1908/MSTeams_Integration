package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/oauth2"
)

// GenerateSecureKey creates a cryptographically secure random key of the specified length
func GenerateSecureKey(length int) (string, error) {
	key := make([]byte, length)
	_, err := rand.Read(key)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate session ID: %v", err)
	}
	return hex.EncodeToString(bytes)
}

// clearAuthSessionCookie removes the authentication session cookie
func clearAuthSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "auth-session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// startNewAuthSession initiates a new authentication session
func startNewAuthSession(session *sessions.Session, r *http.Request, w http.ResponseWriter) {
	// Clear the existing session
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to save session: %v", err)
		http.Error(w, "Failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect to OAuth provider
	url := oauthConfig.AuthCodeURL("", oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// handleOAuthError checks for and handles OAuth errors in the request
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

// exchangeToken exchanges the OAuth code for a token
func exchangeToken(r *http.Request) (*oauth2.Token, error) {
	ctx := context.Background()
	code := r.URL.Query().Get("code")
	return oauthConfig.Exchange(ctx, code)
}

// storeTokenInSession saves the OAuth token in the session
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

// clearSession removes the current session
func clearSession(session *sessions.Session, r *http.Request, w http.ResponseWriter) {
	session.Options.MaxAge = -1
	session.Save(r, w)
}

// updateEnvFile updates the .env file with new key-value pairs
func updateEnvFile(envVars map[string]string) error {
	// Read the current .env file
	content, err := os.ReadFile(".env")
	if err != nil {
		return fmt.Errorf("failed to read .env file: %w", err)
	}

	// Parse existing environment variables
	lines := strings.Split(string(content), "\n")
	envMap := make(map[string]string)
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	// Update with new values
	for key, value := range envVars {
		envMap[key] = value
	}

	// Write the updated .env file
	file, err := os.Create(".env")
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	for key, value := range envMap {
		_, err := file.WriteString(fmt.Sprintf("%s=%s\n", key, value))
		if err != nil {
			return fmt.Errorf("failed to write to .env file: %w", err)
		}
	}

	return nil
}

// checkTeamExists verifies if a team with the given name exists
func checkTeamExists(token, teamName string) (bool, string, error) {
	url := fmt.Sprintf("%s/me/joinedTeams", graphAPIBaseURL)

	fmt.Printf("Fetching user's teams with URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response status: %d\nResponse body: %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	err = json.NewDecoder(bytes.NewReader(body)).Decode(&result)
	if err != nil {
		return false, "", fmt.Errorf("failed to decode response: %w", err)
	}

	for _, team := range result.Value {
		if team.DisplayName == teamName {
			return true, team.ID, nil
		}
	}

	return false, "", nil
}

// sendWelcomeMessage sends a welcome message to the specified channel
func sendWelcomeMessage(channelID string) error {
	message := "Welcome to the **Culminate Security Reports Channel**, we will send you once an investigation reports in this channel.\n\nIf you have any questions, send our virtual assistant a direct chat message!"
	return sendBotMessage(channelID, message)
}

// listApps retrieves a list of installed Teams apps
func listApps(token string) ([]struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}, error) {
	url := fmt.Sprintf("%s/appCatalogs/teamsApps", graphAPIBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"value"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Value, nil
}

// deleteApp removes a Teams app with the given ID
func deleteApp(token, appID string) error {
	url := fmt.Sprintf("%s/appCatalogs/teamsApps/%s", graphAPIBaseURL, appID)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete app, status code: %d", resp.StatusCode)
	}

	return nil
}

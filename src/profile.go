package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// UserProfile structure to include necessary fields
type UserProfile struct {
	ID    string `json:"id"`
	Email string `json:"mail"`
}

// getUserProfile retrieves the user profile
func getUserProfile(token *oauth2.Token) (UserProfile, error) {
	client := oauthConfig.Client(context.Background(), token)

	// Fetch user profile
	resp, err := client.Get("https://graph.microsoft.com/v1.0/me")
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to get user profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return UserProfile{}, fmt.Errorf("failed to get user profile: status %d", resp.StatusCode)
	}

	var profile UserProfile
	err = json.NewDecoder(resp.Body).Decode(&profile)
	if err != nil {
		return UserProfile{}, fmt.Errorf("failed to decode user profile: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return profile, fmt.Errorf("failed to get organization: status %d", resp.StatusCode)
	}

	return profile, nil
}

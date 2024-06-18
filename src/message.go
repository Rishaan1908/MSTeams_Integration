package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// getTeamID retrieves the team ID for a given team name
func getTeamID(token *oauth2.Token, teamName string) (string, error) {
	url := "https://graph.microsoft.com/v1.0/me/joinedTeams"

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		// Read the response body
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get joined teams: %s, response: %s", resp.Status, string(body))
	}

	// Decode the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Find the team with the specified name
	teams, ok := result["value"].([]interface{})
	if !ok {
		return "", fmt.Errorf("failed to parse teams list from response")
	}

	for _, team := range teams {
		teamMap, ok := team.(map[string]interface{})
		if !ok {
			continue
		}

		if name, ok := teamMap["displayName"].(string); ok && name == teamName {
			if teamID, ok := teamMap["id"].(string); ok {
				return teamID, nil
			}
		}
	}

	return "", fmt.Errorf("team '%s' not found", teamName)
}

// getChannelID retrieves the channel ID for a given team ID and channel name
func getChannelID(token *oauth2.Token, teamID, channelName string) (string, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)

	// Create a new HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusOK {
		// Read the response body
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get channels: %s, response: %s", resp.Status, string(body))
	}

	// Decode the response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// Find the channel with the specified name
	channels, ok := result["value"].([]interface{})
	if !ok {
		return "", fmt.Errorf("failed to parse channels list from response")
	}

	for _, channel := range channels {
		channelMap, ok := channel.(map[string]interface{})
		if !ok {
			continue
		}

		if name, ok := channelMap["displayName"].(string); ok && name == channelName {
			if channelID, ok := channelMap["id"].(string); ok {
				return channelID, nil
			}
		}
	}

	return "", fmt.Errorf("channel '%s' not found", channelName)
}

// sendChannelMessage sends a message to a channel in Microsoft Teams
func sendChannelMessage(token *oauth2.Token, teamID, channelID, subject, message string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages", teamID, channelID)

	// Define message body
	messageBody := map[string]interface{}{
		"subject": subject,
		"body": map[string]string{
			"content": message,
		},
	}

	// Marshal request body to JSON
	jsonBody, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Check for a successful response
	if resp.StatusCode != http.StatusCreated {
		// Read the response body
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message: %s, response: %s", resp.Status, string(body))
	}

	return nil
}

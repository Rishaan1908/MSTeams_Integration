package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// getUserTeams retrieves the list of teams the user is part of
func getUserTeams(token *oauth2.Token) ([]map[string]interface{}, error) {
	url := "https://graph.microsoft.com/v1.0/me/joinedTeams"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get joined teams: %s, response: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	teams, ok := result["value"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to parse teams list from response")
	}

	var parsedTeams []map[string]interface{}
	for _, team := range teams {
		teamMap, ok := team.(map[string]interface{})
		if !ok {
			continue
		}
		parsedTeams = append(parsedTeams, teamMap)
	}

	return parsedTeams, nil
}

// createChannel creates a new channel in a specified team
func createChannel(token *oauth2.Token, teamID, channelName string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)

	channel := map[string]interface{}{
		"displayName": channelName,
		"description": "Channel for Culminate Security Reports",
	}

	jsonBody, err := json.Marshal(channel)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create channel: %s, response: %s", resp.Status, string(body))
	}

	return nil
}

// sendWelcomeMessage sends a welcome message to the newly created channel
func sendWelcomeMessage(token *oauth2.Token, teamID, channelID, message string) error {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels/%s/messages", teamID, channelID)

	msg := map[string]interface{}{
		"body": map[string]interface{}{
			"content": message,
		},
	}

	jsonBody, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message: %s, response: %s", resp.Status, string(body))
	}

	return nil
}

// getChannelID retrieves the ID of the created channel
func getChannelID(token *oauth2.Token, teamID, channelName string) (string, error) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/teams/%s/channels", teamID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get channels: %s, response: %s", resp.Status, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	channels, ok := result["value"].([]interface{})
	if !ok {
		return "", fmt.Errorf("failed to parse channels list from response")
	}

	for _, channel := range channels {
		channelMap, ok := channel.(map[string]interface{})
		if !ok {
			continue
		}
		if channelMap["displayName"] == channelName {
			return channelMap["id"].(string), nil
		}
	}

	return "", fmt.Errorf("channel %s not found", channelName)
}

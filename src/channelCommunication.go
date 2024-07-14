package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Sends the message as the bot to the reports channel
func sendBotMessage(channelID, message string, card ...map[string]interface{}) error {
	botToken, err := getValidBotToken()
	if err != nil {
		return fmt.Errorf("failed to get valid bot token: %w", err)
	}

	url := fmt.Sprintf("https://smba.trafficmanager.net/amer/v3/conversations/%s/activities", channelID)

	var payload map[string]interface{}
	if len(card) > 0 {
		payload = card[0]
	} else {
		payload = map[string]interface{}{
			"type": "message",
			"text": message,
		}
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+botToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to send message (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// JSON format for the investigation card
func createInvestigationCard(time, title, severity, description string) map[string]interface{} {
	return map[string]interface{}{
		"type": "message",
		"attachments": []map[string]interface{}{
			{
				"contentType": "application/vnd.microsoft.card.adaptive",
				"content": map[string]interface{}{
					"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
					"type":    "AdaptiveCard",
					"version": "1.0",
					"body": []map[string]interface{}{
						{
							"type":   "TextBlock",
							"text":   "Culminate Security Investigation Report",
							"weight": "bolder",
							"size":   "large",
						},
						{
							"type":   "TextBlock",
							"text":   fmt.Sprintf("Title: %s", title),
							"weight": "bolder",
							"wrap":   true,
						},
						{
							"type":   "TextBlock",
							"text":   fmt.Sprintf("Time: %s", time),
							"weight": "bolder",
							"wrap":   true,
						},
						{
							"type":   "TextBlock",
							"text":   fmt.Sprintf("Severity: %s", severity),
							"weight": "bolder",
							"wrap":   true,
						},
						{
							"type":      "TextBlock",
							"text":      "",
							"wrap":      true,
							"separator": true,
						},
						{
							"type": "TextBlock",
							"text": description,
							"wrap": true,
						},
					},
				},
			},
		},
	}
}

// Gets the bot token from the Bot Framework API
func getBotToken() (string, time.Time, error) {
	url := "https://login.microsoftonline.com/botframework.com/oauth2/v2.0/token"
	payload := strings.NewReader("grant_type=client_credentials&client_id=" + os.Getenv("CLIENT_ID") + "&client_secret=" + os.Getenv("CLIENT_SECRET") + "&scope=https%3A%2F%2Fapi.botframework.com%2F.default")

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	expiresIn := result["expires_in"].(float64)
	expiresAt := time.Now().Add(time.Duration(expiresIn) * time.Second)

	return result["access_token"].(string), expiresAt, nil
}

// Checks if the current bot token is valid and returns it, or fetches a new one if necessary
func getValidBotToken() (string, error) {
	botTokenMutex.RLock()
	if currentBotToken != nil && time.Now().Before(currentBotToken.ExpiresAt) {
		token := currentBotToken.Token
		botTokenMutex.RUnlock()
		return token, nil
	}
	botTokenMutex.RUnlock()

	botTokenMutex.Lock()
	defer botTokenMutex.Unlock()

	// Double-check in case another goroutine refreshed the token
	if currentBotToken != nil && time.Now().Before(currentBotToken.ExpiresAt) {
		return currentBotToken.Token, nil
	}

	token, expiresAt, err := getBotToken()
	if err != nil {
		return "", fmt.Errorf("failed to get bot token: %w", err)
	}

	currentBotToken = &BotToken{
		Token:     token,
		ExpiresAt: expiresAt,
	}

	return token, nil
}

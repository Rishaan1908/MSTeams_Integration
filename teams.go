package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

// sendMessageToUser sends a message to a specific user via chat
func sendMessageToUser(token *oauth2.Token, userEmail, message string) error {
	client := oauthConfig.Client(context.Background(), token)

	// Create chat request payload
	chatPayload := map[string]interface{}{
		"chatType": "oneOnOne",
		"members": []map[string]interface{}{
			{
				"@odata.type":     "#microsoft.graph.aadUserConversationMember",
				"roles":           []string{"owner"},
				"user@odata.bind": fmt.Sprintf("https://graph.microsoft.com/v1.0/users('%s')", userEmail),
			},
		},
	}

	// Create the chat
	chatBytes, err := json.Marshal(chatPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal chat payload: %w", err)
	}

	chatResp, err := client.Post("https://graph.microsoft.com/v1.0/chats", "application/json", bytes.NewBuffer(chatBytes))
	if err != nil {
		return fmt.Errorf("failed to create chat: %w", err)
	}
	defer chatResp.Body.Close()

	if chatResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(chatResp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf("failed to create chat: status %d, response: %s", chatResp.StatusCode, bodyString)
	}

	var chatResponse map[string]interface{}
	if err := json.NewDecoder(chatResp.Body).Decode(&chatResponse); err != nil {
		return fmt.Errorf("failed to decode chat response: %w", err)
	}

	chatID, ok := chatResponse["id"].(string)
	if !ok {
		return fmt.Errorf("failed to get chat ID from response")
	}

	// Create message payload
	messagePayload := map[string]interface{}{
		"body": map[string]interface{}{
			"content": message,
		},
	}

	// Send the message to the chat
	messageBytes, err := json.Marshal(messagePayload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/chats/%s/messages", chatID)
	messageResp, err := client.Post(url, "application/json", bytes.NewBuffer(messageBytes))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer messageResp.Body.Close()

	if messageResp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(messageResp.Body)
		bodyString := string(bodyBytes)
		return fmt.Errorf("failed to send message: status %d, response: %s", messageResp.StatusCode, bodyString)
	}

	return nil
}

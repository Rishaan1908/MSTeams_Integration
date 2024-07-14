package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/oauth2"
)

const (
	integrationsFile = "integrations.json"
	messagesFile     = "messages.json"
)

// Credentials
type Teams struct {
	TenantID   string        `json:"tenant_id,omitempty"`
	OAuthToken *oauth2.Token `json:"oauth_token,omitempty"`
}

// Define integrationProps
type integrationProps struct {
	LastSyncTime time.Time `json:"last_sync_time,omitempty"`
	// Add other properties specific to your Teams integration
}

type Integration struct {
	Uuid  string           `json:"uuid"`
	Teams *Teams           `json:"teams,omitempty"`
	Props integrationProps `json:"props,omitempty"`
}

type IntegrationRequest struct {
	AccountUuid string       `json:"account_uuid"`
	Integration *Integration `json:"integration"`
	Remark      string       `json:"remark"`
}

// Conversations
type TeamsMessageRow struct {
	EventTime      time.Time `json:"event_time"`
	CaseNumber     int       `json:"case_number"`
	ThreadNumber   int       `json:"thread_number"`
	MessageNumber  int       `json:"message_number"`
	TeamsUserId    string    `json:"teams_user_id"`
	TeamsUserEmail string    `json:"teams_user_email"`
	Message        any       `json:"message"`
	Sender         string    `json:"sender"`
	IsIntended     *bool     `json:"is_intended"`
	ResponseStatus string    `json:"response_status"`
}

// Message request
type TeamsMessageRequest struct {
	AccountUuid    string `json:"accountUuid"`
	CaseNumber     int    `json:"caseNumber"`
	ThreadNumber   int    `json:"threadNumber"`
	MessageNumber  int    `json:"messageNumber,omitempty"`
	TeamsUserId    string `json:"teamsUserId"`
	TeamsUserEmail string `json:"teamsUserEmail"`
	Message        string `json:"message,omitempty"`
	Context        string `json:"context"`
}

// Other existing structs
type Team struct {
	DisplayName        string `json:"displayName"`
	Description        string `json:"description"`
	Visibility         string `json:"visibility"`
	AllowGiphy         bool   `json:"allowGiphy"`
	GiphyContentRating string `json:"giphyContentRating"`
}

// Teams channel
type Channel struct {
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// Bot token validation
type BotToken struct {
	Token     string
	ExpiresAt time.Time
}

// Graph token validation
type GraphToken struct {
	AccessToken string
	ExpiresIn   int64
	ExpiresAt   time.Time
}

// Activity struct for cards
type Activity struct {
	Type         string    `json:"type"`
	Timestamp    time.Time `json:"timestamp"`
	From         User      `json:"from"`
	Conversation struct {
		ID string `json:"id"`
	} `json:"conversation"`
	Value struct {
		UserQuestion string `json:"userQuestion"`
	} `json:"value"`
}

type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// New functions for handling integrations and messages

// Adds a new integration to the database
func AddIntegration(req IntegrationRequest) error {
	integrations, err := readIntegrations()
	if err != nil {
		return err
	}

	integrations = append(integrations, req)
	return writeIntegrations(integrations)
}

// updates an existing integration in the database
func UpdateIntegration(req IntegrationRequest, integrationUuid string) error {
	integrations, err := readIntegrations()
	if err != nil {
		return err
	}

	for i, integration := range integrations {
		if integration.Integration.Uuid == integrationUuid {
			integrations[i] = req
			return writeIntegrations(integrations)
		}
	}

	return fmt.Errorf("integration not found")
}

// sends teams messages
func SendTeamsMessage(req TeamsMessageRequest) error {
	messages, err := readMessages()
	if err != nil {
		return err
	}

	message := TeamsMessageRow{
		EventTime:      time.Now(),
		CaseNumber:     req.CaseNumber,
		ThreadNumber:   req.ThreadNumber,
		MessageNumber:  req.MessageNumber,
		TeamsUserId:    req.TeamsUserId,
		TeamsUserEmail: req.TeamsUserEmail,
		Message:        req.Message,
		Sender:         "Bot",
		IsIntended:     nil,
		ResponseStatus: "Sent",
	}

	messages = append(messages, message)
	return writeMessages(messages)
}

// Reads the messages from user
func RecordUserMessage(activity Activity) error {
	messages, err := readMessages()
	if err != nil {
		return err
	}

	message := TeamsMessageRow{
		EventTime:      activity.Timestamp,
		CaseNumber:     0, // Fetch from database
		ThreadNumber:   0,
		MessageNumber:  0,
		TeamsUserId:    activity.From.ID,
		TeamsUserEmail: "", //Fetch from user profile
		Message:        activity.Value.UserQuestion,
		Sender:         "User",
		IsIntended:     nil,
		ResponseStatus: "Received",
	}

	messages = append(messages, message)
	return writeMessages(messages)
}

// Read integrations
func readIntegrations() ([]IntegrationRequest, error) {
	var integrations []IntegrationRequest
	data, err := os.ReadFile(integrationsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return integrations, nil
		}
		return nil, err
	}
	err = json.Unmarshal(data, &integrations)
	return integrations, err
}

// Write integrations
func writeIntegrations(integrations []IntegrationRequest) error {
	data, err := json.MarshalIndent(integrations, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(integrationsFile, data, 0644)
}

// Read messages
func readMessages() ([]TeamsMessageRow, error) {
	var messages []TeamsMessageRow
	data, err := os.ReadFile(messagesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return messages, nil
		}
		return nil, err
	}
	err = json.Unmarshal(data, &messages)
	return messages, err
}

// Write messages
func writeMessages(messages []TeamsMessageRow) error {
	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(messagesFile, data, 0644)
}

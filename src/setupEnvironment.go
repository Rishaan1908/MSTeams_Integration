package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Main setup function
func setupEnvironment(token string) (string, error) {
	apps, err := listApps(token)

	if err != nil {
		return "", fmt.Errorf("failed to list apps: %v", err)
	}

	for _, app := range apps {
		if app.DisplayName == "Culminate Security" {
			err := deleteApp(token, app.ID)
			if err != nil {
				fmt.Printf("Failed to delete app %s: %v\n", app.ID, err)
			} else {
				fmt.Printf("Deleted app %s\n", app.ID)
			}
		}
	}
	// Load environment variables
	err = godotenv.Load()

	if err != nil {
		return "", fmt.Errorf("error loading .env file: %w", err)
	}

	// Check if team already exists
	teamName := os.Getenv("TEAM_NAME")
	exists, teamID, err := checkTeamExists(token, teamName)

	if err != nil {
		return "", fmt.Errorf("failed to check if team exists: %v", err)
	}

	if exists {
		return fmt.Sprintf("Team '%s' already exists. Please use the existing team. Team ID: %s", teamName, teamID), nil
	}

	// Create new team
	teamID, err = createTeam(token)

	if err != nil {
		return "", err
	}

	// Create channel in the new team
	channelID, err := createChannel(token, teamID)

	if err != nil {
		return "", err
	}

	// Update environment variables

	envVars := map[string]string{
		"TEAM_ID":    teamID,
		"CHANNEL_ID": channelID,
	}

	err = updateEnvFile(envVars)

	if err != nil {
		return "", fmt.Errorf("failed to update .env file: %v", err)
	}

	updateChannelID(channelID)

	// Reload updated environment variables
	err = godotenv.Load()

	if err != nil {
		return "", fmt.Errorf("failed to reload .env file: %v", err)
	}

	// Upload and install custom app
	appID, err := uploadAppToCatalog(token, os.Getenv("APP_ZIP"))

	if err != nil {
		return "", fmt.Errorf("failed to upload app package: %v", err)
	}

	err = installCustomApp(token, teamID, appID)

	if err != nil {
		return "", fmt.Errorf("failed to install custom app: %v", err)
	}

	// Send welcome message and sample report
	err = sendWelcomeMessage(channelID)

	if err != nil {
		log.Printf("Failed to send bot message: %v", err)
	}

	err = sendWelcomeCardAsBot(channelID)

	if err != nil {
		log.Printf("Failed to send welcome card: %v", err)
	}

	return fmt.Sprintf("New team created successfully. Team ID: %s, Channel ID: %s", teamID, channelID), nil
}

// Create the Culminate Security team
func createTeam(token string) (string, error) {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		return "", fmt.Errorf("error loading .env file: %w", err)
	}

	// Get the team picture URL from environment variable
	teamPicture := os.Getenv("TEAM_PICTURE")

	// Define the team struct including the picture URL
	team := struct {
		Template    string `json:"template@odata.bind"`
		DisplayName string `json:"displayName"`
		Description string `json:"description"`
		Visibility  string `json:"visibility"`
		Picture     string `json:"picture,omitempty"`
	}{
		Template:    "https://graph.microsoft.com/v1.0/teamsTemplates('standard')",
		DisplayName: "Culminate Security",
		Description: "No alert left behind with our AI expert investigators",
		Visibility:  "Private",
		Picture:     teamPicture,
	}

	// Marshal the team struct into JSON
	jsonData, err := json.Marshal(team)
	if err != nil {
		return "", err
	}

	// Create a POST request to create the team
	req, err := http.NewRequest("POST", graphAPIBaseURL+"/teams", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, _ := io.ReadAll(resp.Body)

	// Handle response statuses
	if resp.StatusCode == 409 {
		return "", fmt.Errorf("Team already exists")
	}

	if resp.StatusCode != 201 && resp.StatusCode != 202 {
		return "", fmt.Errorf("failed to create team: %s", string(body))
	}

	// For asynchronous creation (202 status), we need to check the operation status
	if resp.StatusCode == 202 {
		location := resp.Header.Get("Location")
		return waitForTeamCreation(token, location)
	}

	// Unmarshal the response body into a map to extract team ID
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse response: %s", string(body))
	}

	// Extract and return the team ID
	return result["id"].(string), nil
}

// waitForTeamCreation polls the provided location URL to check the status of team creation
// Retry up to 30 times with a 2-second interval between each attempt
func waitForTeamCreation(token, location string) (string, error) {
	for i := 0; i < 30; i++ { // Try up to 30 times
		time.Sleep(2 * time.Second) // Wait 2 seconds between each attempt

		// Ensure the location URL is absolute
		if !strings.HasPrefix(location, "https://") {
			location = graphAPIBaseURL + location
		}

		// Create a GET request to check the status
		req, err := http.NewRequest("GET", location, nil)
		if err != nil {
			return "", fmt.Errorf("error creating request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		// Send the request
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", fmt.Errorf("error making request: %v", err)
		}
		defer resp.Body.Close()

		// Decode the response
		var result map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return "", fmt.Errorf("error decoding response: %v", err)
		}

		// Check the status of team creation
		status, ok := result["status"].(string)
		if !ok {
			return "", fmt.Errorf("invalid status in response")
		}

		switch status {
		case "succeeded":
			// If succeeded, return the team ID
			targetResourceId, ok := result["targetResourceId"].(string)
			if !ok {
				return "", fmt.Errorf("invalid targetResourceId in response")
			}
			return targetResourceId, nil
		case "failed":
			// If failed, return an error
			return "", fmt.Errorf("team creation failed: %v", result["error"])
		case "inProgress", "notStarted":
			// If still in progress, continue waiting
		default:
			return "", fmt.Errorf("unknown status: %s", status)
		}
	}

	// timeout error
	return "", fmt.Errorf("timeout waiting for team creation")
}

// Creates a new channel named "Reports"
func createChannel(token, teamID string) (string, error) {
	// Define the channel properties
	channel := Channel{
		DisplayName: "Reports",
		Description: "Channel to receive Culminate Security Reports",
	}

	// Marshal the channel data to JSON
	jsonData, err := json.Marshal(channel)
	if err != nil {
		return "", err
	}

	// Create a POST request to create the channel
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/teams/%s/channels", graphAPIBaseURL, teamID), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if the channel was created successfully
	if resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create channel: %s", string(body))
	}

	// Decode the response to get the channel ID
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	channelID := result["id"].(string)

	return channelID, nil
}

// installCustomApp installs a custom app in the specified team
func installCustomApp(token, teamID, appID string) error {
	url := fmt.Sprintf("%s/teams/%s/installedApps", graphAPIBaseURL, teamID)

	// Prepare the payload for installing the app
	payload := map[string]string{
		"teamsApp@odata.bind": fmt.Sprintf("%s/appCatalogs/teamsApps/%s", graphAPIBaseURL, appID),
	}

	// Marshal the payload to JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	// Create a POST request to install the app
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the app was installed successfully
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to install custom app: %s", string(body))
	}

	return nil
}

// uploadAppToCatalog uploads a custom app package to the Teams app catalog
func uploadAppToCatalog(token string, appZipPath string) (string, error) {
	// Open the app package file
	appZip, err := os.Open(appZipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open app package file: %w", err)
	}
	defer appZip.Close()

	url := fmt.Sprintf("%s/appCatalogs/teamsApps", graphAPIBaseURL)

	// Create a POST request to upload the app package
	req, err := http.NewRequest("POST", url, appZip)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/zip")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload app package: %w", err)
	}
	defer resp.Body.Close()

	// Check if the app was uploaded successfully
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload app package: %s", string(body))
	}

	// Decode the response to get the app ID
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	appID, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("invalid response format")
	}

	return appID, nil
}

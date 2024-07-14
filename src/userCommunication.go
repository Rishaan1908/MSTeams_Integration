package main

// Send welcome card to specified conversation
func sendWelcomeCardToConversation(conversationID, displayName string) error {
	card := createWelcomeCard()
	return sendBotMessage(conversationID, "", card)
}

// Create welcome adaptive card
func createWelcomeCard() map[string]interface{} {
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
							"type": "TextBlock",
							"text": "Feel free to ask me any questions!",
							"wrap": true,
						},
						{
							"type":        "Input.Text",
							"id":          "userQuestion",
							"placeholder": "Ask a question...",
						},
						{
							"type": "ActionSet",
							"actions": []map[string]interface{}{
								{
									"type":  "Action.Submit",
									"title": "Send",
								},
							},
						},
					},
				},
			},
		},
	}
}
func sendWelcomeCardAsBot(channelID string) error {
	card := createWelcomeCard()
	return sendBotMessage(channelID, "", card)
}

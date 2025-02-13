package utils

import (
	"log"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Text string `json:"text"`
}

// SendGoogleChatNotification sends a notification to Google Chat
func SendGoogleChatNotification(webhookURL, message, clientIP, shortURL, originalURL string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	if !strings.HasPrefix(webhookURL, "https://chat.googleapis.com/") {
		return fmt.Errorf("invalid webhook URL format: %s", webhookURL)
	}

	text := fmt.Sprintf("*%s*\n\nClient IP: %s\nShort URL: %s\nOriginal URL: %s\n",
		message, clientIP, shortURL, originalURL)

	chatMsg := ChatMessage{
		Text: text,
	}

	jsonData, err := json.Marshal(chatMsg)
	if err != nil {
		return fmt.Errorf("error marshaling chat message: %v", err)
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to send notification, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func SendGoogleChatNotificationAsync(webhookURL, message, clientIP, shortURL, originalURL string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in notification: %v", r)
			}
		}()

		log.Printf("Attempting to send notification for URL: %s", shortURL)
		err := SendGoogleChatNotification(webhookURL, message, clientIP, shortURL, originalURL)
		if err != nil {
			log.Printf("Failed to send notification: %v", err)
			return
		}
		log.Printf("Successfully sent notification for URL: %s", shortURL)
	}()
}

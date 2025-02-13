package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// SendGoogleChatNotificationAsync sends a notification asynchronously
func SendGoogleChatNotificationAsync(webhookURL, message, clientIP, shortURL, originalURL string) {
	go func() {
		_ = SendGoogleChatNotification(webhookURL, message, clientIP, shortURL, originalURL)
	}()
}

type ChatMessage struct {
	Text string `json:"text"`
}

func SendGoogleChatNotification(webhookURL, message, clientIP, shortURL, originalURL string) error {
	if !strings.HasPrefix(webhookURL, "https://chat.googleapis.com/") {
		return fmt.Errorf("invalid webhook URL format")
	}

	if webhookURL == "" {
		return fmt.Errorf("webhook URL is empty")
	}

	qrCodeURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=%s", shortURL)
	text := fmt.Sprintf("*%s*\n\nClient IP: %s\nShort URL: %s\nOriginal URL: %s\nQR Code: %s",
		message, clientIP, shortURL, originalURL, qrCodeURL)

	chatMsg := ChatMessage{
		Text: text,
	}

	jsonData, err := json.Marshal(chatMsg)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send notification, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

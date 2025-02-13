package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Text string `json:"text"`
}

func SendGoogleChatNotification(webhookURL, message, clientIP, shortURL, originalURL string) error {
	fmt.Println("\n=== Google Chat Notification Debug ===")
	fmt.Printf("1. Webhook URL length: %d\n", len(webhookURL))
	fmt.Printf("2. First 10 chars of webhook: %s\n", webhookURL[:10])
	fmt.Printf("3. Message: %s\n", message)

	// Validate webhook URL format
	if !strings.HasPrefix(webhookURL, "https://chat.googleapis.com/") {
		return fmt.Errorf("invalid webhook URL format")
	}

	fmt.Println("🔔 Starting Google Chat Notification")
	fmt.Printf("Message: %s\n", message)
	fmt.Printf("Client IP: %s\n", clientIP)
	fmt.Printf("Short URL: %s\n", shortURL)

	if webhookURL == "" {
		fmt.Println("❌ ERROR: Webhook URL is empty!")
		return fmt.Errorf("webhook URL is empty")
	}

	// Mask the webhook URL for security but show the length
	maskedURL := fmt.Sprintf("length:%d ...%s", len(webhookURL), webhookURL[len(webhookURL)-30:])
	fmt.Printf("Using webhook URL: %s\n", maskedURL)

	fmt.Printf("Preparing to send notification to webhook: %s\n", webhookURL)

	qrCodeURL := fmt.Sprintf("https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=%s", shortURL)

	text := fmt.Sprintf("*%s*\n\nClient IP: %s\nShort URL: %s\nOriginal URL: %s\nQR Code: %s",
		message, clientIP, shortURL, originalURL, qrCodeURL)

	chatMsg := ChatMessage{
		Text: text,
	}

	jsonData, err := json.Marshal(chatMsg)
	if err != nil {
		fmt.Printf("Error marshaling chat message: %v\n", err)
		return err
	}

	fmt.Printf("Sending notification with payload: %s\n", string(jsonData))

	// Add request debugging
	req, _ := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	fmt.Println("4. Sending request...")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error sending chat notification: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Response status: %d, body: %s\n", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Failed with status %d\n", resp.StatusCode)
		return fmt.Errorf("failed to send notification, status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Println("✅ Chat notification sent successfully")
	fmt.Println("========== ENDING GOOGLE CHAT NOTIFICATION ==========")
	log.Output(2, "Function completed") // Force flush the log buffer
	return nil
}

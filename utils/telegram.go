package utils

import (
    "fmt"
    "log"
    "net/http"
    "net/url"
)

func SendTelegramNotificationAsync(botToken, chatID, message, clientIP, shortURL, originalURL string) {
    go func() {
        err := SendTelegramNotification(botToken, chatID, message, clientIP, shortURL, originalURL)
        if err != nil {
            log.Printf("Failed to send Telegram notification: %v", err)
        }
    }()
}

func SendTelegramNotification(botToken, chatID, message, clientIP, shortURL, originalURL string) error {
    apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
    
    text := fmt.Sprintf("🔗 *%s*\n\n👤 Client IP: `%s`\n📎 Short URL: %s\n🌐 Original URL: %s",
        message, clientIP, shortURL, originalURL)
    
    params := url.Values{}
    params.Add("chat_id", chatID)
    params.Add("text", text)
    params.Add("parse_mode", "Markdown")
    
    resp, err := http.PostForm(apiURL, params)
    if err != nil {
        return fmt.Errorf("error sending telegram message: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("telegram API returned status code: %d", resp.StatusCode)
    }
    
    return nil
}
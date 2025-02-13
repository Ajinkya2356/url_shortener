package controller

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"url-shortener/constants"
	"url-shortener/service"
	"url-shortener/storage"
	"url-shortener/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShortenRequest struct {
	URL   string `json:"url" binding:"required"`
	Alias string `json:"alias,omitempty"`
}

type URLResponse struct {
	ShortURL string `json:"shortURL"`
}

func ShortenURL(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// check if orignal url exists
		var existingURL storage.URL
		urlExists := db.Where("original_url = ?", req.URL).First(&existingURL).Error == nil

		var finalAlias string
		if req.Alias != "" {
			var aliasCheck storage.URL
			if err := db.Where("alias = ?", req.Alias).First(&aliasCheck).Error; err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Alias already taken!"})
				return
			}
			finalAlias = req.Alias
		} else {
			hash := service.GenerateURLShortener(req.URL)
			for {
				var hashCheck storage.URL
				if err := db.Where("alias = ?", hash).First(&hashCheck).Error; err != nil {
					finalAlias = hash
					break
				}
				hash = service.GenerateURLShortener(hash)
			}
		}

		// Get the protocol (http or https)
		protocol := "http"
		if c.Request.TLS != nil {
			protocol = "https"
		}

		if urlExists {
			existingURL.Alias = finalAlias
			if err := db.Save(&existingURL).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update URL"})
				return
			}
			shortURL := fmt.Sprintf("%s://%s/%s", protocol, c.Request.Host, existingURL.Alias)
			fmt.Printf("Generated short URL: %s\n", shortURL) // Debug log
			c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
			return
		}
		newURL := storage.URL{
			OriginalURL: req.URL,
			Alias:       finalAlias,
		}
		if err := db.Create(&newURL).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create URL"})
			return
		}
		shortURL := fmt.Sprintf("%s://%s/%s", protocol, c.Request.Host, newURL.Alias)
		fmt.Printf("Generated short URL: %s\n", shortURL) // Debug log

		// Send notification to Google Chat
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		fmt.Printf("🔗 New URL shortened: %s\n", shortURL)
		fmt.Printf("🔔 Webhook URL present: %v\n", webhookURL != "")

		if webhookURL != "" {
			clientIP := c.ClientIP()
			fmt.Printf("👤 Client IP: %s\n", clientIP)
			err := utils.SendGoogleChatNotification(
				webhookURL,
				constants.URLShortenerSuccess,
				clientIP,
				shortURL,
				req.URL,
			)
			if err != nil {
				fmt.Printf("❌ Notification error: %v\n", err)
			}
		} else {
			fmt.Println("⚠️ GOOGLE_CHAT_WEBHOOK_URL not set")
		}

		c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
	}
}

func ResolveURL(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortURL := c.Param("shortURL")
		fmt.Printf("Resolving URL: %s\n", shortURL) // Debug log
		var url storage.URL

		if err := db.Where("alias = ?", shortURL).First(&url).Error; err != nil {
			fmt.Printf("Error resolving URL: %v\n", err) // Debug log
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}

		fmt.Printf("Redirecting to: %s\n", url.OriginalURL) // Debug log

		// Send notification to Google Chat
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		log.Printf("Attempting to send notification with webhook URL: %v", webhookURL != "")
		if webhookURL != "" {
			protocol := "http"
			if c.Request.TLS != nil {
				protocol = "https"
			}
			clientIP := c.ClientIP()
			fullShortURL := fmt.Sprintf("%s://%s/%s", protocol, c.Request.Host, shortURL)
			err := utils.SendGoogleChatNotification(
				webhookURL,
				constants.URLRedirectSuccess,
				clientIP,
				fullShortURL,
				url.OriginalURL,
			)
			if err != nil {
				log.Printf("Failed to send chat notification: %v", err)
			}
		} else {
			log.Printf("ERROR: GOOGLE_CHAT_WEBHOOK_URL environment variable is not set")
		}

		c.Redirect(http.StatusFound, url.OriginalURL)
	}
}

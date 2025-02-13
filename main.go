package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"url-shortener/storage"
	"url-shortener/utils"
	"url-shortener/constants"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"url-shortener/service"
)

type ShortenRequest struct {
	URL   string `json:"url" binding:"required"`
	Alias string `json:"alias,omitempty"`
}

type URLResponse struct {
	ShortURL string `json:"shortURL"`
}

func init() {
	// Configure detailed logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stdout)

	// Set Gin to debug mode to see all logs
	gin.SetMode(gin.DebugMode)  // Changed from ReleaseMode
}

// First, add a helper function at the top level
func sendNotificationViaWebhook(message, clientIP, shortURL, originalURL string) {
    notificationData := map[string]string{
        "message":     message,
        "clientIP":    clientIP,
        "shortURL":    shortURL,
        "originalURL": originalURL,
    }

    jsonData, err := json.Marshal(notificationData)
    if err != nil {
        log.Printf("Error marshaling notification data: %v", err)
        return
    }

    req, err := http.NewRequest("POST", "http://localhost:8080/test-webhook", bytes.NewBuffer(jsonData))
    if err != nil {
        log.Printf("Error creating request: %v", err)
        return
    }
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 5 * time.Second}
    _, err = client.Do(req)
    if err != nil {
        log.Printf("Error sending notification request: %v", err)
    }
}

func main() {
	db := storage.InitDB()
	router := gin.Default()
	router.Use(gin.Logger())
	router.SetTrustedProxies(nil)

	// Add custom logging middleware
	router.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%v] - %s %s \"%s\" %d %v %s\n",
			param.TimeStamp.Format(time.RFC3339),
			param.Method,
			param.Path,
			param.Request.UserAgent(),
			param.StatusCode,
			param.Latency,
			param.ErrorMessage,
		)
	}))

	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Add request logging middleware
	router.Use(func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Log request details
		log.Printf("Request: %s %s | Status: %d | Duration: %v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	})

	router.StaticFile("/", "./index.html")
	router.POST("/encode", func(c*gin.Context){
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// check if original url exists
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

		protocol := "https"
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
		log.Println("Shortened URL created successfully", shortURL)
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		if webhookURL != "" {
			log.Println("Sending Google Chat notification")
			go sendNotificationViaWebhook(
				constants.URLShortenerSuccess,
				c.ClientIP(),
				shortURL,
				req.URL,
			)
		}
		c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
	})
	router.GET("/:shortURL", func(c*gin.Context){
		shortURL := c.Param("shortURL")
		var url storage.URL

		if err := db.Where("alias = ?", shortURL).First(&url).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		c.Redirect(http.StatusFound, url.OriginalURL)
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		if webhookURL != "" {
			protocol := "http"
			if c.Request.TLS != nil {
				protocol = "https"
			}
			fullShortURL := fmt.Sprintf("%s://%s/%s", protocol, c.Request.Host, shortURL)

			go sendNotificationViaWebhook(
				constants.URLRedirectSuccess,
				c.ClientIP(),
				fullShortURL,
				url.OriginalURL,
			)
		}
	})

	// Enhanced test webhook endpoint
	router.GET("/test-webhook", func(c *gin.Context) {
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		if webhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook URL not set"})
			return
		}

		err := utils.SendGoogleChatNotification(
			webhookURL,
			"Test notification from URL Shortener",
			c.ClientIP(),
			"test-short-url",
			"test-original-url",
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Test notification sent successfully"})
	})

	// Update the test-webhook endpoint to handle POST requests with data
	router.POST("/test-webhook", func(c *gin.Context) {
		var notificationData struct {
			Message     string `json:"message"`
			ClientIP    string `json:"clientIP"`
			ShortURL    string `json:"shortURL"`
			OriginalURL string `json:"originalURL"`
		}

		if err := c.ShouldBindJSON(&notificationData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification data"})
			return
		}

		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		if webhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook URL not set"})
			return
		}

		err := utils.SendGoogleChatNotification(
			webhookURL,
			notificationData.Message,
			notificationData.ClientIP,
			notificationData.ShortURL,
			notificationData.OriginalURL,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Notification sent successfully"})
	})

	router.Run(":8080")
}

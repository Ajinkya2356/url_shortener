package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"url-shortener/storage"
	"url-shortener/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"url-shortener/service"
	"url-shortener/constants"
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
	gin.SetMode(gin.ReleaseMode)  // Changed from ReleaseMode
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
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		if webhookURL != "" {
			utils.SendGoogleChatNotificationAsync(
				webhookURL,
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
	})

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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook URL not configured"})
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

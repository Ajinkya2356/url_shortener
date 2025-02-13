package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"strings"
	"url-shortener/constants"
	"url-shortener/service"
	"url-shortener/storage"
	"url-shortener/utils"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type ShortenRequest struct {
	URL   string `json:"url" binding:"required"`
	Alias string `json:"alias,omitempty"`
}

type URLResponse struct {
	ShortURL string `json:"shortURL"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading file")
	}
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT_ID")
	
	if botToken == "" {
		log.Println("Warning: TELEGRAM_BOT_TOKEN is not set")
	}
	if chatID == "" {
		log.Println("Warning: TELEGRAM_CHAT_ID is not set")
	}
	if botToken != "" && chatID != "" {
		log.Printf("Telegram configuration loaded for chat ID: %s", chatID)
	}
}

func main() {
	db := storage.InitDB()
	router := gin.Default()
	router.Use(gin.Logger())
	router.SetTrustedProxies([]string{"127.0.0.1"})

	getClientIP := func(c *gin.Context) string {
        // Check X-Real-IP header first
        if ip := c.GetHeader("X-Real-IP"); ip != "" {
            return ip
        }
        
        // Check X-Forwarded-For header
        if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
            return strings.Split(ip, ",")[0] // Get the first IP if multiple
        }
        
        // Fallback to RemoteAddr
        return c.Request.RemoteAddr
    }

	// Custom logging middleware:
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

	// Request logging middleware:
	router.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("Request: %s %s | Status: %d | Duration: %v",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start),
		)
	})

	router.StaticFile("/", "./index.html")
	router.POST("/encode", func(c *gin.Context) {
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if original URL exists
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
		if c.Request.Header.Get("X-Forwarded-Proto") == "https" || c.Request.TLS != nil {
			protocol = "https"
		} else if c.Request.Header.Get("X-Forwarded-Proto") != "" {
			protocol = c.Request.Header.Get("X-Forwarded-Proto")
		}

		if urlExists {
			existingURL.Alias = finalAlias
			if err := db.Save(&existingURL).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update URL"})
				return
			}
			shortURL := fmt.Sprintf("%s://%s/%s", protocol, c.Request.Host, existingURL.Alias)
			botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
			chatID := os.Getenv("TELEGRAM_CHAT_ID")
			if botToken != "" && chatID != "" {
				utils.SendTelegramNotificationAsync(
					botToken,
					chatID,
					constants.URLShortenerSuccess,
					getClientIP(c),
					shortURL,
					req.URL,
				)
			}
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
		botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
		chatID := os.Getenv("TELEGRAM_CHAT_ID")
		if botToken != "" && chatID != "" {
			utils.SendTelegramNotificationAsync(
				botToken,
				chatID,
				constants.URLShortenerSuccess,
				getClientIP(c),
				shortURL,
				req.URL,
			)
		}
		c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
	})

	router.GET("/:shortURL", func(c *gin.Context) {
		shortURL := c.Param("shortURL")
		var url storage.URL
		if err := db.Where("alias = ?", shortURL).First(&url).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Short URL not found"})
			return
		}
		c.Redirect(http.StatusFound, url.OriginalURL)
	})

	// /test-webhook route omitted for brevity.

	router.Run(":8080")
}

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"url-shortener/controller"
	"url-shortener/storage"
	"url-shortener/utils"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Configure logging to include file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Ensure logs go to stdout
	log.SetOutput(os.Stdout)

	fmt.Println("Starting URL Shortener service...")

	db := storage.InitDB()
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.StaticFile("/", "./index.html")
	router.Static("/static", "./static")
	router.POST("/encode", controller.ShortenURL(db))
	router.GET("/:shortURL", controller.ResolveURL(db))

	// Add test endpoint for webhook verification
	router.GET("/test-webhook", func(c *gin.Context) {
		webhookURL := os.Getenv("GOOGLE_CHAT_WEBHOOK_URL")
		fmt.Printf("🔧 Test - Webhook URL exists: %v\n", webhookURL != "")
		fmt.Printf("🔧 Test - Webhook URL: %s\n", webhookURL)

		if webhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Webhook URL not set"})
			return
		}

		err := utils.SendGoogleChatNotification(
			webhookURL,
			"Test notification",
			"test-ip",
			"test-short-url",
			"test-original-url",
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Test notification sent"})
	})

	router.Run(":8080")
}

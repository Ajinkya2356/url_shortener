package controller

import (
	"net/http"
	"strings"
	"url-shortener/service"
	"url-shortener/storage"

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

		var shortURL string
		// If alias is provided, check if it's available
		if req.Alias != "" && strings.TrimSpace(req.Alias) != "" {
			var existingURL storage.URL
			if err := db.Where("alias = ?", req.Alias).First(&existingURL).Error; err == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Alias already taken"})
				return
			}
			shortURL = req.Alias
		} else {
			// Generate short URL if no alias provided
			shortURL = service.GenerateURLShortener(req.URL)
			for {
				var existingURL storage.URL
				if err := db.Where("short_url = ?", shortURL).First(&existingURL).Error; err != nil {
					break
				}
				shortURL = service.GenerateURLShortener(shortURL)
			}
		}

		// Check if original URL exists
		var url storage.URL
		if err := db.Where("original_url = ?", req.URL).First(&url).Error; err == nil {
			// URL exists, update the short URL and alias
			url.ShortURL = shortURL
			url.Alias = req.Alias
			if err := db.Save(&url).Error; err != nil {
				if strings.Contains(err.Error(), "unique constraint") {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Alias already taken"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			// Create new entry
			url = storage.URL{
				OriginalURL: req.URL,
				ShortURL:    shortURL,
				Alias:       req.Alias,
			}
			if err := db.Create(&url).Error; err != nil {
				if strings.Contains(err.Error(), "unique constraint") {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Alias already taken"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		fullShortURL := "http://" + c.Request.Host + "/" + shortURL
		c.JSON(http.StatusOK, URLResponse{ShortURL: fullShortURL})
	}
}

func ResolveURL(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortURL := c.Param("shortURL")
		var url storage.URL
		// Check both short URL and alias
		if err := db.Where("short_url = ? OR alias = ?", shortURL, shortURL).First(&url).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, url.OriginalURL)
	}
}

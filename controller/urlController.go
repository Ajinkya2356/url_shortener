package controller

import (
	"net/http"
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

		if urlExists {
			existingURL.Alias = finalAlias
			if err := db.Save(&existingURL).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update URL"})
				return
			}
			shortURL := "http://" + c.Request.Host + "/" + existingURL.Alias
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
		shortURL := "http://" + c.Request.Host + "/" + newURL.Alias
		c.JSON(http.StatusOK, gin.H{"shortURL": shortURL})
	}
}

func ResolveURL(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		shortURL := c.Param("shortURL")
		var url storage.URL
		// Check alias
		if err := db.Where("alias = ?", shortURL).First(&url).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, url.OriginalURL)
	}
}

package controller

import (
	"net/http"
	"url-shortener/service"
	"url-shortener/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ShortenRequest struct {
	URL string `json:"url" binding:"required"`
}

type URLResponse struct {
	ShortURL string `json:"shortURL"`
}

func ShortenURL(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ShortenRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}
		shortURL := service.GenerateURLShortener(req.URL)
		var url storage.URL
		if err := db.Where("short_url = ?", shortURL).First(&url).Error; err == nil {
			// same hash exists in the DB
			shortURL = service.GenerateURLShortener(shortURL)
		}

		if err := db.Where("original_url = ?", req.URL).First(&url).Error; err == nil {
			// URL already exists, update the short URL
			url.ShortURL = shortURL
			if err := db.Save(&url).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			// URL does not exist, create a new entry
			url = storage.URL{
				OriginalURL: req.URL,
				ShortURL:    shortURL,
			}
			if err := db.Create(&url).Error; err != nil {
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
		if err := db.Where("short_url = ?", shortURL).First(&url).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, url.OriginalURL)
	}
}

package main

import (
	"url-shortener/controller"
	"url-shortener/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
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
	router.POST("/encode", controller.ShortenURL(db))
	router.GET("/:shortURL", controller.ResolveURL(db))
	router.Run(":5000")
}

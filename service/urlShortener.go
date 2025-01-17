package service

import (
	"crypto/sha256"
	"encoding/base64"
)

func GenerateURLShortener(originalString string) string {
	hash := sha256.Sum256([]byte(originalString))
	shortURL := base64.URLEncoding.EncodeToString(hash[:])[:8]
	return shortURL
}
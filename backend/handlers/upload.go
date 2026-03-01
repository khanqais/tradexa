package handlers

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/khanqais/tradexa/config"
)

func UploadImage(c *gin.Context) {
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "image file is required",
		})
		return
	}
	defer file.Close()
	//only allow image file types
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only jpg, jpeg, png, webp files are allowed"})
		return
	}
	if header.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size must be under 5MB"})
		return
	}

	uploadResult, err := config.Cloudinary.Upload.Upload(
		context.Background(),
		file,
		uploader.UploadParams{
			Folder: "tradexa/listings",
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":       uploadResult.SecureURL,
		"public_id": uploadResult.PublicID,
	})

}

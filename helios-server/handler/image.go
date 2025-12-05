// Package handler provides HTTP request handlers.
package handler

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	"nfcunha/helios/core/service"

	"github.com/gin-gonic/gin"
)

// ImageHandler handles image-related HTTP requests.
type ImageHandler struct {
	imageService *service.ImageService
}

// NewImageHandler creates a new image handler.
func NewImageHandler(imageService *service.ImageService) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
	}
}

// ListImages handles GET /images
func (h *ImageHandler) ListImages(c *gin.Context) {
	// Parse query parameters
	all := c.DefaultQuery("all", "false") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	images, err := h.imageService.ListImages(ctx, all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to list images",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// InspectImage handles GET /images/:id
func (h *ImageHandler) InspectImage(c *gin.Context) {
	imageID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	detail, err := h.imageService.InspectImage(ctx, imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Failed to inspect image",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// PullImage handles POST /images/pull
func (h *ImageHandler) PullImage(c *gin.Context) {
	var req struct {
		Image string `json:"image" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	progressChan, errChan, err := h.imageService.PullImage(ctx, req.Image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to start image pull",
			"detail": err.Error(),
		})
		return
	}

	// Stream progress updates as Server-Sent Events (SSE)
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")

	c.Stream(func(w io.Writer) bool {
		select {
		case progress, ok := <-progressChan:
			if !ok {
				// Channel closed, pull completed
				c.SSEvent("complete", gin.H{
					"status": "Pull completed successfully",
				})
				return false
			}
			// Send progress update
			c.SSEvent("progress", progress)
			return true

		case err := <-errChan:
			if err != nil {
				c.SSEvent("error", gin.H{
					"error": err.Error(),
				})
			}
			return false

		case <-ctx.Done():
			c.SSEvent("error", gin.H{
				"error": "Pull operation timed out",
			})
			return false
		}
	})
}

// RemoveImage handles DELETE /images/:id
func (h *ImageHandler) RemoveImage(c *gin.Context) {
	imageID := c.Param("id")
	force := c.DefaultQuery("force", "false") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := h.imageService.RemoveImage(ctx, imageID, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to remove image",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image removed successfully",
		"id":      imageID,
	})
}

// BulkRemoveImages handles POST /images/bulk/remove
func (h *ImageHandler) BulkRemoveImages(c *gin.Context) {
	var req struct {
		ImageIDs []string `json:"image_ids" binding:"required"`
		Force    bool     `json:"force"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	if len(req.ImageIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No image IDs provided",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	results := h.imageService.BulkRemoveImages(ctx, req.ImageIDs, req.Force)

	// Count successful and failed operations
	successful := countSuccessful(results)
	failed := countFailed(results)

	c.JSON(http.StatusOK, gin.H{
		"results":    results,
		"total":      len(results),
		"successful": successful,
		"failed":     failed,
	})
}

// PruneImages handles POST /images/prune
func (h *ImageHandler) PruneImages(c *gin.Context) {
	all := c.DefaultQuery("all", "false") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	spaceReclaimed, err := h.imageService.PruneImages(ctx, all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to prune images",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "Images pruned successfully",
		"space_reclaimed":    spaceReclaimed,
		"space_reclaimed_mb": float64(spaceReclaimed) / 1024 / 1024,
	})
}

// SearchImages handles GET /images/search
func (h *ImageHandler) SearchImages(c *gin.Context) {
	term := c.Query("term")
	if term == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Missing search term",
			"detail": "Query parameter 'term' is required",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	results, err := h.imageService.SearchImages(ctx, term, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to search images",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"count":   len(results),
		"term":    term,
	})
}

// GetImageTags handles GET /images/tags
func (h *ImageHandler) GetImageTags(c *gin.Context) {
	imageName := c.Query("image")
	if imageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Missing image name",
			"detail": "Query parameter 'image' is required",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	tags, err := h.imageService.GetImageTags(ctx, imageName, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to fetch image tags",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tags":  tags,
		"count": len(tags),
		"image": imageName,
	})
}

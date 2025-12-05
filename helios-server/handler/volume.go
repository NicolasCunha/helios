// Package handler provides HTTP request handlers.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/helios/core/service"
)

// VolumeHandler handles volume-related HTTP requests.
type VolumeHandler struct {
	volumeService *service.VolumeService
}

// NewVolumeHandler creates a new volume handler.
func NewVolumeHandler(volumeService *service.VolumeService) *VolumeHandler {
	return &VolumeHandler{
		volumeService: volumeService,
	}
}

// ListVolumes handles GET /volumes
func (h *VolumeHandler) ListVolumes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	volumes, err := h.volumeService.ListVolumes(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to list volumes",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"volumes": volumes,
		"count":   len(volumes),
	})
}

// InspectVolume handles GET /volumes/:name
func (h *VolumeHandler) InspectVolume(c *gin.Context) {
	volumeName := c.Param("name")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	detail, err := h.volumeService.InspectVolume(ctx, volumeName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Failed to inspect volume",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// CreateVolume handles POST /volumes
func (h *VolumeHandler) CreateVolume(c *gin.Context) {
	var req service.CreateVolumeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	detail, err := h.volumeService.CreateVolume(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to create volume",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// RemoveVolume handles DELETE /volumes/:name
func (h *VolumeHandler) RemoveVolume(c *gin.Context) {
	volumeName := c.Param("name")
	force := c.DefaultQuery("force", "false") == "true"

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := h.volumeService.RemoveVolume(ctx, volumeName, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to remove volume",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Volume removed successfully",
		"name":    volumeName,
	})
}

// PruneVolumes handles POST /volumes/prune
func (h *VolumeHandler) PruneVolumes(c *gin.Context) {
	// Parse optional filters from request body
	var req struct {
		Filters map[string][]string `json:"filters"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		// No body is fine, use empty filters
		req.Filters = make(map[string][]string)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	spaceReclaimed, volumesDeleted, err := h.volumeService.PruneVolumes(ctx, req.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to prune volumes",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Volumes pruned successfully",
		"space_reclaimed":  spaceReclaimed,
		"space_reclaimed_mb": float64(spaceReclaimed) / 1024 / 1024,
		"volumes_deleted":  volumesDeleted,
		"count":            len(volumesDeleted),
	})
}

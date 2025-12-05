// Package handler provides HTTP handlers for the Helios API.
package handler

import (
	"net/http"
	"strconv"

	"nfcunha/helios/core/service"

	"github.com/gin-gonic/gin"
)

// ContainerHandler handles container-related HTTP requests.
type ContainerHandler struct {
	containerService *service.ContainerService
}

// NewContainerHandler creates a new container handler.
func NewContainerHandler(containerService *service.ContainerService) *ContainerHandler {
	return &ContainerHandler{
		containerService: containerService,
	}
}

// ListContainers handles GET /helios/containers
// Query parameters:
//   - all: boolean (include stopped containers)
//   - limit: integer (max number of results)
//   - filter: string (filter by name)
//   - stats: boolean (include resource stats - default true)
func (h *ContainerHandler) ListContainers(c *gin.Context) {
	// Default to including stats, but allow disabling for performance
	includeStats := c.Query("stats") != "false"

	opts := service.ContainerListOptions{
		All:          c.Query("all") == "true",
		IncludeStats: includeStats,
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			opts.Limit = limit
		}
	}

	if filter := c.Query("filter"); filter != "" {
		opts.Filter = filter
	}

	containers, err := h.containerService.ListContainers(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to list containers",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"containers": containers,
		"count":      len(containers),
	})
}

// GetContainer handles GET /helios/containers/:id
func (h *ContainerHandler) GetContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	container, err := h.containerService.GetContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Container not found",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, container)
}

// StartContainer handles POST /helios/containers/:id/start
func (h *ContainerHandler) StartContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	err := h.containerService.StartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to start container",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Container started successfully",
		"id":      containerID,
	})
}

// StopContainer handles POST /helios/containers/:id/stop
func (h *ContainerHandler) StopContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	err := h.containerService.StopContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to stop container",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Container stopped successfully",
		"id":      containerID,
	})
}

// RestartContainer handles POST /helios/containers/:id/restart
func (h *ContainerHandler) RestartContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	err := h.containerService.RestartContainer(c.Request.Context(), containerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to restart container",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Container restarted successfully",
		"id":      containerID,
	})
}

// RemoveContainer handles DELETE /helios/containers/:id
// Query parameters:
//   - force: boolean (force removal of running container)
func (h *ContainerHandler) RemoveContainer(c *gin.Context) {
	containerID := c.Param("id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Container ID is required",
		})
		return
	}

	force := c.Query("force") == "true"

	err := h.containerService.RemoveContainer(c.Request.Context(), containerID, force)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to remove container",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Container removed successfully",
		"id":      containerID,
	})
}

// BulkStartContainers handles POST /helios/containers/bulk/start
func (h *ContainerHandler) BulkStartContainers(c *gin.Context) {
	var req struct {
		ContainerIDs []string `json:"container_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	results := h.containerService.BulkStartContainers(c.Request.Context(), req.ContainerIDs)

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.ContainerIDs),
		"success": countSuccessful(results),
		"failed":  countFailed(results),
	})
}

// BulkStopContainers handles POST /helios/containers/bulk/stop
func (h *ContainerHandler) BulkStopContainers(c *gin.Context) {
	var req struct {
		ContainerIDs []string `json:"container_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	results := h.containerService.BulkStopContainers(c.Request.Context(), req.ContainerIDs)

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.ContainerIDs),
		"success": countSuccessful(results),
		"failed":  countFailed(results),
	})
}

// BulkRemoveContainers handles POST /helios/containers/bulk/remove
func (h *ContainerHandler) BulkRemoveContainers(c *gin.Context) {
	var req struct {
		ContainerIDs []string `json:"container_ids" binding:"required"`
		Force        bool     `json:"force"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	results := h.containerService.BulkRemoveContainers(c.Request.Context(), req.ContainerIDs, req.Force)

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(req.ContainerIDs),
		"success": countSuccessful(results),
		"failed":  countFailed(results),
	})
}

func countSuccessful(results []service.BulkOperationResult) int {
	count := 0
	for _, r := range results {
		if r.Success {
			count++
		}
	}
	return count
}

func countFailed(results []service.BulkOperationResult) int {
	count := 0
	for _, r := range results {
		if !r.Success {
			count++
		}
	}
	return count
}

// GetDashboardSummary handles GET /helios/dashboard/summary
func (h *ContainerHandler) GetDashboardSummary(c *gin.Context) {
	summary, err := h.containerService.GetDashboardSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to get dashboard summary",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

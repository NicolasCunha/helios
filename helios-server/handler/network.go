// Package handler provides HTTP request handlers.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"nfcunha/helios/core/service"
)

// NetworkHandler handles network-related HTTP requests.
type NetworkHandler struct {
	networkService *service.NetworkService
}

// NewNetworkHandler creates a new network handler.
func NewNetworkHandler(networkService *service.NetworkService) *NetworkHandler {
	return &NetworkHandler{
		networkService: networkService,
	}
}

// ListNetworks handles GET /networks
func (h *NetworkHandler) ListNetworks(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	networks, err := h.networkService.ListNetworks(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to list networks",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"networks": networks,
		"count":    len(networks),
	})
}

// InspectNetwork handles GET /networks/:id
func (h *NetworkHandler) InspectNetwork(c *gin.Context) {
	networkID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	detail, err := h.networkService.InspectNetwork(ctx, networkID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Failed to inspect network",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// CreateNetwork handles POST /networks
func (h *NetworkHandler) CreateNetwork(c *gin.Context) {
	var req service.CreateNetworkRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Invalid request body",
			"detail": err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	detail, err := h.networkService.CreateNetwork(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to create network",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, detail)
}

// RemoveNetwork handles DELETE /networks/:id
func (h *NetworkHandler) RemoveNetwork(c *gin.Context) {
	networkID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	err := h.networkService.RemoveNetwork(ctx, networkID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to remove network",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Network removed successfully",
		"id":      networkID,
	})
}

// PruneNetworks handles POST /networks/prune
func (h *NetworkHandler) PruneNetworks(c *gin.Context) {
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

	_, networksDeleted, err := h.networkService.PruneNetworks(ctx, req.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":  "Failed to prune networks",
			"detail": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Networks pruned successfully",
		"networks_deleted": networksDeleted,
		"count":            len(networksDeleted),
	})
}

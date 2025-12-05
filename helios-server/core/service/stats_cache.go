// Package service provides business logic for Docker resource management.
package service

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
)

// StatsCache manages cached container statistics with background refresh.
type StatsCache struct {
	containerService *ContainerService
	containerStats   map[string]*ContainerStats // containerID -> stats
	dashboardSummary *DashboardSummary
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
}

// NewStatsCache creates a new stats cache and starts background refresh.
func NewStatsCache(containerService *ContainerService) *StatsCache {
	ctx, cancel := context.WithCancel(context.Background())
	cache := &StatsCache{
		containerService: containerService,
		containerStats:   make(map[string]*ContainerStats),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start background refresh
	go cache.refreshLoop()

	return cache
}

// GetContainerStats returns cached stats for a container (may be nil if not yet cached).
func (c *StatsCache) GetContainerStats(containerID string) *ContainerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.containerStats[containerID]
}

// GetAllContainerStats returns all cached container stats.
func (c *StatsCache) GetAllContainerStats() map[string]*ContainerStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*ContainerStats, len(c.containerStats))
	for id, stats := range c.containerStats {
		result[id] = stats
	}
	return result
}

// GetDashboardSummary returns cached dashboard summary.
func (c *StatsCache) GetDashboardSummary() *DashboardSummary {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.dashboardSummary == nil {
		return &DashboardSummary{}
	}

	// Return a copy
	summary := *c.dashboardSummary
	return &summary
}

// refreshLoop continuously refreshes stats in the background.
func (c *StatsCache) refreshLoop() {
	// Initial refresh
	c.refresh()

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.refresh()
		}
	}
}

// refresh fetches fresh stats and updates the cache.
func (c *StatsCache) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get list of running containers
	containers, err := c.containerService.dockerClient.ContainerList(ctx, container.ListOptions{
		All: false, // Only running
	})
	if err != nil {
		log.Printf("Failed to list containers for stats cache: %v", err)
		return
	}

	if len(containers) == 0 {
		c.mu.Lock()
		c.containerStats = make(map[string]*ContainerStats)
		c.dashboardSummary = &DashboardSummary{}
		c.mu.Unlock()
		return
	}

	// Fetch stats for all containers in parallel
	type statsResult struct {
		containerID string
		stats       *ContainerStats
		err         error
	}

	statsChan := make(chan statsResult, len(containers))
	var wg sync.WaitGroup

	for _, container := range containers {
		wg.Add(1)
		go func(containerID string) {
			defer wg.Done()
			stats, err := c.containerService.getContainerStats(ctx, containerID)
			statsChan <- statsResult{
				containerID: containerID,
				stats:       stats,
				err:         err,
			}
		}(container.ID)
	}

	// Wait and close channel
	go func() {
		wg.Wait()
		close(statsChan)
	}()

	// Collect results
	newStats := make(map[string]*ContainerStats)
	summary := &DashboardSummary{}

	for result := range statsChan {
		if result.err != nil {
			log.Printf("Failed to get stats for container %s: %v", result.containerID, result.err)
			continue
		}

		if result.stats != nil {
			newStats[result.containerID] = result.stats

			// Aggregate for dashboard
			summary.TotalCPUPercent += result.stats.CPUPercent
			summary.TotalMemoryUsage += result.stats.MemoryUsage
			summary.TotalMemoryLimit += result.stats.MemoryLimit
			summary.TotalNetworkRx += result.stats.NetworkRx
			summary.TotalNetworkTx += result.stats.NetworkTx
			summary.ContainerCount++
		}
	}

	// Calculate average memory percentage
	if summary.TotalMemoryLimit > 0 {
		summary.TotalMemoryPercent = (float64(summary.TotalMemoryUsage) / float64(summary.TotalMemoryLimit)) * 100.0
	}

	// Update cache
	c.mu.Lock()
	c.containerStats = newStats
	c.dashboardSummary = summary
	c.mu.Unlock()
}

// Stop stops the background refresh loop.
func (c *StatsCache) Stop() {
	c.cancel()
}

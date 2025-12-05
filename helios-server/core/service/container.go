// Package service provides business logic for Docker resource management.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"nfcunha/helios/core/models"
	"nfcunha/helios/core/repository"
	"nfcunha/helios/utils/docker"
	"nfcunha/helios/utils/statsutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

// ContainerService handles container-related operations.
type ContainerService struct {
	dockerClient  *docker.Client
	actionLogRepo *repository.ActionLogRepository
	statsCache    *StatsCache
}

// NewContainerService creates a new container service.
func NewContainerService(dockerClient *docker.Client, actionLogRepo *repository.ActionLogRepository) *ContainerService {
	service := &ContainerService{
		dockerClient:  dockerClient,
		actionLogRepo: actionLogRepo,
	}

	// Initialize stats cache with background refresh
	service.statsCache = NewStatsCache(service)

	return service
}

// ContainerListOptions represents filtering options for listing containers.
type ContainerListOptions struct {
	All          bool   // Include stopped containers
	Limit        int    // Limit number of results
	Filter       string // Filter by name (substring match)
	IncludeStats bool   // Include resource stats (CPU, memory, etc.)
}

// ContainerInfo represents detailed container information with stats.
type ContainerInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	ImageID     string            `json:"image_id"`
	Command     string            `json:"command"`
	Created     int64             `json:"created"`
	State       string            `json:"state"`
	Status      string            `json:"status"`
	Ports       []PortInfo        `json:"ports"`
	Labels      map[string]string `json:"labels"`
	Mounts      []MountInfo       `json:"mounts"`
	NetworkMode string            `json:"network_mode"`
	Stats       *ContainerStats   `json:"stats,omitempty"`
}

// PortInfo represents a container port mapping.
type PortInfo struct {
	IP          string `json:"ip,omitempty"`
	PrivatePort uint16 `json:"private_port"`
	PublicPort  uint16 `json:"public_port,omitempty"`
	Type        string `json:"type"`
}

// MountInfo represents a container mount.
type MountInfo struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}

// ContainerStats represents container resource statistics.
type ContainerStats struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsage   uint64  `json:"memory_usage"`
	MemoryLimit   uint64  `json:"memory_limit"`
	MemoryPercent float64 `json:"memory_percent"`
	NetworkRx     uint64  `json:"network_rx"`
	NetworkTx     uint64  `json:"network_tx"`
	BlockRead     uint64  `json:"block_read"`
	BlockWrite    uint64  `json:"block_write"`
}

// DashboardSummary represents aggregate resource usage statistics.
type DashboardSummary struct {
	TotalCPUPercent    float64 `json:"total_cpu_percent"`
	TotalMemoryUsage   uint64  `json:"total_memory_usage"`
	TotalMemoryLimit   uint64  `json:"total_memory_limit"`
	TotalMemoryPercent float64 `json:"total_memory_percent"`
	TotalNetworkRx     uint64  `json:"total_network_rx"`
	TotalNetworkTx     uint64  `json:"total_network_tx"`
	ContainerCount     int     `json:"container_count"`
}

// ListContainers retrieves a list of containers based on the provided options.
func (s *ContainerService) ListContainers(ctx context.Context, opts ContainerListOptions) ([]ContainerInfo, error) {
	listOpts := container.ListOptions{
		All: opts.All,
	}

	containers, err := s.dockerClient.ContainerList(ctx, listOpts)
	if err != nil {
		log.Printf("Failed to list containers: %v", err)
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	var runningContainers []int // Track indices of running containers

	for _, c := range containers {
		// Apply name filter if specified
		if opts.Filter != "" {
			matched := false
			for _, name := range c.Names {
				if strings.Contains(strings.ToLower(name), strings.ToLower(opts.Filter)) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		info := s.convertToContainerInfo(c)
		result = append(result, info)

		// Track running containers for stats
		if c.State == "running" {
			runningContainers = append(runningContainers, len(result)-1)
		}

		// Apply limit if specified
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}

	// Use cached stats if requested (instant response, no loader!)
	if opts.IncludeStats && len(runningContainers) > 0 {
		cachedStats := s.statsCache.GetAllContainerStats()

		for _, idx := range runningContainers {
			if stats, ok := cachedStats[result[idx].ID]; ok {
				result[idx].Stats = stats
			}
		}
	}

	return result, nil
}

// GetContainer retrieves detailed information about a specific container.
func (s *ContainerService) GetContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	// Get container JSON (detailed info)
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		log.Printf("Failed to inspect container %s: %v", containerID, err)
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Build container info
	info := &ContainerInfo{
		ID:          containerJSON.ID,
		Name:        containerJSON.Name,
		Image:       containerJSON.Config.Image,
		ImageID:     containerJSON.Image,
		Command:     containerJSON.Path,
		Created:     parseTime(containerJSON.Created),
		State:       containerJSON.State.Status,
		Status:      formatStatus(containerJSON.State),
		Labels:      containerJSON.Config.Labels,
		NetworkMode: string(containerJSON.HostConfig.NetworkMode),
	}

	// Parse ports
	for port, bindings := range containerJSON.NetworkSettings.Ports {
		privatePort := port.Int()
		portType := port.Proto()

		if len(bindings) > 0 {
			for _, binding := range bindings {
				info.Ports = append(info.Ports, PortInfo{
					IP:          binding.HostIP,
					PrivatePort: uint16(privatePort),
					PublicPort:  parseUint16(binding.HostPort),
					Type:        portType,
				})
			}
		} else {
			info.Ports = append(info.Ports, PortInfo{
				PrivatePort: uint16(privatePort),
				Type:        portType,
			})
		}
	}

	// Parse mounts
	for _, mount := range containerJSON.Mounts {
		info.Mounts = append(info.Mounts, MountInfo{
			Type:        string(mount.Type),
			Name:        mount.Name,
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			RW:          mount.RW,
		})
	}

	// Get stats if container is running
	if containerJSON.State.Running {
		stats, err := s.getContainerStats(ctx, containerID)
		if err != nil {
			log.Printf("Failed to get stats for container %s: %v", containerID, err)
		} else {
			info.Stats = stats
		}
	}

	return info, nil
}

// StartContainer starts a stopped container.
func (s *ContainerService) StartContainer(ctx context.Context, containerID string) error {
	// Get container name for logging
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return s.logAction("start", "container", containerID, "", false, err)
	}

	// Start the container
	err = s.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return s.logAction("start", "container", containerID, containerJSON.Name, false, err)
	}

	log.Printf("Container %s started successfully", containerJSON.Name)
	return s.logAction("start", "container", containerID, containerJSON.Name, true, nil)
}

// StopContainer stops a running container.
func (s *ContainerService) StopContainer(ctx context.Context, containerID string) error {
	// Get container name for logging
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return s.logAction("stop", "container", containerID, "", false, err)
	}

	// Stop the container with 10 second timeout
	timeout := 10
	err = s.dockerClient.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return s.logAction("stop", "container", containerID, containerJSON.Name, false, err)
	}

	log.Printf("Container %s stopped successfully", containerJSON.Name)
	return s.logAction("stop", "container", containerID, containerJSON.Name, true, nil)
}

// RestartContainer restarts a container.
func (s *ContainerService) RestartContainer(ctx context.Context, containerID string) error {
	// Get container name for logging
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return s.logAction("restart", "container", containerID, "", false, err)
	}

	// Restart the container with 10 second timeout
	timeout := 10
	err = s.dockerClient.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeout,
	})
	if err != nil {
		return s.logAction("restart", "container", containerID, containerJSON.Name, false, err)
	}

	log.Printf("Container %s restarted successfully", containerJSON.Name)
	return s.logAction("restart", "container", containerID, containerJSON.Name, true, nil)
}

// RemoveContainer removes a container (must be stopped first unless force is true).
func (s *ContainerService) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	// Get container name for logging
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return s.logAction("remove", "container", containerID, "", false, err)
	}

	// Remove the container
	err = s.dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false,
	})
	if err != nil {
		return s.logAction("remove", "container", containerID, containerJSON.Name, false, err)
	}

	log.Printf("Container %s removed successfully", containerJSON.Name)
	return s.logAction("remove", "container", containerID, containerJSON.Name, true, nil)
}

// logAction logs an action to the database.
func (s *ContainerService) logAction(actionType, resourceType, resourceID, resourceName string, success bool, err error) error {
	actionLog := &models.ActionLog{
		ActionType:   actionType,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		ResourceName: resourceName,
		Success:      success,
		ExecutedAt:   time.Now(),
	}

	if err != nil {
		actionLog.ErrorMessage = err.Error()
	}

	if logErr := s.actionLogRepo.Create(actionLog); logErr != nil {
		log.Printf("Failed to log action: %v", logErr)
	}

	return err
}

// getContainerStats retrieves current statistics for a container.
func (s *ContainerService) getContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	statsResponse, err := s.dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, err
	}
	defer statsResponse.Body.Close()

	// Read stats
	statsJSON := &container.StatsResponse{}
	if err := decodeStats(statsResponse.Body, statsJSON); err != nil {
		return nil, err
	}

	// Calculate metrics
	cpuPercent := statsutil.CalculateCPUPercent(statsJSON)
	memoryUsage := statsJSON.MemoryStats.Usage
	memoryLimit := statsJSON.MemoryStats.Limit
	memoryPercent := float64(memoryUsage) / float64(memoryLimit) * 100.0

	stats := &ContainerStats{
		CPUPercent:    cpuPercent,
		MemoryUsage:   memoryUsage,
		MemoryLimit:   memoryLimit,
		MemoryPercent: memoryPercent,
		NetworkRx:     statsutil.GetNetworkRx(statsJSON),
		NetworkTx:     statsutil.GetNetworkTx(statsJSON),
		BlockRead:     statsutil.GetBlockRead(statsJSON),
		BlockWrite:    statsutil.GetBlockWrite(statsJSON),
	}

	return stats, nil
}

// Helper functions

func (s *ContainerService) convertToContainerInfo(c types.Container) ContainerInfo {
	name := ""
	if len(c.Names) > 0 {
		name = c.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}
	}

	info := ContainerInfo{
		ID:      c.ID,
		Name:    name,
		Image:   c.Image,
		ImageID: c.ImageID,
		Command: c.Command,
		Created: c.Created,
		State:   c.State,
		Status:  c.Status,
		Labels:  c.Labels,
	}

	// Parse ports
	for _, port := range c.Ports {
		info.Ports = append(info.Ports, PortInfo{
			IP:          port.IP,
			PrivatePort: port.PrivatePort,
			PublicPort:  port.PublicPort,
			Type:        port.Type,
		})
	}

	// Parse mounts
	for _, mount := range c.Mounts {
		info.Mounts = append(info.Mounts, MountInfo{
			Type:        string(mount.Type),
			Name:        mount.Name,
			Source:      mount.Source,
			Destination: mount.Destination,
			Mode:        mount.Mode,
			RW:          mount.RW,
		})
	}

	return info
}

func decodeStats(reader io.Reader, v interface{}) error {
	return json.NewDecoder(reader).Decode(v)
}

func formatStatus(state *types.ContainerState) string {
	if state.Running {
		startedAt := parseTimeString(state.StartedAt)
		if !startedAt.IsZero() {
			return fmt.Sprintf("Up %s", time.Since(startedAt).Round(time.Second))
		}
		return "Running"
	}
	finishedAt := parseTimeString(state.FinishedAt)
	if !finishedAt.IsZero() {
		if state.ExitCode == 0 {
			return fmt.Sprintf("Exited (%d) %s ago", state.ExitCode, time.Since(finishedAt).Round(time.Second))
		}
		return fmt.Sprintf("Exited (%d) %s ago", state.ExitCode, time.Since(finishedAt).Round(time.Second))
	}
	return fmt.Sprintf("Exited (%d)", state.ExitCode)
}

// parseUint16 parses a string to uint16
func parseUint16(s string) uint16 {
	val, err := strconv.ParseUint(s, 10, 16)
	if err != nil {
		return 0
	}
	return uint16(val)
}

func parseTime(s string) int64 {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func parseTimeString(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try alternative format
		t, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// BulkOperationResult represents the result of a bulk operation on a single container.
type BulkOperationResult struct {
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name,omitempty"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

// BulkStartContainers starts multiple containers in parallel.
func (s *ContainerService) BulkStartContainers(ctx context.Context, containerIDs []string) []BulkOperationResult {
	results := make([]BulkOperationResult, len(containerIDs))

	for i, containerID := range containerIDs {
		result := BulkOperationResult{
			ContainerID: containerID,
		}

		// Get container name
		containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
		if err == nil {
			result.ContainerName = containerJSON.Name
		}

		// Start container
		err = s.StartContainer(ctx, containerID)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results[i] = result
	}

	return results
}

// BulkStopContainers stops multiple containers in parallel.
func (s *ContainerService) BulkStopContainers(ctx context.Context, containerIDs []string) []BulkOperationResult {
	results := make([]BulkOperationResult, len(containerIDs))

	for i, containerID := range containerIDs {
		result := BulkOperationResult{
			ContainerID: containerID,
		}

		// Get container name
		containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
		if err == nil {
			result.ContainerName = containerJSON.Name
		}

		// Stop container
		err = s.StopContainer(ctx, containerID)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results[i] = result
	}

	return results
}

// BulkRemoveContainers removes multiple containers in parallel.
func (s *ContainerService) BulkRemoveContainers(ctx context.Context, containerIDs []string, force bool) []BulkOperationResult {
	results := make([]BulkOperationResult, len(containerIDs))

	for i, containerID := range containerIDs {
		result := BulkOperationResult{
			ContainerID: containerID,
		}

		// Get container name
		containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
		if err == nil {
			result.ContainerName = containerJSON.Name
		}

		// Remove container
		err = s.RemoveContainer(ctx, containerID, force)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		} else {
			result.Success = true
		}

		results[i] = result
	}

	return results
}

// GetDashboardSummary retrieves aggregate resource usage statistics for running containers.
func (s *ContainerService) GetDashboardSummary(ctx context.Context) (*DashboardSummary, error) {
	// Return cached summary (instant response!)
	return s.statsCache.GetDashboardSummary(), nil
}

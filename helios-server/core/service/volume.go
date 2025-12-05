// Package service provides business logic for Docker resource management.
package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"nfcunha/helios/core/models"
	"nfcunha/helios/core/repository"
	"nfcunha/helios/utils/docker"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
)

// VolumeService handles volume-related operations.
type VolumeService struct {
	dockerClient  *docker.Client
	actionLogRepo *repository.ActionLogRepository
}

// NewVolumeService creates a new volume service.
func NewVolumeService(dockerClient *docker.Client, actionLogRepo *repository.ActionLogRepository) *VolumeService {
	return &VolumeService{
		dockerClient:  dockerClient,
		actionLogRepo: actionLogRepo,
	}
}

// VolumeInfo represents volume information with usage data.
type VolumeInfo struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  string            `json:"created_at"`
	Labels     map[string]string `json:"labels"`
	Scope      string            `json:"scope"`
	Options    map[string]string `json:"options"`
	UsageData  *VolumeUsageData  `json:"usage_data,omitempty"`
}

// VolumeUsageData represents volume usage information.
type VolumeUsageData struct {
	Size     int64 `json:"size"`
	RefCount int64 `json:"ref_count"`
}

// VolumeDetail represents detailed volume information.
type VolumeDetail struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  string            `json:"created_at"`
	Labels     map[string]string `json:"labels"`
	Scope      string            `json:"scope"`
	Options    map[string]string `json:"options"`
	Status     map[string]any    `json:"status,omitempty"`
	UsageData  *VolumeUsageData  `json:"usage_data,omitempty"`
}

// CreateVolumeRequest represents the request to create a volume.
type CreateVolumeRequest struct {
	Name       string            `json:"name" binding:"required"`
	Driver     string            `json:"driver"`
	DriverOpts map[string]string `json:"driver_opts"`
	Labels     map[string]string `json:"labels"`
}

// ListVolumes retrieves a list of all volumes.
func (s *VolumeService) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	volumeList, err := s.dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		log.Printf("Failed to list volumes: %v", err)
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	var result []VolumeInfo
	for _, vol := range volumeList.Volumes {
		info := VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			CreatedAt:  vol.CreatedAt,
			Labels:     vol.Labels,
			Scope:      vol.Scope,
			Options:    vol.Options,
		}

		// Add usage data if available
		if vol.UsageData != nil {
			info.UsageData = &VolumeUsageData{
				Size:     vol.UsageData.Size,
				RefCount: vol.UsageData.RefCount,
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// InspectVolume retrieves detailed information about a specific volume.
func (s *VolumeService) InspectVolume(ctx context.Context, volumeName string) (*VolumeDetail, error) {
	vol, err := s.dockerClient.VolumeInspect(ctx, volumeName)
	if err != nil {
		log.Printf("Failed to inspect volume %s: %v", volumeName, err)
		return nil, fmt.Errorf("failed to inspect volume: %w", err)
	}

	detail := &VolumeDetail{
		Name:       vol.Name,
		Driver:     vol.Driver,
		Mountpoint: vol.Mountpoint,
		CreatedAt:  vol.CreatedAt,
		Labels:     vol.Labels,
		Scope:      vol.Scope,
		Options:    vol.Options,
		Status:     vol.Status,
	}

	// Add usage data if available
	if vol.UsageData != nil {
		detail.UsageData = &VolumeUsageData{
			Size:     vol.UsageData.Size,
			RefCount: vol.UsageData.RefCount,
		}
	}

	return detail, nil
}

// CreateVolume creates a new volume.
func (s *VolumeService) CreateVolume(ctx context.Context, req *CreateVolumeRequest) (*VolumeDetail, error) {
	// Set default driver if not specified
	driver := req.Driver
	if driver == "" {
		driver = "local"
	}

	createOptions := volume.CreateOptions{
		Name:       req.Name,
		Driver:     driver,
		DriverOpts: req.DriverOpts,
		Labels:     req.Labels,
	}

	vol, err := s.dockerClient.VolumeCreate(ctx, createOptions)
	if err != nil {
		log.Printf("Failed to create volume %s: %v", req.Name, err)
		s.logAction("create", "volume", "", req.Name, false, err)
		return nil, fmt.Errorf("failed to create volume: %w", err)
	}

	log.Printf("Successfully created volume: %s", vol.Name)
	s.logAction("create", "volume", vol.Name, vol.Name, true, nil)

	// Inspect to get full details
	detail, err := s.InspectVolume(ctx, vol.Name)
	if err != nil {
		// Still return success, but log the error
		log.Printf("Warning: Created volume but failed to inspect: %v", err)
		return &VolumeDetail{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Labels:     vol.Labels,
			Scope:      vol.Scope,
			Options:    vol.Options,
		}, nil
	}

	return detail, nil
}

// RemoveVolume removes a volume.
func (s *VolumeService) RemoveVolume(ctx context.Context, volumeName string, force bool) error {
	err := s.dockerClient.VolumeRemove(ctx, volumeName, force)
	if err != nil {
		log.Printf("Failed to remove volume %s: %v", volumeName, err)
		s.logAction("remove", "volume", volumeName, volumeName, false, err)
		return fmt.Errorf("failed to remove volume: %w", err)
	}

	log.Printf("Successfully removed volume: %s", volumeName)
	s.logAction("remove", "volume", volumeName, volumeName, true, nil)
	return nil
}

// PruneVolumes removes unused volumes and their associated stopped containers.
func (s *VolumeService) PruneVolumes(ctx context.Context, pruneFilters map[string][]string) (uint64, []string, error) {
	// Get all volumes
	volumeList, err := s.dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		log.Printf("Failed to list volumes for pruning: %v", err)
		return 0, nil, fmt.Errorf("failed to list volumes: %w", err)
	}

	// Get all containers (including stopped)
	containers, err := s.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list containers for volume pruning: %v", err)
		return 0, nil, fmt.Errorf("failed to list containers: %w", err)
	}

	// Build a map of volumes used by running containers
	usedByRunning := make(map[string]bool)
	for _, c := range containers {
		if c.State == "running" {
			for _, mount := range c.Mounts {
				if mount.Type == "volume" && mount.Name != "" {
					usedByRunning[mount.Name] = true
				}
			}
		}
	}

	// Remove stopped containers that use volumes not used by running containers
	removedContainers := 0
	for _, c := range containers {
		if c.State != "running" {
			shouldRemove := false
			for _, mount := range c.Mounts {
				if mount.Type == "volume" && mount.Name != "" && !usedByRunning[mount.Name] {
					shouldRemove = true
					break
				}
			}

			if shouldRemove {
				if err := s.dockerClient.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true, RemoveVolumes: false}); err != nil {
					log.Printf("Failed to remove stopped container %s: %v", c.ID, err)
				} else {
					removedContainers++
					log.Printf("Removed stopped container %s for volume cleanup", c.ID[:12])
				}
			}
		}
	}

	log.Printf("Removed %d stopped containers for volume pruning", removedContainers)

	// Now manually remove all volumes not used by running containers
	removedVolumes := []string{}
	var totalReclaimed uint64 = 0

	for _, vol := range volumeList.Volumes {
		// Skip volumes used by running containers
		if usedByRunning[vol.Name] {
			continue
		}

		// Try to remove the volume
		if err := s.dockerClient.VolumeRemove(ctx, vol.Name, false); err != nil {
			log.Printf("Failed to remove volume %s: %v", vol.Name, err)
		} else {
			removedVolumes = append(removedVolumes, vol.Name)
			// Estimate size if UsageData is available
			if vol.UsageData != nil {
				totalReclaimed += uint64(vol.UsageData.Size)
			}
			log.Printf("Removed unused volume: %s", vol.Name)
		}
	}

	log.Printf("Pruned %d volumes, reclaimed space: %d bytes", len(removedVolumes), totalReclaimed)
	s.logAction("prune", "volume", "all", "all", true, nil)
	return totalReclaimed, removedVolumes, nil
}

// logAction logs an action to the database.
func (s *VolumeService) logAction(actionType, resourceType, resourceID, resourceName string, success bool, err error) error {
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

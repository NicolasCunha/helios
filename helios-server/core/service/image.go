// Package service provides business logic for Docker resource management.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"nfcunha/helios/core/models"
	"nfcunha/helios/core/repository"
	"nfcunha/helios/utils/docker"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/registry"
)

// ImageService handles image-related operations.
type ImageService struct {
	dockerClient  *docker.Client
	actionLogRepo *repository.ActionLogRepository
}

// NewImageService creates a new image service.
func NewImageService(dockerClient *docker.Client, actionLogRepo *repository.ActionLogRepository) *ImageService {
	return &ImageService{
		dockerClient:  dockerClient,
		actionLogRepo: actionLogRepo,
	}
}

// ImageInfo represents detailed image information.
type ImageInfo struct {
	ID          string            `json:"id"`
	RepoTags    []string          `json:"repo_tags"`
	RepoDigests []string          `json:"repo_digests"`
	Created     int64             `json:"created"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size"`
	SharedSize  int64             `json:"shared_size"`
	Labels      map[string]string `json:"labels"`
	Containers  int               `json:"containers"`
}

// ImageDetail represents comprehensive image details from inspection.
type ImageDetail struct {
	ID            string            `json:"id"`
	RepoTags      []string          `json:"repo_tags"`
	RepoDigests   []string          `json:"repo_digests"`
	Parent        string            `json:"parent"`
	Comment       string            `json:"comment"`
	Created       string            `json:"created"`
	Container     string            `json:"container"`
	DockerVersion string            `json:"docker_version"`
	Author        string            `json:"author"`
	Architecture  string            `json:"architecture"`
	Os            string            `json:"os"`
	Size          int64             `json:"size"`
	VirtualSize   int64             `json:"virtual_size"`
	Labels        map[string]string `json:"labels"`
	ExposedPorts  []string          `json:"exposed_ports"`
	Env           []string          `json:"env"`
	Cmd           []string          `json:"cmd"`
	Entrypoint    []string          `json:"entrypoint"`
	Volumes       []string          `json:"volumes"`
	WorkingDir    string            `json:"working_dir"`
	User          string            `json:"user"`
	RootFS        *RootFS           `json:"rootfs"`
}

// RootFS represents the rootfs information.
type RootFS struct {
	Type   string   `json:"type"`
	Layers []string `json:"layers"`
}

// PullProgress represents the progress of an image pull operation.
type PullProgress struct {
	Status         string `json:"status"`
	ProgressDetail struct {
		Current int64 `json:"current"`
		Total   int64 `json:"total"`
	} `json:"progressDetail"`
	Progress    string                 `json:"progress"`
	ID          string                 `json:"id"`
	Error       string                 `json:"error,omitempty"`
	ErrorDetail map[string]interface{} `json:"errorDetail,omitempty"`
}

// GetImages retrieves all Docker images.
func (s *ImageService) ListImages(ctx context.Context, all bool) ([]ImageInfo, error) {
	opts := image.ListOptions{
		All: all,
	}

	images, err := s.dockerClient.ImageList(ctx, opts)
	if err != nil {
		log.Printf("Failed to list images: %v", err)
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var result []ImageInfo
	for _, img := range images {
		result = append(result, ImageInfo{
			ID:          img.ID,
			RepoTags:    img.RepoTags,
			RepoDigests: img.RepoDigests,
			Created:     img.Created,
			Size:        img.Size,
			VirtualSize: img.VirtualSize,
			SharedSize:  img.SharedSize,
			Labels:      img.Labels,
			Containers:  int(img.Containers),
		})
	}

	return result, nil
}

// InspectImage retrieves detailed information about a specific image.
func (s *ImageService) InspectImage(ctx context.Context, imageID string) (*ImageDetail, error) {
	inspect, _, err := s.dockerClient.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		log.Printf("Failed to inspect image %s: %v", imageID, err)
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	detail := &ImageDetail{
		ID:            inspect.ID,
		RepoTags:      inspect.RepoTags,
		RepoDigests:   inspect.RepoDigests,
		Parent:        inspect.Parent,
		Comment:       inspect.Comment,
		Created:       inspect.Created,
		Container:     inspect.Container,
		DockerVersion: inspect.DockerVersion,
		Author:        inspect.Author,
		Architecture:  inspect.Architecture,
		Os:            inspect.Os,
		Size:          inspect.Size,
		VirtualSize:   inspect.VirtualSize,
		Labels:        inspect.Config.Labels,
		Env:           inspect.Config.Env,
		Cmd:           inspect.Config.Cmd,
		Entrypoint:    inspect.Config.Entrypoint,
		WorkingDir:    inspect.Config.WorkingDir,
		User:          inspect.Config.User,
	}

	// Extract exposed ports
	if inspect.Config.ExposedPorts != nil {
		for port := range inspect.Config.ExposedPorts {
			detail.ExposedPorts = append(detail.ExposedPorts, string(port))
		}
	}

	// Extract volumes
	if inspect.Config.Volumes != nil {
		for vol := range inspect.Config.Volumes {
			detail.Volumes = append(detail.Volumes, vol)
		}
	}

	// Extract rootfs
	if inspect.RootFS.Type != "" {
		detail.RootFS = &RootFS{
			Type:   inspect.RootFS.Type,
			Layers: inspect.RootFS.Layers,
		}
	}

	return detail, nil
}

// PullImage pulls an image from a registry.
// Returns a channel that provides progress updates.
func (s *ImageService) PullImage(ctx context.Context, imageName string) (<-chan PullProgress, <-chan error, error) {
	// Start pull
	reader, err := s.dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		log.Printf("Failed to start pull for image %s: %v", imageName, err)
		s.logAction("pull", "image", imageName, imageName, false, err)
		return nil, nil, fmt.Errorf("failed to pull image: %w", err)
	}

	log.Printf("Started pulling image %s, streaming progress...", imageName)

	// Create channels for progress and errors
	progressChan := make(chan PullProgress, 10)
	errChan := make(chan error, 1)

	// Parse progress in background
	go func() {
		defer close(progressChan)
		defer close(errChan)
		defer reader.Close()

		decoder := json.NewDecoder(reader)
		hasError := false
		for {
			var progress PullProgress
			if err := decoder.Decode(&progress); err != nil {
				if err == io.EOF {
					// Pull completed successfully (only if no errors occurred)
					if !hasError {
						s.logAction("pull", "image", imageName, imageName, true, nil)
						log.Printf("Successfully pulled image: %s", imageName)
					}
					return
				}
				// If we already sent an error, don't send decode errors
				if !hasError {
					errChan <- fmt.Errorf("failed to decode progress: %w", err)
					s.logAction("pull", "image", imageName, imageName, false, err)
				}
				return
			}

			// Check for errors in progress
			if progress.Error != "" || len(progress.ErrorDetail) > 0 {
				hasError = true
				errMsg := progress.Error
				if len(progress.ErrorDetail) > 0 {
					if detailMsg, ok := progress.ErrorDetail["message"].(string); ok {
						errMsg += ": " + detailMsg
					} else {
						// Fallback: convert the whole map to string
						errMsg += fmt.Sprintf(": %v", progress.ErrorDetail)
					}
				}
				err := fmt.Errorf("%s", errMsg)
				// Send the error progress to frontend before sending to errChan
				select {
				case progressChan <- progress:
				case <-ctx.Done():
				}
				errChan <- err
				s.logAction("pull", "image", imageName, imageName, false, err)
				log.Printf("Failed to pull image %s: %v", imageName, err)
				return
			}

			if strings.Contains(strings.ToLower(progress.Status), "error") {
				hasError = true
				err := fmt.Errorf("pull error: %s", progress.Status)
				// Send the error progress to frontend before sending to errChan
				select {
				case progressChan <- progress:
				case <-ctx.Done():
				}
				errChan <- err
				s.logAction("pull", "image", imageName, imageName, false, err)
				return
			}

			select {
			case progressChan <- progress:
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}
		}
	}()

	return progressChan, errChan, nil
}

// RemoveImage removes an image by ID or name.
func (s *ImageService) RemoveImage(ctx context.Context, imageID string, force bool) error {
	opts := image.RemoveOptions{
		Force:         force,
		PruneChildren: true,
	}

	// Get image info for logging before removal
	imageName := imageID
	if inspect, _, err := s.dockerClient.ImageInspectWithRaw(ctx, imageID); err == nil {
		if len(inspect.RepoTags) > 0 {
			imageName = inspect.RepoTags[0]
		}
	}

	_, err := s.dockerClient.ImageRemove(ctx, imageID, opts)
	if err != nil {
		log.Printf("Failed to remove image %s: %v", imageID, err)
		s.logAction("remove", "image", imageID, imageName, false, err)
		return fmt.Errorf("failed to remove image: %w", err)
	}

	log.Printf("Successfully removed image: %s", imageName)
	s.logAction("remove", "image", imageID, imageName, true, nil)
	return nil
}

// BulkRemoveImages removes multiple images by their IDs.
func (s *ImageService) BulkRemoveImages(ctx context.Context, imageIDs []string, force bool) []BulkOperationResult {
	results := make([]BulkOperationResult, 0, len(imageIDs))

	for _, imageID := range imageIDs {
		// Get image name for result
		imageName := imageID
		if inspect, _, err := s.dockerClient.ImageInspectWithRaw(ctx, imageID); err == nil {
			if len(inspect.RepoTags) > 0 {
				imageName = inspect.RepoTags[0]
			}
		}

		result := BulkOperationResult{
			ContainerID:   imageID,
			ContainerName: imageName,
			Success:       true,
		}

		err := s.RemoveImage(ctx, imageID, force)
		if err != nil {
			result.Success = false
			result.Error = err.Error()
		}

		results = append(results, result)
	}

	return results
}

// PruneImages removes unused images and their associated stopped containers.
func (s *ImageService) PruneImages(ctx context.Context, all bool) (uint64, error) {
	if all {
		// Get all images
		images, err := s.dockerClient.ImageList(ctx, image.ListOptions{})
		if err != nil {
			log.Printf("Failed to list images for pruning: %v", err)
			return 0, fmt.Errorf("failed to list images: %w", err)
		}

		// Get all containers (including stopped)
		containers, err := s.dockerClient.ContainerList(ctx, container.ListOptions{All: true})
		if err != nil {
			log.Printf("Failed to list containers for pruning: %v", err)
			return 0, fmt.Errorf("failed to list containers: %w", err)
		}

		// Build a map of images that are being used by running containers
		usedByRunning := make(map[string]bool)
		for _, c := range containers {
			if c.State == "running" {
				usedByRunning[c.ImageID] = true
			}
		}

		// Remove stopped containers for images not used by running containers
		removedContainers := 0
		for _, c := range containers {
			if c.State != "running" && !usedByRunning[c.ImageID] {
				// Remove this stopped container
				if err := s.dockerClient.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true}); err != nil {
					log.Printf("Failed to remove stopped container %s: %v", c.ID, err)
				} else {
					removedContainers++
					log.Printf("Removed stopped container %s using image %s", c.ID[:12], c.ImageID[:12])
				}
			}
		}

		log.Printf("Removed %d stopped containers", removedContainers)

		// Now manually remove all images not used by running containers
		removedImages := 0
		var totalReclaimed uint64 = 0

		for _, img := range images {
			// Skip images used by running containers
			if usedByRunning[img.ID] {
				continue
			}

			// Try to remove the image
			deleteResponse, err := s.dockerClient.ImageRemove(ctx, img.ID, image.RemoveOptions{Force: false, PruneChildren: true})
			if err != nil {
				log.Printf("Failed to remove image %s: %v", img.ID[:12], err)
			} else {
				for _, item := range deleteResponse {
					if item.Deleted != "" {
						removedImages++
						totalReclaimed += uint64(img.Size)
						log.Printf("Removed unused image: %s", img.ID[:12])
						break
					}
				}
			}
		}

		log.Printf("Pruned %d images, reclaimed space: %d bytes", removedImages, totalReclaimed)
		s.logAction("prune", "image", "all", "all", true, nil)
		return totalReclaimed, nil
	}

	// For non-"all" mode, just remove dangling images
	pruneFilters := filters.NewArgs()
	pruneFilters.Add("dangling", "true")

	report, err := s.dockerClient.ImagesPrune(ctx, pruneFilters)
	if err != nil {
		log.Printf("Failed to prune images: %v", err)
		s.logAction("prune", "image", "all", "all", false, err)
		return 0, fmt.Errorf("failed to prune images: %w", err)
	}

	log.Printf("Pruned images, reclaimed space: %d bytes", report.SpaceReclaimed)
	s.logAction("prune", "image", "all", "all", true, nil)
	return report.SpaceReclaimed, nil
}

// SearchImages searches for images in a registry.
func (s *ImageService) SearchImages(ctx context.Context, term string, limit int) ([]registry.SearchResult, error) {
	opts := registry.SearchOptions{
		Limit: limit,
	}

	results, err := s.dockerClient.ImageSearch(ctx, term, opts)
	if err != nil {
		log.Printf("Failed to search images for term %s: %v", term, err)
		return nil, fmt.Errorf("failed to search images: %w", err)
	}

	return results, nil
}

// GetImageTags fetches available tags for an image from Docker Hub.
func (s *ImageService) GetImageTags(ctx context.Context, imageName string, limit int) ([]string, error) {
	// Prepare repository name
	repository := imageName

	// Check if it's an official image (no slash means it's in library/)
	if !strings.Contains(imageName, "/") {
		repository = "library/" + imageName
	}

	// Create HTTP client with timeout from context
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build Docker Hub API URL
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/%s/tags?page_size=%d", repository, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker hub returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var result struct {
		Results []struct {
			Name string `json:"name"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	tags := make([]string, 0, len(result.Results))
	for _, tag := range result.Results {
		tags = append(tags, tag.Name)
	}

	return tags, nil
}

// logAction logs an action to the database.
func (s *ImageService) logAction(actionType, resourceType, resourceID, resourceName string, success bool, err error) error {
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

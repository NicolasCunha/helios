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

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
)

// NetworkService handles network-related operations.
type NetworkService struct {
	dockerClient  *docker.Client
	actionLogRepo *repository.ActionLogRepository
}

// NewNetworkService creates a new network service.
func NewNetworkService(dockerClient *docker.Client, actionLogRepo *repository.ActionLogRepository) *NetworkService {
	return &NetworkService{
		dockerClient:  dockerClient,
		actionLogRepo: actionLogRepo,
	}
}

// NetworkInfo represents network information.
type NetworkInfo struct {
	ID         string                              `json:"id"`
	Name       string                              `json:"name"`
	Driver     string                              `json:"driver"`
	Scope      string                              `json:"scope"`
	Internal   bool                                `json:"internal"`
	Attachable bool                                `json:"attachable"`
	Ingress    bool                                `json:"ingress"`
	IPAM       network.IPAM                        `json:"ipam"`
	Containers map[string]network.EndpointResource `json:"containers,omitempty"`
	Options    map[string]string                   `json:"options,omitempty"`
	Labels     map[string]string                   `json:"labels,omitempty"`
	Created    string                              `json:"created,omitempty"`
}

// NetworkDetail represents detailed network information.
type NetworkDetail struct {
	ID         string                              `json:"id"`
	Name       string                              `json:"name"`
	Driver     string                              `json:"driver"`
	Scope      string                              `json:"scope"`
	Internal   bool                                `json:"internal"`
	Attachable bool                                `json:"attachable"`
	Ingress    bool                                `json:"ingress"`
	EnableIPv6 bool                                `json:"enable_ipv6"`
	IPAM       network.IPAM                        `json:"ipam"`
	Containers map[string]network.EndpointResource `json:"containers"`
	Options    map[string]string                   `json:"options"`
	Labels     map[string]string                   `json:"labels"`
	ConfigFrom network.ConfigReference             `json:"config_from,omitempty"`
	ConfigOnly bool                                `json:"config_only"`
	Created    string                              `json:"created"`
}

// CreateNetworkRequest represents the request to create a network.
type CreateNetworkRequest struct {
	Name       string            `json:"name" binding:"required"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Internal   bool              `json:"internal"`
	Attachable bool              `json:"attachable"`
	Ingress    bool              `json:"ingress"`
	EnableIPv6 bool              `json:"enable_ipv6"`
	IPAM       *network.IPAM     `json:"ipam"`
	Options    map[string]string `json:"options"`
	Labels     map[string]string `json:"labels"`
}

// ListNetworks retrieves a list of all networks.
func (s *NetworkService) ListNetworks(ctx context.Context) ([]NetworkInfo, error) {
	networks, err := s.dockerClient.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		log.Printf("Failed to list networks: %v", err)
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var result []NetworkInfo
	for _, net := range networks {
		// NetworkList doesn't populate Containers, need to inspect each network
		inspected, err := s.dockerClient.NetworkInspect(ctx, net.ID, network.InspectOptions{})
		if err != nil {
			log.Printf("Failed to inspect network %s: %v", net.ID, err)
			// Continue with basic info if inspect fails
			result = append(result, NetworkInfo{
				ID:         net.ID,
				Name:       net.Name,
				Driver:     net.Driver,
				Scope:      net.Scope,
				Internal:   net.Internal,
				Attachable: net.Attachable,
				Ingress:    net.Ingress,
				IPAM:       net.IPAM,
				Containers: make(map[string]network.EndpointResource),
				Options:    net.Options,
				Labels:     net.Labels,
				Created:    net.Created.String(),
			})
			continue
		}

		result = append(result, NetworkInfo{
			ID:         inspected.ID,
			Name:       inspected.Name,
			Driver:     inspected.Driver,
			Scope:      inspected.Scope,
			Internal:   inspected.Internal,
			Attachable: inspected.Attachable,
			Ingress:    inspected.Ingress,
			IPAM:       inspected.IPAM,
			Containers: inspected.Containers,
			Options:    inspected.Options,
			Labels:     inspected.Labels,
			Created:    inspected.Created.String(),
		})
	}

	return result, nil
}

// InspectNetwork retrieves detailed information about a specific network.
func (s *NetworkService) InspectNetwork(ctx context.Context, networkID string) (*NetworkDetail, error) {
	net, err := s.dockerClient.NetworkInspect(ctx, networkID, network.InspectOptions{
		Verbose: true,
	})
	if err != nil {
		log.Printf("Failed to inspect network %s: %v", networkID, err)
		return nil, fmt.Errorf("failed to inspect network: %w", err)
	}

	detail := &NetworkDetail{
		ID:         net.ID,
		Name:       net.Name,
		Driver:     net.Driver,
		Scope:      net.Scope,
		Internal:   net.Internal,
		Attachable: net.Attachable,
		Ingress:    net.Ingress,
		EnableIPv6: net.EnableIPv6,
		IPAM:       net.IPAM,
		Containers: net.Containers,
		Options:    net.Options,
		Labels:     net.Labels,
		ConfigFrom: net.ConfigFrom,
		ConfigOnly: net.ConfigOnly,
		Created:    net.Created.String(),
	}

	return detail, nil
}

// CreateNetwork creates a new network.
func (s *NetworkService) CreateNetwork(ctx context.Context, req *CreateNetworkRequest) (*NetworkDetail, error) {
	// Set default driver if not specified
	driver := req.Driver
	if driver == "" {
		driver = "bridge"
	}

	createOptions := network.CreateOptions{
		Driver:     driver,
		Scope:      req.Scope,
		Internal:   req.Internal,
		Attachable: req.Attachable,
		Ingress:    req.Ingress,
		EnableIPv6: &req.EnableIPv6,
		Options:    req.Options,
		Labels:     req.Labels,
	}

	// Add IPAM configuration if provided
	if req.IPAM != nil {
		createOptions.IPAM = req.IPAM
	}

	response, err := s.dockerClient.NetworkCreate(ctx, req.Name, createOptions)
	if err != nil {
		log.Printf("Failed to create network %s: %v", req.Name, err)
		s.logAction("create", "network", "", req.Name, false, err)
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	if response.Warning != "" {
		log.Printf("Warning while creating network %s: %s", req.Name, response.Warning)
	}

	log.Printf("Successfully created network: %s (ID: %s)", req.Name, response.ID)
	s.logAction("create", "network", response.ID, req.Name, true, nil)

	// Inspect to get full details
	detail, err := s.InspectNetwork(ctx, response.ID)
	if err != nil {
		// Still return success, but log the error
		log.Printf("Warning: Created network but failed to inspect: %v", err)
		return &NetworkDetail{
			ID:     response.ID,
			Name:   req.Name,
			Driver: driver,
		}, nil
	}

	return detail, nil
}

// RemoveNetwork removes a network.
func (s *NetworkService) RemoveNetwork(ctx context.Context, networkID string) error {
	// Get network info for logging before removal
	networkName := networkID
	if net, err := s.dockerClient.NetworkInspect(ctx, networkID, network.InspectOptions{}); err == nil {
		networkName = net.Name
	}

	err := s.dockerClient.NetworkRemove(ctx, networkID)
	if err != nil {
		log.Printf("Failed to remove network %s: %v", networkID, err)
		s.logAction("remove", "network", networkID, networkName, false, err)
		return fmt.Errorf("failed to remove network: %w", err)
	}

	log.Printf("Successfully removed network: %s", networkName)
	s.logAction("remove", "network", networkID, networkName, true, nil)
	return nil
}

// PruneNetworks removes unused networks.
func (s *NetworkService) PruneNetworks(ctx context.Context, pruneFilters map[string][]string) (uint64, []string, error) {
	// Convert filter map to filters.Args
	filterArgs := filters.NewArgs()
	for key, values := range pruneFilters {
		for _, value := range values {
			filterArgs.Add(key, value)
		}
	}

	report, err := s.dockerClient.NetworksPrune(ctx, filterArgs)
	if err != nil {
		log.Printf("Failed to prune networks: %v", err)
		s.logAction("prune", "network", "all", "all", false, err)
		return 0, nil, fmt.Errorf("failed to prune networks: %w", err)
	}

	networkNames := make([]string, 0)
	for _, net := range report.NetworksDeleted {
		networkNames = append(networkNames, net)
	}

	log.Printf("Pruned networks, removed: %v", networkNames)
	s.logAction("prune", "network", "all", "all", true, nil)
	return 0, networkNames, nil
}

// logAction logs an action to the database.
func (s *NetworkService) logAction(actionType, resourceType, resourceID, resourceName string, success bool, err error) error {
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

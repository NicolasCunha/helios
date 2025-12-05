// Package docker provides a wrapper around the Docker SDK client.
package docker

import (
	"context"
	"log"

	"github.com/docker/docker/client"
)

// Client wraps the Docker SDK client with additional functionality.
type Client struct {
	*client.Client
}

// NewClient creates a new Docker client using environment variables.
// It connects to the Docker daemon via the socket specified in DOCKER_HOST
// or defaults to unix:///var/run/docker.sock
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Printf("Failed to create Docker client: %v", err)
		return nil, err
	}

	log.Println("Docker client created successfully")
	return &Client{Client: cli}, nil
}

// Ping verifies connection to the Docker daemon.
func (c *Client) Ping(ctx context.Context) error {
	_, err := c.Client.Ping(ctx)
	if err != nil {
		log.Printf("Docker daemon ping failed: %v", err)
		return err
	}
	return nil
}

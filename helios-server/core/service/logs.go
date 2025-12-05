// Package service provides business logic for Docker resource management.
package service

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/docker/docker/api/types/container"
	"nfcunha/helios/utils/docker"
)

// LogService handles container log operations.
type LogService struct {
	dockerClient *docker.Client
}

// NewLogService creates a new log service.
func NewLogService(dockerClient *docker.Client) *LogService {
	return &LogService{
		dockerClient: dockerClient,
	}
}

// LogStreamOptions represents options for streaming container logs.
type LogStreamOptions struct {
	Follow     bool   // Follow log output
	Tail       string // Number of lines to show from the end (default "all")
	Since      string // Show logs since timestamp
	Until      string // Show logs before timestamp
	Timestamps bool   // Show timestamps
}

// StreamLogs streams container logs to the provided writer.
// Returns a channel that will be closed when streaming is complete or an error occurs.
func (s *LogService) StreamLogs(ctx context.Context, containerID string, opts LogStreamOptions, writer io.Writer) (<-chan error, error) {
	// Build log options
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     opts.Follow,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}

	if opts.Since != "" {
		logOpts.Since = opts.Since
	}
	if opts.Until != "" {
		logOpts.Until = opts.Until
	}

	// Get logs from Docker
	reader, err := s.dockerClient.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		log.Printf("Failed to get logs for container %s: %v", containerID, err)
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}

	// Create error channel
	errChan := make(chan error, 1)

	// Stream logs in background
	go func() {
		defer close(errChan)
		defer reader.Close()

		// Docker log format has an 8-byte header: [STREAM_TYPE, 0, 0, 0, SIZE1, SIZE2, SIZE3, SIZE4]
		// We need to skip this header and write only the actual log content
		buf := make([]byte, 32*1024) // 32KB buffer

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				// Read header (8 bytes)
				header := make([]byte, 8)
				_, err := io.ReadFull(reader, header)
				if err != nil {
					if err == io.EOF {
						return
					}
					errChan <- fmt.Errorf("failed to read log header: %w", err)
					return
				}

				// Extract payload size from header (big-endian)
				size := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])

				if size == 0 {
					continue
				}

				// Read payload
				if size > uint32(len(buf)) {
					buf = make([]byte, size)
				}

				n, err := io.ReadFull(reader, buf[:size])
				if err != nil {
					if err == io.EOF {
						return
					}
					errChan <- fmt.Errorf("failed to read log payload: %w", err)
					return
				}

				// Write to output
				if _, err := writer.Write(buf[:n]); err != nil {
					errChan <- fmt.Errorf("failed to write log data: %w", err)
					return
				}

				// Flush if writer supports it
				if flusher, ok := writer.(interface{ Flush() error }); ok {
					flusher.Flush()
				}
			}
		}
	}()

	return errChan, nil
}

// GetLogs retrieves all container logs and returns them as a string.
func (s *LogService) GetLogs(ctx context.Context, containerID string, opts LogStreamOptions) (string, error) {
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: opts.Timestamps,
		Tail:       opts.Tail,
	}

	if opts.Since != "" {
		logOpts.Since = opts.Since
	}
	if opts.Until != "" {
		logOpts.Until = opts.Until
	}

	reader, err := s.dockerClient.ContainerLogs(ctx, containerID, logOpts)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Read all logs, stripping Docker headers
	var result []byte
	buf := make([]byte, 32*1024)

	for {
		// Read header
		header := make([]byte, 8)
		_, err := io.ReadFull(reader, header)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read log header: %w", err)
		}

		// Extract payload size
		size := uint32(header[4])<<24 | uint32(header[5])<<16 | uint32(header[6])<<8 | uint32(header[7])

		if size == 0 {
			continue
		}

		// Read payload
		if size > uint32(len(buf)) {
			buf = make([]byte, size)
		}

		n, err := io.ReadFull(reader, buf[:size])
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", fmt.Errorf("failed to read log payload: %w", err)
		}

		result = append(result, buf[:n]...)
	}

	return string(result), nil
}

// CreateLogArchive creates a ZIP archive of container logs.
func (s *LogService) CreateLogArchive(ctx context.Context, containerID string, writer io.Writer) error {
	// Get container info for filename
	containerJSON, err := s.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	// Get logs
	logs, err := s.GetLogs(ctx, containerID, LogStreamOptions{
		Timestamps: true,
		Tail:       "all",
	})
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}

	// Create ZIP archive
	zipWriter := zip.NewWriter(writer)
	defer zipWriter.Close()

	// Create log file in archive
	containerName := containerJSON.Name
	if len(containerName) > 0 && containerName[0] == '/' {
		containerName = containerName[1:]
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s_%s.log", containerName, timestamp)

	fileWriter, err := zipWriter.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	// Write logs to zip
	if _, err := fileWriter.Write([]byte(logs)); err != nil {
		return fmt.Errorf("failed to write logs to zip: %w", err)
	}

	log.Printf("Created log archive for container %s (%d bytes)", containerName, len(logs))
	return nil
}

// StreamLogsWithWriter is a convenience method that handles the writer lifecycle.
type LogWriter struct {
	writer io.Writer
}

func (lw *LogWriter) Write(p []byte) (n int, err error) {
	return lw.writer.Write(p)
}

func (lw *LogWriter) Flush() error {
	if flusher, ok := lw.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// StdoutStderr splits Docker multiplexed stream into stdout and stderr.
func StdoutStderr(reader io.Reader) (stdout, stderr io.Reader) {
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()

	go func() {
		defer stdoutWriter.Close()
		defer stderrWriter.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 {
				// First byte indicates stream type: 1=stdout, 2=stderr
				if line[0] == 1 {
					stdoutWriter.Write(line[8:])
					stdoutWriter.Write([]byte("\n"))
				} else if line[0] == 2 {
					stderrWriter.Write(line[8:])
					stderrWriter.Write([]byte("\n"))
				}
			}
		}
	}()

	return stdoutReader, stderrReader
}

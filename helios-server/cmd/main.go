// Package main is the entry point for the Helios Docker Management Dashboard server.
// It initializes the Docker client, database, and HTTP server.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nfcunha/helios/core/models"
	"nfcunha/helios/core/repository"
	"nfcunha/helios/core/service"
	"nfcunha/helios/database"
	"nfcunha/helios/handler"
	"nfcunha/helios/utils/config"
	"nfcunha/helios/utils/docker"
	"nfcunha/helios/utils/statsutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Println("Starting Helios Docker Management Dashboard...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	if err := database.Initialize(cfg.Database.Path); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()
	log.Println("Database initialized successfully")

	// Initialize Docker client
	dockerClient, err := docker.NewClient()
	if err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}
	defer dockerClient.Close()

	// Test Docker connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := dockerClient.Ping(ctx); err != nil {
		log.Fatalf("Failed to connect to Docker daemon: %v", err)
	}
	log.Println("Docker client initialized successfully")

	// Create repository instances
	healthCheckRepo := repository.NewHealthCheckLogRepository(database.GetDB())
	actionLogRepo := repository.NewActionLogRepository(database.GetDB())

	// Create service instances
	containerService := service.NewContainerService(dockerClient, actionLogRepo)
	logService := service.NewLogService(dockerClient)
	imageService := service.NewImageService(dockerClient, actionLogRepo)
	volumeService := service.NewVolumeService(dockerClient, actionLogRepo)
	networkService := service.NewNetworkService(dockerClient, actionLogRepo)

	// Start health checker if enabled
	if cfg.HealthCheck.Enabled {
		go startHealthChecker(dockerClient, healthCheckRepo, &cfg.HealthCheck)
	}

	// Set Gin mode
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
		log.Println("Running in RELEASE mode")
	} else {
		gin.SetMode(gin.DebugMode)
		log.Println("Running in DEBUG mode")
	}

	// Create Gin engine
	engine := gin.New()
	engine.Use(gin.Recovery())
	if cfg.Server.Mode != "release" {
		engine.Use(gin.Logger())
	}

	// Add CORS middleware
	engine.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Basic health endpoint for Phase 1
	helios := engine.Group("/helios")
	{
		helios.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status": "healthy",
				"time":   time.Now(),
			})
		})

		// Container management endpoints (Phase 2)
		containerHandler := handler.NewContainerHandler(containerService)

		// Dashboard summary endpoint
		helios.GET("/dashboard/summary", containerHandler.GetDashboardSummary)

		containers := helios.Group("/containers")
		{
			containers.GET("", containerHandler.ListContainers)
			containers.GET("/:id", containerHandler.GetContainer)
			containers.POST("/:id/start", containerHandler.StartContainer)
			containers.POST("/:id/stop", containerHandler.StopContainer)
			containers.POST("/:id/restart", containerHandler.RestartContainer)
			containers.DELETE("/:id", containerHandler.RemoveContainer)

			// Bulk operations
			bulk := containers.Group("/bulk")
			{
				bulk.POST("/start", containerHandler.BulkStartContainers)
				bulk.POST("/stop", containerHandler.BulkStopContainers)
				bulk.POST("/remove", containerHandler.BulkRemoveContainers)
			}

			// Log streaming endpoints (Phase 3)
			logHandler := handler.NewLogHandler(logService)
			containers.GET("/:id/logs", logHandler.StreamLogs)
			containers.GET("/:id/logs/download", logHandler.DownloadLogs)
		}

		// Image management endpoints (Phase 4)
		imageHandler := handler.NewImageHandler(imageService)
		images := helios.Group("/images")
		{
			images.GET("", imageHandler.ListImages)
			images.GET("/search", imageHandler.SearchImages)
			images.GET("/tags", imageHandler.GetImageTags)
			images.GET("/:id", imageHandler.InspectImage)
			images.POST("/pull", imageHandler.PullImage)
			images.POST("/prune", imageHandler.PruneImages)
			images.DELETE("/:id", imageHandler.RemoveImage)

			// Bulk operations
			bulk := images.Group("/bulk")
			{
				bulk.POST("/remove", imageHandler.BulkRemoveImages)
			}
		}

		// Volume management endpoints (Phase 5)
		volumeHandler := handler.NewVolumeHandler(volumeService)
		volumes := helios.Group("/volumes")
		{
			volumes.GET("", volumeHandler.ListVolumes)
			volumes.GET("/:name", volumeHandler.InspectVolume)
			volumes.POST("", volumeHandler.CreateVolume)
			volumes.POST("/prune", volumeHandler.PruneVolumes)
			volumes.DELETE("/:name", volumeHandler.RemoveVolume)
		}

		// Network management endpoints (Phase 5)
		networkHandler := handler.NewNetworkHandler(networkService)
		networks := helios.Group("/networks")
		{
			networks.GET("", networkHandler.ListNetworks)
			networks.GET("/:id", networkHandler.InspectNetwork)
			networks.POST("", networkHandler.CreateNetwork)
			networks.POST("/prune", networkHandler.PruneNetworks)
			networks.DELETE("/:id", networkHandler.RemoveNetwork)
		}
	}

	// Create HTTP server
	addr := cfg.Server.Host + ":" + cfg.Server.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in background
	go func() {
		log.Printf("Helios server listening on %s", addr)
		log.Println("API available at: /helios")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped gracefully")
}

// startHealthChecker runs the health check loop at the configured interval.
func startHealthChecker(dockerClient *docker.Client, repo *repository.HealthCheckLogRepository, cfg *config.HealthCheckConfig) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Printf("Health checker started (interval: %v, CPU threshold: %.1f%%, Memory threshold: %.1f%%)",
		cfg.Interval, cfg.CPUThreshold, cfg.MemoryThreshold)

	for {
		<-ticker.C
		log.Println("Running health check...")

		containers, err := dockerClient.ContainerList(context.Background(), container.ListOptions{})
		if err != nil {
			log.Printf("Failed to list containers: %v", err)
			continue
		}

		for _, c := range containers {
			checkContainer(dockerClient, repo, c, cfg)
		}
	}
}

// checkContainer performs health check on a single container.
func checkContainer(dockerClient *docker.Client, repo *repository.HealthCheckLogRepository, c types.Container, cfg *config.HealthCheckConfig) {
	ctx := context.Background()

	// Get container name (remove leading slash)
	containerName := c.Names[0]
	if len(containerName) > 0 && containerName[0] == '/' {
		containerName = containerName[1:]
	}

	// Get container stats
	stats, err := dockerClient.ContainerStats(ctx, c.ID, false)
	if err != nil {
		log.Printf("Failed to get stats for container %s: %v", containerName, err)
		// Log error to database
		healthLog := &models.HealthCheckLog{
			ContainerID:   c.ID,
			ContainerName: containerName,
			Status:        "error",
			ErrorMessage:  err.Error(),
			CheckedAt:     time.Now(),
		}
		if err := repo.Create(healthLog); err != nil {
			log.Printf("Failed to store health check log: %v", err)
		}
		return
	}
	defer stats.Body.Close()

	// Parse stats
	var statsData container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&statsData); err != nil {
		log.Printf("Failed to decode stats for container %s: %v", containerName, err)
		return
	}

	// Calculate CPU percentage
	cpuPercent := statsutil.CalculateCPUPercent(&statsData)

	// Calculate memory percentage
	memoryPercent := float64(statsData.MemoryStats.Usage) / float64(statsData.MemoryStats.Limit) * 100.0

	// Determine status
	status := "healthy"
	if cpuPercent > cfg.CPUThreshold || memoryPercent > cfg.MemoryThreshold {
		status = "resource_critical"
		log.Printf("Container %s is resource critical (CPU: %.2f%%, Memory: %.2f%%)", containerName, cpuPercent, memoryPercent)
	}

	// Store health check log
	healthLog := &models.HealthCheckLog{
		ContainerID:         c.ID,
		ContainerName:       containerName,
		Status:              status,
		ResourceCPU:         cpuPercent,
		ResourceMemory:      statsData.MemoryStats.Usage,
		ResourceMemoryLimit: statsData.MemoryStats.Limit,
		ResourceNetworkRx:   statsutil.GetNetworkRx(&statsData),
		ResourceNetworkTx:   statsutil.GetNetworkTx(&statsData),
		CheckedAt:           time.Now(),
	}

	if err := repo.Create(healthLog); err != nil {
		log.Printf("Failed to store health check log: %v", err)
	}
}

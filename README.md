> This is a personal learning project. You’re welcome to use it as a reference or as a starting point for your own ideas. However, I strongly recommend using more mature and stable solutions available in the market rather than relying on this project for production needs.

# Helios 🌞

**Helios** is a modern, lightweight Docker container management dashboard built with Go and React. It provides real-time monitoring, resource tracking, and comprehensive container, image, volume, and network management through a clean web interface.

[![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)](https://hub.docker.com/r/cunhanicolas/helios)
[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/react-%2320232a.svg?style=for-the-badge&logo=react&logoColor=%2361DAFB)](https://reactjs.org/)

## ✨ Features

### Container Management
- **Real-time Monitoring**: Live stats for CPU, memory, network, and disk I/O
- **Background Caching**: 3-second background refresh eliminates loading delays
- **Bulk Operations**: Start, stop, restart, or remove multiple containers simultaneously
- **Log Streaming**: WebSocket-based real-time log viewing with download capability
- **Detailed Inspection**: Full container configuration, environment variables, mounts, and network settings

### Image Management
- **Pull Images**: Download images with real-time progress tracking
- **Search Registry**: Find images on Docker Hub
- **Remove & Prune**: Clean up unused images to reclaim disk space
- **Multi-architecture**: Full inspect details including layers and rootfs

### Volume & Network Management
- **Volume Operations**: Create, inspect, and remove Docker volumes
- **Network Configuration**: Manage bridge, overlay, and custom networks
- **Resource Pruning**: One-click cleanup of unused resources
- **Usage Tracking**: Monitor volume and network utilization

### Health Monitoring
- **Automatic Checks**: Periodic health checks for all running containers
- **Resource Alerts**: Flags containers exceeding CPU/Memory thresholds
- **Historical Logging**: SQLite database stores health check, action, and event logs

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```yaml
services:
  helios:
    image: cunhanicolas/helios:latest
    container_name: helios
    restart: unless-stopped
    ports:
      - "5000:5000"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - helios-data:/app/data
    environment:
      - HELIOS_SERVER_MODE=release
      - HELIOS_CPU_THRESHOLD=80.0
      - HELIOS_MEMORY_THRESHOLD=80.0

volumes:
  helios-data:
```

Start with: `docker compose up -d`

**Note:** The container automatically detects your host's Docker socket GID at runtime and configures permissions accordingly. No manual configuration needed!

Access at: [http://localhost:5000](http://localhost:5000)

### Using Docker CLI

```bash
docker run -d \
  --name helios \
  -p 5000:5000 \
  -e HELIOS_SERVER_PORT=8081 \
  -e HELIOS_SERVER_MODE=release \
  -e HELIOS_CPU_THRESHOLD=80.0 \
  -e HELIOS_MEMORY_THRESHOLD=80.0 \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  -v helios-data:/app/data \
  --restart unless-stopped \
  cunhanicolas/helios:latest
```

## ⚙️ Configuration

All configuration is done via environment variables with the `HELIOS_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `HELIOS_SERVER_PORT` | `8081` | Backend server port (internal) |
| `HELIOS_SERVER_MODE` | `debug` | Gin mode: `debug`, `release`, or `test` |
| `HELIOS_DB_PATH` | `/app/data/helios.db` | SQLite database file path |
| `HELIOS_HEALTH_CHECK_ENABLED` | `true` | Enable automatic health checks |
| `HELIOS_HEALTH_CHECK_INTERVAL` | `30` | Check interval in seconds |
| `HELIOS_CPU_THRESHOLD` | `90.0` | CPU threshold for alerts (%) |
| `HELIOS_MEMORY_THRESHOLD` | `90.0` | Memory threshold for alerts (%) |
| `HELIOS_LOG_RETENTION_DAYS` | `30` | Days to retain logs in database |

## 🏗️ Architecture

Helios uses a single-container deployment with NGINX and Supervisord:

- **NGINX** (Port 5000): Serves React frontend, proxies /helios/* to backend
- **Go Backend** (Port 8081): RESTful API, Docker SDK, stats caching
- **Stats Cache**: Background refresh every 3 seconds for instant UI updates
- **Health Checker**: Monitors containers, logs resource usage
- **SQLite Database**: Stores health, action, and event logs

## 📡 API Endpoints

All API endpoints are under `/helios`:

- `GET /helios/containers` - List containers
- `GET /helios/containers/:id` - Container details
- `POST /helios/containers/:id/{start|stop|restart}` - Container actions
- `GET /helios/dashboard/summary` - Dashboard metrics
- `WS /helios/logs/:id/stream` - Log streaming
- `GET /helios/images` - List images
- `GET /helios/volumes` - List volumes
- `GET /helios/networks` - List networks

See full API documentation in [DEPLOYMENT.md](./DEPLOYMENT.md)

## �� Security

- Mount Docker socket as **read-only** (`:ro`)
- No built-in authentication - use reverse proxy (Traefik, Caddy) in production
- Helios user (UID 1001) added to docker group for socket access

## 🐛 Troubleshooting

**Stats not updating?**
```bash
docker logs helios  # Check for errors
docker restart helios
```

**Permission denied?**
```bash
# Ensure docker socket is accessible
ls -l /var/run/docker.sock
```

**Web interface not loading?**
```bash
# Check services are running
docker exec helios supervisorctl status
```

## 🔧 Development

```bash
# Backend
cd helios-server
go build -o helios ./cmd
./helios

# Frontend
cd helios-ui
npm install
npm run dev
```

Frontend dev server: http://localhost:3000
Backend API: http://localhost:5000/helios

## 📝 License

MIT License - see LICENSE file for details

## 🙏 Acknowledgments

Built with Docker SDK, Gin Web Framework, React, Tailwind CSS, and Lucide Icons

---

Made with ❤️ by [Nicolas Cunha](https://github.com/nfcunha)

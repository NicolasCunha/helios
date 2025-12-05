#!/bin/bash
set -e

# Get the GID of the Docker socket
DOCKER_SOCKET_GID=$(stat -c '%g' /var/run/docker.sock 2>/dev/null || echo "")

if [ -n "$DOCKER_SOCKET_GID" ]; then
    echo "Docker socket GID: $DOCKER_SOCKET_GID"
    
    # Check if docker group already exists with correct GID
    EXISTING_GID=$(getent group docker | cut -d: -f3 || echo "")
    
    if [ "$EXISTING_GID" != "$DOCKER_SOCKET_GID" ]; then
        # Remove existing docker group if it exists with different GID
        if [ -n "$EXISTING_GID" ]; then
            echo "Removing docker group with GID $EXISTING_GID"
            groupdel docker 2>/dev/null || true
        fi
        
        # Create docker group with host's GID
        echo "Creating docker group with GID $DOCKER_SOCKET_GID"
        groupadd -g "$DOCKER_SOCKET_GID" docker
        
        # Add helios user to docker group
        usermod -aG docker helios
    fi
else
    echo "Warning: Docker socket not found, creating docker group with default GID"
    groupadd -g 984 docker 2>/dev/null || true
    usermod -aG docker helios 2>/dev/null || true
fi

# Ensure data directory has correct ownership
chown -R helios:helios /app/data 2>/dev/null || true

# Execute the main command
exec "$@"

#!/bin/bash
set -e

# Function to check if Docker is running
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        echo "🚨 Docker is not running. Starting Docker..."
        open -a Docker
        sleep 10  # Give Docker some time to start
        while ! docker info >/dev/null 2>&1; do
            echo "⏳ Waiting for Docker to start..."
            sleep 5
        done
    fi
    echo "✅ Docker is running!"
}

# Check and start Docker if needed
check_docker

# Check if Redis container is already running
if docker ps --format '{{.Names}}' | grep -q '^context-redis$'; then
    echo "⚡ Redis container is already running."
else
    echo "🔄 Starting Redis container..."
    docker run --rm --name context-redis -p 6379:6379 -d redis:7-alpine
fi

# Wait until Redis is ready
echo "⏳ Waiting for Redis to become available..."
until docker exec context-redis redis-cli ping | grep -q "PONG"; do
    sleep 1
done

echo "✅ Redis is up and running!"

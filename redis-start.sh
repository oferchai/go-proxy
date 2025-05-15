#!/bin/bash

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Error: .env file not found"
    exit 1
fi

# Load environment variables
source .env

# Start Redis container
docker compose up -d

# Show status
echo "Redis is starting..."
sleep 3
docker compose ps

# Show how to connect
echo -e "\nTo connect to Redis:"
echo "redis-cli -h localhost -p 6379 -a \$REDIS_PASSWORD" 
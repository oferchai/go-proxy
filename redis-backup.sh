#!/bin/bash

# Create backups directory if it doesn't exist
mkdir -p backups

# Get current timestamp
timestamp=$(date +%Y%m%d_%H%M%S)

# Create backup
docker run --rm \
  --volumes-from $(docker-compose ps -q redis) \
  -v $(pwd)/backups:/backup \
  alpine \
  tar czf /backup/redis_backup_${timestamp}.tar.gz /data

echo "Backup created: backups/redis_backup_${timestamp}.tar.gz" 
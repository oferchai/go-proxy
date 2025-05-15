#!/bin/bash

if [ -z "$1" ]; then
    echo "Usage: ./redis-restore.sh <backup_file>"
    exit 1
fi

# Stop Redis container
docker-compose stop redis

# Restore data
docker run --rm \
  --volumes-from $(docker-compose ps -q redis) \
  -v $(pwd)/backups:/backup \
  alpine \
  sh -c "cd /data && tar xzf /backup/$(basename $1)"

# Start Redis container
docker-compose start redis

echo "Restore completed" 
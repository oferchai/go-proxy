#!/bin/bash

# Stop Redis container
docker compose down

# To keep the data volume, don't use -v flag
# To remove volume: docker compose down -v 
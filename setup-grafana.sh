#!/bin/bash

# Wait for Grafana to be ready
until $(curl --output /dev/null --silent --head --fail http://localhost:3000); do
    printf '.'
    sleep 5
done

# Add Redis data source
curl -X POST -H "Content-Type: application/json" -d '{
    "name":"Redis",
    "type":"redis",
    "url":"redis:6379",
    "access":"proxy",
    "isDefault":true
}' http://localhost:3000/api/datasources

# Import dashboard
curl -X POST -H "Content-Type: application/json" -d @grafana/dashboards/proxy-stats.json http://localhost:3000/api/dashboards/db 
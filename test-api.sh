#!/bin/bash

echo "Testing hourly stats API..."

# Test current day
echo -e "\nQuerying current day (9 AM - 5 PM):"
curl -s "http://localhost:8080/api/stats/hourly?date=$(date +%Y-%m-%d)&from_hour=9&to_hour=17" | jq .

# Test specific date
echo -e "\nQuerying specific date (March 22, 2024):"
curl -s "http://localhost:8080/api/stats/hourly?date=2024-03-22&from_hour=9&to_hour=17" | jq .

# Test using POST
echo -e "\nTesting POST request:"
curl -s -X POST http://localhost:8080/api/stats/hourly \
-H "Content-Type: application/json" \
-d '{
    "date": "2024-03-22",
    "from_hour": 9,
    "to_hour": 17
}' | jq . 
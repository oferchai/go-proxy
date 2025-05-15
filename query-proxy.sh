#!/bin/bash

# Get today's date in the correct format
TODAY=$(date +"%Y-%m-%d")
START_TIME="${TODAY} 00:00:00"
END_TIME="${TODAY} 23:59:59"

# Make the API call
curl -X POST http://localhost:3000/api/timeframe \
-H "Content-Type: application/json" \
-d "{
    \"start_time\": \"${START_TIME}\",
    \"end_time\": \"${END_TIME}\"
}"

# Add newline for better formatting
echo 
#!/bin/bash
# Heroku log simulator - 
echo "Starting log generation"
METHODS=("GET" "POST" "PUT" "DELETE" "PATCH")
PATHS=(
  "/api/users" "/api/orders" "/api/products" "/api/dashboard" "/api/settings"
  "/api/auth/login" "/api/auth/logout" "/api/profile" "/api/notifications"
  "/api/reports" "/api/analytics" "/api/search" "/api/upload" "/api/download"
  "/login" "/signup" "/dashboard" "/profile" "/settings" "/logout"
  "/products" "/orders" "/cart" "/checkout" "/payment" "/admin"
)
IPS=(
  "8.8.8.8"         
  "1.1.1.1"         
  "185.199.108.153"
  "151.101.1.140"   
  "172.217.14.206"  
  "13.107.42.14"   
  "104.16.132.229" 
  "54.230.159.121"  # US
  "2.16.254.1"      # UK
  "82.165.177.154"  # Netherlands
  "87.250.250.242"  # Russia
  "202.106.0.20"    # China
)

DYNOS=("web.1" "web.2")
generate_status() {
  local rand=$((RANDOM % 100))
  if [ $rand -lt 75 ]; then
    echo $((200 + RANDOM % 5))  # 200, 201, 202, 203, 204
  elif [ $rand -lt 85 ]; then
    local codes=(301 302 304)
    echo ${codes[$((RANDOM % 3))]}
  elif [ $rand -lt 96 ]; then
    local codes=(400 401 403 404 422 429)
    echo ${codes[$((RANDOM % 6))]}
  else
    local codes=(500 502 503 504)
    echo ${codes[$((RANDOM % 4))]}
  fi
}
generate_response_time() {
  local rand=$((RANDOM % 100))
  if [ $rand -lt 65 ]; then
    echo $((15 + RANDOM % 85))
  elif [ $rand -lt 88 ]; then
    echo $((100 + RANDOM % 300))
  elif [ $rand -lt 97 ]; then
    echo $((400 + RANDOM % 1600))
  else
    echo $((2000 + RANDOM % 5000))
  fi
}
for i in {1..10}; do
  method=${METHODS[$((RANDOM % ${#METHODS[@]}))]}
  path=${PATHS[$((RANDOM % ${#PATHS[@]}))]}
  ip=${IPS[$((RANDOM % ${#IPS[@]}))]}
  dyno=${DYNOS[$((RANDOM % ${#DYNOS[@]}))]}
  status=$(generate_status)
  service_time=$(generate_response_time)
  connect_time=$((1 + RANDOM % 8))
  bytes=$((250 + RANDOM % 4000))
  current_hour=$(date +%H)
  current_min=$(date +%M)
  rand_min=$(( (current_min - RANDOM % 60 + 60) % 60 ))
  timestamp="2025-07-19T${current_hour}:$(printf "%02d" $rand_min):$(printf "%02d" $((RANDOM % 60))).$(printf "%06d" $((RANDOM % 1000000)))+00:00"
  request_id="req-$(date +%s)-$i-$((RANDOM % 9999))"
  frame_id="frame-$(date +%s)-$i"
  log_entry="$timestamp heroku[router]: at=info method=$method path=\"$path\" host=myapp.herokuapp.com request_id=$request_id fwd=\"$ip\" dyno=$dyno connect=${connect_time}ms service=${service_time}ms status=$status bytes=$bytes protocol=https"
  curl -s -X POST http://localhost:5000/logdrains \
    -H "Content-Type: application/logplex-1" \
    -H "User-Agent: Logplex/v123" \
    -H "Logplex-Msg-Count: 1" \
    -H "Logplex-Frame-Id: $frame_id" \
    -d "$log_entry"
  
  sleep 0.03
done

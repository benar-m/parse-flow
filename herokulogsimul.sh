#!/bin/bash
# Real Heroku logs injection script
echo "Starting real Heroku log injection..."
SERVER_URL="http://localhost:5000/logdrains"
declare -a REAL_LOGS=(
    "<134>1 2025-03-18T07:39:02.817989+00:00 host heroku web.1 - source=web.1 dyno=heroku.440712070.e5b1c909-743a-4285-86ae-bd0eeebe8dab sample#load_avg_1m=0.00 sample#load_avg_5m=0.00 sample#load_avg_15m=0.00"
    "<134>1 2025-03-18T07:39:02.834764+00:00 host heroku web.1 - source=web.1 dyno=heroku.440712070.e5b1c909-743a-4285-86ae-bd0eeebe8dab sample#memory_total=134.04MB sample#memory_rss=123.68MB sample#memory_cache=10.36MB sample#memory_swap=0.00MB sample#memory_pgpgin=1537pages sample#memory_pgpgout=122pages sample#memory_quota=512.00MB"
    "<158>1 2025-03-18T07:42:05.674768+00:00 host heroku router - at=info method=GET path=\"/\" host=www.kustudyhub.live request_id=776469a1-9108-41fc-9784-2469683ba7ae fwd=\"102.213.49.41\" dyno=web.1 connect=0ms service=1ms status=200 bytes=5778 protocol=https"
    "<158>1 2025-03-18T07:42:06.252822+00:00 host heroku router - at=info method=GET path=\"/static/css/home.css\" host=www.kustudyhub.live request_id=0de51742-1c7b-485e-9faa-640b41f4614d fwd=\"102.213.49.41\" dyno=web.1 connect=0ms service=1ms status=304 bytes=161 protocol=https"
    "<158>1 2025-03-18T07:42:08.011021+00:00 host heroku router - at=info method=GET path=\"/api/units/\" host=www.kustudyhub.live request_id=6036c7c8-e708-4971-b723-a0ed6c50f2ff fwd=\"102.213.49.41\" dyno=web.1 connect=0ms service=1311ms status=200 bytes=14555 protocol=https"
    "<190>1 2025-03-18T07:42:05.674526+00:00 host app web.1 - 10.1.52.53 - - [18/Mar/2025:07:42:05 +0000] \"GET / HTTP/1.1\" 200 5496 \"-\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<190>1 2025-03-18T07:42:06.252562+00:00 host app web.1 - 10.1.52.53 - - [18/Mar/2025:07:42:06 +0000] \"GET /static/css/home.css HTTP/1.1\" 304 0 \"https://www.kustudyhub.live/\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:42:06.254201+00:00 host heroku router - at=info method=GET path=\"/static/js/slider.js\" host=www.kustudyhub.live request_id=ee329c3e-1e3c-4112-b328-57f8f8f32a13 fwd=\"102.213.49.41\" dyno=web.1 connect=0ms service=0ms status=304 bytes=161 protocol=https"
    "<158>1 2025-03-18T07:42:06.771706+00:00 host heroku router - at=info method=GET path=\"/sw.js\" host=www.kustudyhub.live request_id=a4ea626f-7b31-41a8-8253-ef421f3a5d84 fwd=\"102.213.49.41\" dyno=web.1 connect=0ms service=1ms status=200 bytes=471 protocol=https"
    "<190>1 2025-03-18T07:42:06.771267+00:00 host app web.1 - 10.1.52.53 - - [18/Mar/2025:07:42:06 +0000] \"GET /sw.js HTTP/1.1\" 200 192 \"https://www.kustudyhub.live/\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:42:58.06297+00:00 host heroku router - at=info method=GET path=\"/\" host=www.kustudyhub.live request_id=5fe205db-38bc-4968-893f-b1576f836426 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=1ms status=200 bytes=5778 protocol=https"
    "<190>1 2025-03-18T07:42:58.062754+00:00 host app web.1 - 10.1.55.141 - - [18/Mar/2025:07:42:58 +0000] \"GET / HTTP/1.1\" 200 5496 \"-\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:43:01.136294+00:00 host heroku router - at=info method=GET path=\"/api/units\" host=www.kustudyhub.live request_id=ab878592-d9f1-4451-b97f-fe490cbd7c12 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=1ms status=301 bytes=308 protocol=https"
    "<158>1 2025-03-18T07:43:02.809686+00:00 host heroku router - at=info method=GET path=\"/favicon.ico\" host=www.kustudyhub.live request_id=c460183f-0ba7-452d-bba5-46dac331a559 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=1ms status=404 bytes=467 protocol=https"
    "<190>1 2025-03-18T07:43:02.80932+00:00 host app web.1 - 10.1.28.44 - - [18/Mar/2025:07:43:02 +0000] \"GET /favicon.ico HTTP/1.1\" 404 179 \"https://www.kustudyhub.live/\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:43:22.500402+00:00 host heroku router - at=info method=GET path=\"/units/?unitId=118\" host=www.kustudyhub.live request_id=5ba5189e-8f61-4463-b815-37ed8e65dec2 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=1ms status=200 bytes=3509 protocol=https"
    "<190>1 2025-03-18T07:43:22.500125+00:00 host app web.1 - 10.1.28.44 - - [18/Mar/2025:07:43:22 +0000] \"GET /units/?unitId=118 HTTP/1.1\" 200 3227 \"https://www.kustudyhub.live/\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:43:25.881848+00:00 host heroku router - at=info method=GET path=\"/api/units/\" host=www.kustudyhub.live request_id=bbdc7b23-ddb6-4c70-868f-23f9e8763905 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=95ms status=200 bytes=14555 protocol=https"
    "<190>1 2025-03-18T07:43:25.881173+00:00 host app web.1 - 10.1.55.141 - - [18/Mar/2025:07:43:25 +0000] \"GET /api/units/ HTTP/1.1\" 200 14280 \"https://www.kustudyhub.live/units/?unitId=118\" \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Mobile Safari/537.36\""
    "<158>1 2025-03-18T07:43:26.901781+00:00 host heroku router - at=info method=GET path=\"/api/unit/118/pdfs/\" host=www.kustudyhub.live request_id=c1332765-d961-4983-abc4-b07895ff23af fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=208ms status=200 bytes=2855 protocol=https"
    "<172>1 2025-03-18T07:44:02.283977+00:00 host heroku logplex - Error L10 (output buffer overflow): 1 messages dropped since 2025-03-18T07:43:08.99958+00:00."
    "<158>1 2025-03-18T07:44:20.123753+00:00 host heroku router - at=info method=GET path=\"/units/?unitId=207\" host=www.kustudyhub.live request_id=d930fe48-12d2-4ef3-a1d1-57dad9831766 fwd=\"102.213.49.42\" dyno=web.1 connect=0ms service=1ms status=200 bytes=3509 protocol=https"
    "<158>1 2025-03-18T07:44:22.053357+00:00 host heroku router - at=info method=GET path=\"/api/unit/207/pdfs/\" host=www.kustudyhub.live request_id=31d81b35-0348-4997-bc76-560f973106af fwd=\"102.213.49.42\" dyno=web.1 connect=0ms service=189ms status=200 bytes=554 protocol=https"
)

declare -a ERROR_LOGS=(
    "<158>1 2025-03-18T07:43:02.809686+00:00 host heroku router - at=info method=GET path=\"/favicon.ico\" host=www.kustudyhub.live request_id=c460183f-0ba7-452d-bba5-46dac331a559 fwd=\"41.89.10.241\" dyno=web.1 connect=0ms service=1ms status=404 bytes=467 protocol=https"
    "<158>1 2025-03-18T07:45:00.123456+00:00 host heroku router - at=error method=POST path=\"/api/users\" host=www.kustudyhub.live request_id=error-test-123 fwd=\"192.168.1.1\" dyno=web.1 connect=0ms service=5000ms status=500 bytes=0 protocol=https"
    "<158>1 2025-03-18T07:45:01.654321+00:00 host heroku router - at=error method=GET path=\"/api/timeout\" host=www.kustudyhub.live request_id=timeout-test-456 fwd=\"10.0.0.1\" dyno=web.1 connect=2000ms service=30000ms status=503 bytes=0 protocol=https"
    "<172>1 2025-03-18T07:44:02.283977+00:00 host heroku logplex - Error L10 (output buffer overflow): 5 messages dropped since 2025-03-18T07:43:08.99958+00:00."
)

# Function to generate realistic frame ID
generate_frame_id() {
    echo "frame-$(date +%s)-$((RANDOM % 10000))"
}

# Function to send log with proper Heroku headers
send_log() {
    local log_message="$1"
    local frame_id=$(generate_frame_id)
    
    echo "Sending: ${log_message:0:80}..." # Show first 80 chars
    
    curl -s -X POST "$SERVER_URL" \
        -H "Content-Type: application/logplex-1" \
        -H "User-Agent: Logplex/v123" \
        -H "Logplex-Msg-Count: 1" \
        -H "Logplex-Frame-Id: $frame_id" \
        -d "$log_message"
    
    local exit_code=$?
    if [ $exit_code -eq 0 ]; then
        echo "✓ Sent successfully"
    else
        echo "✗ Failed to send (exit code: $exit_code)"
    fi
}

# Parse command line arguments
RATE=1        # logs per second
DURATION=30   # seconds
MODE="mixed"  # mixed, normal, errors, memory, router

while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--rate)
            RATE="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -m|--mode)
            MODE="$2"
            shift 2
            ;;
        -u|--url)
            SERVER_URL="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -r, --rate RATE        Logs per second (default: 1)"
            echo "  -d, --duration SEC     Duration in seconds (default: 30)"
            echo "  -m, --mode MODE        Mode: mixed, normal, errors, memory, router (default: mixed)"
            echo "  -u, --url URL          Server URL (default: http://localhost:5000/logdrains)"
            echo "  -h, --help             Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "Configuration:"
echo "  Server URL: $SERVER_URL"
echo "  Rate: $RATE logs/second"
echo "  Duration: $DURATION seconds"
echo "  Mode: $MODE"
echo "  Total logs: $((RATE * DURATION))"
echo ""

# Check if server is running
echo "Checking server connectivity..."
if ! curl -s --connect-timeout 5 "$SERVER_URL" > /dev/null 2>&1; then
    echo "❌ Cannot connect to server at $SERVER_URL"
    echo "Make sure your log processor is running!"
    exit 1
fi
echo ""
SLEEP_INTERVAL=$(echo "scale=3; 1.0 / $RATE" | bc -l)

echo "Starting log injection (Ctrl+C to stop)..."
echo "Sleep interval: ${SLEEP_INTERVAL}s"
echo ""
count=0
start_time=$(date +%s)

while [ $count -lt $((RATE * DURATION)) ]; do
    case $MODE in
        "normal")
            log_index=$((RANDOM % (${#REAL_LOGS[@]} - 4)))  # Exclude error logs
            log="${REAL_LOGS[$log_index]}"
            ;;
        "errors")
            log_index=$((RANDOM % ${#ERROR_LOGS[@]}))
            log="${ERROR_LOGS[$log_index]}"
            ;;
        "memory")
            memory_logs=()
            for log_entry in "${REAL_LOGS[@]}"; do
                if [[ "$log_entry" == *"sample#memory"* ]] || [[ "$log_entry" == *"sample#load_avg"* ]]; then
                    memory_logs+=("$log_entry")
                fi
            done
            if [ ${#memory_logs[@]} -gt 0 ]; then
                log_index=$((RANDOM % ${#memory_logs[@]}))
                log="${memory_logs[$log_index]}"
            else
                log="${REAL_LOGS[0]}"
            fi
            ;;
        "router")
            router_logs=()
            for log_entry in "${REAL_LOGS[@]}"; do
                if [[ "$log_entry" == *"heroku router"* ]]; then
                    router_logs+=("$log_entry")
                fi
            done
            if [ ${#router_logs[@]} -gt 0 ]; then
                log_index=$((RANDOM % ${#router_logs[@]}))
                log="${router_logs[$log_index]}"
            else
                log="${REAL_LOGS[2]}"  # Fallback to known router log
            fi
            ;;
        *)
            all_logs=("${REAL_LOGS[@]}" "${ERROR_LOGS[@]}")
            log_index=$((RANDOM % ${#all_logs[@]}))
            log="${all_logs[$log_index]}"
            ;;
    esac
    
    current_timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.%6N+00:00")
    log=$(echo "$log" | sed -E "s/[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}\+[0-9]{2}:[0-9]{2}/$current_timestamp/")
    
    send_log "$log"
    
    count=$((count + 1))
    echo "Progress: $count/$((RATE * DURATION)) logs sent"
    echo ""
    
    # Sleep between requests
    sleep "$SLEEP_INTERVAL"
done

end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
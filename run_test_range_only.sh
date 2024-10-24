#!/bin/bash

# Function to run a test with retry
run_test_with_retry() {
    local test_command="$1"
    local output_file="$2"
    local max_attempts=2

    for ((attempt=1; attempt<=max_attempts; attempt++))
    do
        echo "Attempt $attempt of $max_attempts: $test_command"
        if $test_command 2>&1 | tee "$output_file"; then
            echo "Test passed on attempt $attempt"
            return 0
        else
            echo "Test failed on attempt $attempt"
            if [ $attempt -lt $max_attempts ]; then
                echo "Retrying..."
                sleep 5  # Wait for 5 seconds before retrying
            fi
        fi
    done

    echo "Test failed after $max_attempts attempts"
    return 1
}

# Read ranges from range.txt
readarray -t ranges < range.txt

for range in "${ranges[@]}"
do
    # Split the range into START_NUM and END_NUM
    IFS='-' read -ra NUMS <<< "$range"
    export START_NUM="${NUMS[0]}"
    export END_NUM="${NUMS[1]}"
    
    echo "Running Test Range: $START_NUM-$END_NUM"
    
    run_test_with_retry "go test -run ^TestHitRate$ blockConcur/test -v -timeout 30m -count=1" "./res/hitRate${START_NUM}_${END_NUM}.txt"
done

#!/bin/bash
LOG_FILE="/tmp/$(basename "$1").log"
rm -f "$LOG_FILE"

{
    echo "----------"
    echo "pwd: $(pwd)"
    echo "args: "
    for var in "$@"
    do
        echo "$var" 
    done
    echo "----------"
} | tee -a "$LOG_FILE"

ps aux | tee /tmp/log
pwd > /tmp/log3

$@ 2>&1 | tee -a "$LOG_FILE"
exit "${PIPESTATUS[0]}"
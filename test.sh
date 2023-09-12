#!/bin/bash
echo "----------"
echo "args: "
for var in "$@"
do
    echo "$var"
done
echo "----------"

ps aux | tee /tmp/log
pwd > /tmp/log3

$@ 2>&1 | tee "/tmp/$(basename "$1").log"
exit "${PIPESTATUS[0]}"
#!/usr/bin/env bash
# Starts the course site on port 8000 in the background, unless it's already running.
cd "$(dirname "$0")/.."
curl -fs -o /dev/null localhost:8000 && exit 0
detach=$(command -v setsid || true)
$detach nohup make course > /tmp/course.log 2>&1 < /dev/null &

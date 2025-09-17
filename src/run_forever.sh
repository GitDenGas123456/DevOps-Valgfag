#!/bin/bash

PYTHON=${PYTHON:-python3}
PYTHON_SCRIPT_PATH=$1

while true; do
    $PYTHON "$PYTHON_SCRIPT_PATH"
    if [ $? -ne 0 ]; then
        echo "Script crashed with exit code $?. Restarting..." >&2
        sleep 1
    fi
done

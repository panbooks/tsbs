#!/bin/bash

# Generate 10K data points for all supported formats and use cases
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BULK_DATA_DIR=${BULK_DATA_DIR:-${SCRIPT_DIR}}
SCALE=${SCALE:-"10"}
MAX_DATA_POINTS=${MAX_DATA_POINTS:-1000000}
TS_END=${TS_END:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
TS_START=${TS_START:-$(date -u -d "1 hours ago" +"%Y-%m-%dT%H:%M:%SZ")}

# All supported formats and use cases
FORMATS=${FORMATS:-"influx"}
#USE_CASES=${USE_CASES:-"cpu-only devops iot"}
USE_CASES=${USE_CASES:-"cpu-only"}

export BULK_DATA_DIR SCALE MAX_DATA_POINTS TS_START TS_END

for USE_CASE in ${USE_CASES}; do
    echo "=== Generating data for use case: ${USE_CASE} ==="
    FORMATS="${FORMATS}" USE_CASE="${USE_CASE}" "${SCRIPT_DIR}/../generate_data.sh"
done

#!/bin/bash

# Generate 10K data points
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

BULK_DATA_DIR=${BULK_DATA_DIR:-${SCRIPT_DIR}}
MAX_DATA_POINTS=10000
TS_END=${TS_END:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
TS_START=${TS_START:-$(date -u -d "6 hours ago" +"%Y-%m-%dT%H:%M:%SZ")}

export BULK_DATA_DIR MAX_DATA_POINTS TS_START TS_END
exec "${SCRIPT_DIR}/../generate_data.sh"

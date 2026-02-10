#!/bin/bash
# Generate sample devops data in InfluxDB line protocol format for TsunamiDB
#
# Usage:
#   ./scripts/generate_tsunamidb.sh [OPTIONS]
#
# Options (via environment variables):
#   HOURS       - Hours of data to generate (default: 3, counting back from now)
#   SCALE       - Number of simulated hosts (default: 10)
#   USE_CASE    - Use case: cpu-only, devops, iot (default: devops)
#   INTERVAL    - Log interval (default: 10s)
#   SEED        - PRNG seed for reproducibility (default: 123)
#   OUTPUT      - Output file path (default: /tmp/tsunamidb-data.txt)

set -euo pipefail

HOURS="${HOURS:-3}"
SCALE="${SCALE:-10}"
USE_CASE="${USE_CASE:-devops}"
INTERVAL="${INTERVAL:-10s}"
SEED="${SEED:-123}"
OUTPUT="${OUTPUT:-/tmp/tsunamidb-data.txt}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/../bin/tsbs_generate_data"

if [ ! -x "$BINARY" ]; then
    echo "Binary not found. Building tsbs_generate_data..."
    (cd "$SCRIPT_DIR/.." && go build -o bin/tsbs_generate_data ./cmd/tsbs_generate_data)
fi

TS_END=$(date -u +%Y-%m-%dT%H:%M:%SZ)
TS_START=$(date -u -d "${HOURS} hours ago" +%Y-%m-%dT%H:%M:%SZ)

echo "Generating ${USE_CASE} data:"
echo "  Time range : ${TS_START} -> ${TS_END} (${HOURS}h)"
echo "  Scale      : ${SCALE} hosts"
echo "  Interval   : ${INTERVAL}"
echo "  Output     : ${OUTPUT}"

"$BINARY" \
    --format="influx" \
    --use-case="$USE_CASE" \
    --scale="$SCALE" \
    --timestamp-start="$TS_START" \
    --timestamp-end="$TS_END" \
    --seed="$SEED" \
    --log-interval="$INTERVAL" \
    > "$OUTPUT"

LINES=$(wc -l < "$OUTPUT")
SIZE=$(du -h "$OUTPUT" | cut -f1)

echo "Done: ${LINES} rows, ${SIZE}"

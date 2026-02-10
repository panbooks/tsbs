#!/bin/bash
# Load generated data into TsunamiDB (InfluxDB-compatible API)
#
# Usage:
#   ./scripts/load_tsunamidb.sh [OPTIONS]
#
# Options (via environment variables):
#   INPUT       - Input file path (default: /tmp/tsunamidb-data.txt)
#   DB_NAME     - Database name (default: benchmark)
#   URL         - TsunamiDB URL (default: http://localhost:8086)
#   WORKERS     - Number of parallel workers (default: 4)
#   BATCH_SIZE  - Batch size per write (default: 5000)

set -euo pipefail

INPUT="${INPUT:-/tmp/tsunamidb-data.txt}"
DB_NAME="${DB_NAME:-benchmark}"
URL="${URL:-http://localhost:8086}"
WORKERS="${WORKERS:-4}"
BATCH_SIZE="${BATCH_SIZE:-5000}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/../bin/tsbs_load_influx"

if [ ! -x "$BINARY" ]; then
    echo "Binary not found. Building tsbs_load_influx..."
    (cd "$SCRIPT_DIR/.." && go build -o bin/tsbs_load_influx ./cmd/tsbs_load_influx)
fi

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file not found: ${INPUT}"
    echo "Run ./scripts/generate_tsunamidb.sh first."
    exit 1
fi

LINES=$(wc -l < "$INPUT")
SIZE=$(du -h "$INPUT" | cut -f1)

echo "Loading data into TsunamiDB:"
echo "  Input      : ${INPUT} (${LINES} rows, ${SIZE})"
echo "  Database   : ${DB_NAME}"
echo "  URL        : ${URL}"
echo "  Workers    : ${WORKERS}"
echo "  Batch size : ${BATCH_SIZE}"

export no_proxy=localhost,127.0.0.1

"$BINARY" \
    --db-name="$DB_NAME" \
    --urls="$URL" \
    --workers="$WORKERS" \
    --batch-size="$BATCH_SIZE" \
    < "$INPUT"

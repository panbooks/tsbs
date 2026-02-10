#!/bin/bash
# Load generated data into MynaDB (InfluxDB-compatible API)
#
# MynaDB supports:
#   - Gzip Content-Encoding (--gzip=true)
#   - Large batch sizes (up to 250MB body)
#   - No CREATE DATABASE via query API (--do-create-db=false)
#
# Usage:
#   ./scripts/load_mynadb.sh [OPTIONS]
#
# Options (via environment variables):
#   INPUT       - Input file path (default: /tmp/mynadb-data.txt)
#   DB_NAME     - Database name (default: benchmark)
#   URL         - MynaDB URL (default: http://localhost:7000)
#   WORKERS     - Number of parallel workers (default: 4)
#   BATCH_SIZE  - Batch size per write (default: 5000)
#   GZIP        - Enable gzip compression (default: true)

set -euo pipefail

INPUT="${INPUT:-/tmp/mynadb-data.txt}"
DB_NAME="${DB_NAME:-benchmark}"
URL="${URL:-http://localhost:7000}"
WORKERS="${WORKERS:-4}"
BATCH_SIZE="${BATCH_SIZE:-5000}"
GZIP="${GZIP:-true}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY="${SCRIPT_DIR}/../bin/tsbs_load_influx"

if [ ! -x "$BINARY" ]; then
    echo "Binary not found. Building tsbs_load_influx..."
    (cd "$SCRIPT_DIR/.." && go build -o bin/tsbs_load_influx ./cmd/tsbs_load_influx)
fi

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file not found: ${INPUT}"
    echo "Run ./scripts/generate_mynadb.sh first."
    exit 1
fi

LINES=$(wc -l < "$INPUT")
SIZE=$(du -h "$INPUT" | cut -f1)

echo "Loading data into MynaDB:"
echo "  Input      : ${INPUT} (${LINES} rows, ${SIZE})"
echo "  Database   : ${DB_NAME}"
echo "  URL        : ${URL}"
echo "  Workers    : ${WORKERS}"
echo "  Batch size : ${BATCH_SIZE}"
echo "  Gzip       : ${GZIP}"

export no_proxy=localhost,127.0.0.1

"$BINARY" \
    --db-name="$DB_NAME" \
    --urls="$URL" \
    --do-create-db=false \
    --gzip="$GZIP" \
    --workers="$WORKERS" \
    --batch-size="$BATCH_SIZE" \
    < "$INPUT"

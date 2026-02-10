#!/bin/bash

# Load 100K data points
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

FORMAT=${FORMAT:-"influx"}
BULK_DATA_DIR=${BULK_DATA_DIR:-${SCRIPT_DIR}}

export BULK_DATA_DIR
exec "${SCRIPT_DIR}/../load/load_${FORMAT}.sh"

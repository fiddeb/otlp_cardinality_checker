#!/bin/bash
set -e

# Script to start local ClickHouse server for development
# Assumes ClickHouse binary is in external/ directory

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/config/clickhouse-config.xml"
DATA_DIR="$PROJECT_ROOT/data/clickhouse"
CLICKHOUSE_BIN="$PROJECT_ROOT/external/clickhouse"

# Check if ClickHouse binary exists
if [ ! -f "$CLICKHOUSE_BIN" ]; then
    echo "Error: ClickHouse binary not found at $CLICKHOUSE_BIN"
    echo ""
    echo "Please download ClickHouse:"
    echo "  curl -LO 'https://builds.clickhouse.com/master/macos/clickhouse'"
    echo "  chmod +x clickhouse"
    echo "  mv clickhouse $CLICKHOUSE_BIN"
    echo ""
    exit 1
fi

# Create data directory
mkdir -p "$DATA_DIR"

echo "Starting ClickHouse server..."
echo "  Config: $CONFIG_FILE"
echo "  Data:   $DATA_DIR"
echo "  Ports:  TCP 9000, HTTP 8123"
echo ""

# Start ClickHouse server
exec "$CLICKHOUSE_BIN" server --config-file="$CONFIG_FILE"

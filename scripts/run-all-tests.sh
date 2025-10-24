#!/bin/bash
# run-all-tests.sh - Comprehensive test suite for OTLP Cardinality Checker
#
# Usage:
#   ./scripts/run-all-tests.sh [quick|comprehensive|stress]
#
# Modes:
#   quick        - 5 seconds per test, 2 VUs
#   comprehensive - 60 seconds per test, 10 VUs (default)
#   stress       - 120 seconds per test, 50 VUs

set -e

MODE="${1:-comprehensive}"
API_ENDPOINT="${API_ENDPOINT:-http://localhost:8080}"

# ANSI colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configure based on mode
case "$MODE" in
  quick)
    VUS=2
    DURATION="5s"
    NUM_METRICS=100
    NUM_SPANS=50
    NUM_MODULES=50
    CARDINALITY=20
    ;;
  comprehensive)
    VUS=10
    DURATION="60s"
    NUM_METRICS=1000
    NUM_SPANS=100
    NUM_MODULES=100
    CARDINALITY=50
    ;;
  stress)
    VUS=50
    DURATION="120s"
    NUM_METRICS=5000
    NUM_SPANS=500
    NUM_MODULES=500
    CARDINALITY=100
    ;;
  *)
    echo -e "${RED}❌ Invalid mode: $MODE${NC}"
    echo "Usage: $0 [quick|comprehensive|stress]"
    exit 1
    ;;
esac

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  OTLP Cardinality Checker - Test Suite                       ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Mode:${NC} $MODE"
echo -e "${BLUE}VUs:${NC} $VUS"
echo -e "${BLUE}Duration:${NC} $DURATION"
echo -e "${BLUE}API Endpoint:${NC} $API_ENDPOINT"
echo ""

# Check if server is running
if ! curl -s "$API_ENDPOINT/api/v1/stats" > /dev/null 2>&1; then
    echo -e "${RED}❌ Server not responding at $API_ENDPOINT${NC}"
    echo "Please start the server first:"
    echo "  ./otlp-cardinality-checker"
    exit 1
fi

# Get initial stats
echo -e "${YELLOW}📊 Initial state:${NC}"
INITIAL_METRICS=$(curl -s "$API_ENDPOINT/api/v1/metrics" 2>/dev/null | jq -r '.total // 0')
INITIAL_SPANS=$(curl -s "$API_ENDPOINT/api/v1/spans" 2>/dev/null | jq -r '.total // 0')
INITIAL_LOGS=$(curl -s "$API_ENDPOINT/api/v1/logs" 2>/dev/null | jq -r '.total // 0')
INITIAL_MEMORY=$(curl -s "$API_ENDPOINT/api/v1/health" 2>/dev/null | jq -r '.memory.sys_mb // 0')

echo "  Metrics: $INITIAL_METRICS"
echo "  Spans: $INITIAL_SPANS"
echo "  Logs: $INITIAL_LOGS"
echo "  Memory: ${INITIAL_MEMORY} MB"
echo ""

# Run metrics test
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📊 Testing Metrics Ingestion...${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
k6 run --vus $VUS --duration $DURATION \
  -e NUM_METRICS=$NUM_METRICS \
  -e CARDINALITY=$CARDINALITY \
  scripts/load-test-metrics.js

echo ""

# Run traces test
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🔍 Testing Traces Ingestion...${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
k6 run --vus $VUS --duration $DURATION \
  -e NUM_SPANS=$NUM_SPANS \
  -e CARDINALITY=$CARDINALITY \
  scripts/load-test-traces.js

echo ""

# Run logs test
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📝 Testing Logs Ingestion...${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
k6 run --vus $VUS --duration $DURATION \
  -e NUM_MODULES=$NUM_MODULES \
  -e CARDINALITY=$CARDINALITY \
  scripts/load-test-logs.js

echo ""

# Analyze for noisy neighbors
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🔎 Analyzing for Noisy Neighbors...${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
./scripts/find-noisy-neighbors.sh "$API_ENDPOINT"

echo ""

# Show final stats
echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  Final Statistics                                            ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

FINAL_METRICS=$(curl -s "$API_ENDPOINT/api/v1/metrics" 2>/dev/null | jq -r '.total // 0')
FINAL_SPANS=$(curl -s "$API_ENDPOINT/api/v1/spans" 2>/dev/null | jq -r '.total // 0')
FINAL_LOGS=$(curl -s "$API_ENDPOINT/api/v1/logs" 2>/dev/null | jq -r '.total // 0')
FINAL_MEMORY=$(curl -s "$API_ENDPOINT/api/v1/health" 2>/dev/null | jq -r '.memory.sys_mb // 0')
SERVICES_COUNT=$(curl -s "$API_ENDPOINT/api/v1/services" 2>/dev/null | jq -r '.total // 0')

echo "  Metrics: $FINAL_METRICS"
echo "  Spans: $FINAL_SPANS"
echo "  Logs: $FINAL_LOGS"
echo "  Services: $SERVICES_COUNT"
echo "  Memory: ${FINAL_MEMORY} MB"

# Calculate deltas
METRICS_DELTA=$((FINAL_METRICS - INITIAL_METRICS))
SPANS_DELTA=$((FINAL_SPANS - INITIAL_SPANS))
LOGS_DELTA=$((FINAL_LOGS - INITIAL_LOGS))
MEMORY_DELTA=$((FINAL_MEMORY - INITIAL_MEMORY))

echo ""
echo -e "${YELLOW}📈 Changes:${NC}"
echo -e "  Metrics: ${GREEN}+$METRICS_DELTA${NC}"
echo -e "  Spans: ${GREEN}+$SPANS_DELTA${NC}"
echo -e "  Logs: ${GREEN}+$LOGS_DELTA${NC}"
echo -e "  Memory growth: ${GREEN}+${MEMORY_DELTA} MB${NC}"

TOTAL_ITEMS=$((METRICS_DELTA + SPANS_DELTA + LOGS_DELTA))
if [ $TOTAL_ITEMS -gt 0 ] && [ $MEMORY_DELTA -gt 0 ]; then
  AVG_MEMORY=$((MEMORY_DELTA * 1024 / TOTAL_ITEMS))
  echo -e "  Average per item: ${CYAN}~${AVG_MEMORY} KB${NC}"
fi

echo ""
echo -e "${GREEN}✅ Test suite complete!${NC}"
echo ""

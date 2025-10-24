#!/bin/bash
# Find Noisy Neighbors - Services causing high cardinality or high volume
# Usage: ./scripts/find-noisy-neighbors.sh [API_ENDPOINT] [THRESHOLD]

API_ENDPOINT=${1:-"http://localhost:8080"}
THRESHOLD=${2:-30}  # Cardinality threshold

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  Noisy Neighbor Detection - OTLP Cardinality Checker         ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "API Endpoint: $API_ENDPOINT"
echo "Cardinality Threshold: $THRESHOLD"
echo ""

# Get overall stats by counting items from endpoints
METRICS_COUNT=$(curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" 2>/dev/null | jq -r '.total // 0')
SPANS_COUNT=$(curl -s "${API_ENDPOINT}/api/v1/spans?limit=10000" 2>/dev/null | jq -r '.total // 0')
LOGS_COUNT=$(curl -s "${API_ENDPOINT}/api/v1/logs?limit=10000" 2>/dev/null | jq -r '.total // 0')
MEMORY=$(curl -s "${API_ENDPOINT}/api/v1/health" 2>/dev/null | jq -r '.memory.sys_mb // 0')

echo "📊 Current State:"
echo "   Metrics: $METRICS_COUNT"
echo "   Spans: $SPANS_COUNT"  
echo "   Logs: $LOGS_COUNT"
echo "   Memory: ${MEMORY} MB"
echo ""

echo "═══════════════════════════════════════════════════════════════"
echo "1️⃣  Services by Total Sample Volume"
echo "═══════════════════════════════════════════════════════════════"

# Combine data from metrics, traces, and logs
{
  # Metrics
  if [ "$METRICS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r '
    [.data[] | .services | to_entries[] | {service: .key, count: .value, type: "metrics"}]
    ' 
  else
    echo "[]"
  fi
  
  # Traces
  if [ "$SPANS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/spans?limit=10000" | jq -r '
    [.data[] | .services | to_entries[] | {service: .key, count: .value, type: "traces"}]
    '
  else
    echo "[]"
  fi
  
  # Logs
  if [ "$LOGS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/logs?limit=10000" | jq -r '
    [.data[] | .services | to_entries[] | {service: .key, count: .value, type: "logs"}]
    '
  else
    echo "[]"
  fi
} | jq -s 'add | 
group_by(.service) |
map({
  service: .[0].service,
  total_samples: ([.[].count] | add),
  metrics_samples: ([.[] | select(.type == "metrics") | .count] | add // 0),
  traces_samples: ([.[] | select(.type == "traces") | .count] | add // 0),
  logs_samples: ([.[] | select(.type == "logs") | .count] | add // 0),
  signal_types: ([.[].type] | unique)
}) |
sort_by(.total_samples) | reverse | .[0:10]
' | jq -r '.[] | "  📊 \(.service):\n     Total: \(.total_samples) samples\n     Metrics: \(.metrics_samples) | Traces: \(.traces_samples) | Logs: \(.logs_samples)\n     Signal types: \(.signal_types | join(", "))\n"'

echo "═══════════════════════════════════════════════════════════════"
echo "2️⃣  High Cardinality Attributes (> ${THRESHOLD})"
echo "═══════════════════════════════════════════════════════════════"

# Combine high cardinality from metrics, traces, and logs
{
  # Metrics - check label_keys
  if [ "$METRICS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      . as $item |
      (.label_keys // {} | to_entries[] | select(.value.estimated_cardinality > $threshold)) as $attr |
      {
        name: $item.name,
        attribute: $attr.key,
        cardinality: $attr.value.estimated_cardinality,
        type: "metric",
        services: ($item.services | keys | join(", "))
      }
    ]
    '
  else
    echo "[]"
  fi
  
  # Traces - check attribute_keys
  if [ "$SPANS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/spans?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      . as $item |
      (.attribute_keys // {} | to_entries[] | select(.value.estimated_cardinality > $threshold)) as $attr |
      {
        name: $item.name,
        attribute: $attr.key,
        cardinality: $attr.value.estimated_cardinality,
        type: "span",
        services: ($item.services | keys | join(", "))
      }
    ]
    '
  else
    echo "[]"
  fi
  
  # Logs - check attribute_keys
  if [ "$LOGS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/logs?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      . as $item |
      (.attribute_keys // {} | to_entries[] | select(.value.estimated_cardinality > $threshold)) as $attr |
      {
        name: ("severity_" + $item.severity),
        attribute: $attr.key,
        cardinality: $attr.value.estimated_cardinality,
        type: "log",
        services: ($item.services | keys | join(", "))
      }
    ]
    '
  else
    echo "[]"
  fi
} | jq -s 'add | sort_by(.cardinality) | reverse | .[0:10]' | jq -r '
if length == 0 then
  "  ✅ No high cardinality attributes found\n"
else
  .[] | "  ⚠️  [\(.type)] \(.name).\(.attribute):\n     Cardinality: \(.cardinality)\n     Services: \(.services)\n"
end
'

echo "═══════════════════════════════════════════════════════════════"
echo "3️⃣  Services Contributing to High Cardinality"
echo "═══════════════════════════════════════════════════════════════"

# Combine services from all signal types
{
  # Metrics
  if [ "$METRICS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      select((.label_keys // {} | to_entries | any(.value.estimated_cardinality > $threshold))) |
      . as $item |
      .services | to_entries[] | {
        service: .key,
        samples: .value,
        item_name: $item.name,
        type: "metric"
      }
    ]
    '
  else
    echo "[]"
  fi
  
  # Traces
  if [ "$SPANS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/spans?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      select((.attribute_keys // {} | to_entries | any(.value.estimated_cardinality > $threshold))) |
      . as $item |
      .services | to_entries[] | {
        service: .key,
        samples: .value,
        item_name: $item.name,
        type: "span"
      }
    ]
    '
  else
    echo "[]"
  fi
  
  # Logs
  if [ "$LOGS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/logs?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
    [.data[] | 
      select((.attribute_keys // {} | to_entries | any(.value.estimated_cardinality > $threshold))) |
      . as $item |
      .services | to_entries[] | {
        service: .key,
        samples: .value,
        item_name: ("severity_" + $item.severity),
        type: "log"
      }
    ]
    '
  else
    echo "[]"
  fi
} | jq -s 'add | 
group_by(.service) |
map({
  service: .[0].service,
  high_card_items: length,
  total_samples: ([.[].samples] | add),
  by_type: (group_by(.type) | map({type: .[0].type, count: length}) | from_entries),
  examples: .[0:3] | map("\(.type):\(.item_name)")
}) |
sort_by(.total_samples) | reverse | .[0:10]
' | jq -r '
if length == 0 then
  "  ✅ No services contributing to high cardinality\n"
else
  .[] | "  🔥 \(.service):\n     High-card items: \(.high_card_items) (\(.by_type | to_entries | map("\(.key):\(.value)") | join(", ")))\n     Total samples: \(.total_samples)\n     Examples: \(.examples | join(", "))\n"
end
'

echo "═══════════════════════════════════════════════════════════════"
echo "4️⃣  Items with Most Unique Services (Multi-tenant Issues)"
echo "═══════════════════════════════════════════════════════════════"

# Combine data from all signal types
{
  # Metrics
  if [ "$METRICS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r '
    .data | map({
      name: .name,
      type: "metric",
      service_count: (.services | length),
      total_samples: .sample_count,
      services: (.services | keys)
    })
    '
  else
    echo "[]"
  fi
  
  # Traces
  if [ "$SPANS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/spans?limit=10000" | jq -r '
    .data | map({
      name: .name,
      type: "span",
      service_count: (.services | length),
      total_samples: .sample_count,
      services: (.services | keys)
    })
    '
  else
    echo "[]"
  fi
  
  # Logs
  if [ "$LOGS_COUNT" -gt 0 ]; then
    curl -s "${API_ENDPOINT}/api/v1/logs?limit=10000" | jq -r '
    .data | map({
      name: ("severity_" + .severity),
      type: "log",
      service_count: (.services | length),
      total_samples: .sample_count,
      services: (.services | keys)
    })
    '
  else
    echo "[]"
  fi
} | jq -s 'add | sort_by(.service_count) | reverse | .[0:10]' | jq -r '
.[] | "  🏢 [\(.type)] \(.name):\n     Services: \(.service_count)\n     Samples: \(.total_samples)\n     List: \(.services | join(", "))\n"'

echo "═══════════════════════════════════════════════════════════════"
echo "5️⃣  Summary - Top Noisy Neighbors"
echo "═══════════════════════════════════════════════════════════════"

# Get top 5 services by sample volume
TOP_SERVICES=$(curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r '
[.data[] | .services | to_entries[] | {service: .key, count: .value}] |
group_by(.service) |
map({
  service: .[0].service,
  total_samples: ([.[].count] | add)
}) |
sort_by(.total_samples) | reverse | .[0:5] | .[] | .service
' | tr '\n' ' ')

echo ""
echo "🎯 Recommended Actions:"
echo ""

if [ -z "$TOP_SERVICES" ]; then
    echo "  ✅ No concerning patterns detected"
else
    echo "  Top services to investigate: $TOP_SERVICES"
    echo ""
    echo "  Next steps:"
    echo "  1. Review metrics for top services:"
    for service in $TOP_SERVICES; do
        echo "     curl '${API_ENDPOINT}/api/v1/services/${service}/overview' | jq ."
    done
    echo ""
    echo "  2. Check for label explosion in specific metrics"
    echo "  3. Consider adding label filters or sampling for high-volume services"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════"

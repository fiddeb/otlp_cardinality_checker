#!/bin/bash
# Find Noisy Neighbors - Services causing high cardinality or high volume
# Usage: ./scripts/find-noisy-neighbors.sh [API_ENDPOINT]

API_ENDPOINT=${1:-"http://localhost:8080"}
THRESHOLD=${2:-30}  # Cardinality threshold

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║  Noisy Neighbor Detection - OTLP Cardinality Checker         ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo ""
echo "API Endpoint: $API_ENDPOINT"
echo "Cardinality Threshold: $THRESHOLD"
echo ""

echo "═══════════════════════════════════════════════════════════════"
echo "1️⃣  Services by Total Sample Volume"
echo "═══════════════════════════════════════════════════════════════"
curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r '
[.data[] | .services | to_entries[] | {service: .key, count: .value}] |
group_by(.service) |
map({
  service: .[0].service,
  total_samples: ([.[].count] | add),
  metrics_count: length
}) |
sort_by(.total_samples) | reverse | .[0:10]
' | jq -r '.[] | "  📊 \(.service):\n     Samples: \(.total_samples)\n     Metrics: \(.metrics_count)\n"'

echo "═══════════════════════════════════════════════════════════════"
echo "2️⃣  High Cardinality Labels (> ${THRESHOLD})"
echo "═══════════════════════════════════════════════════════════════"
curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
[.data[] | 
  . as $metric |
  (.label_keys // {} | to_entries[] | select(.value.estimated_cardinality > $threshold)) as $label |
  {
    metric: $metric.name,
    label: $label.key,
    cardinality: $label.value.estimated_cardinality,
    services: ($metric.services | to_entries | map({service: .key, samples: .value}))
  }
] | sort_by(.cardinality) | reverse | .[0:10]
' | jq -r '
if length == 0 then
  "  ✅ No high cardinality labels found\n"
else
  .[] | "  ⚠️  \(.metric).\(.label):\n     Cardinality: \(.cardinality)\n     Services: \(.services | map(.service) | join(", "))\n"
end
'

echo "═══════════════════════════════════════════════════════════════"
echo "3️⃣  Services Contributing to High Cardinality"
echo "═══════════════════════════════════════════════════════════════"
curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r --argjson threshold "$THRESHOLD" '
[.data[] | 
  select((.label_keys // {} | to_entries | any(.value.estimated_cardinality > $threshold))) |
  . as $metric |
  .services | to_entries[] | {
    service: .key,
    samples: .value,
    metric: $metric.name,
    high_card_labels: [$metric.label_keys | to_entries[] | select(.value.estimated_cardinality > $threshold) | .key]
  }
] |
group_by(.service) |
map({
  service: .[0].service,
  high_card_metrics: length,
  total_samples: ([.[].samples] | add),
  examples: .[0:3]
}) |
sort_by(.total_samples) | reverse | .[0:10]
' | jq -r '
if length == 0 then
  "  ✅ No services contributing to high cardinality\n"
else
  .[] | "  🔥 \(.service):\n     High-card metrics: \(.high_card_metrics)\n     Total samples: \(.total_samples)\n     Examples: \(.examples | map(.metric) | join(", "))\n"
end
'

echo "═══════════════════════════════════════════════════════════════"
echo "4️⃣  Metrics with Most Unique Services (Multi-tenant Issues)"
echo "═══════════════════════════════════════════════════════════════"
curl -s "${API_ENDPOINT}/api/v1/metrics?limit=10000" | jq -r '
.data | 
sort_by(.services | length) | reverse | .[0:10] |
map({
  metric: .name,
  service_count: (.services | length),
  total_samples: .sample_count,
  services: (.services | keys)
})
' | jq -r '.[] | "  🏢 \(.metric):\n     Services: \(.service_count)\n     Samples: \(.total_samples)\n     List: \(.services | join(", "))\n"'

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

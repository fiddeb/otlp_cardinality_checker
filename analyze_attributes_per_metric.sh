#!/bin/bash

# Script för att räkna antal resource attributes och label attributes per metric
# Hanterar pagination för alla metrics

echo "Analyserar antal attributes per metric..."

offset=0
limit=1000
temp_file=$(mktemp)

while true; do
  echo "Bearbetar metrics $offset-$((offset + limit - 1))..."
  
  response=$(curl -s "http://localhost:8080/api/v1/metrics?limit=$limit&offset=$offset")
  
  # Extrahera metric-namn och antal attributes
  echo "$response" | jq -r '
    .data[] |
    {
      name: .name,
      label_count: (.label_keys | length),
      resource_count: (.resource_keys | length),
      total_attributes: ((.label_keys | length) + (.resource_keys | length)),
      sample_count: .sample_count
    } |
    "\(.total_attributes):\(.label_count):\(.resource_count):\(.sample_count):\(.name)"
  ' >> "$temp_file"
  
  # Kolla om det finns fler sidor
  has_more=$(echo "$response" | jq -r '.has_more')
  if [ "$has_more" != "true" ]; then
    break
  fi
  
  offset=$((offset + limit))
done

echo ""
echo "TOP 20 METRICS MED FLEST TOTALA ATTRIBUTES:"
echo "==========================================="
echo "Format: [Totalt] [Labels] [Resources] [Samples] Metric-namn"
sort -nr "$temp_file" | head -20 | while IFS=':' read -r total label resource samples name; do
  printf "%2d attributes (%2d labels + %2d resources) | %'8d samples | %s\n" "$total" "$label" "$resource" "$samples" "$name"
done

echo ""
echo "TOP 20 METRICS MED FLEST LABEL ATTRIBUTES:"
echo "=========================================="
sort -t: -k2 -nr "$temp_file" | head -20 | while IFS=':' read -r total label resource samples name; do
  printf "%2d labels (%2d resources) | %'8d samples | %s\n" "$label" "$resource" "$samples" "$name"
done

echo ""
echo "TOP 20 METRICS MED FLEST RESOURCE ATTRIBUTES:"
echo "============================================="
sort -t: -k3 -nr "$temp_file" | head -20 | while IFS=':' read -r total label resource samples name; do
  printf "%2d resources (%2d labels) | %'8d samples | %s\n" "$resource" "$label" "$samples" "$name"
done

echo ""
echo "STATISTIK:"
echo "=========="
total_metrics=$(wc -l < "$temp_file")
avg_labels=$(awk -F: '{sum+=$2} END {printf "%.1f", sum/NR}' "$temp_file")
avg_resources=$(awk -F: '{sum+=$3} END {printf "%.1f", sum/NR}' "$temp_file")
avg_total=$(awk -F: '{sum+=$1} END {printf "%.1f", sum/NR}' "$temp_file")

echo "Totalt antal metrics: $total_metrics"
echo "Genomsnitt labels per metric: $avg_labels"
echo "Genomsnitt resource attributes per metric: $avg_resources"
echo "Genomsnitt totala attributes per metric: $avg_total"

echo ""
echo "METRICS MED MÅNGA ATTRIBUTES (>15 totalt):"
echo "=========================================="
awk -F: '$1 > 15 {printf "%2d attributes (%2d + %2d) | %s\n", $1, $2, $3, $5}' "$temp_file" | sort -nr

# Cleanup
rm "$temp_file"

echo ""
echo "Analys klar!"
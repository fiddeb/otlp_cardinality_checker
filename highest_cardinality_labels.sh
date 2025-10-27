#!/bin/bash

# Script för att hitta labeln med högst estimated_cardinality bland alla metrics
# Hanterar pagination för alla 8411+ metrics

echo "Analyserar alla metrics för att hitta högsta estimated_cardinality..."

offset=0
limit=1000
temp_file=$(mktemp)

while true; do
  echo "Bearbetar metrics $offset-$((offset + limit - 1))..."
  
  response=$(curl -s "http://localhost:8080/api/v1/metrics?limit=$limit&offset=$offset")
  
  # Extrahera alla label keys med estimated_cardinality från alla metrics på denna sida
  echo "$response" | jq -r '
    .data[] as $metric |
    $metric.label_keys | to_entries[] |
    "\(.value.estimated_cardinality):\(.key):\($metric.name)"
  ' >> "$temp_file"
  
  # Kolla om det finns fler sidor
  has_more=$(echo "$response" | jq -r '.has_more')
  if [ "$has_more" != "true" ]; then
    break
  fi
  
  offset=$((offset + limit))
done

echo ""
echo "TOP 20 LABELS MED HÖGST ESTIMATED_CARDINALITY:"
echo "=============================================="
sort -nr "$temp_file" | head -20 | while IFS=':' read -r cardinality label metric; do
  printf "%'d cardinality: %s (i metric: %s)\n" "$cardinality" "$label" "$metric"
done

echo ""
echo "TOP 10 UNIKA LABELS (högsta cardinality per label-namn):"
echo "======================================================="
sort -nr "$temp_file" | awk -F: '!seen[$2]++ {print $1":"$2":"$3}' | head -10 | while IFS=':' read -r cardinality label metric; do
  printf "%'d cardinality: %s (första förekomst i: %s)\n" "$cardinality" "$label" "$metric"
done

echo ""
echo "LABELS MED CARDINALITY > 1000:"
echo "=============================="
sort -nr "$temp_file" | awk -F: '$1 > 1000 {print $1":"$2":"$3}' | while IFS=':' read -r cardinality label metric; do
  printf "%'d cardinality: %s (i metric: %s)\n" "$cardinality" "$label" "$metric"
done

# Cleanup
rm "$temp_file"

echo ""
echo "Analys klar!"
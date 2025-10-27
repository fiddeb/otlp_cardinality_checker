#!/bin/bash

# Script för att hitta metrics med högst sample count
# Hanterar pagination för alla 8411+ metrics

echo "Hämtar alla metrics och sorterar efter sample count..."

offset=0
limit=1000
all_metrics=()

# Temporär fil för att samla alla metrics
temp_file=$(mktemp)

while true; do
  echo "Hämtar metrics $offset-$((offset + limit - 1))..."
  
  response=$(curl -s "http://localhost:8080/api/v1/metrics?limit=$limit&offset=$offset")
  
  # Extrahera sample_count och name från denna sida
  echo "$response" | jq -r '.data[] | "\(.sample_count):\(.name)"' >> "$temp_file"
  
  # Kolla om det finns fler sidor
  has_more=$(echo "$response" | jq -r '.has_more')
  if [ "$has_more" != "true" ]; then
    break
  fi
  
  offset=$((offset + limit))
done

echo ""
echo "TOP 20 METRICS MED FLEST SAMPLES:"
echo "=================================="
sort -nr "$temp_file" | head -20 | while IFS=':' read -r count name; do
  printf "%'d samples: %s\n" "$count" "$name"
done

echo ""
echo "BOTTOM 10 METRICS MED MINST SAMPLES:"
echo "===================================="
sort -n "$temp_file" | head -10 | while IFS=':' read -r count name; do
  printf "%'d samples: %s\n" "$count" "$name"
done

# Cleanup
rm "$temp_file"

echo ""
echo "Totalt antal metrics analyserade: $(wc -l < "$temp_file" 2>/dev/null || echo "Okänt")"
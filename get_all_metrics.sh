#!/bin/bash

# Script för att hämta alla metric-namn från OTLP cardinality checker
# Använder pagination för att få alla 8411+ metrics

echo "Hämtar alla metrics..."

offset=0
limit=1000
all_metrics=()

while true; do
  echo "Hämtar metrics $offset-$((offset + limit - 1))..."
  
  response=$(curl -s "http://localhost:8080/api/v1/metrics?limit=$limit&offset=$offset")
  
  # Extrahera metric-namn från denna sida
  metrics=$(echo "$response" | jq -r '.data[].name')
  
  # Lägg till i array
  while IFS= read -r metric; do
    if [ -n "$metric" ]; then
      all_metrics+=("$metric")
    fi
  done <<< "$metrics"
  
  # Kolla om det finns fler sidor
  has_more=$(echo "$response" | jq -r '.has_more')
  if [ "$has_more" != "true" ]; then
    break
  fi
  
  offset=$((offset + limit))
done

echo ""
echo "Totalt antal metrics hämtade: ${#all_metrics[@]}"
echo ""
echo "Alla metric-namn:"
printf '%s\n' "${all_metrics[@]}"
#!/usr/bin/env python3
"""
Send logs from a text file to OTLP endpoint.
Each line becomes a separate log entry.
"""

import json
import time
import sys
from datetime import datetime
import base64
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

OTLP_ENDPOINT = "http://localhost:4318/v1/logs"

def create_otlp_payload(log_body, severity="INFO"):
    """Create OTLP log payload"""
    now_ns = int(time.time() * 1_000_000_000)
    
    return {
        "resourceLogs": [
            {
                "resource": {
                    "attributes": [
                        {"key": "service.name", "value": {"stringValue": "real-logs"}},
                        {"key": "host.name", "value": {"stringValue": "combo"}},
                        {"key": "deployment.environment", "value": {"stringValue": "production"}}
                    ]
                },
                "scopeLogs": [
                    {
                        "scope": {
                            "name": "file-log-sender",
                            "version": "1.0.0"
                        },
                        "logRecords": [
                            {
                                "timeUnixNano": str(now_ns),
                                "observedTimeUnixNano": str(now_ns),
                                "severityNumber": 9,  # INFO
                                "severityText": severity,
                                "body": {
                                    "stringValue": log_body
                                },
                                "attributes": [
                                    {"key": "log.level", "value": {"stringValue": severity.lower()}},
                                    {"key": "log.source", "value": {"stringValue": "file"}}
                                ]
                            }
                        ]
                    }
                ]
            }
        ]
    }

def send_logs(file_path, batch_size=100):
    """Read file and send logs in batches"""
    print(f"Reading logs from: {file_path}")
    
    with open(file_path, 'r', encoding='utf-8') as f:
        lines = [line.strip() for line in f if line.strip()]
    
    total_lines = len(lines)
    print(f"Found {total_lines} log lines")
    print(f"Sending to: {OTLP_ENDPOINT}")
    print()
    
    sent = 0
    failed = 0
    start_time = time.time()
    
    for i in range(0, total_lines, batch_size):
        batch = lines[i:i+batch_size]
        
        # Send each line in the batch
        for line in batch:
            payload = create_otlp_payload(line)
            
            try:
                data = json.dumps(payload).encode('utf-8')
                req = Request(
                    OTLP_ENDPOINT,
                    data=data,
                    headers={"Content-Type": "application/json"}
                )
                
                response = urlopen(req, timeout=5)
                
                if response.status == 200:
                    sent += 1
                else:
                    failed += 1
                    print(f"Failed to send log (status {response.status}): {line[:80]}...")
                    
            except (HTTPError, URLError) as e:
                failed += 1
                print(f"Error sending log: {e}")
        
        # Progress update
        if (i + batch_size) % 500 == 0:
            elapsed = time.time() - start_time
            rate = sent / elapsed if elapsed > 0 else 0
            print(f"Progress: {sent}/{total_lines} logs sent ({rate:.0f} logs/sec)")
    
    # Final stats
    elapsed = time.time() - start_time
    rate = sent / elapsed if elapsed > 0 else 0
    
    print()
    print("=" * 60)
    print(f"✓ Complete!")
    print(f"  Sent:    {sent:,} logs")
    print(f"  Failed:  {failed:,} logs")
    print(f"  Time:    {elapsed:.2f}s")
    print(f"  Rate:    {rate:.0f} logs/sec")
    print("=" * 60)
    print()
    print("Query results:")
    print(f"  curl http://localhost:8080/api/v1/logs/INFO | jq '.body_templates'")

if __name__ == "__main__":
    file_path = sys.argv[1] if len(sys.argv) > 1 else "tmp/logs.txt"
    batch_size = int(sys.argv[2]) if len(sys.argv) > 2 else 100
    
    send_logs(file_path, batch_size)

// scripts/k6-send-logs.js
// k6 load test script that reads ./tmp/data/logs.json and posts log batches
// Usage examples (see README below):
// K6_VUS=50 K6_DURATION=1m BATCH=10 OTEL_COLLECTOR_URL=http://localhost:4318/v1/logs k6 run scripts/k6-send-logs.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';

// Configuration via env vars (or defaults)
const TARGET = __ENV.OTEL_COLLECTOR_URL || 'http://localhost:4318/v1/logs';
const VUS = __ENV.K6_VUS ? parseInt(__ENV.K6_VUS) : 50;
const DURATION = __ENV.K6_DURATION || '1m';
const BATCH = __ENV.BATCH ? parseInt(__ENV.BATCH) : 10;
const RANDOM_SLEEP_MS = __ENV.RANDOM_SLEEP_MS ? parseInt(__ENV.RANDOM_SLEEP_MS) : 100;

export const options = {
  vus: VUS,
  duration: DURATION,
  thresholds: {
    http_req_duration: ['p(95)<1000'],
    http_req_failed: ['rate<0.05'],
  },
};

// Load the JSON file once per k6 instance. File path is relative to the script.
const logsData = new SharedArray('logs', function () {
  // Ensure your file is present at scripts relative path: ./tmp/data/logs.json
  const text = open('./data/logs.json');
  try {
    const parsed = JSON.parse(text);
    // If top-level is array of OTLP payloads or log records, normalize to array
    if (Array.isArray(parsed)) return parsed;
    return [parsed];
  } catch (e) {
    console.error('Failed to parse data/logs.json', e);
    return [];
  }
});

function pickRandomBatch(batchSize) {
  const out = [];
  for (let i = 0; i < batchSize; i++) {
    const idx = Math.floor(Math.random() * logsData.length);
    out.push(logsData[idx]);
  }
  return out;
}

// Helper: construct a conservative OTLP JSON envelope when input isn't already an envelope
function buildEnvelopeFromBatch(batch) {
  // If the batch item already looks like an OTLP envelope (has resourceLogs), merge
  const envelope = { resourceLogs: [] };
  for (const item of batch) {
    if (item && typeof item === 'object') {
      if (item.resourceLogs) {
        envelope.resourceLogs = envelope.resourceLogs.concat(item.resourceLogs);
        continue;
      }
      // If the item looks like a full LogRecord or scoped object, wrap it
      if (item.logRecords || item.scopeLogs) {
        // crude merging
        if (item.scopeLogs) {
          envelope.resourceLogs.push({ resource: item.resource || {}, scopeLogs: item.scopeLogs });
        } else {
          envelope.resourceLogs.push({ resource: item.resource || {}, scopeLogs: [{ scope: item.scope || {}, logRecords: item.logRecords || [item] }] });
        }
        continue;
      }
      // Fallback: treat item as a single LogRecord and wrap it under a minimal envelope
      envelope.resourceLogs.push({ resource: {}, scopeLogs: [{ scope: {}, logRecords: [item] }] });
    }
  }
  return envelope;
}

export default function () {
  if (logsData.length === 0) {
    console.error('No logs loaded from tmp/data/logs.json - aborting iteration');
    sleep(1);
    return;
  }

  // Choose a batch to post
  const batch = pickRandomBatch(BATCH);

  // If the first item already seems like a full OTLP envelope and batch size == 1, send it
  let payload;
  if (batch.length === 1 && batch[0] && (batch[0].resourceLogs || batch[0].resourceSpans || batch[0].resourceMetrics)) {
    payload = batch[0];
  } else {
    payload = buildEnvelopeFromBatch(batch);
  }

  const params = { headers: { 'Content-Type': 'application/json' }, tags: { name: 'otlp-logs' } };
  const res = http.post(TARGET, JSON.stringify(payload), params);

  check(res, {
    'status is 2xx': (r) => r.status >= 200 && r.status < 300,
  });

  // small randomized sleep to create a realistic arrival pattern
  sleep(Math.random() * (RANDOM_SLEEP_MS / 1000));
}

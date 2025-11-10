// K6 Mixed Load Test for ClickHouse Backend
// 70% write operations (OTLP), 30% read operations (REST API)
// Tests realistic production workload with concurrent reads and writes

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const metricsWritten = new Counter('metrics_written');
const spansWritten = new Counter('spans_written');
const logsWritten = new Counter('logs_written');
const apiReads = new Counter('api_reads');
const writeDuration = new Trend('write_duration');
const readDuration = new Trend('read_duration');

// Test configuration
export const options = {
  scenarios: {
    // Write scenarios (70% of load)
    write_metrics: {
      executor: 'constant-arrival-rate',
      rate: 70, // 70 metrics/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 50,
      maxVUs: 100,
      exec: 'writeMetrics',
    },
    write_spans: {
      executor: 'constant-arrival-rate',
      rate: 35, // 35 spans/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 25,
      maxVUs: 50,
      exec: 'writeSpans',
    },
    write_logs: {
      executor: 'constant-arrival-rate',
      rate: 20, // 20 logs/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 15,
      maxVUs: 30,
      exec: 'writeLogs',
    },
    
    // Read scenarios (30% of load)
    read_metrics: {
      executor: 'constant-arrival-rate',
      rate: 20, // 20 reads/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 10,
      maxVUs: 20,
      exec: 'readMetrics',
      startTime: '10s', // Start reads after some writes
    },
    read_spans: {
      executor: 'constant-arrival-rate',
      rate: 10, // 10 reads/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 5,
      maxVUs: 10,
      exec: 'readSpans',
      startTime: '10s',
    },
    read_logs: {
      executor: 'constant-arrival-rate',
      rate: 10, // 10 reads/sec
      timeUnit: '1s',
      duration: '3m',
      preAllocatedVUs: 5,
      maxVUs: 10,
      exec: 'readLogs',
      startTime: '10s',
    },
  },
  
  thresholds: {
    'http_req_failed': ['rate<0.01'], // <1% errors
    'write_duration': ['p95<1000'], // 95% under 1s
    'read_duration': ['p95<200'], // 95% under 200ms
    'http_req_duration': ['p95<1000'],
  },
};

const BASE_URL = 'http://localhost:4318';
const API_URL = 'http://localhost:8080';

// Generate OTLP metric payload
function generateMetricPayload() {
  const metricName = `http_requests_total_${Math.floor(Math.random() * 50)}`;
  const method = ['GET', 'POST', 'PUT', 'DELETE'][Math.floor(Math.random() * 4)];
  const status = [200, 201, 400, 404, 500][Math.floor(Math.random() * 5)];
  
  return JSON.stringify({
    resourceMetrics: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: `service-${Math.floor(Math.random() * 10)}` }
        }]
      },
      scopeMetrics: [{
        metrics: [{
          name: metricName,
          unit: '1',
          sum: {
            dataPoints: [{
              asInt: Math.floor(Math.random() * 1000),
              attributes: [
                { key: 'method', value: { stringValue: method } },
                { key: 'status', value: { stringValue: status.toString() } }
              ],
              timeUnixNano: Date.now() * 1000000
            }],
            aggregationTemporality: 'AGGREGATION_TEMPORALITY_CUMULATIVE'
          }
        }]
      }]
    }]
  });
}

// Generate OTLP span payload
function generateSpanPayload() {
  const operationName = `operation_${Math.floor(Math.random() * 20)}`;
  
  return JSON.stringify({
    resourceSpans: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: `service-${Math.floor(Math.random() * 10)}` }
        }]
      },
      scopeSpans: [{
        spans: [{
          name: operationName,
          kind: Math.floor(Math.random() * 6),
          traceId: Array(32).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
          spanId: Array(16).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
          startTimeUnixNano: (Date.now() - 1000) * 1000000,
          endTimeUnixNano: Date.now() * 1000000,
          attributes: [
            { key: 'http.method', value: { stringValue: 'GET' } },
            { key: 'http.status_code', value: { intValue: 200 } }
          ]
        }]
      }]
    }]
  });
}

// Generate OTLP log payload
function generateLogPayload() {
  const severities = ['INFO', 'WARN', 'ERROR', 'DEBUG'];
  const severity = severities[Math.floor(Math.random() * severities.length)];
  const messages = [
    'User logged in successfully',
    'Request processed',
    'Database query executed',
    'Cache hit',
    'Configuration loaded'
  ];
  const message = messages[Math.floor(Math.random() * messages.length)];
  
  return JSON.stringify({
    resourceLogs: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: `service-${Math.floor(Math.random() * 10)}` }
        }]
      },
      scopeLogs: [{
        logRecords: [{
          timeUnixNano: Date.now() * 1000000,
          severityText: severity,
          severityNumber: { INFO: 9, WARN: 13, ERROR: 17, DEBUG: 5 }[severity],
          body: { stringValue: message },
          attributes: [
            { key: 'user.id', value: { stringValue: `user_${Math.floor(Math.random() * 100)}` } }
          ]
        }]
      }]
    }]
  });
}

// Write scenarios
export function writeMetrics() {
  const payload = generateMetricPayload();
  const startTime = Date.now();
  
  const res = http.post(`${BASE_URL}/v1/metrics`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - startTime;
  writeDuration.add(duration);
  
  const success = check(res, {
    'metric write status 200': (r) => r.status === 200,
  });
  
  if (success) {
    metricsWritten.add(1);
  }
}

export function writeSpans() {
  const payload = generateSpanPayload();
  const startTime = Date.now();
  
  const res = http.post(`${BASE_URL}/v1/traces`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - startTime;
  writeDuration.add(duration);
  
  const success = check(res, {
    'span write status 200': (r) => r.status === 200,
  });
  
  if (success) {
    spansWritten.add(1);
  }
}

export function writeLogs() {
  const payload = generateLogPayload();
  const startTime = Date.now();
  
  const res = http.post(`${BASE_URL}/v1/logs`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - startTime;
  writeDuration.add(duration);
  
  const success = check(res, {
    'log write status 200': (r) => r.status === 200,
  });
  
  if (success) {
    logsWritten.add(1);
  }
}

// Read scenarios
export function readMetrics() {
  const startTime = Date.now();
  
  const res = http.get(`${API_URL}/api/v1/metrics?limit=20`);
  
  const duration = Date.now() - startTime;
  readDuration.add(duration);
  
  const success = check(res, {
    'read metrics status 200': (r) => r.status === 200,
    'read metrics has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    apiReads.add(1);
  }
}

export function readSpans() {
  const startTime = Date.now();
  
  const res = http.get(`${API_URL}/api/v1/spans?limit=20`);
  
  const duration = Date.now() - startTime;
  readDuration.add(duration);
  
  const success = check(res, {
    'read spans status 200': (r) => r.status === 200,
    'read spans has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true && Array.isArray(body.data);
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    apiReads.add(1);
  }
}

export function readLogs() {
  const startTime = Date.now();
  
  // Read log patterns
  const severities = ['INFO', 'WARN', 'ERROR', 'DEBUG'];
  const severity = severities[Math.floor(Math.random() * severities.length)];
  const res = http.get(`${API_URL}/api/v1/logs/${severity}?limit=10`);
  
  const duration = Date.now() - startTime;
  readDuration.add(duration);
  
  const success = check(res, {
    'read logs status 200': (r) => r.status === 200,
    'read logs has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.success === true;
      } catch (e) {
        return false;
      }
    },
  });
  
  if (success) {
    apiReads.add(1);
  }
}

// Summary handler
export function handleSummary(data) {
  const metricsCount = data.metrics.metrics_written ? data.metrics.metrics_written.values.count : 0;
  const spansCount = data.metrics.spans_written ? data.metrics.spans_written.values.count : 0;
  const logsCount = data.metrics.logs_written ? data.metrics.logs_written.values.count : 0;
  const readsCount = data.metrics.api_reads ? data.metrics.api_reads.values.count : 0;
  
  const totalWrites = metricsCount + spansCount + logsCount;
  const totalOps = totalWrites + readsCount;
  const testDurationSec = data.state.testRunDurationMs / 1000;
  
  const writeP95 = data.metrics.write_duration ? data.metrics.write_duration.values['p(95)'] : 0;
  const readP95 = data.metrics.read_duration ? data.metrics.read_duration.values['p(95)'] : 0;
  
  const httpReqFailed = data.metrics.http_req_failed ? data.metrics.http_req_failed.values.rate : 0;
  const successRate = (1 - httpReqFailed) * 100;
  
  console.log(`
=== Mixed Load Test Summary ===
Test duration: ${testDurationSec.toFixed(1)}s

Write operations (70%):
  - Metrics: ${metricsCount} (${(metricsCount / testDurationSec).toFixed(0)}/sec)
  - Spans: ${spansCount} (${(spansCount / testDurationSec).toFixed(0)}/sec)
  - Logs: ${logsCount} (${(logsCount / testDurationSec).toFixed(0)}/sec)
  - Total writes: ${totalWrites} (${(totalWrites / testDurationSec).toFixed(0)}/sec)
  - Write p95 latency: ${writeP95.toFixed(0)}ms

Read operations (30%):
  - API reads: ${readsCount} (${(readsCount / testDurationSec).toFixed(0)}/sec)
  - Read p95 latency: ${readP95.toFixed(0)}ms

Overall:
  - Total operations: ${totalOps} (${(totalOps / testDurationSec).toFixed(0)}/sec)
  - Success rate: ${successRate.toFixed(2)}%
  - Write/Read ratio: ${(totalWrites / readsCount).toFixed(1)}:1
`);
  
  return {
    'k6-clickhouse-mixed-results.json': JSON.stringify({
      test_duration_sec: testDurationSec,
      writes: {
        metrics: metricsCount,
        spans: spansCount,
        logs: logsCount,
        total: totalWrites,
        throughput_per_sec: totalWrites / testDurationSec,
        p95_latency_ms: writeP95,
      },
      reads: {
        total: readsCount,
        throughput_per_sec: readsCount / testDurationSec,
        p95_latency_ms: readP95,
      },
      overall: {
        total_operations: totalOps,
        throughput_per_sec: totalOps / testDurationSec,
        success_rate: successRate / 100,
        write_read_ratio: totalWrites / readsCount,
      },
    }, null, 2),
    'stdout': '', // Don't duplicate summary
  };
}

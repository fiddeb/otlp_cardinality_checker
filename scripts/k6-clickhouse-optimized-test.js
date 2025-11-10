import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics to track throughput
const metricsWritten = new Counter('metrics_written');
const spansWritten = new Counter('spans_written');
const logsWritten = new Counter('logs_written');
const writeDuration = new Trend('write_duration');

// Test configuration - Optimized for 10,000+ signals/sec
export const options = {
  scenarios: {
    // Metrics: 60% of load
    metrics_optimized: {
      executor: 'ramping-vus',
      exec: 'sendMetric',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 100 },  // Quick warmup
        { duration: '30s', target: 400 },  // Ramp to 400 VUs
        { duration: '30s', target: 600 },  // Push to 600 VUs
        { duration: '20s', target: 600 },  // Hold at max
        { duration: '10s', target: 0 },    // Cool down
      ],
    },
    // Spans: 30% of load
    spans_optimized: {
      executor: 'ramping-vus',
      exec: 'sendSpan',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 50 },
        { duration: '30s', target: 200 },
        { duration: '30s', target: 300 },
        { duration: '20s', target: 300 },
        { duration: '10s', target: 0 },
      ],
    },
    // Logs: 10% of load
    logs_optimized: {
      executor: 'ramping-vus',
      exec: 'sendLog',
      startVUs: 0,
      stages: [
        { duration: '10s', target: 30 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 150 },
        { duration: '20s', target: 150 },
        { duration: '10s', target: 0 },
      ],
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.05'],    // <5% failure rate
    'write_duration': ['p(95)<1000'],     // p95 < 1s
    'http_req_duration': ['p(99)<2000'],  // p99 < 2s
  },
};

// Base URL for OTLP endpoints
const BASE_URL = 'http://localhost:4318';

// Generate random metric payload
function generateMetricPayload() {
  const metricId = Math.floor(Math.random() * 200); // 200 unique metrics
  const methodId = Math.floor(Math.random() * 10);  // 10 methods
  const statusId = Math.floor(Math.random() * 5);   // 5 status codes
  
  return JSON.stringify({
    resourceMetrics: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: `service_${Math.floor(Math.random() * 5)}` }},
          { key: 'host.name', value: { stringValue: `host_${Math.floor(Math.random() * 10)}` }},
        ],
      },
      scopeMetrics: [{
        metrics: [{
          name: `http_requests_metric_${metricId}`,
          unit: 'requests',
          gauge: {
            dataPoints: [{
              asInt: Math.floor(Math.random() * 1000),
              timeUnixNano: Date.now() * 1000000,
              attributes: [
                { key: 'method', value: { stringValue: `GET_${methodId}` }},
                { key: 'status', value: { intValue: 200 + statusId }},
              ],
            }],
          },
        }],
      }],
    }],
  });
}

// Generate random span payload
function generateSpanPayload() {
  const operationId = Math.floor(Math.random() * 100); // 100 unique operations
  const serviceId = Math.floor(Math.random() * 5);     // 5 services
  
  return JSON.stringify({
    resourceSpans: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: `span_service_${serviceId}` }},
        ],
      },
      scopeSpans: [{
        spans: [{
          traceId: '5b8aa5a2d2c872e8321cf37308d69df2',
          spanId: '051581bf3cb55c13',
          name: `operation_${operationId}`,
          kind: 2,
          startTimeUnixNano: Date.now() * 1000000,
          endTimeUnixNano: (Date.now() + 100) * 1000000,
          attributes: [
            { key: 'http.method', value: { stringValue: 'POST' }},
            { key: 'http.status_code', value: { intValue: 200 }},
          ],
        }],
      }],
    }],
  });
}

// Generate random log payload
function generateLogPayload() {
  const severities = ['INFO', 'WARN', 'ERROR', 'DEBUG'];
  const severity = severities[Math.floor(Math.random() * severities.length)];
  const serviceId = Math.floor(Math.random() * 5);
  const userId = Math.floor(Math.random() * 1000);
  
  return JSON.stringify({
    resourceLogs: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: `log_service_${serviceId}` }},
        ],
      },
      scopeLogs: [{
        logRecords: [{
          timeUnixNano: Date.now() * 1000000,
          severityText: severity,
          body: { stringValue: `User ${userId} performed action at ${new Date().toISOString()}` },
          attributes: [
            { key: 'user.id', value: { intValue: userId }},
            { key: 'action', value: { stringValue: 'login' }},
          ],
        }],
      }],
    }],
  });
}

// Send metric
export function sendMetric() {
  const start = Date.now();
  const payload = generateMetricPayload();
  
  const res = http.post(
    `${BASE_URL}/v1/metrics`,
    payload,
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: '10s',
    }
  );
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'metric write successful': (r) => r.status === 200,
  });
  
  if (success) {
    metricsWritten.add(1);
  }
}

// Send span
export function sendSpan() {
  const start = Date.now();
  const payload = generateSpanPayload();
  
  const res = http.post(
    `${BASE_URL}/v1/traces`,
    payload,
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: '10s',
    }
  );
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'span write successful': (r) => r.status === 200,
  });
  
  if (success) {
    spansWritten.add(1);
  }
}

// Send log
export function sendLog() {
  const start = Date.now();
  const payload = generateLogPayload();
  
  const res = http.post(
    `${BASE_URL}/v1/logs`,
    payload,
    {
      headers: { 'Content-Type': 'application/json' },
      timeout: '10s',
    }
  );
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'log write successful': (r) => r.status === 200,
  });
  
  if (success) {
    logsWritten.add(1);
  }
}

// Summary handler
export function handleSummary(data) {
  const metricsCount = data.metrics.metrics_written?.values?.count || 0;
  const spansCount = data.metrics.spans_written?.values?.count || 0;
  const logsCount = data.metrics.logs_written?.values?.count || 0;
  const totalSignals = metricsCount + spansCount + logsCount;
  
  // Test duration in seconds (100 seconds total)
  const testDuration = 100;
  const throughput = totalSignals / testDuration;
  
  const p95Latency = data.metrics.write_duration?.values['p(95)'] || 0;
  const successRate = 1 - (data.metrics.http_req_failed?.values?.rate || 0);
  
  console.log('\n=== Optimized Throughput Test Summary ===');
  console.log(`Total throughput: ${Math.round(throughput)} signals/sec`);
  console.log(`  - Metrics: ${Math.round(metricsCount/testDuration)}/sec`);
  console.log(`  - Spans: ${Math.round(spansCount/testDuration)}/sec`);
  console.log(`  - Logs: ${Math.round(logsCount/testDuration)}/sec`);
  console.log(`Write latency p95: ${Math.round(p95Latency)}ms`);
  console.log(`Success rate: ${(successRate * 100).toFixed(2)}%`);
  
  return {
    'stdout': JSON.stringify({
      total_requests: data.metrics.http_reqs?.values?.count || 0,
      total_signals: totalSignals,
      throughput_per_sec: throughput,
      p95_latency_ms: Math.round(p95Latency),
      success_rate: successRate,
      metrics_per_sec: Math.round(metricsCount/testDuration),
      spans_per_sec: Math.round(spansCount/testDuration),
      logs_per_sec: Math.round(logsCount/testDuration),
    }, null, 2),
    'k6-optimized-results.json': JSON.stringify(data, null, 2),
  };
}

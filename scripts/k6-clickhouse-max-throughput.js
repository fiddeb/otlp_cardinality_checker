import http from 'k6/http';
import { check } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const metricsWritten = new Counter('metrics_written');
const spansWritten = new Counter('spans_written');
const logsWritten = new Counter('logs_written');
const writeDuration = new Trend('write_duration', true);

// Test configuration - ramp up to find max throughput
export const options = {
  scenarios: {
    metrics_max: {
      executor: 'ramping-vus',
      exec: 'sendMetrics',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },   // Warm up
        { duration: '1m', target: 100 },   // Increase load
        { duration: '1m', target: 200 },   // Push harder
        { duration: '1m', target: 300 },   // Maximum load
        { duration: '30s', target: 0 },    // Cool down
      ],
      gracefulStop: '30s',
    },
    spans_max: {
      executor: 'ramping-vus',
      exec: 'sendSpans',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 25 },
        { duration: '1m', target: 50 },
        { duration: '1m', target: 100 },
        { duration: '1m', target: 150 },
        { duration: '30s', target: 0 },
      ],
      gracefulStop: '30s',
    },
    logs_max: {
      executor: 'ramping-vus',
      exec: 'sendLogs',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 15 },
        { duration: '1m', target: 30 },
        { duration: '1m', target: 60 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    'http_req_failed': ['rate<0.05'],        // Allow 5% failure at max load
    'write_duration': ['p(95)<1000'],        // p95 under 1 second at max
    'http_req_duration': ['p(95)<1000'],     // p95 request duration under 1s
  },
};

const BASE_URL = 'http://localhost:4318';

// Generate OTLP metric payload
function generateMetricPayload() {
  const metricName = `load_test_metric_${Math.floor(Math.random() * 100)}`;
  return JSON.stringify({
    resourceMetrics: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: 'load-test-svc' }
        }]
      },
      scopeMetrics: [{
        metrics: [{
          name: metricName,
          description: 'Load test metric',
          unit: 'ms',
          gauge: {
            dataPoints: [{
              asDouble: Math.random() * 1000,
              timeUnixNano: String(Date.now() * 1000000),
              attributes: [
                { key: 'method', value: { stringValue: 'GET' } },
                { key: 'status', value: { stringValue: '200' } }
              ]
            }]
          }
        }]
      }]
    }]
  });
}

// Generate OTLP span payload
function generateSpanPayload() {
  const spanName = `operation_${Math.floor(Math.random() * 50)}`;
  return JSON.stringify({
    resourceSpans: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: 'load-test-svc' }
        }]
      },
      scopeSpans: [{
        spans: [{
          traceId: Array.from({length: 32}, () => Math.floor(Math.random() * 16).toString(16)).join(''),
          spanId: Array.from({length: 16}, () => Math.floor(Math.random() * 16).toString(16)).join(''),
          name: spanName,
          kind: 2,
          startTimeUnixNano: String(Date.now() * 1000000),
          endTimeUnixNano: String((Date.now() + 100) * 1000000),
          attributes: [
            { key: 'http.method', value: { stringValue: 'POST' } },
            { key: 'http.url', value: { stringValue: '/api/endpoint' } }
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
  return JSON.stringify({
    resourceLogs: [{
      resource: {
        attributes: [{
          key: 'service.name',
          value: { stringValue: 'load-test-svc' }
        }]
      },
      scopeLogs: [{
        logRecords: [{
          timeUnixNano: String(Date.now() * 1000000),
          severityText: severity,
          severityNumber: 9,
          body: { stringValue: `Log message ${Math.random()} with some data` },
          attributes: [
            { key: 'log.level', value: { stringValue: severity } }
          ]
        }]
      }]
    }]
  });
}

export function sendMetrics() {
  const start = Date.now();
  const payload = generateMetricPayload();
  
  const res = http.post(`${BASE_URL}/v1/metrics`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'metrics: status is 200': (r) => r.status === 200,
  });
  
  if (success) {
    metricsWritten.add(1);
  }
}

export function sendSpans() {
  const start = Date.now();
  const payload = generateSpanPayload();
  
  const res = http.post(`${BASE_URL}/v1/traces`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'spans: status is 200': (r) => r.status === 200,
  });
  
  if (success) {
    spansWritten.add(1);
  }
}

export function sendLogs() {
  const start = Date.now();
  const payload = generateLogPayload();
  
  const res = http.post(`${BASE_URL}/v1/logs`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });
  
  const duration = Date.now() - start;
  writeDuration.add(duration);
  
  const success = check(res, {
    'logs: status is 200': (r) => r.status === 200,
  });
  
  if (success) {
    logsWritten.add(1);
  }
}

export function handleSummary(data) {
  const metricsRate = data.metrics.metrics_written.values.rate;
  const spansRate = data.metrics.spans_written.values.rate;
  const logsRate = data.metrics.logs_written.values.rate;
  const totalRate = metricsRate + spansRate + logsRate;
  
  console.log('\n=== Max Throughput Test Summary ===');
  console.log(`Total throughput: ${totalRate.toFixed(0)} signals/sec`);
  console.log(`  - Metrics: ${metricsRate.toFixed(0)}/sec`);
  console.log(`  - Spans: ${spansRate.toFixed(0)}/sec`);
  console.log(`  - Logs: ${logsRate.toFixed(0)}/sec`);
  console.log(`Write latency p95: ${data.metrics.write_duration.values['p(95)'].toFixed(0)}ms`);
  console.log(`Success rate: ${(data.metrics.checks.values.rate * 100).toFixed(2)}%`);
  
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'k6-max-throughput-results.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data, opts) {
  return JSON.stringify({
    total_requests: data.metrics.http_reqs.values.count,
    total_signals: data.metrics.metrics_written.values.count + 
                   data.metrics.spans_written.values.count + 
                   data.metrics.logs_written.values.count,
    throughput_per_sec: data.metrics.metrics_written.values.rate + 
                        data.metrics.spans_written.values.rate + 
                        data.metrics.logs_written.values.rate,
    p95_latency_ms: data.metrics.write_duration.values['p(95)'],
    success_rate: data.metrics.checks.values.rate,
  }, null, 2);
}

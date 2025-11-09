import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const metricsWritten = new Counter('metrics_written');
const spansWritten = new Counter('spans_written');
const logsWritten = new Counter('logs_written');
const writeErrors = new Counter('write_errors');
const writeSuccessRate = new Rate('write_success_rate');
const writeDuration = new Trend('write_duration');

// Test configuration
export const options = {
  scenarios: {
    metrics_load: {
      executor: 'constant-arrival-rate',
      rate: 100,        // 100 requests/sec
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 100,
      exec: 'sendMetrics',
    },
    spans_load: {
      executor: 'constant-arrival-rate',
      rate: 50,         // 50 requests/sec
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 25,
      maxVUs: 50,
      exec: 'sendSpans',
      startTime: '0s',
    },
    logs_load: {
      executor: 'constant-arrival-rate',
      rate: 30,         // 30 requests/sec
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 20,
      maxVUs: 40,
      exec: 'sendLogs',
      startTime: '0s',
    },
  },
  thresholds: {
    'write_success_rate': ['rate>0.99'],  // 99% success rate
    'write_duration': ['p(95)<500'],       // 95% under 500ms
    'http_req_failed': ['rate<0.01'],      // Less than 1% failures
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:4318';

// Generate random service names
const services = ['payment-svc', 'user-svc', 'order-svc', 'notification-svc', 'inventory-svc'];
const methods = ['GET', 'POST', 'PUT', 'DELETE'];
const statuses = ['200', '201', '400', '404', '500'];
const severities = ['DEBUG', 'INFO', 'WARN', 'ERROR'];

function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// Send metrics via OTLP HTTP
export function sendMetrics() {
  const service = randomChoice(services);
  const method = randomChoice(methods);
  const status = randomChoice(statuses);
  const timestamp = Date.now() * 1000000; // nanoseconds

  const payload = JSON.stringify({
    resourceMetrics: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: service } },
          { key: 'service.version', value: { stringValue: '1.0.0' } },
          { key: 'deployment.environment', value: { stringValue: 'load-test' } },
        ]
      },
      scopeMetrics: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0'
        },
        metrics: [{
          name: 'http_request_duration_ms',
          description: 'HTTP request duration',
          unit: 'ms',
          histogram: {
            dataPoints: [{
              count: 10,
              sum: randomInt(100, 5000),
              bucketCounts: [1, 2, 3, 2, 1, 1],
              explicitBounds: [100, 250, 500, 1000, 2500, 5000],
              attributes: [
                { key: 'http.method', value: { stringValue: method } },
                { key: 'http.status_code', value: { stringValue: status } },
                { key: 'http.route', value: { stringValue: '/api/v1/users' } },
              ],
              timeUnixNano: String(timestamp),
            }],
            aggregationTemporality: 2, // CUMULATIVE
          }
        }]
      }]
    }]
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/v1/metrics`, payload, params);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'metrics: status is 200': (r) => r.status === 200,
  });

  writeSuccessRate.add(success);
  writeDuration.add(duration);

  if (success) {
    metricsWritten.add(1);
  } else {
    writeErrors.add(1);
    console.error(`Metrics write failed: ${res.status} - ${res.body}`);
  }
}

// Send spans via OTLP HTTP
export function sendSpans() {
  const service = randomChoice(services);
  const method = randomChoice(methods);
  const timestamp = Date.now() * 1000000; // nanoseconds
  const duration = randomInt(10, 1000) * 1000000; // nanoseconds

  const payload = JSON.stringify({
    resourceSpans: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: service } },
          { key: 'deployment.environment', value: { stringValue: 'load-test' } },
        ]
      },
      scopeSpans: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0'
        },
        spans: [{
          traceId: Array(32).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
          spanId: Array(16).fill(0).map(() => Math.floor(Math.random() * 16).toString(16)).join(''),
          name: `${method} /api/v1/users`,
          kind: 2, // SERVER
          startTimeUnixNano: String(timestamp),
          endTimeUnixNano: String(timestamp + duration),
          attributes: [
            { key: 'http.method', value: { stringValue: method } },
            { key: 'http.url', value: { stringValue: '/api/v1/users' } },
            { key: 'http.status_code', value: { intValue: randomInt(200, 500) } },
          ],
        }]
      }]
    }]
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/v1/traces`, payload, params);
  const duration_ms = Date.now() - startTime;

  const success = check(res, {
    'spans: status is 200': (r) => r.status === 200,
  });

  writeSuccessRate.add(success);
  writeDuration.add(duration_ms);

  if (success) {
    spansWritten.add(1);
  } else {
    writeErrors.add(1);
    console.error(`Spans write failed: ${res.status}`);
  }
}

// Send logs via OTLP HTTP
export function sendLogs() {
  const service = randomChoice(services);
  const severity = randomChoice(severities);
  const timestamp = Date.now() * 1000000; // nanoseconds

  const logMessages = [
    'User authentication successful',
    'Database query executed in 45ms',
    'Cache miss for key: user_profile_123',
    'API request received from client',
    'Payment processed successfully',
  ];

  const payload = JSON.stringify({
    resourceLogs: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: service } },
          { key: 'deployment.environment', value: { stringValue: 'load-test' } },
        ]
      },
      scopeLogs: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0'
        },
        logRecords: [{
          timeUnixNano: String(timestamp),
          severityText: severity,
          severityNumber: severities.indexOf(severity) * 4 + 1,
          body: { stringValue: randomChoice(logMessages) },
          attributes: [
            { key: 'log.level', value: { stringValue: severity } },
            { key: 'user.id', value: { stringValue: `user_${randomInt(1, 1000)}` } },
          ],
        }]
      }]
    }]
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    timeout: '10s',
  };

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/v1/logs`, payload, params);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'logs: status is 200': (r) => r.status === 200,
  });

  writeSuccessRate.add(success);
  writeDuration.add(duration);

  if (success) {
    logsWritten.add(1);
  } else {
    writeErrors.add(1);
    console.error(`Logs write failed: ${res.status}`);
  }
}

export function handleSummary(data) {
  return {
    'stdout': JSON.stringify(data, null, 2),
    'k6-clickhouse-write-results.json': JSON.stringify(data),
  };
}

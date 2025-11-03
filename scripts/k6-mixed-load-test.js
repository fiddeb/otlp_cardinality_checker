import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const otlpRequests = new Counter('otlp_requests');
const otlpErrors = new Counter('otlp_errors');
const otlpDuration = new Trend('otlp_duration');

// Load test stages - gradual ramp up
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm up: 10 VUs
    { duration: '1m', target: 50 },    // Ramp up to 50 VUs
    { duration: '2m', target: 100 },   // Ramp up to 100 VUs
    { duration: '3m', target: 100 },   // Stay at 100 VUs
    { duration: '1m', target: 200 },   // Spike to 200 VUs
    { duration: '2m', target: 200 },   // Sustain spike
    { duration: '1m', target: 50 },    // Ramp down
    { duration: '30s', target: 0 },    // Cool down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'http_req_failed': ['rate<0.05'],   // Less than 5% errors
  },
};

const BASE_URL = 'http://localhost:4318';

// Service names - mix of low and high cardinality
const services = [
  'web-frontend',
  'api-gateway', 
  'user-service',
  'payment-service',
  'inventory-service',
];

// Generate high cardinality attributes
function generateHighCardinalityAttributes() {
  const userId = `user_${Math.floor(Math.random() * 10000)}`; // 10k unique users
  const sessionId = `session_${Math.floor(Math.random() * 50000)}`; // 50k unique sessions
  const requestId = `req_${Date.now()}_${Math.random().toString(36).substring(7)}`;
  
  return {
    'user.id': userId,
    'session.id': sessionId,
    'request.id': requestId,
  };
}

// Generate low cardinality attributes
function generateLowCardinalityAttributes() {
  const methods = ['GET', 'POST', 'PUT', 'DELETE'];
  const statusCodes = ['200', '201', '400', '404', '500'];
  const environments = ['production', 'staging', 'development'];
  
  return {
    'http.method': methods[Math.floor(Math.random() * methods.length)],
    'http.status_code': statusCodes[Math.floor(Math.random() * statusCodes.length)],
    'environment': environments[Math.floor(Math.random() * environments.length)],
  };
}

// Mix low and high cardinality
function generateMixedAttributes() {
  return {
    ...generateLowCardinalityAttributes(),
    ...generateHighCardinalityAttributes(),
  };
}

// Send OTLP Metrics
function sendMetrics(serviceName, cardinalityType) {
  const timeNow = Date.now() * 1000000; // nanoseconds
  
  const attributes = cardinalityType === 'high' 
    ? generateMixedAttributes() 
    : generateLowCardinalityAttributes();

  const payload = JSON.stringify({
    resourceMetrics: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: serviceName } },
          { key: 'host.name', value: { stringValue: `host_${Math.floor(Math.random() * 10)}` } },
        ],
      },
      scopeMetrics: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0',
        },
        metrics: [
          // Counter - request count
          {
            name: 'http.server.requests',
            description: 'Total HTTP requests',
            unit: '1',
            sum: {
              dataPoints: [{
                attributes: Object.entries(attributes).map(([key, value]) => ({
                  key,
                  value: { stringValue: value },
                })),
                startTimeUnixNano: timeNow - 60000000000,
                timeUnixNano: timeNow,
                asInt: Math.floor(Math.random() * 1000),
              }],
              aggregationTemporality: 2, // CUMULATIVE
              isMonotonic: true,
            },
          },
          // Gauge - current connections
          {
            name: 'http.server.active_connections',
            description: 'Active HTTP connections',
            unit: '1',
            gauge: {
              dataPoints: [{
                attributes: Object.entries(attributes).map(([key, value]) => ({
                  key,
                  value: { stringValue: value },
                })),
                timeUnixNano: timeNow,
                asInt: Math.floor(Math.random() * 100),
              }],
            },
          },
          // Histogram - request duration
          {
            name: 'http.server.duration',
            description: 'HTTP request duration',
            unit: 'ms',
            histogram: {
              dataPoints: [{
                attributes: Object.entries(attributes).map(([key, value]) => ({
                  key,
                  value: { stringValue: value },
                })),
                startTimeUnixNano: timeNow - 60000000000,
                timeUnixNano: timeNow,
                count: Math.floor(Math.random() * 1000),
                sum: Math.random() * 10000,
                bucketCounts: [10, 50, 200, 400, 300, 40, 10],
                explicitBounds: [0.1, 0.5, 1, 5, 10, 50],
              }],
              aggregationTemporality: 2,
            },
          },
        ],
      }],
    }],
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/metrics`, payload, params);
  const duration = Date.now() - start;

  otlpRequests.add(1);
  otlpDuration.add(duration);

  const success = check(res, {
    'metrics: status is 200': (r) => r.status === 200,
  });

  if (!success) {
    otlpErrors.add(1);
  }
}

// Send OTLP Traces
function sendTraces(serviceName, cardinalityType) {
  const timeNow = Date.now() * 1000000;
  const traceId = Array.from({ length: 16 }, () => 
    Math.floor(Math.random() * 256).toString(16).padStart(2, '0')
  ).join('');
  const spanId = Array.from({ length: 8 }, () => 
    Math.floor(Math.random() * 256).toString(16).padStart(2, '0')
  ).join('');

  const attributes = cardinalityType === 'high' 
    ? generateMixedAttributes() 
    : generateLowCardinalityAttributes();

  const payload = JSON.stringify({
    resourceSpans: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: serviceName } },
          { key: 'host.name', value: { stringValue: `host_${Math.floor(Math.random() * 10)}` } },
        ],
      },
      scopeSpans: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0',
        },
        spans: [{
          traceId,
          spanId,
          name: 'HTTP ' + attributes['http.method'] + ' /api/endpoint',
          kind: 3, // CLIENT
          startTimeUnixNano: timeNow - 100000000,
          endTimeUnixNano: timeNow,
          attributes: Object.entries(attributes).map(([key, value]) => ({
            key,
            value: { stringValue: value },
          })),
          status: {
            code: attributes['http.status_code'] === '200' ? 1 : 2, // OK : ERROR
          },
        }],
      }],
    }],
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/traces`, payload, params);
  const duration = Date.now() - start;

  otlpRequests.add(1);
  otlpDuration.add(duration);

  const success = check(res, {
    'traces: status is 200': (r) => r.status === 200,
  });

  if (!success) {
    otlpErrors.add(1);
  }
}

// Send OTLP Logs
function sendLogs(serviceName, cardinalityType) {
  const timeNow = Date.now() * 1000000;

  const attributes = cardinalityType === 'high' 
    ? generateMixedAttributes() 
    : generateLowCardinalityAttributes();

  const severities = [
    { name: 'INFO', number: 9 },
    { name: 'WARN', number: 13 },
    { name: 'ERROR', number: 17 },
    { name: 'DEBUG', number: 5 },
  ];
  const severity = severities[Math.floor(Math.random() * severities.length)];

  const logMessages = [
    `Request processed successfully with method ${attributes['http.method']} and status ${attributes['http.status_code']}`,
    `User authentication completed for ${attributes['user.id'] || 'unknown'}`,
    `Database query executed in ${Math.floor(Math.random() * 100)}ms`,
    `Cache hit for key ${Math.random().toString(36).substring(7)}`,
    `External API call to service completed`,
  ];

  const payload = JSON.stringify({
    resourceLogs: [{
      resource: {
        attributes: [
          { key: 'service.name', value: { stringValue: serviceName } },
          { key: 'host.name', value: { stringValue: `host_${Math.floor(Math.random() * 10)}` } },
        ],
      },
      scopeLogs: [{
        scope: {
          name: 'k6-load-test',
          version: '1.0.0',
        },
        logRecords: [{
          timeUnixNano: timeNow,
          severityNumber: severity.number,
          severityText: severity.name,
          body: {
            stringValue: logMessages[Math.floor(Math.random() * logMessages.length)],
          },
          attributes: Object.entries(attributes).map(([key, value]) => ({
            key,
            value: { stringValue: value },
          })),
        }],
      }],
    }],
  });

  const params = {
    headers: { 'Content-Type': 'application/json' },
  };

  const start = Date.now();
  const res = http.post(`${BASE_URL}/v1/logs`, payload, params);
  const duration = Date.now() - start;

  otlpRequests.add(1);
  otlpDuration.add(duration);

  const success = check(res, {
    'logs: status is 200': (r) => r.status === 200,
  });

  if (!success) {
    otlpErrors.add(1);
  }
}

// Main test function
export default function () {
  const service = services[Math.floor(Math.random() * services.length)];
  
  // 70% low cardinality, 30% high cardinality
  const cardinalityType = Math.random() < 0.7 ? 'low' : 'high';
  
  // Random mix of signal types
  const signalType = Math.random();
  
  if (signalType < 0.4) {
    // 40% metrics
    sendMetrics(service, cardinalityType);
  } else if (signalType < 0.7) {
    // 30% logs
    sendLogs(service, cardinalityType);
  } else {
    // 30% traces
    sendTraces(service, cardinalityType);
  }

  // Small sleep to avoid hammering too hard
  sleep(0.1);
}

// Summary at the end
export function handleSummary(data) {
  const requests = data.metrics.otlp_requests?.values?.count || 0;
  const errors = data.metrics.otlp_errors?.values?.count || 0;
  const duration = data.state.testRunDurationMs / 1000;
  
  return {
    'stdout': JSON.stringify({
      duration: duration,
      requests: requests,
      errors: errors,
      errorRate: requests > 0 ? ((errors / requests * 100).toFixed(2) + '%') : '0.00%',
      avgDuration: (data.metrics.otlp_duration?.values?.avg || 0).toFixed(2) + 'ms',
      p95Duration: (data.metrics.otlp_duration?.values['p(95)'] || 0).toFixed(2) + 'ms',
      p99Duration: (data.metrics.otlp_duration?.values['p(99)'] || 0).toFixed(2) + 'ms',
      requestsPerSecond: duration > 0 ? (requests / duration).toFixed(2) : '0.00',
    }, null, 2),
  };
}

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const readsSuccess = new Counter('reads_success');
const readErrors = new Counter('read_errors');
const readSuccessRate = new Rate('read_success_rate');
const readDuration = new Trend('read_duration');

// Test configuration
export const options = {
  scenarios: {
    read_load: {
      executor: 'constant-arrival-rate',
      rate: 50,        // 50 requests/sec
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 25,
      maxVUs: 50,
    },
  },
  thresholds: {
    'read_success_rate': ['rate>0.95'],   // 95% success rate
    'read_duration': ['p(95)<200'],       // 95% under 200ms
    'http_req_failed': ['rate<0.05'],     // Less than 5% failures
  },
};

const API_URL = __ENV.API_URL || 'http://localhost:8080';

// List of endpoints to test
const endpoints = [
  '/api/v1/metrics',
  '/api/v1/metrics?limit=10',
  '/api/v1/metrics?service=payment-svc',
  '/api/v1/spans',
  '/api/v1/spans?limit=10',
  '/api/v1/logs',
  '/api/v1/logs?limit=10',
  '/api/v1/services',
  '/api/v1/summary',
  '/health',
];

function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

export default function() {
  const endpoint = randomChoice(endpoints);
  
  const params = {
    headers: {
      'Accept': 'application/json',
    },
    timeout: '5s',
  };

  const startTime = Date.now();
  const res = http.get(`${API_URL}${endpoint}`, params);
  const duration = Date.now() - startTime;

  const success = check(res, {
    'read: status is 200': (r) => r.status === 200,
    'read: has json response': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch (e) {
        return false;
      }
    },
    'read: response time < 500ms': (r) => duration < 500,
  });

  readSuccessRate.add(success);
  readDuration.add(duration);

  if (success) {
    readsSuccess.add(1);
  } else {
    readErrors.add(1);
    console.error(`Read failed for ${endpoint}: ${res.status}`);
  }

  sleep(0.1); // Small delay between requests
}

export function handleSummary(data) {
  console.log('\n=== Load Test Summary ===\n');
  console.log(`Total Requests: ${data.metrics.http_reqs.values.count}`);
  console.log(`Success Rate: ${(data.metrics.read_success_rate.values.rate * 100).toFixed(2)}%`);
  console.log(`Read Duration p95: ${data.metrics.read_duration.values['p(95)']}ms`);
  console.log(`Read Duration avg: ${data.metrics.read_duration.values.avg}ms`);
  
  return {
    'stdout': JSON.stringify(data, null, 2),
    'k6-clickhouse-read-results.json': JSON.stringify(data),
  };
}

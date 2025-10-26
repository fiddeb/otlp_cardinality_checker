// K6 Load Test for OTLP Cardinality Checker
// Install k6: brew install k6
// Run: k6 run scripts/load-test-metrics.js
// Run with options: k6 run --vus 10 --duration 60s scripts/load-test-metrics.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const metricsCreated = new Counter('metrics_created');
const memoryUsage = new Trend('memory_usage_mb');

// Configuration
export const options = {
    // Scenarios for different load patterns
    scenarios: {
        // Steady load test
        steady_load: {
            executor: 'constant-vus',
            vus: 10,
            duration: '60s',
        },
        // Ramp up test (commented out - uncomment to use)
        // ramp_up: {
        //     executor: 'ramping-vus',
        //     startVUs: 0,
        //     stages: [
        //         { duration: '30s', target: 10 },
        //         { duration: '60s', target: 50 },
        //         { duration: '30s', target: 0 },
        //     ],
        // },
    },
    thresholds: {
        http_req_failed: ['rate<0.01'], // <1% errors
        http_req_duration: ['p(95)<500'], // 95% under 500ms
    },
};

// Test configuration
const OTLP_ENDPOINT = __ENV.OTLP_ENDPOINT || 'http://localhost:4318';
const API_ENDPOINT = __ENV.API_ENDPOINT || 'http://localhost:8080';
const NUM_METRICS = parseInt(__ENV.NUM_METRICS || '1000');
const CARDINALITY = parseInt(__ENV.CARDINALITY || '50');

// Generate random metric data
function generateMetricBatch(vu, iter) {
    const timestamp = Date.now() * 1000000; // nanoseconds
    const serviceNum = (vu % 10);
    const serviceName = `service-${serviceNum}`;
    
    const metrics = [];
    const batchSize = 10;
    
    for (let i = 0; i < batchSize; i++) {
        // Hybrid approach: sequential base + small random offset
        // This ensures better coverage while maintaining some randomness
        const baseMetric = (iter * batchSize + i);
        const randomOffset = Math.floor(Math.random() * 100);
        const metricNum = (baseMetric + randomOffset) % NUM_METRICS;
        const labelValue = Math.floor(Math.random() * CARDINALITY);
        
        metrics.push({
            name: `test_metric_${metricNum}`,
            sum: {
                aggregation_temporality: 2,
                is_monotonic: true,
                data_points: [{
                    attributes: [
                        { key: 'label1', value: { string_value: `value_${labelValue}` } },
                        { key: 'label2', value: { string_value: `value_${Math.floor(Math.random() * 10)}` } },
                        { key: 'method', value: { string_value: 'GET' } },
                        { key: 'endpoint', value: { string_value: `/api/v${Math.floor(Math.random() * 3) + 1}` } },
                    ],
                    as_int: Math.floor(Math.random() * 1000),
                    time_unix_nano: timestamp,
                }],
            },
        });
    }
    
    return {
        resource_metrics: [{
            resource: {
                attributes: [
                    { key: 'service.name', value: { string_value: serviceName } },
                    { key: 'host.name', value: { string_value: `host_${vu % 5}` } },
                ],
            },
            scope_metrics: [{
                scope: {
                    name: 'k6-load-test',
                    version: '1.0.0',
                },
                metrics: metrics,
            }],
        }],
    };
}

// Setup function - runs once before test
export function setup() {
    console.log('='.repeat(50));
    console.log('K6 Load Test for OTLP Cardinality Checker (Write-Only Mode)');
    console.log('='.repeat(50));
    console.log(`OTLP Endpoint: ${OTLP_ENDPOINT}`);
    console.log(`Metrics:       ${NUM_METRICS} unique metrics`);
    console.log(`Cardinality:   ${CARDINALITY} values per label`);
    console.log('='.repeat(50));
}

// Main test function - runs for each VU iteration
export default function() {
    const vu = __VU;
    const iter = __ITER;
    
    // Send metric batch
    const payload = JSON.stringify(generateMetricBatch(vu, iter));
    const params = {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'SendMetrics' },
    };
    
    const response = http.post(`${OTLP_ENDPOINT}/v1/metrics`, payload, params);
    
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    metricsCreated.add(10); // 10 metrics per batch
    
    // Small sleep to avoid overwhelming the server
    sleep(0.1);
}

// Teardown function - runs once after test
export function teardown() {
    console.log('\n' + '='.repeat(50));
    console.log('Test Complete');
    console.log('='.repeat(50));
    console.log('Query the API manually to see results:');
    console.log(`  curl ${API_ENDPOINT}/api/v1/metrics`);
    console.log(`  curl ${API_ENDPOINT}/api/v1/services`);
    console.log('='.repeat(50));
}

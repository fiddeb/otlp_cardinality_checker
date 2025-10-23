// K6 Stress Test - Test with increasing load
// Run: k6 run scripts/stress-test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');

export const options = {
    stages: [
        { duration: '30s', target: 10 },   // Ramp up to 10 VUs
        { duration: '1m', target: 50 },    // Ramp up to 50 VUs
        { duration: '1m', target: 100 },   // Ramp up to 100 VUs
        { duration: '30s', target: 0 },    // Ramp down
    ],
    thresholds: {
        errors: ['rate<0.05'],              // <5% errors
        http_req_duration: ['p(95)<1000'],  // 95% under 1s
        api_latency: ['p(99)<500'],         // 99% API calls under 500ms
    },
};

const OTLP_ENDPOINT = __ENV.OTLP_ENDPOINT || 'http://localhost:4218';
const API_ENDPOINT = __ENV.API_ENDPOINT || 'http://localhost:8080';

export default function() {
    // Send metrics
    const payload = JSON.stringify({
        resource_metrics: [{
            resource: {
                attributes: [
                    { key: 'service.name', value: { string_value: `stress_test_${__VU}` } },
                ],
            },
            scope_metrics: [{
                metrics: [{
                    name: `stress_metric_${Math.floor(Math.random() * 5000)}`,
                    sum: {
                        data_points: [{
                            attributes: [
                                { key: 'high_card', value: { string_value: `value_${__ITER}` } },
                                { key: 'label', value: { string_value: 'test' } },
                            ],
                            as_int: 1,
                            time_unix_nano: Date.now() * 1000000,
                        }],
                    },
                }],
            }],
        }],
    });
    
    const response = http.post(`${OTLP_ENDPOINT}/v1/metrics`, payload, {
        headers: { 'Content-Type': 'application/json' },
    });
    
    errorRate.add(response.status !== 200);
    
    // Check API every 50 iterations
    if (__ITER % 50 === 0) {
        const start = Date.now();
        const apiResponse = http.get(`${API_ENDPOINT}/api/v1/metrics?limit=10`);
        apiLatency.add(Date.now() - start);
        
        check(apiResponse, {
            'API still responsive': (r) => r.status === 200,
        });
    }
    
    sleep(0.05);
}

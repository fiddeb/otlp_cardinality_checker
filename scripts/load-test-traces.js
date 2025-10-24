// K6 Load Test for OTLP Traces
// Install k6: brew install k6
// Run: k6 run scripts/load-test-traces.js
// Run with options: k6 run --vus 10 --duration 60s scripts/load-test-traces.js
// Custom config: k6 run --vus 50 --duration 120s -e NUM_SPANS=10000 -e CARDINALITY=100 scripts/load-test-traces.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const tracesCreated = new Counter('traces_created');

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
    },
    thresholds: {
        http_req_failed: ['rate<0.01'], // <1% errors
        http_req_duration: ['p(95)<500'], // 95% under 500ms
    },
};

// Test configuration
const OTLP_ENDPOINT = __ENV.OTLP_ENDPOINT || 'http://localhost:4318';
const API_ENDPOINT = __ENV.API_ENDPOINT || 'http://localhost:8080';
const NUM_SPANS = parseInt(__ENV.NUM_SPANS || '100');
const CARDINALITY = parseInt(__ENV.CARDINALITY || '50');

// Generate random trace data
function generateTraceBatch(vu, iter) {
    const timestamp = Date.now() * 1000000; // nanoseconds
    const serviceNum = (vu % 10);
    const serviceName = `trace-service-${serviceNum}`;
    
    const spans = [];
    const batchSize = 10;
    
    for (let i = 0; i < batchSize; i++) {
        // Hybrid approach: sequential base + small random offset
        const baseSpan = (iter * batchSize + i);
        const randomOffset = Math.floor(Math.random() * 100);
        const spanNum = (baseSpan + randomOffset) % NUM_SPANS;
        const userValue = Math.floor(Math.random() * CARDINALITY);
        
        // Generate trace and span IDs
        const traceId = spanNum.toString(16).padStart(32, '0');
        const spanId = (spanNum + i).toString(16).padStart(16, '0');
        
        spans.push({
            trace_id: traceId,
            span_id: spanId,
            name: `span_operation_${spanNum % 50}`,
            kind: (spanNum % 5) + 1, // 1=Internal, 2=Server, 3=Client, 4=Producer, 5=Consumer
            start_time_unix_nano: timestamp,
            end_time_unix_nano: timestamp + 1000000000, // +1 second
            attributes: [
                { key: 'http.method', value: { string_value: ['GET', 'POST', 'PUT', 'DELETE'][spanNum % 4] } },
                { key: 'http.url', value: { string_value: `/api/v${Math.floor(Math.random() * 3) + 1}/resource` } },
                { key: 'http.status_code', value: { int_value: [200, 201, 204, 400, 404, 500][spanNum % 6] } },
                { key: 'user_id', value: { string_value: `user_${userValue}` } },
            ],
        });
    }
    
    return {
        resource_spans: [{
            resource: {
                attributes: [
                    { key: 'service.name', value: { string_value: serviceName } },
                    { key: 'host.name', value: { string_value: `host_${vu % 5}` } },
                    { key: 'deployment.environment', value: { string_value: ['production', 'staging', 'development'][vu % 3] } },
                ],
            },
            scope_spans: [{
                scope: {
                    name: 'k6-trace-test',
                    version: '1.0.0',
                },
                spans: spans,
            }],
        }],
    };
}

// Setup function - runs once before test
export function setup() {
    console.log('='.repeat(50));
    console.log('K6 Load Test for OTLP Traces');
    console.log('='.repeat(50));
    console.log(`OTLP Endpoint: ${OTLP_ENDPOINT}`);
    console.log(`API Endpoint:  ${API_ENDPOINT}`);
    console.log(`Span Names:    ${NUM_SPANS} unique operations`);
    console.log(`Cardinality:   ${CARDINALITY} values per attribute`);
    console.log('='.repeat(50));
    
    // Get baseline spans
    const baseline = http.get(`${API_ENDPOINT}/api/v1/spans`);
    if (baseline.status === 200) {
        const data = JSON.parse(baseline.body);
        console.log(`Baseline spans: ${data.total}`);
        return { baselineSpans: data.total };
    }
    return { baselineSpans: 0 };
}

// Main test function - runs for each VU iteration
export default function(data) {
    const vu = __VU;
    const iter = __ITER;
    
    // Send trace batch
    const payload = JSON.stringify(generateTraceBatch(vu, iter));
    const params = {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'SendTraces' },
    };
    
    const response = http.post(`${OTLP_ENDPOINT}/v1/traces`, payload, params);
    
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    tracesCreated.add(10); // 10 spans per batch
    
    // Every 100 iterations, check API responsiveness
    if (iter % 100 === 0) {
        const apiCheck = http.get(`${API_ENDPOINT}/api/v1/spans?limit=10`, {
            tags: { name: 'CheckAPI' },
        });
        
        check(apiCheck, {
            'API responsive': (r) => r.status === 200,
        });
    }
    
    sleep(0.1); // 100ms pause between batches
}

// Teardown function - runs once after test
export function teardown(data) {
    console.log('\n' + '='.repeat(50));
    console.log('Test Complete - Collecting Statistics');
    console.log('='.repeat(50));
    
    // Get final stats
    const final = http.get(`${API_ENDPOINT}/api/v1/spans`);
    if (final.status === 200) {
        const stats = JSON.parse(final.body);
        const newSpans = stats.total - data.baselineSpans;
        
        console.log(`Final span names: ${stats.total}`);
        console.log(`New span names created: ${newSpans}`);
        
        // Get service stats
        const services = http.get(`${API_ENDPOINT}/api/v1/services`);
        if (services.status === 200) {
            const serviceData = JSON.parse(services.body);
            console.log(`Services tracked: ${serviceData.total || 'undefined'}`);
        }
        
        // Check for high cardinality attributes
        if (stats.data && stats.data.length > 0) {
            console.log('='.repeat(50));
            let highCardCount = 0;
            for (const span of stats.data.slice(0, 100)) { // Check first 100 spans
                for (const [key, meta] of Object.entries(span.attribute_keys)) {
                    if (meta.estimated_cardinality > 40) {
                        console.log(`⚠️  High cardinality: ${span.name}.${key} = ${meta.estimated_cardinality}`);
                        highCardCount++;
                        if (highCardCount >= 5) break; // Show max 5 examples
                    }
                }
                if (highCardCount >= 5) break;
            }
            if (highCardCount > 0) {
                console.log(`Total high cardinality attributes: ${highCardCount}`);
            }
        }
    }
    
    console.log('='.repeat(50));
}

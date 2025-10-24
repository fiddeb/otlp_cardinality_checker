// K6 Load Test for OTLP Logs
// Install k6: brew install k6
// Run: k6 run scripts/load-test-logs.js
// Run with options: k6 run --vus 10 --duration 60s scripts/load-test-logs.js
// Custom config: k6 run --vus 50 --duration 120s -e NUM_MODULES=1000 -e CARDINALITY=100 scripts/load-test-logs.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Custom metrics
const logsCreated = new Counter('logs_created');

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
const NUM_MODULES = parseInt(__ENV.NUM_MODULES || '100');
const CARDINALITY = parseInt(__ENV.CARDINALITY || '50');

// Severity levels mapping
const SEVERITIES = [
    { number: 9, text: 'INFO' },
    { number: 13, text: 'WARN' },
    { number: 17, text: 'ERROR' },
    { number: 5, text: 'DEBUG' },
];

// Generate random log data
function generateLogBatch(vu, iter) {
    const timestamp = Date.now() * 1000000; // nanoseconds
    const serviceNum = (vu % 10);
    const serviceName = `log-service-${serviceNum}`;
    
    const logRecords = [];
    const batchSize = 10;
    
    for (let i = 0; i < batchSize; i++) {
        // Hybrid approach: sequential base + small random offset
        const baseLog = (iter * batchSize + i);
        const randomOffset = Math.floor(Math.random() * 100);
        const logNum = (baseLog + randomOffset) % NUM_MODULES;
        const traceValue = Math.floor(Math.random() * CARDINALITY);
        
        const severity = SEVERITIES[logNum % 4];
        
        logRecords.push({
            time_unix_nano: timestamp + (i * 1000000), // Spread logs over time
            severity_number: severity.number,
            severity_text: severity.text,
            body: { string_value: `Log message from ${serviceName} - event ${logNum}` },
            attributes: [
                { key: 'log.level', value: { string_value: severity.text.toLowerCase() } },
                { key: 'module', value: { string_value: `module_${logNum % 20}` } },
                { key: 'trace_id', value: { string_value: `trace_${traceValue}` } },
                { key: 'user_id', value: { string_value: `user_${Math.floor(Math.random() * CARDINALITY)}` } },
                { key: 'request_id', value: { string_value: `req_${logNum}` } },
            ],
        });
    }
    
    return {
        resource_logs: [{
            resource: {
                attributes: [
                    { key: 'service.name', value: { string_value: serviceName } },
                    { key: 'host.name', value: { string_value: `host_${vu % 5}` } },
                    { key: 'deployment.environment', value: { string_value: ['production', 'staging', 'development'][vu % 3] } },
                ],
            },
            scope_logs: [{
                scope: {
                    name: 'k6-log-test',
                    version: '1.0.0',
                },
                log_records: logRecords,
            }],
        }],
    };
}

// Setup function - runs once before test
export function setup() {
    console.log('='.repeat(50));
    console.log('K6 Load Test for OTLP Logs');
    console.log('='.repeat(50));
    console.log(`OTLP Endpoint: ${OTLP_ENDPOINT}`);
    console.log(`API Endpoint:  ${API_ENDPOINT}`);
    console.log(`Modules:       ${NUM_MODULES} unique modules`);
    console.log(`Cardinality:   ${CARDINALITY} values per attribute`);
    console.log('='.repeat(50));
    
    // Get baseline logs
    const baseline = http.get(`${API_ENDPOINT}/api/v1/logs`);
    if (baseline.status === 200) {
        const data = JSON.parse(baseline.body);
        console.log(`Baseline log severities: ${data.total}`);
        return { baselineLogs: data.total };
    }
    return { baselineLogs: 0 };
}

// Main test function - runs for each VU iteration
export default function(data) {
    const vu = __VU;
    const iter = __ITER;
    
    // Send log batch
    const payload = JSON.stringify(generateLogBatch(vu, iter));
    const params = {
        headers: { 'Content-Type': 'application/json' },
        tags: { name: 'SendLogs' },
    };
    
    const response = http.post(`${OTLP_ENDPOINT}/v1/logs`, payload, params);
    
    check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });
    
    logsCreated.add(10); // 10 log records per batch
    
    // Every 100 iterations, check API responsiveness
    if (iter % 100 === 0) {
        const apiCheck = http.get(`${API_ENDPOINT}/api/v1/logs?limit=10`, {
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
    const final = http.get(`${API_ENDPOINT}/api/v1/logs`);
    if (final.status === 200) {
        const stats = JSON.parse(final.body);
        const newLogs = stats.total - data.baselineLogs;
        
        console.log(`Final log severities: ${stats.total}`);
        console.log(`New severities created: ${newLogs}`);
        
        // Show severity breakdown
        if (stats.data && stats.data.length > 0) {
            console.log('\nSeverity Breakdown:');
            for (const log of stats.data) {
                console.log(`  ${log.severity_text}: ${log.record_count} records`);
            }
        }
        
        // Get service stats
        const services = http.get(`${API_ENDPOINT}/api/v1/services`);
        if (services.status === 200) {
            const serviceData = JSON.parse(services.body);
            console.log(`\nServices tracked: ${serviceData.total || 'undefined'}`);
        }
        
        // Check for high cardinality attributes
        if (stats.data && stats.data.length > 0) {
            console.log('='.repeat(50));
            let highCardCount = 0;
            for (const log of stats.data) {
                for (const [key, meta] of Object.entries(log.attribute_keys)) {
                    if (meta.estimated_cardinality > 40) {
                        console.log(`⚠️  High cardinality: ${log.severity_text}.${key} = ${meta.estimated_cardinality}`);
                        highCardCount++;
                        if (highCardCount >= 5) break; // Show max 5 examples
                    }
                }
                if (highCardCount >= 5) break;
            }
            if (highCardCount > 0) {
                console.log(`Total high cardinality attributes: ${highCardCount}+`);
            }
        }
    }
    
    console.log('='.repeat(50));
}

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
    const serviceName = `service-${serviceNum}`;
    
    const logRecords = [];
    const batchSize = 10;
    
    // Different log message templates to create varied patterns
    // Use predictable values that will be replaced by template extraction regex
    const logTemplates = [
        (logNum, serviceName) => `Log message from ${serviceName} - event ${logNum}`,
        (logNum) => `User ${logNum} logged in from 192.168.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
        (logNum) => `Processing request ${logNum} took ${Math.floor(Math.random() * 1000)}ms`,
        (logNum) => `Database query executed in ${Math.floor(Math.random() * 500)}ms - affected ${Math.floor(Math.random() * 100)} rows`,
        (logNum) => `Cache hit for key cache_key_${logNum} - returned ${Math.floor(Math.random() * 10000)}B`,
        (logNum) => `HTTP ${['GET', 'POST', 'PUT', 'DELETE'][Math.floor(Math.random() * 4)]} /api/v1/resource/${logNum} - status ${[200, 201, 404, 500][Math.floor(Math.random() * 4)]}`,
        (logNum) => `Order ${logNum} was ${['placed', 'shipped', 'delivered', 'cancelled'][Math.floor(Math.random() * 4)]} successfully`,
        (logNum) => `Payment transaction ${logNum} completed - amount $${(Math.random() * 1000).toFixed(2)}`,
        (logNum) => `File ${logNum}.log uploaded - size ${Math.floor(Math.random() * 10)}MB`,
        // Email pattern will match the email address
        (logNum) => `Email sent to user_${logNum}@example.com successfully`
    ];
    
    for (let i = 0; i < batchSize; i++) {
        // Hybrid approach: sequential base + small random offset
        const baseLog = (iter * batchSize + i);
        const randomOffset = Math.floor(Math.random() * 100);
        const logNum = (baseLog + randomOffset) % NUM_MODULES;
        const traceValue = Math.floor(Math.random() * CARDINALITY);
        
        const severity = SEVERITIES[logNum % 4];
        
        // Select a template based on logNum to get good distribution
        const templateIndex = logNum % logTemplates.length;
        const logMessage = logTemplates[templateIndex](logNum, serviceName);
        
        // Generate trace_id and span_id (16 and 8 bytes as hex strings)
        const traceId = Math.floor(Math.random() * 1e16).toString(16).padStart(32, '0');
        const spanId = Math.floor(Math.random() * 1e16).toString(16).padStart(16, '0');
        
        const logRecord = {
            time_unix_nano: timestamp + (i * 1000000), // Spread logs over time
            severity_number: severity.number,
            severity_text: severity.text,
            body: { string_value: logMessage },
            attributes: [
                { key: 'log.level', value: { string_value: severity.text.toLowerCase() } },
                { key: 'module', value: { string_value: `module_${logNum % 20}` } },
                { key: 'user_id', value: { string_value: `user_${Math.floor(Math.random() * CARDINALITY)}` } },
                { key: 'request_id', value: { string_value: `req_${logNum}` } },
            ],
            trace_id: traceId,
            span_id: spanId,
            dropped_attributes_count: Math.floor(Math.random() * 5), // 0-4 dropped attributes
        };
        
        // Add event.name attribute to some logs
        if (logNum % 3 === 0) {
            logRecord.attributes.push({
                key: 'event.name',
                value: { string_value: ['user.login', 'user.logout', 'payment.completed', 'order.placed'][logNum % 4] }
            });
        }
        
        logRecords.push(logRecord);
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
    console.log('K6 Load Test for OTLP Logs (Write-Only Mode)');
    console.log('='.repeat(50));
    console.log(`OTLP Endpoint: ${OTLP_ENDPOINT}`);
    console.log(`Modules:       ${NUM_MODULES} unique modules`);
    console.log(`Cardinality:   ${CARDINALITY} values per attribute`);
    console.log('='.repeat(50));
}

// Main test function - runs for each VU iteration
export default function() {
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
    
    sleep(0.1); // 100ms pause between batches
}

// Teardown function - runs once after test
export function teardown() {
    console.log('\n' + '='.repeat(50));
    console.log('Test Complete');
    console.log('='.repeat(50));
    console.log('Query the API manually to see results:');
    console.log(`  curl ${API_ENDPOINT}/api/v1/logs`);
    console.log(`  curl ${API_ENDPOINT}/api/v1/services`);
    console.log('='.repeat(50));
}

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

// Realistic span names based on microservices architecture
const spanTemplates = [
    // Web API endpoints
    { name: 'GET /api/v1/user/{userId}', kind: 2, service: 'service-1', attrs: ['http.method', 'http.route', 'http.status_code', 'user_id'] },
    { name: 'POST /api/v1/auth/login', kind: 2, service: 'service-1', attrs: ['http.method', 'http.route', 'http.status_code', 'client_ip'] },
    { name: 'POST /api/v1/order/create', kind: 2, service: 'service-1', attrs: ['http.method', 'http.route', 'http.status_code', 'user_id'] },
    { name: 'PUT /api/v1/profile/update', kind: 2, service: 'api-gateway', attrs: ['http.method', 'http.route', 'http.status_code', 'user_id'] },
    { name: 'GET /api/v1/search/products', kind: 2, service: 'api-gateway', attrs: ['http.method', 'http.route', 'http.status_code', 'query'] },
    
    // Order service operations
    { name: 'order-service/createOrder', kind: 1, service: 'service-1', attrs: ['order_id', 'user_id', 'total_amount'] },
    { name: 'order-service/validateStock', kind: 3, service: 'service-1', attrs: ['product_id', 'quantity'] },
    { name: 'order-service/processPayment', kind: 3, service: 'service-1', attrs: ['order_id', 'payment_method'] },
    { name: 'order-service/sendConfirmationEmail', kind: 3, service: 'order-service', attrs: ['order_id', 'user_email'] },
    
    // Product service operations
    { name: 'product-service/getProductDetails', kind: 1, service: 'product-service', attrs: ['product_id', 'category'] },
    { name: 'product-service/getProductsByCategory', kind: 1, service: 'product-service', attrs: ['category', 'limit'] },
    { name: 'product-service/updateProductStock', kind: 1, service: 'product-service', attrs: ['product_id', 'stock_delta'] },
    
    // User service operations
    { name: 'user-service/getUserProfile', kind: 1, service: 'user-service', attrs: ['user_id'] },
    { name: 'user-service/registerNewUser', kind: 1, service: 'user-service', attrs: ['email', 'signup_method'] },
    { name: 'user-service/resetPassword', kind: 1, service: 'user-service', attrs: ['user_id', 'reset_token'] },
    
    // Database operations
    { name: 'db/query: SELECT FROM users WHERE id = ?', kind: 3, service: 'user-service', attrs: ['db.system', 'db.statement', 'db.operation'] },
    { name: 'db/query: INSERT INTO orders VALUES (...)', kind: 3, service: 'order-service', attrs: ['db.system', 'db.statement', 'db.operation'] },
    { name: 'db/query: UPDATE products SET stock = ? WHERE id = ?', kind: 3, service: 'product-service', attrs: ['db.system', 'db.statement', 'db.operation'] },
    { name: 'db/query: DELETE FROM sessions WHERE userId = ?', kind: 3, service: 'user-service', attrs: ['db.system', 'db.statement', 'db.operation'] },
    
    // Cache operations
    { name: 'cache/get: product-details-cache', kind: 3, service: 'product-service', attrs: ['cache.hit', 'cache.key'] },
    { name: 'cache/set: user-session-cache', kind: 3, service: 'user-service', attrs: ['cache.key', 'cache.ttl'] },
    
    // External services
    { name: 'payment-gateway/processPayment', kind: 3, service: 'order-service', attrs: ['payment.provider', 'payment.method', 'amount'] },
    { name: 'sms-provider/sendSmsNotification', kind: 3, service: 'notification-service', attrs: ['phone_number', 'message_type'] },
    
    // Message queues
    { name: 'queue/publish: order-created-event', kind: 4, service: 'order-service', attrs: ['messaging.system', 'messaging.destination'] },
    { name: 'queue/receive: process-payment-message', kind: 5, service: 'payment-processor', attrs: ['messaging.system', 'messaging.source'] },
];

// Generate random trace data with realistic span names
function generateTraceBatch(vu, iter) {
    const timestamp = Date.now() * 1000000; // nanoseconds
    
    const spans = [];
    const batchSize = 10;
    
    for (let i = 0; i < batchSize; i++) {
        // Use both iter and i to get different spans over time
        const spanIndex = (iter * batchSize + i) % spanTemplates.length;
        const spanTemplate = spanTemplates[spanIndex];
        const userValue = Math.floor(Math.random() * CARDINALITY);
        
        // Generate trace and span IDs
        const traceId = Math.floor(Math.random() * 1000000000000).toString(16).padStart(32, '0');
        const spanId = Math.floor(Math.random() * 10000000000).toString(16).padStart(16, '0');
        
        // Build attributes based on span template
        const attributes = [];
        spanTemplate.attrs.forEach(attr => {
            switch(attr) {
                case 'http.method':
                    attributes.push({ key: 'http.method', value: { string_value: ['GET', 'POST', 'PUT', 'DELETE'][Math.floor(Math.random() * 4)] } });
                    break;
                case 'http.route':
                    attributes.push({ key: 'http.route', value: { string_value: spanTemplate.name } });
                    break;
                case 'http.status_code':
                    attributes.push({ key: 'http.status_code', value: { int_value: [200, 201, 204, 400, 404, 500][Math.floor(Math.random() * 6)] } });
                    break;
                case 'user_id':
                    attributes.push({ key: 'user_id', value: { string_value: `user_${userValue}` } });
                    break;
                case 'order_id':
                    attributes.push({ key: 'order_id', value: { string_value: `order_${Math.floor(Math.random() * 10000)}` } });
                    break;
                case 'product_id':
                    attributes.push({ key: 'product_id', value: { string_value: `prod_${Math.floor(Math.random() * 500)}` } });
                    break;
                case 'db.system':
                    attributes.push({ key: 'db.system', value: { string_value: 'postgresql' } });
                    break;
                case 'db.statement':
                    attributes.push({ key: 'db.statement', value: { string_value: spanTemplate.name } });
                    break;
                case 'db.operation':
                    attributes.push({ key: 'db.operation', value: { string_value: spanTemplate.name.split(' ')[2] } });
                    break;
                case 'cache.hit':
                    attributes.push({ key: 'cache.hit', value: { bool_value: Math.random() > 0.3 } });
                    break;
                case 'cache.key':
                    attributes.push({ key: 'cache.key', value: { string_value: spanTemplate.name.split(': ')[1] } });
                    break;
                case 'messaging.system':
                    attributes.push({ key: 'messaging.system', value: { string_value: 'kafka' } });
                    break;
                case 'messaging.destination':
                    attributes.push({ key: 'messaging.destination', value: { string_value: 'order-events' } });
                    break;
                case 'payment.provider':
                    attributes.push({ key: 'payment.provider', value: { string_value: ['stripe', 'paypal', 'klarna'][Math.floor(Math.random() * 3)] } });
                    break;
                default:
                    attributes.push({ key: attr, value: { string_value: `value_${Math.floor(Math.random() * 100)}` } });
            }
        });
        
        spans.push({
            trace_id: traceId,
            span_id: spanId,
            name: spanTemplate.name,
            kind: spanTemplate.kind,
            start_time_unix_nano: timestamp,
            end_time_unix_nano: timestamp + Math.floor(Math.random() * 1000000000), // Random duration up to 1 second
            attributes: attributes,
        });
    }
    
    return {
        resource_spans: [{
            resource: {
                attributes: [
                    { key: 'service.name', value: { string_value: spanTemplates[iter % spanTemplates.length].service } },
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
    console.log('K6 Load Test for OTLP Traces (Realistic Microservices)');
    console.log('='.repeat(50));
    console.log(`OTLP Endpoint: ${OTLP_ENDPOINT}`);
    console.log(`Span Templates: ${spanTemplates.length} unique operations`);
    console.log(`Services:       ${[...new Set(spanTemplates.map(t => t.service))].join(', ')}`);
    console.log(`Cardinality:    ${CARDINALITY} values per attribute`);
    console.log('='.repeat(50));
}

// Main test function - runs for each VU iteration
export default function() {
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
    
    sleep(0.1); // 100ms pause between batches
}

// Teardown function - runs once after test
export function teardown() {
    console.log('\n' + '='.repeat(50));
    console.log('Test Complete');
    console.log('='.repeat(50));
    console.log('Query the API manually to see results:');
    console.log(`  curl ${API_ENDPOINT}/api/v1/spans`);
    console.log(`  curl ${API_ENDPOINT}/api/v1/services`);
    console.log('='.repeat(50));
}

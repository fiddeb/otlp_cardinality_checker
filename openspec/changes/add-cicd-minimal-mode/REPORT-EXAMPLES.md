# OCC Report Examples

Realistic examples of OCC minimal mode reports for an e-commerce application.

## Scenario

**Application:** E-commerce platform  
**Test duration:** 5 minutes  
**Telemetry generated:**
- 87 unique metrics
- 34 unique span names  
- 12 log patterns
- 156 unique attributes
- 2.45M metric samples, 1.85M spans, 425K logs

---

## Basic Mode Report (Default)

```
================================================================================
OCC Telemetry Analysis Report
================================================================================
Generated: 2026-01-25 14:23:45 UTC
Duration: 5m 0s
OCC Version: 0.2.0
Verbosity: basic (top 20 items per section)

SUMMARY
================================================================================
Metrics:           87 total
Span names:        34 total  
Log patterns:      12 total
Unique attributes: 156 total
High cardinality:  8 items ⚠️

Samples received:
  Metrics: 2,450,000 data points
  Spans:   1,850,000 spans
  Logs:     425,000 log records

STATUS: ⚠️  WARNING - High cardinality issues detected

TOP METRICS BY CARDINALITY (showing 5 of 87)
================================================================================

1. ✗ order_processing_duration_ms [CRITICAL]
   Type: histogram
   Labels: customer_id, order_id, warehouse_id, payment_method, shipping_zone
   Cardinality: 45,230 ⚠️
   Samples: 125,400
   Recommendation: Remove customer_id and order_id - use aggregated dimensions

2. ⚠ user_session_active [WARNING]
   Type: gauge
   Labels: user_id, session_id, device_type, app_version, country_code
   Cardinality: 18,500
   Samples: 450,000

3. ⚠ product_view_count [WARNING]  
   Type: counter
   Labels: product_sku, user_segment, referrer_url, ab_test_variant
   Cardinality: 12,800
   Samples: 890,000

4. ✓ http_request_duration_seconds [OK]
   Type: histogram
   Labels: method, route, status_code, region
   Cardinality: 240
   Samples: 650,000

5. ✓ cache_hit_ratio [OK]
   Type: gauge
   Labels: cache_type, region
   Cardinality: 12
   Samples: 334,600

... 82 more metrics (use --report-verbosity verbose to see all)

TOP SPANS BY CARDINALITY (showing 5 of 34)
================================================================================

1. ✗ db.query.orders [CRITICAL]
   Attributes: db.system, db.operation, db.statement, user.id, order.id
   Cardinality: 67,500 ⚠️
   Spans: 450,000
   Recommendation: Do not include full SQL in db.statement - use parameterized

2. ⚠ POST /api/v1/checkout [WARNING]
   Attributes: http.method, http.route, user.id, cart.items, payment.processor
   Cardinality: 8,900
   Spans: 125,000

3. ⚠ redis.get [WARNING]
   Attributes: db.system, db.operation, cache.key, user.id
   Cardinality: 5,600
   Spans: 680,000

4. ✓ GET /api/v1/products/{id} [OK]
   Attributes: http.method, http.route, http.status_code, cache.hit
   Cardinality: 450
   Spans: 340,000

5. ✓ payment.process [OK]
   Attributes: payment.provider, payment.method, payment.status
   Cardinality: 18
   Spans: 95,000

... 29 more spans

TOP LOGS BY CARDINALITY (showing 5 of 12)
================================================================================

1. ⚠ Order processing error [WARNING]
   Pattern: "Order {order_id} processing failed: {error_msg}"
   Attributes: order.id, error.type, error.message, user.id, timestamp
   Cardinality: 3,200
   Logs: 8,500

2. ✓ User login successful [OK]
   Pattern: "User logged in successfully"
   Attributes: user.id, login.method, source.ip, user_agent
   Cardinality: 1,850
   Logs: 45,000

3. ✓ Cache miss [OK]
   Pattern: "Cache miss for key: {key}"
   Attributes: cache.type, cache.key, cache.ttl
   Cardinality: 890
   Logs: 125,000

4. ✓ Payment processed [OK]
   Pattern: "Payment successful"
   Attributes: payment.method, payment.amount_range, payment.currency
   Cardinality: 24
   Logs: 95,000

5. ✓ Product indexed [OK]
   Pattern: "Product indexed in search"
   Attributes: product.category, index.status
   Cardinality: 8
   Logs: 15,000

... 7 more log patterns

ATTRIBUTES (CROSS-SIGNAL ANALYSIS) - showing 10 of 156
================================================================================

1. ✗ user_id [HIGH CARDINALITY RISK]
   Signals: metrics(3), spans(8), logs(4)
   Unique values: ~18,500
   Impact: CRITICAL
   Used in:
   - Metrics: user_session_active, user_cart_value, user_events_total
   - Spans: POST /api/v1/checkout, redis.get, db.query.orders, ...
   - Logs: User login successful, Order processing error, ...
   ⚠️  Consider using user segments instead of individual IDs

2. ✗ order_id [HIGH CARDINALITY RISK]
   Signals: metrics(2), spans(5), logs(2)
   Unique values: ~45,000
   Impact: CRITICAL
   Used in:
   - Metrics: order_processing_duration_ms, order_value_histogram
   - Spans: POST /api/v1/orders, db.query.orders, ...
   - Logs: Order processing error, Order completed
   ⚠️  Never use order IDs in metrics - aggregate to order status or type

3. ⚠ db.statement [WARNING]
   Signals: spans(12)
   Unique values: ~67,000
   Impact: HIGH
   Used in: db.query.orders, db.query.products, db.query.users, ...
   ⚠️  Use parameterized queries, not full SQL statements

4. ⚠ product_sku [WARNING]
   Signals: metrics(2), spans(3)
   Unique values: ~12,000
   Impact: MEDIUM
   ⚠️  Consider product categories instead of individual SKUs

5. ✓ http.method [OK]
   Signals: metrics(5), spans(18)
   Unique values: 7 (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS)
   Impact: LOW

6. ✓ payment.method [OK]
   Signals: metrics(2), spans(3), logs(1)
   Unique values: 6 (credit_card, paypal, apple_pay, ...)
   Impact: LOW

7. ✓ region [OK]
   Signals: metrics(8), spans(12)
   Unique values: 4 (us-east, us-west, eu-central, ap-southeast)
   Impact: LOW

8. ✓ cache.type [OK]
   Signals: metrics(3), spans(5), logs(2)
   Unique values: 3 (redis, memcached, in-memory)
   Impact: LOW

9. ✓ http.status_code [OK]
   Signals: metrics(4), spans(15)
   Unique values: 12 (200, 201, 204, 400, 401, 403, 404, 422, 500, ...)
   Impact: LOW

10. ✓ db.system [OK]
    Signals: spans(12), logs(2)
    Unique values: 3 (postgresql, redis, elasticsearch)
    Impact: LOW

... 146 more attributes

RECOMMENDATIONS
================================================================================
⚠️  CRITICAL ISSUES (must fix before production):

  • Remove 'order_id' from metric order_processing_duration_ms
    → Use order_status or order_type instead
  
  • Remove 'customer_id' from metric order_processing_duration_ms
    → Use customer_segment (new/returning/vip) instead
  
  • Remove 'user_id' from metric user_session_active
    → Track session count per user_segment instead
  
  • Fix span db.query.orders - do not include full SQL in db.statement
    → Use operation type and table name only
  
⚠️  WARNINGS (should review):

  • Consider aggregating product_sku to product_category in metrics
  • Review cache.key cardinality in redis.get spans
  • Reduce attributes in POST /api/v1/checkout span
  
✓ GOOD PRACTICES DETECTED:

  • Proper use of semantic conventions for HTTP attributes
  • Bounded cardinality in payment and region dimensions
  • Appropriate use of status codes and cache types

OPENTELEMETRY SEMANTIC CONVENTIONS
================================================================================
✓ Using standard conventions: http.method, http.route, http.status_code
⚠  Non-standard: order.id (use order.status), user.id (use user.segment)
ℹ  See: https://opentelemetry.io/docs/specs/semconv/

NEXT STEPS
================================================================================
For detailed cardinality analysis with full UI:

  1. Export session from this run:
     occ start --minimal --session-export ci-analysis.json

  2. Load in full OCC UI:
     occ start
     occ session load ci-analysis.json
     
  3. Access web interface:
     http://localhost:3000

  4. Explore:
     • Time-series cardinality trends
     • Attribute value distributions  
     • Cross-signal correlations
     • Pattern detection

================================================================================
Report complete. Exit code: 1 (warnings detected)
Use --exit-on-threshold flag to fail CI/CD on cardinality issues
================================================================================
```

---

## Verbose Mode Report (Partial)

```
================================================================================
OCC Telemetry Analysis Report
================================================================================
Generated: 2026-01-25 14:23:45 UTC
Duration: 5m 0s
OCC Version: 0.2.0
Verbosity: verbose (all items, sorted by cardinality)

SUMMARY
================================================================================
[Same as basic mode...]

ALL METRICS (sorted by cardinality descending)
================================================================================

1. ✗ order_processing_duration_ms [CRITICAL]
   Type: histogram
   Labels: customer_id, order_id, warehouse_id, payment_method, shipping_zone
   Cardinality: 45,230 ⚠️
   Samples: 125,400
   Buckets: [50ms, 100ms, 250ms, 500ms, 1s, 2.5s, 5s, 10s+]
   P50: 450ms | P95: 2.1s | P99: 4.8s
   
2. ⚠ user_session_active [WARNING]
   Type: gauge
   Labels: user_id, session_id, device_type, app_version, country_code
   Cardinality: 18,500
   Samples: 450,000
   Current value range: 0-1
   
3. ⚠ product_view_count [WARNING]
   Type: counter
   Labels: product_sku, user_segment, referrer_url, ab_test_variant
   Cardinality: 12,800
   Samples: 890,000
   Total: 2,456,789 views
   
4. ⚠ shopping_cart_value_usd [WARNING]
   Type: histogram
   Labels: user_segment, country_code, currency, cart_size_bucket
   Cardinality: 8,400
   Samples: 234,000
   Buckets: [10, 25, 50, 100, 250, 500, 1000+]
   
5. ✓ api_request_duration_seconds [OK]
   Type: histogram
   Labels: service, endpoint, method, status_code
   Cardinality: 2,100
   Samples: 1,250,000
   
6. ✓ database_connection_pool_size [OK]
   Type: gauge
   Labels: db_host, db_name, pool_type
   Cardinality: 840
   Samples: 78,000

7. ✓ http_request_duration_seconds [OK]
   Type: histogram
   Labels: method, route, status_code, region
   Cardinality: 240
   Samples: 650,000

8. ✓ payment_transaction_amount_usd [OK]
   Type: histogram
   Labels: payment_method, currency, country_code
   Cardinality: 180
   Samples: 95,000

... [continues with all 87 metrics]

ALL SPANS (sorted by cardinality descending)
================================================================================

1. ✗ db.query.orders [CRITICAL]
   Attributes: 
   - db.system: postgresql
   - db.operation: SELECT, INSERT, UPDATE, DELETE
   - db.statement: [67,500 unique values] ⚠️
   - db.name: ecommerce_prod
   - db.table: orders
   - user.id: [18,500 unique values]
   - order.id: [45,000 unique values]
   Cardinality: 67,500 ⚠️
   Spans: 450,000
   Avg duration: 25ms
   P95 duration: 180ms
   P99 duration: 450ms

2. ⚠ POST /api/v1/checkout [WARNING]
   Attributes:
   - http.method: POST
   - http.route: /api/v1/checkout
   - http.status_code: 200, 201, 400, 422, 500
   - user.id: [18,500 unique values]
   - cart.item_count: 1-50
   - payment.processor: stripe, paypal, adyen
   - promotion.code: [varies]
   Cardinality: 8,900
   Spans: 125,000
   Avg duration: 340ms
   P95 duration: 890ms
   P99 duration: 1.2s

3. ⚠ redis.get [WARNING]
   Attributes:
   - db.system: redis
   - db.operation: GET
   - cache.key: [5,600 unique keys]
   - cache.hit: true, false
   - user.id: [18,500 unique values]
   Cardinality: 5,600
   Spans: 680,000
   Avg duration: 2ms
   P99 duration: 15ms

... [continues with all 34 spans]

ALL ATTRIBUTES (cross-signal, sorted by impact)
================================================================================

1. ✗ db.statement [CRITICAL CARDINALITY]
   Unique values: ~67,500
   Signal distribution:
   - Spans: 12 span names
   
   Used in spans:
   • db.query.orders (450k spans, cardinality: 67,500)
   • db.query.products (340k spans, cardinality: 12,400)  
   • db.query.users (180k spans, cardinality: 8,900)
   • db.query.inventory (120k spans, cardinality: 5,600)
   • db.query.payments (95k spans, cardinality: 3,200)
   • db.query.sessions (78k spans, cardinality: 2,100)
   • db.query.carts (56k spans, cardinality: 1,800)
   • db.query.reviews (34k spans, cardinality: 890)
   • db.query.wishlist (23k spans, cardinality: 450)
   • db.query.recommendations (12k spans, cardinality: 234)
   • db.query.analytics (8k spans, cardinality: 123)
   • db.query.audit (4k spans, cardinality: 67)
   
   Example values:
   • "SELECT * FROM orders WHERE user_id = 'usr_12345' AND status = 'pending'"
   • "SELECT * FROM orders WHERE order_id = 'ord_54321'"
   • "INSERT INTO orders (user_id, total) VALUES ('usr_98765', 149.99)"
   
   ⚠️  CRITICAL: Do not include parameter values in db.statement
   ✓ FIX: Use "SELECT * FROM orders WHERE user_id = ? AND status = ?"

... [continues with all 156 attributes in full detail]

================================================================================
Report complete. Total: 87 metrics + 34 spans + 12 logs + 156 attributes
================================================================================
```

---

## Comparison Table

| Feature | Basic Mode | Verbose Mode |
|---------|------------|---------------|
| **Summary stats** | ✓ Full | ✓ Full |
| **Metrics shown** | Top 20 of 87 | All 87 |
| **Spans shown** | Top 20 of 34 | All 34 |
| **Logs shown** | Top 20 of 12 | All 12 |
| **Attributes** | Top 20 of 156 | All 156 with examples |
| **Details level** | Name, cardinality, severity | + value examples, percentiles |
| **File size** | ~5KB | ~80KB |
| **Read time** | 2-3 minutes | 15-20 minutes |
| **Best for** | CI feedback, alerts | Documentation, deep analysis |
| **Default** | Yes | No (opt-in) |

---

## Usage

### Generate Basic Report
```bash
occ start --minimal --duration 5m --report-output report.txt
# or explicitly:
occ start --minimal --duration 5m \
  --report-verbosity basic \
  --report-max-items 20
```

### Generate Verbose Report
```bash
occ start --minimal --duration 5m \
  --report-output report-verbose.txt \
  --report-verbosity verbose \
  --report-max-items 0  # 0 = unlimited
```

### Generate Report + Session Export
```bash
occ start --minimal --duration 5m \
  --report-output report.txt \
  --session-export session.json

# Later: load session in UI for interactive analysis
occ start
occ session load session.json
# Browse at http://localhost:3000
```

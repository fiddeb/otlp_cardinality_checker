# Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the OTLP Cardinality Checker.

## Files

- `deployment.yaml` - Deployment configuration
- `service.yaml` - Service configuration
- `ingress.yaml` - Ingress configuration (optional)

## Quick Start

### 1. Build Docker Image

```bash
# Build the image
docker build -t otlp-cardinality-checker:latest .

# Tag for your registry (if pushing to remote)
docker tag otlp-cardinality-checker:latest your-registry/otlp-cardinality-checker:latest
docker push your-registry/otlp-cardinality-checker:latest
```

### 2. Deploy to Kubernetes

```bash
# Deploy all resources
kubectl apply -f k8s/

# Or deploy individually
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### 3. Verify Deployment

```bash
# Check pods
kubectl get pods -l app=otlp-cardinality-checker

# Check service
kubectl get svc otlp-cardinality-checker

# Check logs
kubectl logs -l app=otlp-cardinality-checker -f
```

### 4. Access the Service

```bash
# Port-forward to access locally
kubectl port-forward svc/otlp-cardinality-checker 8080:8080 4318:4318

# Test API
curl http://localhost:8080/api/v1/health

# Test OTLP endpoint (from OpenTelemetry Collector)
# Configure your collector to export to:
# endpoint: http://otlp-cardinality-checker:4318
```

## Configuration

### Resource Limits

Default resource configuration:

```yaml
requests:
  memory: "128Mi"
  cpu: "100m"
limits:
  memory: "512Mi"
  cpu: "500m"
```

Adjust based on your workload:
- For 10,000 metrics: 256-512 MB memory
- For 50,000 metrics: 512 MB - 1 GB memory

### Replicas

The deployment is configured with 1 replica since the application uses in-memory storage.

For high availability with multiple replicas, you would need to:
1. Implement distributed storage (Redis, PostgreSQL)
2. Or accept that each replica has independent metadata

### Health Checks

- **Liveness Probe**: Checks if the application is alive (restarts if failing)
- **Readiness Probe**: Checks if the application can accept traffic

Both probes hit the `/api/v1/health` endpoint.

## OpenTelemetry Collector Configuration

Configure your OpenTelemetry Collector to export to this service:

```yaml
exporters:
  otlphttp:
    endpoint: http://otlp-cardinality-checker:4318
    compression: gzip

service:
  pipelines:
    metrics:
      receivers: [prometheus, kafka]
      processors: [batch]
      exporters: [otlphttp]
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp]
    logs:
      receivers: [filelog]
      processors: [batch]
      exporters: [otlphttp]
```

## Ingress (Optional)

If you need external access to the API:

```bash
kubectl apply -f k8s/ingress.yaml
```

Make sure you have an Ingress controller installed (e.g., nginx-ingress, traefik).

## Monitoring

### Prometheus ServiceMonitor

If using Prometheus Operator, you can add a ServiceMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: otlp-cardinality-checker
spec:
  selector:
    matchLabels:
      app: otlp-cardinality-checker
  endpoints:
  - port: api
    path: /api/v1/metrics
    interval: 30s
```

## Troubleshooting

### Pod not starting

```bash
# Check pod status
kubectl describe pod -l app=otlp-cardinality-checker

# Check logs
kubectl logs -l app=otlp-cardinality-checker --tail=50
```

### Service not accessible

```bash
# Check endpoints
kubectl get endpoints otlp-cardinality-checker

# Test from another pod
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://otlp-cardinality-checker:8080/api/v1/health
```

### High memory usage

```bash
# Check resource usage
kubectl top pod -l app=otlp-cardinality-checker

# Increase memory limits if needed
kubectl edit deployment otlp-cardinality-checker
```

## Cleanup

```bash
# Delete all resources
kubectl delete -f k8s/

# Or delete individually
kubectl delete deployment otlp-cardinality-checker
kubectl delete service otlp-cardinality-checker
```

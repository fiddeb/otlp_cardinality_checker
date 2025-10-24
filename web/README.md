# OTLP Cardinality Checker - Web UI

Simple React UI for analyzing OpenTelemetry cardinality.

## Features

- **Dashboard** - Overview of all telemetry with service list
- **High Cardinality Detection** - Find metrics/spans/logs with high cardinality attributes (adjustable threshold)
- **Metrics View** - Dedicated analysis for metrics with filtering by type, samples, and cardinality
- **Traces View** - Dedicated analysis for spans with filtering by kind, samples, and cardinality
- **Logs View** - Severity-based log analysis with visual breakdowns and cardinality insights
- **Compare View** - Side-by-side comparison of up to 4 metrics/spans/logs to identify differences
- **Service Explorer** - View all telemetry (metrics, spans, logs) for a specific service
- **Details View** - Deep dive into specific metric/span/log with full attribute breakdown

## Development

```bash
# Install dependencies
npm install

# Start dev server (proxies API to localhost:8080)
npm run dev

# Build for production
npm run build
```

## Production

The Go server can serve the built static files. Add to your Go server:

```go
// Serve UI
http.Handle("/", http.FileServer(http.Dir("./web/dist")))
```

Then build the UI and copy to your server:

```bash
cd web
npm run build
# dist/ folder contains the built files
```

## Architecture

- **React 18**: UI framework
- **Vite**: Build tool and dev server
- **No external CSS framework**: Clean, simple CSS
- **No state management library**: Uses React hooks only

Simple and easy to maintain.

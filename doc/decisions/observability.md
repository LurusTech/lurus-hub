# ADR: Observability Stack

**Status**: Accepted (partial implementation)
**Date**: 2026-02-03
**Relates to**: ADR-API-008 (architecture.md), Epic 4

## Context

Production Lurus API needs metrics, tracing, and alerting for incident response and performance optimization. The team is 2 people running on a single K3s node, so the stack must be lightweight and optional (feature-flagged).

### Current State

| Capability | Status |
|------------|--------|
| Structured logging (slog, JSON in prod) | Implemented |
| pprof profiling (`ENABLE_PPROF=true`) | Implemented |
| Active connections counter (StatsMiddleware) | Implemented |
| Prometheus metrics (`/metrics`) | Implemented |
| Performance benchmarks (relay latency) | Implemented |
| Distributed tracing | Not implemented |
| Alerting rules | Not implemented |

## Decision

Adopt OpenTelemetry (OTel) SDK as the instrumentation layer, with Prometheus for metrics and Jaeger for tracing. All observability features are gated behind environment variables and disabled by default.

### Architecture

```
Application (OTel SDK)
  |- Metrics -> Prometheus Exporter -> /metrics -> Prometheus -> Grafana
  |- Traces  -> OTLP Exporter -> Jaeger (staging: deployed)
  '- Logs    -> slog + trace_id injection -> stdout -> Loki (future)
```

### Key Metrics Exposed

- `http_request_duration_seconds` (histogram, by method/path/status)
- `http_requests_total` (counter)
- `active_connections` (gauge)
- `relay_request_duration_seconds` (histogram, by model/provider)
- Go runtime metrics (goroutines, GC, memory)

### Feature Flags

| Env Var | Default | Controls |
|---------|---------|----------|
| `ENABLE_METRIC` | `false` | Prometheus `/metrics` endpoint |
| `OTEL_TRACING_ENABLED` | `false` | Distributed tracing export |
| `ENABLE_PPROF` | `false` | Go pprof endpoints |

### Sampling Strategy

- 10% normal requests
- 100% errors and P99 latency outliers
- 0% health check endpoints (`/api/status`)

## Consequences

- (+) Zero overhead when disabled (feature flags)
- (+) Industry-standard stack (Prometheus + Grafana + Jaeger)
- (+) OTel SDK allows swapping backends without code changes
- (-) Distributed tracing adds ~1-2ms per request when enabled
- (-) Prometheus scraping requires ServiceMonitor in K8s

## References

- Prometheus endpoint: `internal/lifecycle/metrics.go`
- Benchmarks: `internal/adapter/handler/benchmark_test.go`
- Staging Jaeger: deployed in `infra` namespace

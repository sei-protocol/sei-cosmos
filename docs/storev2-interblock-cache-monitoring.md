# StorageV2 Inter-Block Cache Monitoring Guide

This guide explains how to monitor the effectiveness of the inter-block cache in StorageV2 and determine if it's providing performance benefits.

## Quick Start

### 1. Check if Cache is Enabled

```bash
# Check logs for cache enablement
grep "Inter-block cache" /path/to/app.log

# Expected output:
# Inter-block cache enabled for storev2
```

### 2. Basic Metrics Check

```bash
# If using Prometheus metrics endpoint
curl http://localhost:26660/metrics | grep storev2_interblock_cache
```

## Key Metrics to Monitor

### Cache Status Metrics

- `storev2_interblock_cache_enabled`: Should be `1` when cache is active
- `storev2_interblock_cache_active`: Confirms cache is being used during operations
- `storev2_interblock_cache_potential_stores`: Number of IAVL stores that could benefit

### Performance Metrics

- `storev2_interblock_cache_hits_total`: Cache hits per store (higher is better)
- `storev2_interblock_cache_misses_total`: Cache misses per store (lower is better)
- `storev2_interblock_cache_get_store_latency`: Total store access latency
- `storev2_interblock_cache_unwrap_latency`: Cache lookup latency

## Effectiveness Analysis

### 1. Calculate Cache Hit Rate

```promql
# Overall hit rate across all stores
rate(storev2_interblock_cache_hits_total[5m]) /
(rate(storev2_interblock_cache_hits_total[5m]) + rate(storev2_interblock_cache_misses_total[5m]))
```

**Interpretation:**
- **>80%**: Excellent cache effectiveness
- **50-80%**: Good, but room for improvement
- **<50%**: Poor, investigate cache configuration or workload patterns

### 2. Performance Impact Assessment

```promql
# 95th percentile store access latency
histogram_quantile(0.95, rate(storev2_interblock_cache_get_store_latency_bucket[5m]))

# Cache lookup overhead
histogram_quantile(0.95, rate(storev2_interblock_cache_unwrap_latency_bucket[5m]))
```

**What to Look For:**
- Cache lookup overhead should be <1ms
- Overall store access should be faster with cache enabled
- Compare latencies before/after enabling cache

### 3. Store-Level Analysis

```promql
# Hit rate per store
rate(storev2_interblock_cache_hits_total[5m]) by (store_key) /
(rate(storev2_interblock_cache_hits_total[5m]) by (store_key) +
 rate(storev2_interblock_cache_misses_total[5m]) by (store_key))
```

This helps identify which stores benefit most from caching.

## Common Issues and Troubleshooting

### Issue: Low Hit Rate (<50%)

**Possible Causes:**
1. **Workload not cache-friendly**: Random access patterns don't benefit from caching
2. **Cache size too small**: Cache evicting items too quickly
3. **High write volume**: Frequent invalidations reduce hit rate

**Investigation Steps:**
```bash
# Check cache size configuration
grep -i "cache" /path/to/config.toml

# Monitor cache evictions (if your cache implementation reports them)
curl http://localhost:26660/metrics | grep evict
```

### Issue: High Cache Lookup Latency

**Possible Causes:**
1. **Cache implementation inefficient**: Consider alternative cache backend
2. **Lock contention**: Cache access patterns causing blocking

**Investigation:**
```promql
# Compare cache vs direct access latency
histogram_quantile(0.95, rate(storev2_interblock_cache_unwrap_latency_bucket[5m])) -
histogram_quantile(0.95, rate(storev2_interblock_cache_get_store_latency_bucket[5m]))
```

### Issue: Cache Not Being Used

**Check List:**
1. Verify `storev2_interblock_cache_enabled = 1`
2. Confirm `storev2_interblock_cache_store_wrapped > 0`
3. Check application configuration for cache setup

## Sample Monitoring Dashboard

### Grafana Panel Queries

```json
{
  "panels": [
    {
      "title": "Cache Hit Rate",
      "expr": "rate(storev2_interblock_cache_hits_total[5m]) / (rate(storev2_interblock_cache_hits_total[5m]) + rate(storev2_interblock_cache_misses_total[5m]))",
      "format": "percent"
    },
    {
      "title": "Store Access Latency (95th percentile)",
      "expr": "histogram_quantile(0.95, rate(storev2_interblock_cache_get_store_latency_bucket[5m]))",
      "format": "ms"
    },
    {
      "title": "Cache Operations per Second",
      "expr": "rate(storev2_interblock_cache_hits_total[5m]) + rate(storev2_interblock_cache_misses_total[5m])",
      "format": "ops"
    }
  ]
}
```

## Alerting Rules

```yaml
groups:
- name: storev2_cache_alerts
  rules:
  - alert: InterBlockCacheLowHitRate
    expr: |
      (
        rate(storev2_interblock_cache_hits_total[10m]) /
        (rate(storev2_interblock_cache_hits_total[10m]) + rate(storev2_interblock_cache_misses_total[10m]))
      ) < 0.3
    for: 5m
    annotations:
      summary: "StorageV2 inter-block cache hit rate is below 30%"
      description: "Cache hit rate has been below 30% for 5 minutes, indicating poor cache effectiveness"

  - alert: InterBlockCacheDisabled
    expr: storev2_interblock_cache_enabled == 0
    for: 1m
    annotations:
      summary: "StorageV2 inter-block cache is disabled"
      description: "The inter-block cache is not enabled, missing potential performance benefits"
```

## Performance Testing

### Before/After Cache Comparison

1. **Baseline measurement** (cache disabled):
   ```bash
   # Record performance metrics for 10 minutes
   curl -s http://localhost:26660/metrics | grep storev2 > baseline_metrics.txt
   ```

2. **Enable cache** and measure again:
   ```bash
   # After cache is enabled and warmed up
   curl -s http://localhost:26660/metrics | grep storev2 > cached_metrics.txt
   ```

3. **Compare results**:
   - Store access latency should decrease
   - Overall throughput may increase
   - CPU usage patterns may change

### Load Testing with Cache

```bash
# Example load test command (adjust for your application)
ab -n 10000 -c 10 http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest
```

Monitor cache metrics during load testing to see how cache performs under stress.

## Summary

The inter-block cache is effective when:
1. ✅ Hit rate consistently above 70%
2. ✅ Store access latency reduced compared to no-cache baseline
3. ✅ Cache lookup overhead minimal (<1ms)
4. ✅ No significant increase in memory usage or CPU overhead

Regular monitoring of these metrics will help ensure the cache continues to provide performance benefits as your application evolves.
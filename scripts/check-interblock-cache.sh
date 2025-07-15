#!/bin/bash

# StorageV2 Inter-Block Cache Status Checker
# This script provides a quick overview of inter-block cache status and effectiveness

set -e

# Configuration
METRICS_URL="${METRICS_URL:-http://localhost:26660/metrics}"
LOG_FILE="${LOG_FILE:-/var/log/app.log}"

echo "🔍 StorageV2 Inter-Block Cache Status Check"
echo "============================================="

# Function to get metric value
get_metric() {
    local metric_name="$1"
    curl -s "$METRICS_URL" 2>/dev/null | grep "^${metric_name}" | awk '{print $2}' | head -1
}

# Function to calculate percentage
calc_percentage() {
    local numerator="$1"
    local denominator="$2"
    if [ "$denominator" != "0" ] && [ -n "$denominator" ]; then
        echo "scale=2; ($numerator / $denominator) * 100" | bc -l
    else
        echo "0"
    fi
}

echo "📊 Cache Status:"
echo "---------------"

# Check if cache is enabled
enabled=$(get_metric "storev2_interblock_cache_enabled")
if [ "$enabled" = "1" ]; then
    echo "✅ Cache Status: ENABLED"
else
    echo "❌ Cache Status: DISABLED"
    echo ""
    echo "💡 To enable inter-block cache, configure it in your application."
    exit 0
fi

# Check if cache is active
active=$(get_metric "storev2_interblock_cache_active")
if [ "$active" = "1" ]; then
    echo "✅ Cache Active: YES"
else
    echo "⚠️  Cache Active: NO"
fi

# Store information
potential_stores=$(get_metric "storev2_interblock_cache_potential_stores")
wrapped_stores=$(get_metric "storev2_interblock_cache_store_wrapped_total")

echo "📦 Store Information:"
echo "--------------------"
echo "Potential IAVL Stores: ${potential_stores:-N/A}"
echo "Wrapped Stores: ${wrapped_stores:-N/A}"

echo ""
echo "📈 Performance Metrics:"
echo "----------------------"

# Get hit/miss counts
hits_total=$(curl -s "$METRICS_URL" 2>/dev/null | grep "storev2_interblock_cache_hits_total" | awk '{sum += $2} END {print sum}')
misses_total=$(curl -s "$METRICS_URL" 2>/dev/null | grep "storev2_interblock_cache_misses_total" | awk '{sum += $2} END {print sum}')

hits_total=${hits_total:-0}
misses_total=${misses_total:-0}
total_requests=$((hits_total + misses_total))

if [ "$total_requests" -gt 0 ]; then
    hit_rate=$(calc_percentage "$hits_total" "$total_requests")
    echo "Cache Hit Rate: ${hit_rate}%"
    echo "Total Hits: $hits_total"
    echo "Total Misses: $misses_total"
    echo "Total Requests: $total_requests"

    # Interpret hit rate
    if [ "$(echo "$hit_rate >= 80" | bc -l)" = "1" ]; then
        echo "✅ Hit Rate Assessment: EXCELLENT"
    elif [ "$(echo "$hit_rate >= 50" | bc -l)" = "1" ]; then
        echo "⚠️  Hit Rate Assessment: GOOD"
    else
        echo "❌ Hit Rate Assessment: POOR - Consider reviewing cache configuration"
    fi
else
    echo "No cache requests recorded yet"
fi

echo ""
echo "📝 Recent Log Messages:"
echo "----------------------"

if [ -f "$LOG_FILE" ]; then
    echo "Recent cache-related log entries:"
    grep -i "inter-block cache" "$LOG_FILE" 2>/dev/null | tail -5 || echo "No cache-related log messages found"
else
    echo "Log file not found at: $LOG_FILE"
    echo "Set LOG_FILE environment variable to specify log location"
fi

echo ""
echo "🔧 Quick Actions:"
echo "----------------"

if [ "$enabled" != "1" ]; then
    echo "• Enable inter-block cache in application configuration"
elif [ "$total_requests" -eq 0 ]; then
    echo "• Cache is enabled but not being used - check application activity"
elif [ "$(echo "${hit_rate:-0} < 50" | bc -l)" = "1" ] && [ "$total_requests" -gt 100 ]; then
    echo "• Low hit rate detected - consider:"
    echo "  - Reviewing cache size configuration"
    echo "  - Analyzing access patterns"
    echo "  - Checking for cache eviction issues"
else
    echo "• Cache appears to be working well!"
fi

echo ""
echo "For detailed monitoring guide, see: docs/storev2-interblock-cache-monitoring.md"
echo "Metrics endpoint: $METRICS_URL"
# Analytics Module for Audit Service

This module provides statistical anomaly detection, behavioral baseline management, and compliance scoring for the OpenPAM platform.

## Architecture

### Core Components

1. **Detector** (`detector.go`) - Statistical anomaly detection using Z-score and IQR methods
   - Behavioral anomaly detection
   - Temporal anomaly detection (off-hours access, unusual times)
   - Spatial anomaly detection (unusual targets)
   - Volumetric anomaly detection (command/session volume)

2. **MetricsStore** (`metrics_collector.go`) - Collection and aggregation of session metrics
   - Session metrics recording
   - Time-bucketed aggregation
   - Sliding window metrics
   - Command frequency analysis

3. **BaselineManager** (`baseline.go`) - Behavioral baseline establishment and maintenance
   - User behavioral baselines
   - Hourly and weekday patterns
   - Target access statistics
   - Baseline confidence scoring

4. **ComplianceEngine** (`compliance.go`) - Compliance scoring and reporting
   - Multi-framework support (SOC2, ISO27001, PCI-DSS, HIPAA, NIST-800-53, GDPR)
   - Control category assessment
   - Finding generation
   - Recommendation generation

5. **Scheduler** (`scheduler.go`) - Cron-based task scheduling
   - Anomaly detection runs
   - Baseline updates
   - Compliance scans
   - Report generation

6. **AnomalyStore** (`anomalies/store.go`) - Anomaly persistence and queries
   - CRUD operations for anomalies
   - Duplicate detection and merging
   - Filtering and pagination
   - Aggregation queries

7. **ReportGenerator** (`reports/generator.go`) - Report generation
   - Multiple output formats (JSON, HTML, CSV, PDF)
   - Template-based rendering
   - Async report generation

## Integration

### Adding to main.go

```go
import (
    "github.com/openpam/openpam/internal/audit/analytics"
    auditreports "github.com/openpam/openpam/internal/audit/reports"
    "github.com/openpam/openpam/internal/audit/anomalies"
    "github.com/redis/go-redis/v9"
)

// In main function after DB and Redis initialization:
redisClient := redis.NewClient(&redis.Options{
    Addr:     config.RedisHost + ":" + fmt.Sprint(config.RedisPort),
    Password: config.RedisPassword,
    DB:       config.RedisDB,
})

// Initialize analytics module
analyticsModule, err := analytics.InitializeModule(analytics.ModuleConfig{
    DB:                   db.DB,
    RedisClient:          redisClient,
    Logger:               logger,
    ReportStoragePath:    config.ReportStoragePath,
    ReportStorageBaseURL: config.ReportStorageBaseURL,
    ReportMaxFileSize:    config.ReportStorageMaxSize,
    ReportRetentionDays:  config.ReportRetentionDays,
    EnableScheduler:      true,
    SchedulerConfig:      analytics.DefaultSchedulerConfig(),
})
if err != nil {
    log.Fatal().Err(err).Msg("Failed to initialize analytics module")
}

// Start the module
go func() {
    if err := analyticsModule.Start(context.Background(), analytics.DefaultSchedulerConfig()); err != nil {
        logger.Error().Err(err).Msg("Failed to start analytics scheduler")
    }
}()

// Graceful shutdown
defer analyticsModule.Stop()
```

## API Endpoints

The analytics module adds the following endpoints:

### Anomalies
- `GET /api/v1/analytics/anomalies` - List anomalies
- `GET /api/v1/analytics/anomalies/:id` - Get anomaly details
- `PUT /api/v1/analytics/anomalies/:id` - Update anomaly
- `DELETE /api/v1/analytics/anomalies/:id` - Delete anomaly
- `GET /api/v1/analytics/anomalies/stats` - Get anomaly statistics
- `POST /api/v1/analytics/anomalies/detect` - Run anomaly detection
- `POST /api/v1/analytics/anomalies/:id/merge` - Merge duplicate anomalies

### Compliance
- `GET /api/v1/analytics/compliance/:framework` - Get compliance score
- `POST /api/v1/analytics/compliance/:framework/generate` - Generate compliance report

### Reports
- `POST /api/v1/analytics/reports/generate` - Generate a report
- `GET /api/v1/analytics/reports/templates` - List report templates
- `GET /api/v1/analytics/reports/download/:tenant_id/:filename` - Download report
- `POST /api/v1/analytics/reports/schedule` - Schedule recurring report

### Baselines
- `GET /api/v1/analytics/baselines/:user_id` - Get user baseline
- `POST /api/v1/analytics/baselines/:user_id/rebuild` - Rebuild baseline
- `GET /api/v1/analytics/baselines/stats` - Get baseline statistics

## Database Schema

See migrations/audit/005_baselines.up.sql for the complete database schema.

Key tables:
- `user_baselines` - Behavioral baselines for anomaly detection
- `session_analytics` - Detailed session metrics
- `daily_user_metrics` - Aggregated daily metrics
- `anomaly_detections` - Detected anomalies (already exists)

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ANALYTICS_SCHEDULER_ENABLED` | Enable cron scheduler | `true` |
| `ANALYTICS_BASELINE_RETENTION_DAYS` | Baseline retention | `90` |
| `ANALYTICS_METRICS_RETENTION_DAYS` | Metrics retention | `30` |
| `REPORT_STORAGE_PATH` | Report storage path | `/var/lib/openpam/reports` |
| `REPORT_STORAGE_BASE_URL` | Report base URL | `/api/v1/analytics/reports/download` |
| `REPORT_MAX_SIZE_MB` | Max report size | `100` |

### Scheduler Configuration

Default schedules (can be customized):
- Anomaly detection: Every 30 minutes
- Baseline update: Daily at 2 AM
- Compliance scan: Weekly on Monday at 3 AM
- Metrics aggregation: Every 15 minutes
- Report generation: Daily at 4 AM

## Usage Examples

### Detecting Anomalies

```go
detector := analyticsModule.Detector

config := analytics.DefaultDetectionConfig()
result, err := detector.DetectAnomalies(ctx, tenantID, config)

// Convert and store
anomalies := detector.ConvertToModelAnomalies(result)
for _, anomaly := range anomalies {
    analyticsModule.AnomalyStore.Create(ctx, &anomaly)
}
```

### Building User Baselines

```go
baselineMgr := analyticsModule.BaselineManager

now := time.Now()
baseline, err := baselineMgr.BuildUserBaseline(
    ctx,
    tenantID,
    userID,
    now.AddDate(0, 0, -90), // 90 day lookback
    now,
    "daily",
)
```

### Generating Compliance Reports

```go
compliance := analyticsModule.Compliance

score, err := compliance.CalculateComplianceScore(
    ctx,
    tenantID,
    "SOC2",
    periodStart,
    periodEnd,
)

report := compliance.ConvertToComplianceReport(score, generatedBy)
```

## Security Considerations

1. **Multi-tenancy**: All queries are scoped by tenant_id with RLS policies
2. **Data encryption**: Indicators stored in anomalies may contain sensitive data
3. **Audit trail**: All baseline changes and detections are logged
4. **Rate limiting**: Report generation endpoints should be rate-limited

## Performance

- Redis caching for frequently accessed baselines and metrics
- Sliding window counters for real-time metrics
- Materialized views for dashboard queries
- Batch processing for scheduled tasks

## Monitoring

Key metrics to monitor:
- Anomaly detection latency
- Baseline calculation time
- Report generation queue depth
- Cache hit/miss ratios
- Scheduler task execution times

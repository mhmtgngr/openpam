import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { analyticsApi } from '../../api/analytics';
import { AlertOctagon, AlertTriangle } from 'lucide-react';

interface AnomalyDetection {
  id: string;
  type: string;
  severity: string;
  title?: string;
  description: string;
  detected_at: string;
  confidence_score: number;
  status: string;
}

interface AnomalyAlertsProps {
  tenantId: string;
  status?: string;
  limit?: number;
}

export const AnomalyAlerts: React.FC<AnomalyAlertsProps> = ({
  tenantId,
  status,
  limit = 10,
}) => {
  const [anomalies, setAnomalies] = useState<AnomalyDetection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filterStatus, setFilterStatus] = useState<string | undefined>(status);

  useEffect(() => {
    const fetchAnomalies = async () => {
      setLoading(true);
      setError(null);
      try {
        const params: Record<string, string> = {};
        if (filterStatus) params.status = filterStatus;
        if (limit) params.limit = String(limit);

        const response = await analyticsApi.listAnomalies(params);
        setAnomalies((response as { data?: typeof anomalies }).data || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load anomalies');
      } finally {
        setLoading(false);
      }
    };

    fetchAnomalies();
  }, [tenantId, filterStatus, limit]);

  const handleStatusChange = (newStatus: string) => {
    setFilterStatus(newStatus === filterStatus ? undefined : newStatus);
  };

  const handleAssign = async (anomalyId: string) => {
    try {
      // Would call API to assign to current user
      await analyticsApi.updateAnomalyStatus(anomalyId, {
        status: 'investigating',
      });
      // Refresh list
      setAnomalies(anomalies.map(a =>
        a.id === anomalyId
          ? { ...a, status: 'investigating' as const }
          : a
      ));
    } catch (err) {
      console.error('Failed to assign anomaly:', err);
    }
  };

  if (loading) {
    return <LoadingState message="Loading anomaly alerts..." />;
  }

  if (error) {
    return (
      <Card>
        <div className="card-body">
          <div className="text-center text-danger-400">
            <p className="font-semibold">Error Loading Anomalies</p>
            <p className="text-sm text-gray-400">{error}</p>
          </div>
        </div>
      </Card>
    );
  }

  const criticalCount = anomalies.filter(a => a.severity === 'critical' && a.status === 'open').length;
  const highCount = anomalies.filter(a => a.severity === 'high' && a.status === 'open').length;

  return (
    <div className="space-y-4">
      {/* Summary */}
      {(criticalCount > 0 || highCount > 0) && (
        <Card className={`border-2 ${criticalCount > 0 ? 'border-danger-500 bg-danger-500/10' : 'border-warning-500 bg-warning-500/10'}`}>
          <div className="card-body">
            <div className="flex items-center gap-4">
              <div className={`rounded-lg p-3 ${criticalCount > 0 ? 'bg-danger-500/20 text-danger-400' : 'bg-warning-500/20 text-warning-400'}`}>
                {criticalCount > 0 ? <AlertOctagon className="h-6 w-6" /> : <AlertTriangle className="h-6 w-6" />}
              </div>
              <div>
                <p className="font-semibold text-white">
                  {criticalCount > 0 ? 'Critical Anomalies Require Attention' : 'High Severity Anomalies'}
                </p>
                <p className="text-sm text-gray-300">
                  {criticalCount > 0
                    ? `${criticalCount} critical anomaly(ies) need immediate investigation`
                    : `${highCount} high severity anomaly(ies) pending review`}
                </p>
              </div>
            </div>
          </div>
        </Card>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex items-center gap-2 flex-wrap">
            <span className="text-sm font-medium text-gray-300">Filter:</span>
            {['open', 'investigating', 'resolved', 'false_positive'].map((s) => (
              <button
                key={s}
                onClick={() => handleStatusChange(s)}
                className={`px-3 py-1 rounded-full text-sm capitalize transition-colors ${
                  filterStatus === s
                    ? 'bg-primary-600 text-white'
                    : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                }`}
              >
                {s.replace('_', ' ')}
              </button>
            ))}
          </div>
        </div>
      </Card>

      {/* Anomaly List */}
      {anomalies.length === 0 ? (
        <Card>
          <div className="card-body">
            <EmptyState message="No anomalies detected" />
          </div>
        </Card>
      ) : (
        <div className="space-y-3">
          {anomalies.map((anomaly) => (
            <AnomalyCard
              key={anomaly.id}
              anomaly={anomaly}
              onAssign={() => handleAssign(anomaly.id)}
            />
          ))}
        </div>
      )}
    </div>
  );
};

interface AnomalyCardProps {
  anomaly: AnomalyDetection;
  onAssign: () => void;
}

const AnomalyCard: React.FC<AnomalyCardProps> = ({ anomaly, onAssign }) => (
  <Card className={`border ${anomaly.severity === 'critical' ? 'border-danger-500' : anomaly.severity === 'high' ? 'border-warning-500' : ''}`}>
    <div className="card-body">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <SeverityBadge severity={anomaly.severity} />
            <StatusBadge status={anomaly.status} />
            <span className="text-xs text-gray-400 capitalize">
              {anomaly.type.replace(/_/g, ' ')}
            </span>
          </div>

          <p className="text-white mb-2">{anomaly.description}</p>

          <div className="flex items-center gap-4 text-xs text-gray-400">
            <span>
              Detected: {new Date(anomaly.detected_at).toLocaleString()}
            </span>
            <span>
              Confidence: {Math.round(anomaly.confidence_score * 100)}%
            </span>
          </div>
        </div>

        {anomaly.status === 'open' && (
          <Button onClick={onAssign} size="sm" variant="secondary">
            Assign
          </Button>
        )}
      </div>
    </div>
  </Card>
);

const SeverityBadge: React.FC<{ severity: string }> = ({ severity }) => {
  const variants: Record<string, 'danger' | 'warning' | 'success' | 'neutral'> = {
    critical: 'danger',
    high: 'danger',
    medium: 'warning',
    low: 'neutral',
  };

  return (
    <Badge variant={variants[severity] || 'neutral'}>
      {severity.toUpperCase()}
    </Badge>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const variants: Record<string, 'danger' | 'warning' | 'success' | 'neutral'> = {
    open: 'danger',
    investigating: 'warning',
    resolved: 'success',
    false_positive: 'neutral',
    ignored: 'neutral',
  };

  return (
    <Badge variant={variants[status] || 'danger'}>
      {status.replace('_', ' ').toUpperCase()}
    </Badge>
  );
};

interface EmptyStateProps {
  message: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message }) => (
  <div className="text-center py-8 text-gray-500">
    <p>{message}</p>
  </div>
);

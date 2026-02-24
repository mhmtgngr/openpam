import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { Badge } from '../common/Badge';
import { Button } from '../common/Button';
import { analyticsApi } from '../../api/analytics';

interface AnomalyDetection {
  id: string;
  tenant_id: string;
  user_id?: string;
  type: string;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'open' | 'investigating' | 'resolved' | 'false_positive' | 'ignored';
  description: string;
  detected_at: string;
  assigned_to?: string;
  resolution_notes?: string;
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
        setAnomalies(response.anomalies || []);
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
      <Card className="p-6">
        <div className="text-center text-red-600">
          <p className="font-semibold">Error Loading Anomalies</p>
          <p className="text-sm">{error}</p>
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
        <Card className={`p-4 ${criticalCount > 0 ? 'border-red-500 border-2 bg-red-50' : 'border-yellow-500 border-2 bg-yellow-50'}`}>
          <div className="flex items-center gap-4">
            <span className="text-3xl">
              {criticalCount > 0 ? '🚨' : '⚠️'}
            </span>
            <div>
              <p className="font-semibold">
                {criticalCount > 0 ? 'Critical Anomalies Require Attention' : 'High Severity Anomalies'}
              </p>
              <p className="text-sm text-gray-700">
                {criticalCount > 0
                  ? `${criticalCount} critical anomaly(ies) need immediate investigation`
                  : `${highCount} high severity anomaly(ies) pending review`}
              </p>
            </div>
          </div>
        </Card>
      )}

      {/* Filters */}
      <Card className="p-4">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-medium">Filter:</span>
          {['open', 'investigating', 'resolved', 'false_positive'].map((s) => (
            <button
              key={s}
              onClick={() => handleStatusChange(s)}
              className={`px-3 py-1 rounded-full text-sm capitalize ${
                filterStatus === s
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
              }`}
            >
              {s.replace('_', ' ')}
            </button>
          ))}
        </div>
      </Card>

      {/* Anomaly List */}
      {anomalies.length === 0 ? (
        <Card className="p-6">
          <EmptyState message="No anomalies detected" />
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
  <Card className={`p-4 ${
    anomaly.severity === 'critical' ? 'border-red-500 border-2' :
    anomaly.severity === 'high' ? 'border-orange-500 border' : ''
  }`}>
    <div className="flex items-start justify-between">
      <div className="flex-1">
        <div className="flex items-center gap-2 mb-2">
          <SeverityBadge severity={anomaly.severity} />
          <StatusBadge status={anomaly.status} />
          <span className="text-xs text-gray-600 capitalize">
            {anomaly.type.replace(/_/g, ' ')}
          </span>
        </div>

        <p className="text-gray-900 mb-2">{anomaly.description}</p>

        <div className="flex items-center gap-4 text-xs text-gray-600">
          <span>
            Detected: {new Date(anomaly.detected_at).toLocaleString()}
          </span>
          {anomaly.user_id && (
            <span>User ID: {anomaly.user_id.slice(0, 8)}...</span>
          )}
          {anomaly.assigned_to && (
            <span>Assigned to: {anomaly.assigned_to.slice(0, 8)}...</span>
          )}
        </div>

        {anomaly.resolution_notes && (
          <div className="mt-2 p-2 bg-gray-100 rounded text-sm">
            <span className="font-medium">Resolution:</span> {anomaly.resolution_notes}
          </div>
        )}
      </div>

      {anomaly.status === 'open' && (
        <Button onClick={onAssign} size="sm" variant="secondary">
          Assign
        </Button>
      )}
    </div>
  </Card>
);

const SeverityBadge: React.FC<{ severity: string }> = ({ severity }) => {
  const colors: Record<string, string> = {
    critical: 'bg-red-100 text-red-800',
    high: 'bg-orange-100 text-orange-800',
    medium: 'bg-yellow-100 text-yellow-800',
    low: 'bg-blue-100 text-blue-800',
  };

  return (
    <Badge className={colors[severity] || colors.low}>
      {severity.toUpperCase()}
    </Badge>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const colors: Record<string, string> = {
    open: 'bg-red-100 text-red-800',
    investigating: 'bg-yellow-100 text-yellow-800',
    resolved: 'bg-green-100 text-green-800',
    false_positive: 'bg-gray-100 text-gray-800',
    ignored: 'bg-gray-200 text-gray-600',
  };

  return (
    <Badge className={colors[status] || colors.open}>
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

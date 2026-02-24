import React, { useState, useEffect } from 'react';
import { Card } from '../common/Card';
import { LoadingState } from '../common/LoadingState';
import { Badge } from '../common/Badge';
import { analyticsApi } from '../../api/analytics';

interface FrameworkStatus {
  framework: string;
  compliant: number;
  total: number;
  status: 'compliant' | 'non_compliant' | 'partial';
  last_evaluated: string;
}

interface ComplianceSummaryData {
  overall_score: number;
  control_count: number;
  compliant_count: number;
  non_compliant_count: number;
  last_assessed: string;
}

interface ComplianceSummaryProps {
  tenantId: string;
}

export const ComplianceSummary: React.FC<ComplianceSummaryProps> = ({
  tenantId,
}) => {
  const [summary, setSummary] = useState<ComplianceSummaryData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchSummary = async () => {
      setLoading(true);
      setError(null);
      try {
        const response = await analyticsApi.getComplianceSummary();
        setSummary(response.data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load compliance summary');
      } finally {
        setLoading(false);
      }
    };

    fetchSummary();
  }, [tenantId]);

  if (loading) {
    return <LoadingState message="Loading compliance summary..." />;
  }

  if (error) {
    return (
      <Card>
        <div className="card-body">
          <div className="text-center text-danger-400">
            <p className="font-semibold">Error Loading Compliance</p>
            <p className="text-sm text-gray-400">{error}</p>
          </div>
        </div>
      </Card>
    );
  }

  if (!summary) {
    return (
      <Card>
        <div className="card-body">
          <EmptyState message="No compliance data available" />
        </div>
      </Card>
    );
  }

  const overallStatus = getOverallStatus(summary.overall_score);

  return (
    <div className="space-y-6">
      {/* Overall Compliance Score */}
      <Card>
        <div className="card-body">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-semibold text-white">Overall Compliance</h3>
              <p className="text-sm text-gray-400">
                {summary.control_count} controls assessed
              </p>
            </div>
            <div className="flex items-center gap-4">
              <div className="text-right">
                <p className="text-4xl font-bold text-white">{summary.overall_score}%</p>
                <Badge variant={overallStatus === 'compliant' ? 'success' : overallStatus === 'partial' ? 'warning' : 'danger'}>
                  {overallStatus.replace('_', ' ').toUpperCase()}
                </Badge>
              </div>
              <CircularProgress value={summary.overall_score} status={overallStatus} />
            </div>
          </div>
        </div>
      </Card>

      {/* Compliance Details */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <div className="card-body">
            <p className="text-sm text-gray-400">Compliant</p>
            <p className="text-2xl font-semibold text-success-400">{summary.compliant_count}</p>
          </div>
        </Card>
        <Card>
          <div className="card-body">
            <p className="text-sm text-gray-400">Non-Compliant</p>
            <p className="text-2xl font-semibold text-danger-400">{summary.non_compliant_count}</p>
          </div>
        </Card>
        <Card>
          <div className="card-body">
            <p className="text-sm text-gray-400">Last Assessed</p>
            <p className="text-sm font-medium text-white">{new Date(summary.last_assessed).toLocaleDateString()}</p>
          </div>
        </Card>
      </div>
    </div>
  );
};

interface FrameworkCardProps {
  framework: FrameworkStatus;
}

const FrameworkCard: React.FC<FrameworkCardProps> = ({ framework }) => (
  <Card>
    <div className="card-body">
      <div className="flex items-start justify-between mb-3">
        <div>
          <h4 className="font-semibold text-white">{framework.framework}</h4>
          <p className="text-xs text-gray-400">
            Last evaluated: {new Date(framework.last_evaluated).toLocaleDateString()}
          </p>
        </div>
        <Badge variant={
          framework.status === 'compliant' ? 'success' :
          framework.status === 'partial' ? 'warning' : 'danger'
        }>
          {framework.status.replace('_', ' ').toUpperCase()}
        </Badge>
      </div>

      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-gray-300">Compliant Controls</span>
          <span className="font-medium text-white">{framework.compliant} / {framework.total}</span>
        </div>
        <div className="w-full bg-gray-700 rounded-full h-2">
          <div
            className={`h-2 rounded-full ${
              framework.status === 'compliant' ? 'bg-success-500' :
              framework.status === 'partial' ? 'bg-warning-500' : 'bg-danger-500'
            }`}
            style={{ width: `${(framework.compliant / framework.total) * 100}%` }}
          />
        </div>
      </div>
    </div>
  </Card>
);

interface CircularProgressProps {
  value: number;
  status: string;
}

const CircularProgress: React.FC<CircularProgressProps> = ({ value, status }) => {
  const radius = 40;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (value / 100) * circumference;

  const color =
    status === 'compliant' ? '#10b981' : // success-500
    status === 'partial' ? '#f59e0b' :   // warning-500
    '#ef4444';                           // danger-500

  const trackColor = '#374151'; // gray-700

  return (
    <svg width="120" height="120" className="transform -rotate-90">
      <circle
        cx="60"
        cy="60"
        r={radius}
        stroke={trackColor}
        strokeWidth="8"
        fill="none"
      />
      <circle
        cx="60"
        cy="60"
        r={radius}
        stroke={color}
        strokeWidth="8"
        fill="none"
        strokeDasharray={circumference}
        strokeDashoffset={offset}
        strokeLinecap="round"
      />
    </svg>
  );
};

function getOverallStatus(compliance: number): string {
  if (compliance >= 95) return 'compliant';
  if (compliance >= 70) return 'partial';
  return 'non_compliant';
}

interface EmptyStateProps {
  message: string;
}

const EmptyState: React.FC<EmptyStateProps> = ({ message }) => (
  <div className="text-center py-8 text-gray-500">
    <p>{message}</p>
  </div>
);

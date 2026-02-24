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
  frameworks: Record<string, FrameworkStatus>;
  overall_compliance: number;
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
        setSummary(response.summary);
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
      <Card className="p-6">
        <div className="text-center text-red-600">
          <p className="font-semibold">Error Loading Compliance</p>
          <p className="text-sm">{error}</p>
        </div>
      </Card>
    );
  }

  if (!summary || Object.keys(summary.frameworks).length === 0) {
    return (
      <Card className="p-6">
        <EmptyState message="No compliance data available" />
      </Card>
    );
  }

  const overallStatus = getOverallStatus(summary.overall_compliance);

  return (
    <div className="space-y-6">
      {/* Overall Compliance Score */}
      <Card className="p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-semibold">Overall Compliance</h3>
            <p className="text-sm text-gray-600">
              Across {Object.keys(summary.frameworks).length} frameworks
            </p>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-right">
              <p className="text-4xl font-bold">{summary.overall_compliance}%</p>
              <Badge variant={overallStatus === 'compliant' ? 'success' : overallStatus === 'partial' ? 'warning' : 'danger'}>
                {overallStatus.replace('_', ' ').toUpperCase()}
              </Badge>
            </div>
            <CircularProgress value={summary.overall_compliance} status={overallStatus} />
          </div>
        </div>
      </Card>

      {/* Framework Status Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {Object.values(summary.frameworks).map((framework) => (
          <FrameworkCard key={framework.framework} framework={framework} />
        ))}
      </div>
    </div>
  );
};

interface FrameworkCardProps {
  framework: FrameworkStatus;
}

const FrameworkCard: React.FC<FrameworkCardProps> = ({ framework }) => (
  <Card className="p-4">
    <div className="flex items-start justify-between mb-3">
      <div>
        <h4 className="font-semibold">{framework.framework}</h4>
        <p className="text-xs text-gray-600">
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
        <span>Compliant Controls</span>
        <span className="font-medium">{framework.compliant} / {framework.total}</span>
      </div>
      <div className="w-full bg-gray-200 rounded-full h-2">
        <div
          className={`h-2 rounded-full ${
            framework.status === 'compliant' ? 'bg-green-500' :
            framework.status === 'partial' ? 'bg-yellow-500' : 'bg-red-500'
          }`}
          style={{ width: `${(framework.compliant / framework.total) * 100}%` }}
        />
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
    status === 'compliant' ? '#10b981' : // green-500
    status === 'partial' ? '#f59e0b' :   // yellow-500
    '#ef4444';                           // red-500

  return (
    <svg width="120" height="120" className="transform -rotate-90">
      <circle
        cx="60"
        cy="60"
        r={radius}
        stroke="#e5e7eb"
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

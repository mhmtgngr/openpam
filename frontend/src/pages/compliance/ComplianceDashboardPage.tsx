/**
 * ComplianceDashboard Page - Main compliance monitoring dashboard
 * Provides comprehensive view of compliance status across frameworks
 */

import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import {
  ShieldCheck,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  FileText,
  Download,
  RefreshCw,
  Settings,
  Plus,
  TrendingUp,
  Activity,
  Calendar,
  Filter,
} from 'lucide-react';
import { complianceApi, reportsApi } from '@/api/compliance';
import { reportsApi as baseReportsApi } from '@/api/reports';
import { Card, CardHeader } from '@/components/common';
import { Badge, StatusBadge } from '@/components/common';
import { Button } from '@/components/common';
import { Select } from '@/components/common';
import { LoadingState } from '@/components/common';
import { ComplianceGauge, ComplianceBar, ComplianceBreakdown } from '@/components/analytics/ComplianceGauge';
import { AnomalyChart } from '@/components/analytics/AnomalyChart';
import { ExceptionRequestDialog } from '@/components/analytics/ExceptionRequestDialog';
import { toast } from 'react-hot-toast';
import { format } from 'date-fns';
import type { ComplianceFramework, ComplianceControl, ComplianceException } from '@/types';
import clsx from 'clsx';

const frameworkOptions: Array<{ value: ComplianceFramework; label: string }> = [
  { value: 'soc2', label: 'SOC 2 Type II' },
  { value: 'iso27001', label: 'ISO 27001:2022' },
  { value: 'pci_dss', label: 'PCI DSS 4.0' },
  { value: 'hipaa', label: 'HIPAA Security Rule' },
  { value: 'gdpr', label: 'GDPR Compliance' },
  { value: 'nerc_cip', label: 'NERC CIP' },
];

const periodOptions = [
  { value: '30', label: 'Last 30 Days' },
  { value: '90', label: 'Last Quarter' },
  { value: '365', label: 'Last Year' },
];

interface StatCardProps {
  title: string;
  value: string | number;
  total?: number;
  icon: React.ElementType;
  color: 'success' | 'warning' | 'danger';
}

const StatCard: React.FC<StatCardProps> = ({ title, value, total, icon: Icon, color }) => {
  const colorClasses = {
    success: 'text-success-400 bg-success-400/10',
    warning: 'text-warning-400 bg-warning-400/10',
    danger: 'text-danger-400 bg-danger-400/10',
  };

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-gray-400">{title}</p>
            <p className="mt-2 text-2xl font-bold text-white">
              {value}
              {total !== undefined && <span className="text-lg text-gray-500">/{total}</span>}
            </p>
          </div>
          <div className={clsx('rounded-lg p-3', colorClasses[color])}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};

export const ComplianceDashboardPage: React.FC = () => {
  const navigate = useNavigate();

  const [framework, setFramework] = useState<ComplianceFramework>('soc2');
  const [period, setPeriod] = useState(30);
  const [showExceptionDialog, setShowExceptionDialog] = useState(false);

  // Query: Dashboard Data
  const { data: dashboard, isLoading, refetch } = useQuery({
    queryKey: ['complianceDashboard', framework, period],
    queryFn: () => complianceApi.getDashboard(framework),
    refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
  });

  // Query: Anomalies for the chart
  const { data: anomalies } = useQuery({
    queryKey: ['anomalies', period],
    queryFn: () => complianceApi.getAnomalies({
      start_date: new Date(Date.now() - period * 24 * 60 * 60 * 1000).toISOString(),
      end_date: new Date().toISOString(),
    }),
    refetchInterval: 2 * 60 * 1000,
  });

  const controls = dashboard?.controls || [];
  const exceptions = dashboard?.exceptions || [];

  const statusCounts = controls.reduce(
    (acc, control) => {
      acc[control.status] = (acc[control.status] || 0) + 1;
      return acc;
    },
    {} as Record<string, number>
  );

  const overallScore = dashboard?.overall_score || 0;
  const categoryBreakdown: Array<{ name: string; score: number; count: number }> = React.useMemo(() => {
    const categories = new Map<string, { score: number; count: number; total: number }>();
    controls.forEach((control) => {
      const cat = control.category || 'Other';
      const existing = categories.get(cat) || { score: 0, count: 0, total: 0 };
      existing.score += control.score;
      existing.count += 1;
      existing.total += 100;
      categories.set(cat, existing);
    });
    return Array.from(categories.entries()).map(([name, data]) => ({
      name,
      score: Math.round(data.score / data.count),
      count: data.count,
    }));
  }, [controls]);

  const handleRunAssessment = async () => {
    try {
      await complianceApi.runAssessment(framework);
      toast.success('Compliance assessment started');
      refetch();
    } catch (error) {
      // Error handled by interceptor
    }
  };

  const handleGenerateReport = async () => {
    try {
      const result = await reportsApi.generateReport({
        framework,
        period_start: format(new Date(Date.now() - period * 24 * 60 * 60 * 1000), 'yyyy-MM-dd'),
        period_end: format(new Date(), 'yyyy-MM-dd'),
        format: 'pdf',
      });

      if (result.download_url) {
        window.open(result.download_url, '_blank');
      }
    } catch (error) {
      // Error handled by interceptor
    }
  };

  const handleExceptionSubmit = async (data: {
    control_id: string;
    reason: string;
    business_justification: string;
    expires_at?: string;
  }) => {
    try {
      await complianceApi.createException(data);
      toast.success('Exception request submitted');
      refetch();
    } catch (error) {
      // Error handled by interceptor
    }
  };

  if (isLoading) {
    return <LoadingState />;
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Compliance Dashboard</h1>
          <p className="mt-1 text-sm text-gray-400">
            Monitor compliance status across frameworks and manage exceptions
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Select
            options={frameworkOptions}
            value={framework}
            onChange={(e) => setFramework(e.target.value as ComplianceFramework)}
            className="w-48"
          />
          <Button
            variant="secondary"
            leftIcon={<RefreshCw className="h-4 w-4" />}
            onClick={() => refetch()}
          >
            Refresh
          </Button>
          <Button
            variant="primary"
            leftIcon={<FileText className="h-4 w-4" />}
            onClick={handleGenerateReport}
          >
            Generate Report
          </Button>
        </div>
      </div>

      {/* Stats Overview */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Overall Score"
          value={`${overallScore}%`}
          icon={ShieldCheck}
          color={overallScore >= 80 ? 'success' : overallScore >= 60 ? 'warning' : 'danger'}
        />
        <StatCard
          title="Compliant Controls"
          value={statusCounts.compliant || 0}
          total={controls.length}
          icon={CheckCircle}
          color="success"
        />
        <StatCard
          title="Non-Compliant"
          value={statusCounts.non_compliant || 0}
          icon={XCircle}
          color="danger"
        />
        <StatCard
          title="Active Exceptions"
          value={exceptions.length}
          icon={AlertTriangle}
          color="warning"
        />
      </div>

      {/* Main Dashboard Grid */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Compliance Gauge */}
        <Card className="lg:col-span-1">
          <CardHeader title="Compliance Score" icon={ShieldCheck} />
          <div className="card-body flex items-center justify-center">
            <ComplianceGauge score={overallScore} framework={framework} />
          </div>
        </Card>

        {/* Compliance Breakdown by Category */}
        <Card className="lg:col-span-2">
          <CardHeader title="Control Status Breakdown" icon={Activity} />
          <div className="card-body space-y-4">
            <ComplianceBreakdown data={categoryBreakdown} />
            <div className="mt-4 space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-success-500" />
                  Compliant
                </span>
                <span className="font-medium">{statusCounts.compliant || 0}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-warning-500" />
                  Partial
                </span>
                <span className="font-medium">{statusCounts.partial || 0}</span>
              </div>
              <div className="flex items-center justify-between text-sm">
                <span className="flex items-center gap-2">
                  <div className="h-3 w-3 rounded-full bg-danger-500" />
                  Non-Compliant
                </span>
                <span className="font-medium">{statusCounts.non_compliant || 0}</span>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* Controls List */}
      <Card>
        <CardHeader
          title="Controls"
          subtitle={`Showing ${controls.length} controls for ${frameworkOptions.find(f => f.value === framework)?.label}`}
          action={
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => navigate('/compliance/controls')}
            >
              Manage Controls
            </Button>
          }
        />
        <div className="card-body">
          <div className="space-y-2">
            {controls.map((control) => (
              <div
                key={control.id}
                className="flex items-center justify-between rounded-lg border border-gray-700 bg-gray-900/50 p-4 hover:bg-gray-800/50"
              >
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <StatusBadge status={control.status} />
                    <h4 className="font-medium text-white">{control.name}</h4>
                    {control.category && (
                      <Badge variant="neutral">{control.category}</Badge>
                    )}
                  </div>
                  {control.description && (
                    <p className="mt-1 text-sm text-gray-400">{control.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <div className="text-lg font-semibold text-white">{control.score}%</div>
                    <div className="text-xs text-gray-500">{control.evidence_count || 0} evidence</div>
                  </div>
                  <ComplianceBar score={control.score} />
                </div>
              </div>
            ))}
          </div>
        </div>
      </Card>

      {/* Anomalies Chart */}
      {anomalies && anomalies.data && anomalies.data.length > 0 && (
        <Card>
          <CardHeader title="Anomaly Trends" icon={TrendingUp} />
          <div className="card-body">
            <AnomalyChart data={anomalies.data} />
          </div>
        </Card>
      )}

      {/* Exceptions */}
      {exceptions.length > 0 && (
        <Card>
          <CardHeader
            title="Compliance Exceptions"
            subtitle="Approved exceptions and waivers"
            action={
              <Badge variant="warning">{exceptions.length} active</Badge>
            }
          />
          <div className="card-body">
            <div className="space-y-3">
              {exceptions.map((exception) => (
                <div
                  key={exception.id}
                  className="flex items-start justify-between rounded-lg bg-gray-900/50 p-4"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <AlertTriangle className="h-5 w-5 text-warning-400" />
                      <h4 className="font-medium text-white">{exception.control_name}</h4>
                      <Badge variant="warning">{exception.status}</Badge>
                    </div>
                    <p className="mt-1 text-sm text-gray-400">{exception.reason}</p>
                    <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                      <span>Approved by: {exception.approved_by}</span>
                      <span>
                        Expires: {exception.expires_at ?
                          format(new Date(exception.expires_at), 'MMM d, yyyy') : 'Never'}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}

      {/* Exception Dialog */}
      <ExceptionRequestDialog
        isOpen={showExceptionDialog}
        onClose={() => setShowExceptionDialog(false)}
        framework={framework}
        controls={controls}
        onSubmit={handleExceptionSubmit}
      />
    </div>
  );
};

export default ComplianceDashboardPage;

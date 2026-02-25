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
  const categoryBreakdown = React.useMemo(() => {
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
            className="w-40"
          />
          <Select
            options={periodOptions}
            value={String(period)}
            onChange={(e) => setPeriod(Number(e.target.value))}
            className="w-32"
          />
          <Button
            variant="secondary"
            leftIcon={<RefreshCw className="h-4 w-4" />}
            onClick={() => refetch()}
          >
            Refresh
          </Button>
          <Button
            variant="secondary"
            leftIcon={<Download className="h-4 w-4" />}
            onClick={handleGenerateReport}
          >
            Export Report
          </Button>
          <Button
            variant="primary"
            leftIcon={<ShieldCheck className="h-4 w-4" />}
            onClick={handleRunAssessment}
          >
            Run Assessment
          </Button>
        </div>
      </div>

      {/* Overview Stats */}
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
          total={controls.length}
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

      {/* Main Gauge and Category Breakdown */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Overall Score Gauge */}
        <Card>
          <CardHeader title="Compliance Score" />
          <div className="card-body flex justify-center">
            <ComplianceGauge
              score={overallScore}
              size="lg"
              label={`${framework.toUpperCase()} Compliance`}
            />
          </div>
          {dashboard?.last_updated && (
            <div className="card-body border-t border-gray-800 pt-4">
              <p className="text-xs text-gray-500 text-center">
                Last assessed: {format(new Date(dashboard.last_updated), 'MMM d, yyyy HH:mm')}
              </p>
            </div>
          )}
        </Card>

        {/* Category Breakdown */}
        <Card className="lg:col-span-2">
          <CardHeader title="Compliance by Category" />
          <div className="card-body">
            {categoryBreakdown.length > 0 ? (
              <ComplianceBreakdown
                categories={categoryBreakdown.map((cat) => ({
                  name: cat.name,
                  score: cat.score,
                  count: cat.count,
                  total: cat.count,
                }))}
              />
            ) : (
              <div className="text-center text-gray-500 py-8">
                No category data available
              </div>
            )}
          </div>
        </Card>
      </div>

      {/* Anomaly Timeline */}
      {anomalies && anomalies.data.length > 0 && (
        <AnomalyChart
          anomalies={anomalies.data}
          onAnomalyClick={(anomaly) => navigate(`/analytics/anomalies/${anomaly.id}`)}
        />
      )}

      {/* Controls List */}
      <Card>
        <CardHeader
          title="Control Status"
          subtitle="Detailed compliance control assessment"
          action={
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => setShowExceptionDialog(true)}
            >
              Request Exception
            </Button>
          }
        />
        <div className="card-body">
          {isLoading ? (
            <LoadingState message="Loading controls..." />
          ) : controls.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              No controls found for this framework
            </div>
          ) : (
            <div className="space-y-3">
              {controls.map((control) => {
                const statusConfig = {
                  compliant: { variant: 'success' as const, icon: CheckCircle, label: 'Compliant' },
                  non_compliant: { variant: 'danger' as const, icon: XCircle, label: 'Non-Compliant' },
                  partial: { variant: 'warning' as const, icon: AlertTriangle, label: 'Partial' },
                  not_applicable: { variant: 'neutral' as const, icon: Clock, label: 'N/A' },
                };
                const config = statusConfig[control.status] || statusConfig.not_applicable;
                const StatusIcon = config.icon;

                return (
                  <div
                    key={control.id}
                    className="flex items-start justify-between rounded-lg border border-gray-800 p-4 hover:bg-gray-800/30 transition-colors"
                  >
                    <div className="flex items-start gap-4 flex-1">
                      <div className={clsx('rounded-lg p-2', {
                        'bg-success-500/20 text-success-400': control.status === 'compliant',
                        'bg-danger-500/20 text-danger-400': control.status === 'non_compliant',
                        'bg-warning-500/20 text-warning-400': control.status === 'partial',
                        'bg-gray-500/20 text-gray-400': control.status === 'not_applicable',
                      })}>
                        <StatusIcon className="h-5 w-5" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h4 className="font-medium text-white">{control.name}</h4>
                          <Badge variant={config.variant}>{config.label}</Badge>
                          {control.category && (
                            <Badge variant="neutral" className="text-xs">{control.category}</Badge>
                          )}
                        </div>
                        <p className="mt-1 text-sm text-gray-400">{control.description}</p>
                        <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                          <span>Score: {control.score}%</span>
                          <span>•</span>
                          <span>{control.evidence_count} evidence items</span>
                          <span>•</span>
                          <span>
                            Assessed: {format(new Date(control.last_assessed_at), 'MMM d, yyyy')}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="ml-4 text-right">
                      <ComplianceGauge score={control.score} size="sm" showLabel={false} />
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </Card>

      {/* Active Exceptions */}
      {exceptions.length > 0 && (
        <Card>
          <CardHeader
            title="Active Exceptions"
            subtitle="Temporary compliance exceptions"
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

export default ComplianceDashboardPage;

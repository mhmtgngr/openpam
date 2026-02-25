import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
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
  AlertCircle,
} from 'lucide-react';
import { complianceApi, type ComplianceFramework } from '@/api/compliance';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge, StatusBadge } from '@/components/common';
import { Button } from '@/components/common';
import { Select } from '@/components/common';
import { LoadingState, EmptyState } from '@/components/common';
import { Modal } from '@/components/common';
import { Textarea } from '@/components/common';
import { Input } from '@/components/common';
import { DatePicker } from '@/components/common';
import type { ComplianceControl, ComplianceException } from '@/types';
import clsx from 'clsx';

const frameworkOptions = [
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'pci_dss', label: 'PCI DSS' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'gdpr', label: 'GDPR' },
  { value: 'custom', label: 'Custom' },
];

const statusConfig: Record<string, { variant: 'success' | 'danger' | 'warning' | 'neutral'; icon: React.ElementType; label: string }> = {
  compliant: { variant: 'success', icon: CheckCircle, label: 'Compliant' },
  non_compliant: { variant: 'danger', icon: XCircle, label: 'Non-Compliant' },
  partial: { variant: 'warning', icon: AlertTriangle, label: 'Partial' },
  not_applicable: { variant: 'neutral', icon: Clock, label: 'N/A' },
};

interface ComplianceScoreProps {
  score: number;
  label: string;
  size?: 'sm' | 'md' | 'lg';
}

const ComplianceScore: React.FC<ComplianceScoreProps> = ({ score, label, size = 'md' }) => {
  const sizeClasses = {
    sm: { circle: 'h-20 w-20', text: 'text-lg', label: 'text-xs' },
    md: { circle: 'h-32 w-32', text: 'text-3xl', label: 'text-sm' },
    lg: { circle: 'h-48 w-48', text: 'text-4xl', label: 'text-base' },
  };

  const getColor = (score: number): string => {
    if (score >= 80) return 'text-success-400';
    if (score >= 60) return 'text-warning-400';
    return 'text-danger-400';
  };

  const getTrackColor = (score: number): string => {
    if (score >= 80) return 'stroke-success-500';
    if (score >= 60) return 'stroke-warning-500';
    return 'stroke-danger-500';
  };

  const circumference = 2 * Math.PI * 54;
  const strokeDashoffset = circumference - (score / 100) * circumference;

  return (
    <div className="flex flex-col items-center">
      <div className={clsx('relative', sizeClasses[size].circle)}>
        <svg className="h-full w-full rotate-[-90deg]" viewBox="0 0 120 120">
          <circle
            cx="60"
            cy="60"
            r="54"
            fill="none"
            className="stroke-gray-700"
            strokeWidth="8"
          />
          <circle
            cx="60"
            cy="60"
            r="54"
            fill="none"
            className={getTrackColor(score)}
            strokeWidth="8"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
          />
        </svg>
        <div className="absolute inset-0 flex items-center justify-center">
          <span className={clsx('font-bold', sizeClasses[size].text, getColor(score))}>
            {score}%
          </span>
        </div>
      </div>
      <span className={clsx('mt-2 text-gray-400', sizeClasses[size].label)}>{label}</span>
    </div>
  );
};

interface ExceptionModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: { control_id: string; reason: string; expires_at?: string }) => void;
  controls: ComplianceControl[];
}

const ExceptionModal: React.FC<ExceptionModalProps> = ({ isOpen, onClose, onSubmit, controls }) => {
  const [controlId, setControlId] = useState('');
  const [reason, setReason] = useState('');
  const [expiresAt, setExpiresAt] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({
      control_id: controlId,
      reason,
      expires_at: expiresAt || undefined,
    });
    onClose();
  };

  const nonCompliantControls = controls.filter((c) => c.status === 'non_compliant' || c.status === 'partial');

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Request Compliance Exception">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-300">Control</label>
          <Select
            options={[
              { value: '', label: 'Select a control...' },
              ...nonCompliantControls.map((c) => ({ value: c.id, label: `${c.name} - ${c.status}` })),
            ]}
            value={controlId}
            onChange={(e) => setControlId(e.target.value)}
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-300">Reason</label>
          <Textarea
            placeholder="Explain why an exception is needed..."
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            rows={3}
            required
          />
        </div>
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-300">Expiration Date (Optional)</label>
          <Input
            type="date"
            value={expiresAt}
            onChange={(e) => setExpiresAt(e.target.value)}
          />
        </div>
        <div className="flex justify-end gap-3">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary">
            Submit Request
          </Button>
        </div>
      </form>
    </Modal>
  );
};

export const ComplianceDashboard: React.FC = () => {
  const [framework, setFramework] = useState<ComplianceFramework>('soc2');
  const [showExceptionModal, setShowExceptionModal] = useState(false);

  const { data: dashboard, isLoading, refetch } = useQuery({
    queryKey: ['complianceDashboard', framework],
    queryFn: () =>
      complianceApi.getDashboard(framework),
    refetchInterval: 15 * 60 * 1000, // Refresh every 15 minutes
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

  const handleRunAssessment = async () => {
    try {
      await complianceApi.runAssessment(framework);
      // Show success toast and refresh
      refetch();
    } catch {
      // Error handled by interceptor
    }
  };

  const handleGenerateReport = async () => {
    try {
      const result = await complianceApi.generateReport({
        framework,
        period_start: format(new Date(Date.now() - 90 * 24 * 60 * 60 * 1000), 'yyyy-MM-dd'),
        period_end: format(new Date(), 'yyyy-MM-dd'),
        format: 'pdf',
      });

      if (result.download_url) {
        window.open(result.download_url, '_blank');
      }
    } catch {
      // Error handled by interceptor
    }
  };

  const handleExceptionSubmit = async (data: { control_id: string; reason: string; expires_at?: string }) => {
    try {
      await complianceApi.createException(data);
      refetch();
    } catch {
      // Error handled by interceptor
    }
  };

  if (isLoading) {
    return (
      <div className="card">
        <div className="card-body">
          <LoadingState message="Loading compliance dashboard..." />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <CardHeader
        title="Compliance Dashboard"
        subtitle="Track compliance status and manage exceptions"
        action={
          <div className="flex items-center gap-3">
            <Select
              options={frameworkOptions}
              value={framework}
              onChange={(e) => setFramework(e.target.value as ComplianceFramework)}
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
              Report
            </Button>
            <Button
              variant="primary"
              leftIcon={<ShieldCheck className="h-4 w-4" />}
              onClick={handleRunAssessment}
            >
              Run Assessment
            </Button>
          </div>
        }
      />

      {/* Overall Score */}
      <Card>
        <div className="card-body">
          <div className="flex flex-col items-center md:flex-row md:items-start md:justify-around">
            <ComplianceScore
              score={dashboard?.overall_score || 0}
              label="Overall Compliance"
              size="lg"
            />

            {/* Status Breakdown */}
            <div className="mt-6 grid grid-cols-2 gap-4 md:mt-0">
              <div className="text-center">
                <p className="text-3xl font-bold text-success-400">{statusCounts.compliant || 0}</p>
                <p className="text-sm text-gray-400">Compliant</p>
              </div>
              <div className="text-center">
                <p className="text-3xl font-bold text-warning-400">{statusCounts.partial || 0}</p>
                <p className="text-sm text-gray-400">Partial</p>
              </div>
              <div className="text-center">
                <p className="text-3xl font-bold text-danger-400">{statusCounts.non_compliant || 0}</p>
                <p className="text-sm text-gray-400">Non-Compliant</p>
              </div>
              <div className="text-center">
                <p className="text-3xl font-bold text-gray-400">{statusCounts.not_applicable || 0}</p>
                <p className="text-sm text-gray-400">N/A</p>
              </div>
            </div>
          </div>
        </div>
        {dashboard?.last_updated && (
          <CardFooter className="text-sm text-gray-500">
            Last updated: {format(new Date(dashboard.last_updated), 'MMM d, yyyy HH:mm')}
          </CardFooter>
        )}
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
                      <AlertCircle className="h-5 w-5 text-warning-400" />
                      <h4 className="font-medium text-white">{exception.control_name}</h4>
                      <Badge variant="warning">{exception.status}</Badge>
                    </div>
                    <p className="mt-1 text-sm text-gray-400">{exception.reason}</p>
                    <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                      <span>Approved by: {exception.approved_by}</span>
                      <span>
                        Expires: {exception.expires_at ? format(new Date(exception.expires_at), 'MMM d, yyyy') : 'Never'}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </Card>
      )}

      {/* Controls List */}
      <Card>
        <CardHeader
          title="Controls"
          subtitle="Detailed compliance control status"
          action={
            <Button
              variant="secondary"
              leftIcon={<Plus className="h-4 w-4" />}
              onClick={() => setShowExceptionModal(true)}
              disabled={!controls.some((c) => c.status === 'non_compliant' || c.status === 'partial')}
            >
              Request Exception
            </Button>
          }
        />
        <div className="card-body">
          {controls.length === 0 ? (
            <EmptyState title="No controls available for this framework" />
          ) : (
            <div className="space-y-4">
              {controls.map((control) => {
                const config = statusConfig[control.status] || statusConfig.not_applicable;
                const StatusIcon = config.icon;

                return (
                  <div
                    key={control.id}
                    className="flex items-start justify-between rounded-lg border border-gray-800 p-4 hover:bg-gray-800/30"
                  >
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <div className={clsx('rounded-lg p-2', {
                          'bg-success-500/20 text-success-400': control.status === 'compliant',
                          'bg-danger-500/20 text-danger-400': control.status === 'non_compliant',
                          'bg-warning-500/20 text-warning-400': control.status === 'partial',
                          'bg-gray-500/20 text-gray-400': control.status === 'not_applicable',
                        })}>
                          <StatusIcon className="h-5 w-5" />
                        </div>
                        <div>
                          <h4 className="font-medium text-white">{control.name}</h4>
                          <p className="mt-1 text-sm text-gray-400">{control.description}</p>
                          {control.category && (
                            <Badge variant="neutral" className="mt-2 text-xs">
                              {control.category}
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="ml-4 flex flex-col items-end gap-2">
                      <Badge variant={config.variant}>{config.label}</Badge>
                      <div className="text-right">
                        <p className="text-lg font-semibold text-white">{control.score}%</p>
                        <p className="text-xs text-gray-500">
                          {control.evidence_count} evidence items
                        </p>
                      </div>
                      <p className="text-xs text-gray-500">
                        Assessed: {format(new Date(control.last_assessed_at), 'MMM d, yyyy')}
                      </p>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </Card>

      {/* Exception Modal */}
      <ExceptionModal
        isOpen={showExceptionModal}
        onClose={() => setShowExceptionModal(false)}
        onSubmit={handleExceptionSubmit}
        controls={controls}
      />
    </div>
  );
};

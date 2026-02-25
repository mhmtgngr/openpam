import React, { useState, useEffect } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useParams, useNavigate } from 'react-router-dom';
import { format } from 'date-fns';
import {
  AlertTriangle,
  AlertOctagon,
  AlertCircle,
  Activity,
  ChevronLeft,
  Eye,
  CheckCircle,
  XCircle,
  Search,
  User,
  Server,
  Clock,
  Calendar,
  FileText,
  Save,
  Loader2,
} from 'lucide-react';
import { anomalyApi, type AnomalyUpdateData } from '@/api/anomaly';
import { Card, CardHeader } from '@/components/common';
import { Badge, Button, Input, Textarea } from '@/components/common';
import { LoadingState } from '@/components/common';
import type { AnomalyDetection, AnomalySeverity } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const severityConfig: Record<AnomalySeverity, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType; color: string; bgClass: string }> = {
  critical: { variant: 'danger', icon: AlertOctagon, color: 'text-danger-400', bgClass: 'bg-danger-500/20' },
  high: { variant: 'danger', icon: AlertTriangle, color: 'text-danger-400', bgClass: 'bg-danger-500/20' },
  medium: { variant: 'warning', icon: AlertCircle, color: 'text-warning-400', bgClass: 'bg-warning-500/20' },
  low: { variant: 'neutral', icon: Activity, color: 'text-gray-400', bgClass: 'bg-gray-500/20' },
};

const statusConfig: Record<string, { variant: 'danger' | 'warning' | 'success' | 'neutral'; icon: React.ElementType; label: string }> = {
  open: { variant: 'danger', icon: AlertCircle, label: 'Open' },
  investigating: { variant: 'warning', icon: Eye, label: 'Investigating' },
  resolved: { variant: 'success', icon: CheckCircle, label: 'Resolved' },
  false_positive: { variant: 'neutral', icon: XCircle, label: 'False Positive' },
};

const statusOptions = [
  { value: 'open', label: 'Open' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'false_positive', label: 'False Positive' },
];

const typeLabels: Record<string, string> = {
  unusual_access_time: 'Unusual Access Time',
  unusual_location: 'Unusual Location',
  privileged_escalation: 'Privileged Escalation',
  bulk_data_access: 'Bulk Data Access',
  command_injection: 'Command Injection',
  ransomware_indicators: 'Ransomware Indicators',
  impossible_travel: 'Impossible Travel',
  account_takeover: 'Account Takeover',
  credential_theft: 'Credential Theft',
  excessive_failed_logins: 'Excessive Failed Logins',
};

interface DetailRowProps {
  icon: React.ElementType;
  label: string;
  value: React.ReactNode;
}

const DetailRow: React.FC<DetailRowProps> = ({ icon: Icon, label, value }) => (
  <div className="flex items-start gap-3">
    <Icon className="mt-0.5 h-4 w-4 text-gray-500" />
    <div className="flex-1">
      <p className="text-xs text-gray-500">{label}</p>
      <p className="text-sm text-white">{value}</p>
    </div>
  </div>
);

interface IndicatorCardProps {
  indicator: {
    type: string;
    description: string;
    value: number | string;
    threshold?: number | string;
    confidence: number;
  };
}

const IndicatorCard: React.FC<IndicatorCardProps> = ({ indicator }) => {
  const confidencePercent = Math.round(indicator.confidence * 100);
  const barColor =
    confidencePercent >= 80 ? 'bg-danger-500' :
    confidencePercent >= 60 ? 'bg-warning-500' :
    'bg-success-500';

  return (
    <div className="rounded-lg bg-gray-900/50 border border-gray-800 p-4">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <p className="text-sm font-medium text-white">{indicator.type}</p>
          <p className="mt-1 text-xs text-gray-400">{indicator.description}</p>
        </div>
        <div className="ml-4 text-right">
          <p className="text-lg font-semibold text-white">
            {typeof indicator.value === 'number' ? indicator.value.toFixed(2) : indicator.value}
          </p>
          {indicator.threshold !== undefined && (
            <p className="text-xs text-gray-500">Threshold: {indicator.threshold}</p>
          )}
        </div>
      </div>
      <div className="mt-3">
        <div className="mb-1 flex items-center justify-between">
          <span className="text-xs text-gray-500">Confidence</span>
          <span className="text-xs font-medium text-white">{confidencePercent}%</span>
        </div>
        <div className="h-2 w-full overflow-hidden rounded-full bg-gray-800">
          <div
            className={clsx('h-full rounded-full transition-all', barColor)}
            style={{ width: `${confidencePercent}%` }}
          />
        </div>
      </div>
    </div>
  );
};

export const AnomalyDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  // Form state
  const [status, setStatus] = useState<'open' | 'investigating' | 'resolved' | 'false_positive'>('open');
  const [assignedTo, setAssignedTo] = useState('');
  const [resolutionNotes, setResolutionNotes] = useState('');
  const [isEditing, setIsEditing] = useState(false);

  // Fetch anomaly details
  const {
    data: anomalyResponse,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['anomaly', id],
    queryFn: () => anomalyApi.get(id || ''),
    enabled: !!id,
  });

  const anomaly = anomalyResponse;

  // Update form state when anomaly data changes
  useEffect(() => {
    if (anomaly) {
      setStatus(anomaly.status);
      setAssignedTo(anomaly.assigned_to || '');
      setResolutionNotes(anomaly.resolution_notes || '');
    }
  }, [anomaly]);

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: (data: AnomalyUpdateData) =>
      anomalyApi.update(id || '', data),
    onSuccess: () => {
      toast.success('Anomaly updated successfully');
      setIsEditing(false);
      queryClient.invalidateQueries({ queryKey: ['anomaly', id] });
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
      queryClient.invalidateQueries({ queryKey: ['anomalySummary'] });
    },
    onError: () => {
      toast.error('Failed to update anomaly');
    },
  });

  // Acknowledge mutation
  const acknowledgeMutation = useMutation({
    mutationFn: () => anomalyApi.acknowledge(id || ''),
    onSuccess: () => {
      toast.success('Anomaly acknowledged - investigation started');
      queryClient.invalidateQueries({ queryKey: ['anomaly', id] });
      queryClient.invalidateQueries({ queryKey: ['anomalies'] });
    },
    onError: () => {
      toast.error('Failed to acknowledge anomaly');
    },
  });

  const handleSave = () => {
    if (!anomaly) return;

    const data: AnomalyUpdateData = {};
    if (status !== anomaly.status) data.status = status;
    if (assignedTo !== (anomaly.assigned_to || '')) data.assigned_to = assignedTo || undefined;
    if (resolutionNotes !== (anomaly.resolution_notes || '')) data.resolution_notes = resolutionNotes;

    if (Object.keys(data).length > 0) {
      updateMutation.mutate(data);
    } else {
      setIsEditing(false);
    }
  };

  const handleAcknowledge = () => {
    acknowledgeMutation.mutate();
  };

  const handleCancel = () => {
    if (anomaly) {
      setStatus(anomaly.status);
      setAssignedTo(anomaly.assigned_to || '');
      setResolutionNotes(anomaly.resolution_notes || '');
    }
    setIsEditing(false);
  };

  if (isLoading) {
    return (
      <div className="flex min-h-[400px] items-center justify-center">
        <LoadingState message="Loading anomaly details..." />
      </div>
    );
  }

  if (error || !anomaly) {
    return (
      <div className="flex min-h-[400px] flex-col items-center justify-center">
        <AlertOctagon className="h-16 w-16 text-gray-600" />
        <h2 className="mt-4 text-lg font-medium text-white">Failed to load anomaly</h2>
        <p className="mt-2 text-sm text-gray-400">The anomaly could not be found or an error occurred</p>
        <Button
          variant="primary"
          className="mt-4"
          onClick={() => navigate(-1)}
        >
          Go Back
        </Button>
      </div>
    );
  }

  const severityInfo = severityConfig[anomaly.severity as AnomalySeverity];
  const StatusIcon = statusConfig[anomaly.status].icon;
  const SeverityIcon = severityInfo.icon;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={() => navigate(-1)}
            className="rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-800 hover:text-white"
          >
            <ChevronLeft className="h-5 w-5" />
          </button>
          <div className={clsx('rounded-lg p-3', severityInfo.bgClass, severityInfo.color)}>
            <SeverityIcon className="h-6 w-6" />
          </div>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-xl font-bold text-white">{anomaly.title}</h1>
              <Badge variant={severityInfo.variant}>{anomaly.severity}</Badge>
              <Badge variant={statusConfig[anomaly.status].variant}>
                <StatusIcon className="mr-1 h-3 w-3" />
                {statusConfig[anomaly.status].label}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-gray-400">{anomaly.description}</p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          {anomaly.status === 'open' && (
            <Button
              variant="secondary"
              size="sm"
              leftIcon={<Search className="h-4 w-4" />}
              onClick={handleAcknowledge}
              disabled={acknowledgeMutation.isPending}
            >
              {acknowledgeMutation.isPending ? 'Acknowledging...' : 'Acknowledge'}
            </Button>
          )}
          {!isEditing ? (
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setIsEditing(true)}
            >
              Update Status
            </Button>
          ) : (
            <div className="flex items-center gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={handleCancel}
                disabled={updateMutation.isPending}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                size="sm"
                leftIcon={updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save className="h-4 w-4" />}
                onClick={handleSave}
                disabled={updateMutation.isPending}
              >
                Save Changes
              </Button>
            </div>
          )}
        </div>
      </div>

      {/* Main Details Grid */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Left Column - Details */}
        <div className="space-y-6 lg:col-span-2">
          {/* Basic Information */}
          <Card>
            <CardHeader title="Basic Information" />
            <div className="card-body space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <DetailRow
                  icon={Calendar}
                  label="Detected At"
                  value={format(new Date(anomaly.detected_at), 'MMM d, yyyy HH:mm:ss')}
                />
                <DetailRow
                  icon={Activity}
                  label="Confidence Score"
                  value={`${Math.round(anomaly.confidence_score * 100)}%`}
                />
                {anomaly.user_id && (
                  <DetailRow
                    icon={User}
                    label="User ID"
                    value={anomaly.user_id}
                  />
                )}
                {anomaly.target_id && (
                  <DetailRow
                    icon={Server}
                    label="Target ID"
                    value={anomaly.target_id}
                  />
                )}
                {anomaly.session_id && (
                  <DetailRow
                    icon={Server}
                    label="Session ID"
                    value={anomaly.session_id}
                  />
                )}
              </div>

              {anomaly.resolved_at && (
                <div className="mt-4 rounded-lg bg-success-500/10 border border-success-500/30 p-3">
                  <p className="text-sm font-medium text-success-400">
                    Resolved on {format(new Date(anomaly.resolved_at), 'MMM d, yyyy HH:mm')}
                  </p>
                </div>
              )}
            </div>
          </Card>

          {/* Indicators */}
          {anomaly.indicators && anomaly.indicators.length > 0 && (
            <Card>
              <CardHeader title="Detection Indicators" />
              <div className="card-body space-y-3">
                {anomaly.indicators.map((indicator, idx: number) => (
                  <IndicatorCard key={idx} indicator={indicator} />
                ))}
              </div>
            </Card>
          )}
        </div>

        {/* Right Column - Status & Actions */}
        <div className="space-y-6">
          {/* Status Update */}
          <Card>
            <CardHeader title="Status & Assignment" />
            <div className="card-body space-y-4">
              <div>
                <label className="mb-2 block text-sm font-medium text-gray-300">Status</label>
                <select
                  value={status}
                  onChange={(e) => setStatus(e.target.value as any)}
                  disabled={!isEditing}
                  className="w-full rounded-lg border border-gray-700 bg-gray-900 px-3 py-2 text-sm text-white focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-50"
                >
                  {statusOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="mb-2 block text-sm font-medium text-gray-300">Assigned To</label>
                <Input
                  placeholder="Email or user ID"
                  value={assignedTo}
                  onChange={(e) => setAssignedTo(e.target.value)}
                  disabled={!isEditing}
                />
              </div>

              <div>
                <label className="mb-2 block text-sm font-medium text-gray-300">Resolution Notes</label>
                <Textarea
                  placeholder="Add investigation notes or resolution details..."
                  value={resolutionNotes}
                  onChange={(e) => setResolutionNotes(e.target.value)}
                  rows={5}
                  disabled={!isEditing}
                />
              </div>
            </div>
          </Card>

          {/* Related Information */}
          {(anomaly.user_name || anomaly.target_name) && (
            <Card>
              <CardHeader title="Related Entities" />
              <div className="card-body space-y-3">
                {anomaly.user_name && (
                  <div className="flex items-center gap-3 rounded-lg bg-gray-900/50 p-3">
                    <div className="rounded-full bg-primary-500/20 p-2">
                      <User className="h-4 w-4 text-primary-400" />
                    </div>
                    <div>
                      <p className="text-xs text-gray-500">User</p>
                      <p className="text-sm font-medium text-white">{anomaly.user_name}</p>
                    </div>
                  </div>
                )}
                {anomaly.target_name && (
                  <div className="flex items-center gap-3 rounded-lg bg-gray-900/50 p-3">
                    <div className="rounded-full bg-warning-500/20 p-2">
                      <Server className="h-4 w-4 text-warning-400" />
                    </div>
                    <div>
                      <p className="text-xs text-gray-500">Target</p>
                      <p className="text-sm font-medium text-white">{anomaly.target_name}</p>
                    </div>
                  </div>
                )}
                {anomaly.session_id && (
                  <Button
                    variant="secondary"
                    size="sm"
                    className="w-full"
                    onClick={() => navigate(`/sessions/${anomaly.session_id}`)}
                  >
                    View Session Recording
                  </Button>
                )}
              </div>
            </Card>
          )}

          {/* Type Information */}
          <Card>
            <CardHeader title="Anomaly Type" />
            <div className="card-body">
              <p className="text-sm font-medium text-white">
                {typeLabels[anomaly.type] || anomaly.type}
              </p>
              <p className="mt-2 text-xs text-gray-500">
                Type code: <code className="rounded bg-gray-800 px-1 py-0.5">{anomaly.type}</code>
              </p>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default AnomalyDetailPage;

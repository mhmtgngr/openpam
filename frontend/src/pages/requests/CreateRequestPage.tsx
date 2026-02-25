import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useMutation, useQuery } from '@tanstack/react-query';
import { ArrowLeft, Send, Clock, Info } from 'lucide-react';
import { requestsApi } from '@/api/requests';
import { targetsApi } from '@/api/targets';
import { credentialsApi } from '@/api/credentials';
import { Button, Input, Select, Textarea, Card, DatePicker, Toggle } from '@/components/common';
import type { Target, Credential } from '@/types';
import toast from 'react-hot-toast';

const requestTypes = [
  { value: 'session_access', label: 'Session Access' },
  { value: 'credential_checkout', label: 'Credential Checkout' },
  { value: 'elevated_privileges', label: 'Elevated Privileges' },
];

const durationOptions = [
  { value: '15', label: '15 minutes' },
  { value: '30', label: '30 minutes' },
  { value: '60', label: '1 hour' },
  { value: '120', label: '2 hours' },
  { value: '240', label: '4 hours' },
  { value: '480', label: '8 hours' },
];

export const CreateRequestPage: React.FC = () => {
  const navigate = useNavigate();

  const [formData, setFormData] = useState<{
    target_id: string;
    credential_id: string;
    type: 'session_access' | 'credential_checkout' | 'elevated_privileges';
    reason: string;
    duration_minutes: number;
    scheduled_start_at: string;
  }>({
    target_id: '',
    credential_id: '',
    type: 'session_access',
    reason: '',
    duration_minutes: 60,
    scheduled_start_at: '',
  });

  const [isScheduled, setIsScheduled] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch targets
  const { data: targets } = useQuery({
    queryKey: ['targets'],
    queryFn: () => targetsApi.list({ limit: 100, status: 'online' }),
  });

  // Fetch credentials for selected target
  const { data: credentials } = useQuery({
    queryKey: ['credentials', formData.target_id],
    queryFn: () =>
      credentialsApi.list({ target_id: formData.target_id, limit: 50 }),
    enabled: !!formData.target_id,
  });

  // Get selected target details
  const selectedTarget = targets?.data.find((t) => t.id === formData.target_id);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: requestsApi.create,
    onSuccess: (data) => {
      toast.success('Access request created successfully');
      navigate(`/requests/${data.data.id}`);
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.target_id) {
      newErrors.target_id = 'Target is required';
    }
    if (formData.type === 'credential_checkout' && !formData.credential_id) {
      newErrors.credential_id = 'Credential is required for checkout requests';
    }
    if (!formData.reason.trim()) {
      newErrors.reason = 'Reason is required';
    }
    if (formData.reason.length < 10) {
      newErrors.reason = 'Reason must be at least 10 characters';
    }
    if (isScheduled && !formData.scheduled_start_at) {
      newErrors.scheduled_start_at = 'Scheduled start time is required';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    const submitData = {
      target_id: formData.target_id,
      credential_id: formData.credential_id || undefined,
      type: formData.type,
      reason: formData.reason,
      duration_minutes: formData.duration_minutes,
      scheduled_start_at: isScheduled ? formData.scheduled_start_at : undefined,
    };

    createMutation.mutate(submitData);
  };

  const handleTypeChange = (type: string) => {
    setFormData({ ...formData, type: type as any });
  };

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/requests/my">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-white">Request Access</h1>
          <p className="mt-1 text-sm text-gray-400">
            Submit a request for privileged access
          </p>
        </div>
      </div>

      {/* Target Requirement Info */}
      {selectedTarget && (
        <Card>
          <div className="card-body bg-info-400/10 border border-info-400/30">
            <div className="flex gap-3">
              <Info className="h-5 w-5 text-info-400 flex-shrink-0 mt-0.5" />
              <div className="flex-1">
                <h4 className="text-sm font-semibold text-info-300">
                  Access Requirements for {selectedTarget.name}
                </h4>
                <div className="mt-2 grid grid-cols-2 gap-2 text-xs text-info-200/80">
                  <div>
                    <span className="font-medium">Environment:</span>{' '}
                    {selectedTarget.environment}
                  </div>
                  <div>
                    <span className="font-medium">Sensitivity:</span>{' '}
                    {selectedTarget.sensitivity}
                  </div>
                  <div>
                    <span className="font-medium">Approval Required:</span>{' '}
                    {selectedTarget.require_approval ? 'Yes' : 'No'}
                  </div>
                  <div>
                    <span className="font-medium">MFA Required:</span>{' '}
                    {selectedTarget.require_mfa ? 'Yes' : 'No'}
                  </div>
                  <div>
                    <span className="font-medium">Session Recording:</span>{' '}
                    {selectedTarget.allow_recording ? 'Yes' : 'No'}
                  </div>
                  <div>
                    <span className="font-medium">Max Duration:</span>{' '}
                    {selectedTarget.max_session_duration} minutes
                  </div>
                </div>
                {selectedTarget.require_approval && selectedTarget.approver_ids.length > 0 && (
                  <p className="mt-2 text-xs text-info-200/80">
                    <span className="font-medium">Approvers:</span> This request will be sent to{' '}
                    {selectedTarget.approver_ids.length} approver(s)
                  </p>
                )}
              </div>
            </div>
          </div>
        </Card>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Request Details */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Request Details</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Request Type
                </label>
                <div className="grid grid-cols-3 gap-3">
                  {requestTypes.map((type) => (
                    <button
                      key={type.value}
                      type="button"
                      onClick={() => handleTypeChange(type.value)}
                      className={clsx(
                        'p-3 rounded-lg border text-left transition-colors',
                        formData.type === type.value
                          ? 'border-primary-500 bg-primary-600/20'
                          : 'border-gray-700 bg-gray-800 hover:bg-gray-700'
                      )}
                    >
                      <p className="text-sm font-medium text-white">{type.label}</p>
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <Select
                  label="Target"
                  options={[
                    { value: '', label: 'Select a target...' },
                    ...(targets?.data.map((t) => ({
                      value: t.id,
                      label: `${t.name} (${t.host}:${t.port})`,
                    })) || []),
                  ]}
                  value={formData.target_id}
                  onChange={(e) => {
                    setFormData({
                      ...formData,
                      target_id: e.target.value,
                      credential_id: '',
                    });
                  }}
                  error={errors.target_id}
                  required
                />
              </div>

              {formData.type === 'credential_checkout' && (
                <div>
                  <Select
                    label="Credential"
                    options={[
                      { value: '', label: 'Select a credential...' },
                      ...(credentials?.data
                        .filter((c) => c.checkout_enabled)
                        .map((c) => ({
                          value: c.id,
                          label: `${c.name} (${c.username})`,
                        })) || []),
                    ]}
                    value={formData.credential_id}
                    onChange={(e) => setFormData({ ...formData, credential_id: e.target.value })}
                    error={errors.credential_id}
                    required={formData.type === 'credential_checkout'}
                  />
                  {!credentials || credentials.data.length === 0 ? (
                    <p className="mt-1 text-xs text-gray-400">
                      No credentials available for checkout for this target
                    </p>
                  ) : null}
                </div>
              )}

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Duration
                </label>
                <div className="grid grid-cols-6 gap-2">
                  {durationOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() =>
                        setFormData({
                          ...formData,
                          duration_minutes: parseInt(option.value),
                        })
                      }
                      className={clsx(
                        'py-2 px-3 rounded-md text-sm font-medium transition-colors',
                        formData.duration_minutes === parseInt(option.value)
                          ? 'bg-primary-600 text-white'
                          : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                      )}
                    >
                      {option.label}
                    </button>
                  ))}
                </div>
                {selectedTarget && formData.duration_minutes > selectedTarget.max_session_duration && (
                  <p className="mt-1 text-xs text-warning-400">
                    Warning: Requested duration exceeds the maximum allowed duration of{' '}
                    {selectedTarget.max_session_duration} minutes for this target
                  </p>
                )}
              </div>

              <Toggle
                label="Schedule for Later"
                description="Set a specific start time for this access request"
                checked={isScheduled}
                onChange={(e) => setIsScheduled(e.target.checked)}
              />

              {isScheduled && (
                <DatePicker
                  label="Scheduled Start Time"
                  value={formData.scheduled_start_at}
                  onChange={(e) => setFormData({ ...formData, scheduled_start_at: e.target.value })}
                  error={errors.scheduled_start_at}
                  min={new Date().toISOString().slice(0, 16)}
                />
              )}

              <div>
                <Textarea
                  label="Reason for Access"
                  placeholder="Please provide a detailed reason for this access request..."
                  value={formData.reason}
                  onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
                  error={errors.reason}
                  rows={4}
                  required
                />
                <p className="mt-1 text-xs text-gray-400">
                  Minimum 10 characters. This will be reviewed by approvers.
                </p>
              </div>
            </div>
          </div>
        </Card>

        {/* Submit Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/requests/my">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending}
            leftIcon={<Send className="h-4 w-4" />}
          >
            Submit Request
          </Button>
        </div>
      </form>
    </div>
  );
};

function clsx(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(' ');
}

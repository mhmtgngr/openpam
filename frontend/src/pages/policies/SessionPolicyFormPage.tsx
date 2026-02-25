import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Clock, Plus, X, Loader2 } from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Textarea, Card, Toggle } from '@/components/common';
import type { SessionPolicy } from '@/types';
import toast from 'react-hot-toast';

export const SessionPolicyFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState({
    name: '',
    max_duration_minutes: 480,
    require_approval: false,
    require_reason: true,
    require_mfa: true,
    allow_recording: true,
    monitor_keywords: [] as string[],
    blocked_commands: [] as string[],
    idle_timeout_minutes: 15,
    warning_minutes_before_end: 5,
  });

  const [keywordInput, setKeywordInput] = useState('');
  const [commandInput, setCommandInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing policy for editing
  const { data: existingPolicy, isLoading: isLoadingPolicy } = useQuery({
    queryKey: ['sessionPolicy', id],
    queryFn: () => policiesApi.session.get(id!),
    enabled: isEditing,
  });

  // Populate form when editing
  useEffect(() => {
    if (existingPolicy) {
      setFormData({
        name: existingPolicy.name || '',
        max_duration_minutes: existingPolicy.max_duration_minutes || 480,
        require_approval: existingPolicy.require_approval || false,
        require_reason: existingPolicy.require_reason ?? true,
        require_mfa: existingPolicy.require_mfa ?? true,
        allow_recording: existingPolicy.allow_recording ?? true,
        monitor_keywords: existingPolicy.monitor_keywords || [],
        blocked_commands: existingPolicy.blocked_commands || [],
        idle_timeout_minutes: existingPolicy.idle_timeout_minutes || 15,
        warning_minutes_before_end: existingPolicy.warning_minutes_before_end || 5,
      });
    }
  }, [existingPolicy]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: policiesApi.session.create,
    onSuccess: () => {
      toast.success('Session policy created');
      queryClient.invalidateQueries({ queryKey: ['sessionPolicies'] });
      navigate('/policies/session');
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: typeof formData }) =>
      policiesApi.session.update(id, data),
    onSuccess: () => {
      toast.success('Session policy updated');
      queryClient.invalidateQueries({ queryKey: ['sessionPolicies'] });
      navigate('/policies/session');
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (formData.max_duration_minutes < 1) {
      newErrors.max_duration_minutes = 'Must be at least 1 minute';
    }
    if (formData.idle_timeout_minutes < 1) {
      newErrors.idle_timeout_minutes = 'Must be at least 1 minute';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    if (isEditing) {
      updateMutation.mutate({ id: id!, data: formData });
    } else {
      createMutation.mutate(formData);
    }
  };

  const addKeyword = () => {
    const keyword = keywordInput.trim();
    if (keyword && !formData.monitor_keywords.includes(keyword)) {
      setFormData({
        ...formData,
        monitor_keywords: [...formData.monitor_keywords, keyword],
      });
      setKeywordInput('');
    }
  };

  const removeKeyword = (keyword: string) => {
    setFormData({
      ...formData,
      monitor_keywords: formData.monitor_keywords.filter((k) => k !== keyword),
    });
  };

  const addCommand = () => {
    const command = commandInput.trim();
    if (command && !formData.blocked_commands.includes(command)) {
      setFormData({
        ...formData,
        blocked_commands: [...formData.blocked_commands, command],
      });
      setCommandInput('');
    }
  };

  const removeCommand = (command: string) => {
    setFormData({
      ...formData,
      blocked_commands: formData.blocked_commands.filter((c) => c !== command),
    });
  };

  if (isEditing && isLoadingPolicy) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/policies/session">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-600/20 text-info-400">
            <Clock className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">
              {isEditing ? 'Edit Session Policy' : 'Create Session Policy'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              Configure session security and monitoring settings
            </p>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Basic Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <Input
                  label="Policy Name"
                  placeholder="e.g., Standard Session Policy"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div>
                <Input
                  label="Max Session Duration (minutes)"
                  type="number"
                  min="1"
                  value={formData.max_duration_minutes}
                  onChange={(e) =>
                    setFormData({ ...formData, max_duration_minutes: parseInt(e.target.value) || 60 })
                  }
                  error={errors.max_duration_minutes}
                />
              </div>

              <div>
                <Input
                  label="Idle Timeout (minutes)"
                  type="number"
                  min="1"
                  value={formData.idle_timeout_minutes}
                  onChange={(e) =>
                    setFormData({ ...formData, idle_timeout_minutes: parseInt(e.target.value) || 15 })
                  }
                  error={errors.idle_timeout_minutes}
                />
              </div>

              <div>
                <Input
                  label="Warning Before End (minutes)"
                  type="number"
                  min="0"
                  value={formData.warning_minutes_before_end}
                  onChange={(e) =>
                    setFormData({ ...formData, warning_minutes_before_end: parseInt(e.target.value) || 5 })
                  }
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Access Requirements */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Access Requirements</h3>
            <div className="space-y-4">
              <Toggle
                label="Require Approval"
                description="Session access requires approval from authorized users"
                checked={formData.require_approval}
                onChange={(e) => setFormData({ ...formData, require_approval: e.target.checked })}
              />

              <Toggle
                label="Require Reason"
                description="Users must provide a reason for session access"
                checked={formData.require_reason}
                onChange={(e) => setFormData({ ...formData, require_reason: e.target.checked })}
              />

              <Toggle
                label="Require MFA"
                description="Multi-factor authentication required for session access"
                checked={formData.require_mfa}
                onChange={(e) => setFormData({ ...formData, require_mfa: e.target.checked })}
              />

              <Toggle
                label="Allow Recording"
                description="Record sessions for audit and compliance"
                checked={formData.allow_recording}
                onChange={(e) => setFormData({ ...formData, allow_recording: e.target.checked })}
              />
            </div>
          </div>
        </Card>

        {/* Monitoring */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Content Monitoring</h3>

            <div className="mb-4">
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Monitor Keywords
              </label>
              <p className="text-xs text-gray-400 mb-2">
                Alert when these keywords appear during a session
              </p>
              <div className="flex gap-2 mb-2">
                <Input
                  placeholder="Add a keyword to monitor..."
                  value={keywordInput}
                  onChange={(e) => setKeywordInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addKeyword())}
                />
                <Button type="button" onClick={addKeyword}>
                  Add
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {formData.monitor_keywords.map((keyword) => (
                  <span
                    key={keyword}
                    className="inline-flex items-center gap-1 rounded-full bg-warning-600/20 px-3 py-1 text-sm text-warning-400"
                  >
                    {keyword}
                    <button
                      type="button"
                      onClick={() => removeKeyword(keyword)}
                      className="hover:text-warning-300"
                    >
                      ×
                    </button>
                  </span>
                ))}
                {formData.monitor_keywords.length === 0 && (
                  <p className="text-sm text-gray-400">No keywords configured</p>
                )}
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Blocked Commands
              </label>
              <p className="text-xs text-gray-400 mb-2">
                Commands that will be blocked during a session
              </p>
              <div className="flex gap-2 mb-2">
                <Input
                  placeholder="Add a command to block..."
                  value={commandInput}
                  onChange={(e) => setCommandInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addCommand())}
                />
                <Button type="button" onClick={addCommand}>
                  Add
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {formData.blocked_commands.map((command) => (
                  <span
                    key={command}
                    className="inline-flex items-center gap-1 rounded-full bg-danger-600/20 px-3 py-1 text-sm text-danger-400 font-mono"
                  >
                    {command}
                    <button
                      type="button"
                      onClick={() => removeCommand(command)}
                      className="hover:text-danger-300"
                    >
                      ×
                    </button>
                  </span>
                ))}
                {formData.blocked_commands.length === 0 && (
                  <p className="text-sm text-gray-400">No commands blocked</p>
                )}
              </div>
            </div>
          </div>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/policies/session">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending || updateMutation.isPending}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditing ? 'Save Changes' : 'Create Policy'}
          </Button>
        </div>
      </form>
    </div>
  );
};

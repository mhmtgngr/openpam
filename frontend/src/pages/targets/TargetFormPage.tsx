import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, TestTube, Loader2 } from 'lucide-react';
import { targetsApi } from '@/api/targets';
import { foldersApi } from '@/api/folders';
import { usersApi } from '@/api/users';
import { Button, Input, Select, Textarea, Toggle, Card } from '@/components/common';
import type { Target, TargetType } from '@/types';
import toast from 'react-hot-toast';

const targetTypes: { value: TargetType; label: string }[] = [
  { value: 'ssh', label: 'SSH' },
  { value: 'rdp', label: 'RDP' },
  { value: 'database', label: 'Database' },
  { value: 'kubernetes', label: 'Kubernetes' },
  { value: 'web', label: 'Web' },
  { value: 'api', label: 'API' },
];

const environments = [
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
  { value: 'development', label: 'Development' },
  { value: 'test', label: 'Test' },
];

const sensitivities = [
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
];

export const TargetFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  // Form state
  const [formData, setFormData] = useState<{
    name: string;
    type: TargetType;
    host: string;
    port: number;
    description: string;
    environment: 'production' | 'staging' | 'development' | 'test';
    sensitivity: 'high' | 'medium' | 'low';
    platform: string;
    os_version: string;
    tags: string[];
    approver_ids: string[];
    connection_timeout: number;
    max_session_duration: number;
    require_approval: boolean;
    require_reason: boolean;
    require_mfa: boolean;
    allow_recording: boolean;
    auto_rotate_credentials: boolean;
    folder_id: string;
  }>({
    name: '',
    type: 'ssh',
    host: '',
    port: 22,
    description: '',
    environment: 'development',
    sensitivity: 'medium',
    platform: '',
    os_version: '',
    tags: [],
    approver_ids: [],
    connection_timeout: 30,
    max_session_duration: 60,
    require_approval: false,
    require_reason: false,
    require_mfa: true,
    allow_recording: true,
    auto_rotate_credentials: false,
    folder_id: '',
  });

  const [tagInput, setTagInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing target for editing
  const { data: existingTarget, isLoading: isLoadingTarget } = useQuery({
    queryKey: ['target', id],
    queryFn: () => targetsApi.get(id!),
    enabled: isEditing,
  });

  // Fetch folders for organization
  const { data: folders } = useQuery({
    queryKey: ['folders', 'targets'],
    queryFn: () => foldersApi.list('targets'),
  });

  // Fetch users for approvers
  const { data: users } = useQuery({
    queryKey: ['users'],
    queryFn: () => usersApi.list({ limit: 100 }),
  });

  // Populate form when editing
  useEffect(() => {
    if (existingTarget) {
      setFormData({
        name: existingTarget.name || '',
        type: existingTarget.type,
        host: existingTarget.host || '',
        port: existingTarget.port || 22,
        description: existingTarget.description || '',
        environment: existingTarget.environment || 'development',
        sensitivity: existingTarget.sensitivity || 'medium',
        platform: existingTarget.platform || '',
        os_version: existingTarget.os_version || '',
        tags: existingTarget.tags || [],
        approver_ids: existingTarget.approver_ids || [],
        connection_timeout: existingTarget.connection_timeout || 30,
        max_session_duration: existingTarget.max_session_duration || 60,
        require_approval: existingTarget.require_approval || false,
        require_reason: existingTarget.require_reason || false,
        require_mfa: existingTarget.require_mfa ?? true,
        allow_recording: existingTarget.allow_recording ?? true,
        auto_rotate_credentials: existingTarget.auto_rotate_credentials || false,
        folder_id: existingTarget.folder_id || '',
      });
    }
  }, [existingTarget]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: targetsApi.create,
    onSuccess: (data) => {
      toast.success('Target created successfully');
      queryClient.invalidateQueries({ queryKey: ['targets'] });
      navigate(`/targets/${data.id}`);
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<typeof formData> }) =>
      targetsApi.update(id, data),
    onSuccess: () => {
      toast.success('Target updated successfully');
      queryClient.invalidateQueries({ queryKey: ['targets'] });
      queryClient.invalidateQueries({ queryKey: ['target', id] });
    },
  });

  // Test connection mutation
  const testConnectionMutation = useMutation({
    mutationFn: () => targetsApi.testConnection(id || ''),
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!formData.host.trim()) {
      newErrors.host = 'Host is required';
    }
    if (formData.port < 1 || formData.port > 65535) {
      newErrors.port = 'Port must be between 1 and 65535';
    }
    if (formData.connection_timeout < 1) {
      newErrors.connection_timeout = 'Connection timeout must be at least 1 second';
    }
    if (formData.max_session_duration < 1) {
      newErrors.max_session_duration = 'Max session duration must be at least 1 minute';
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

  const handleTestConnection = () => {
    if (isEditing) {
      testConnectionMutation.mutate(undefined, {
        onSuccess: (data) => {
          if (data.status === 'online') {
            toast.success(`Connection successful! Latency: ${data.latency_ms}ms`);
          } else {
            toast.error('Connection failed');
          }
        },
      });
    }
  };

  const handleAddTag = () => {
    const tag = tagInput.trim();
    if (tag && !formData.tags.includes(tag)) {
      setFormData({ ...formData, tags: [...formData.tags, tag] });
      setTagInput('');
    }
  };

  const handleRemoveTag = (tagToRemove: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((t) => t !== tagToRemove),
    });
  };

  const toggleApprover = (userId: string) => {
    setFormData({
      ...formData,
      approver_ids: formData.approver_ids.includes(userId)
        ? formData.approver_ids.filter((id) => id !== userId)
        : [...formData.approver_ids, userId],
    });
  };

  if (isEditing && isLoadingTarget) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/targets">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-white">
            {isEditing ? 'Edit Target' : 'Add New Target'}
          </h1>
          <p className="mt-1 text-sm text-gray-400">
            {isEditing
              ? 'Update target configuration and access policies'
              : 'Configure a new target system for privileged access management'}
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Basic Information</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <Input
                  label="Target Name"
                  placeholder="e.g., Production Database Server"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div>
                <Select
                  label="Target Type"
                  options={targetTypes}
                  value={formData.type}
                  onChange={(e) => setFormData({ ...formData, type: e.target.value as TargetType })}
                />
              </div>

              <div>
                <Select
                  label="Environment"
                  options={environments}
                  value={formData.environment}
                  onChange={(e) =>
                    setFormData({ ...formData, environment: e.target.value as any })
                  }
                />
              </div>

              <div>
                <Input
                  label="Hostname / IP Address"
                  placeholder="e.g., db.example.com or 192.168.1.100"
                  value={formData.host}
                  onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                  error={errors.host}
                  required
                />
              </div>

              <div>
                <Input
                  label="Port"
                  type="number"
                  min="1"
                  max="65535"
                  value={formData.port}
                  onChange={(e) =>
                    setFormData({ ...formData, port: parseInt(e.target.value) || 22 })
                  }
                  error={errors.port}
                />
              </div>

              <div className="md:col-span-2">
                <Select
                  label="Sensitivity Level"
                  options={sensitivities}
                  value={formData.sensitivity}
                  onChange={(e) =>
                    setFormData({ ...formData, sensitivity: e.target.value as any })
                  }
                />
              </div>

              <div className="md:col-span-2">
                <Textarea
                  label="Description"
                  placeholder="Optional description of this target..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                />
              </div>

              <div>
                <Select
                  label="Folder"
                  options={[
                    { value: '', label: 'No Folder' },
                    ...(folders?.map((f) => ({ value: f.id, label: f.name })) || []),
                  ]}
                  value={formData.folder_id}
                  onChange={(e) => setFormData({ ...formData, folder_id: e.target.value })}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Connection Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Connection Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Input
                  label="Platform / OS"
                  placeholder="e.g., Ubuntu 22.04, Windows Server 2019"
                  value={formData.platform}
                  onChange={(e) => setFormData({ ...formData, platform: e.target.value })}
                />
              </div>

              <div>
                <Input
                  label="OS Version"
                  placeholder="e.g., 22.04, 2019"
                  value={formData.os_version}
                  onChange={(e) => setFormData({ ...formData, os_version: e.target.value })}
                />
              </div>

              <div>
                <Input
                  label="Connection Timeout (seconds)"
                  type="number"
                  min="1"
                  value={formData.connection_timeout}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      connection_timeout: parseInt(e.target.value) || 30,
                    })
                  }
                  error={errors.connection_timeout}
                />
              </div>

              <div>
                <Input
                  label="Max Session Duration (minutes)"
                  type="number"
                  min="1"
                  value={formData.max_session_duration}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      max_session_duration: parseInt(e.target.value) || 60,
                    })
                  }
                  error={errors.max_session_duration}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Access Control */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Access Control</h3>
            <div className="space-y-4">
              <Toggle
                label="Require Approval"
                description="Access requests for this target require approval"
                checked={formData.require_approval}
                onChange={(e) =>
                  setFormData({ ...formData, require_approval: e.target.checked })
                }
              />

              <Toggle
                label="Require Reason"
                description="Users must provide a reason when requesting access"
                checked={formData.require_reason}
                onChange={(e) =>
                  setFormData({ ...formData, require_reason: e.target.checked })
                }
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
                onChange={(e) =>
                  setFormData({ ...formData, allow_recording: e.target.checked })
                }
              />

              <Toggle
                label="Auto-rotate Credentials"
                description="Automatically rotate credentials after use"
                checked={formData.auto_rotate_credentials}
                onChange={(e) =>
                  setFormData({ ...formData, auto_rotate_credentials: e.target.checked })
                }
              />
            </div>

            {/* Approvers */}
            {formData.require_approval && (
              <div className="mt-6">
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Approvers
                </label>
                <div className="border border-gray-700 rounded-md p-3 max-h-48 overflow-y-auto">
                  {users && users.data && users.data.length > 0 ? (
                    users.data.map((user) => (
                      <div
                        key={user.id}
                        className="flex items-center justify-between py-2 border-b border-gray-700 last:border-0"
                      >
                        <div>
                          <p className="text-sm text-white">
                            {user.first_name} {user.last_name}
                          </p>
                          <p className="text-xs text-gray-400">{user.email}</p>
                        </div>
                        <input
                          type="checkbox"
                          checked={formData.approver_ids.includes(user.id)}
                          onChange={() => toggleApprover(user.id)}
                          className="h-4 w-4 rounded border-gray-600 bg-gray-700 text-primary-600 focus:ring-primary-500"
                        />
                      </div>
                    ))
                  ) : (
                    <p className="text-sm text-gray-400">No users available</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* Tags */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Tags</h3>
            <div className="flex gap-2 mb-3">
              <Input
                placeholder="Add a tag..."
                value={tagInput}
                onChange={(e) => setTagInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddTag())}
              />
              <Button type="button" onClick={handleAddTag}>
                Add
              </Button>
            </div>
            <div className="flex flex-wrap gap-2">
              {formData.tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 rounded-full bg-primary-600/20 px-3 py-1 text-sm text-primary-400"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() => handleRemoveTag(tag)}
                    className="hover:text-primary-300"
                  >
                    ×
                  </button>
                </span>
              ))}
              {formData.tags.length === 0 && (
                <p className="text-sm text-gray-400">No tags added</p>
              )}
            </div>
          </div>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          {isEditing && (
            <Button
              type="button"
              variant="secondary"
              onClick={handleTestConnection}
              isLoading={testConnectionMutation.isPending}
              leftIcon={<TestTube className="h-4 w-4" />}
            >
              Test Connection
            </Button>
          )}
          <Link to="/targets">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending || updateMutation.isPending}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditing ? 'Save Changes' : 'Create Target'}
          </Button>
        </div>
      </form>
    </div>
  );
};

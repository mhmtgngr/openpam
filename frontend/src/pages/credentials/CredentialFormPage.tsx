import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Eye, EyeOff, RefreshCw, Key as KeyIcon, Loader2 } from 'lucide-react';
import { credentialsApi } from '@/api/credentials';
import { targetsApi } from '@/api/targets';
import { foldersApi } from '@/api/folders';
import { usersApi } from '@/api/users';
import {
  Button,
  Input,
  Select,
  Textarea,
  Toggle,
  Card,
} from '@/components/common';
import type { Credential, CredentialType, RotationPolicy } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const credentialTypes: { value: CredentialType; label: string }[] = [
  { value: 'password', label: 'Password' },
  { value: 'ssh_key', label: 'SSH Key' },
  { value: 'api_key', label: 'API Key' },
  { value: 'certificate', label: 'Certificate' },
  { value: 'database', label: 'Database' },
  { value: 'service_account', label: 'Service Account' },
];

const rotationPolicies: { value: RotationPolicy; label: string }[] = [
  { value: 'manual', label: 'Manual' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
  { value: 'on_checkin', label: 'On Check-in' },
  { value: 'on_expiry', label: 'On Expiry' },
];

const databaseTypes = [
  { value: 'mysql', label: 'MySQL' },
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'mssql', label: 'Microsoft SQL Server' },
  { value: 'oracle', label: 'Oracle' },
  { value: 'mongodb', label: 'MongoDB' },
  { value: 'redis', label: 'Redis' },
];

export const CredentialFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  // Form state
  const [formData, setFormData] = useState<{
    name: string;
    type: CredentialType;
    target_id: string;
    username: string;
    password: string;
    ssh_key: string;
    ssh_key_passphrase: string;
    api_key: string;
    database_name: string;
    database_type: 'mysql' | 'postgresql' | 'mssql' | 'oracle' | 'mongodb' | 'redis';
    rotation_policy: RotationPolicy;
    rotation_schedule: string;
    checkout_enabled: boolean;
    max_checkout_duration: number;
    auto_checkin: boolean;
    require_approval: boolean;
    approver_ids: string[];
    folder_id: string;
    description: string;
    tags: string[];
  }>({
    name: '',
    type: 'password' as CredentialType,
    target_id: '',
    username: '',
    password: '',
    ssh_key: '',
    ssh_key_passphrase: '',
    api_key: '',
    database_name: '',
    database_type: 'postgresql',
    rotation_policy: 'manual' as RotationPolicy,
    rotation_schedule: '',
    checkout_enabled: true,
    max_checkout_duration: 60,
    auto_checkin: false,
    require_approval: false,
    approver_ids: [] as string[],
    folder_id: '',
    description: '',
    tags: [] as string[],
  });

  const [showPassword, setShowPassword] = useState(false);
  const [showSshKey, setShowSshKey] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);
  const [tagInput, setTagInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing credential for editing
  const { data: existingCredential, isLoading: isLoadingCredential } = useQuery({
    queryKey: ['credential', id],
    queryFn: () => credentialsApi.get(id!).then((res) => res.data),
    enabled: isEditing,
  });

  // Fetch targets
  const { data: targets } = useQuery({
    queryKey: ['targets'],
    queryFn: () => targetsApi.list({ limit: 100 }).then((res) => res.data),
  });

  // Fetch folders
  const { data: folders } = useQuery({
    queryKey: ['folders', 'credentials'],
    queryFn: () => foldersApi.list('credentials').then((res) => res.data),
  });

  // Fetch users for approvers
  const { data: users } = useQuery({
    queryKey: ['users'],
    queryFn: () => usersApi.list({ limit: 100 }).then((res) => res.data),
  });

  // Populate form when editing
  useEffect(() => {
    if (existingCredential) {
      setFormData({
        name: existingCredential.name || '',
        type: existingCredential.type,
        target_id: existingCredential.target_id || '',
        username: existingCredential.username || '',
        password: '',
        ssh_key: '',
        ssh_key_passphrase: '',
        api_key: '',
        database_name: existingCredential.database_name || '',
        database_type: (existingCredential.database_type || 'postgresql') as 'mysql' | 'postgresql' | 'mssql' | 'oracle' | 'mongodb' | 'redis',
        rotation_policy: existingCredential.rotation_policy || 'manual',
        rotation_schedule: existingCredential.rotation_schedule || '',
        checkout_enabled: existingCredential.checkout_enabled ?? true,
        max_checkout_duration: existingCredential.max_checkout_duration || 60,
        auto_checkin: existingCredential.auto_checkin || false,
        require_approval: existingCredential.require_approval || false,
        approver_ids: existingCredential.approver_ids || [],
        folder_id: existingCredential.folder_id || '',
        description: existingCredential.description || '',
        tags: existingCredential.tags || [],
      });
    }
  }, [existingCredential]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: credentialsApi.create,
    onSuccess: (data) => {
      toast.success('Credential created successfully');
      queryClient.invalidateQueries({ queryKey: ['credentials'] });
      navigate(`/credentials/${data.data.id}`);
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<typeof formData> }) =>
      credentialsApi.update(id, data),
    onSuccess: () => {
      toast.success('Credential updated successfully');
      queryClient.invalidateQueries({ queryKey: ['credentials'] });
      queryClient.invalidateQueries({ queryKey: ['credential', id] });
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!formData.username.trim()) {
      newErrors.username = 'Username is required';
    }

    // Validate based on type
    if (!isEditing) {
      switch (formData.type) {
        case 'password':
          if (!formData.password) {
            newErrors.password = 'Password is required';
          }
          break;
        case 'ssh_key':
          if (!formData.ssh_key) {
            newErrors.ssh_key = 'SSH key is required';
          }
          break;
        case 'api_key':
          if (!formData.api_key) {
            newErrors.api_key = 'API key is required';
          }
          break;
        case 'database':
          if (!formData.password) {
            newErrors.password = 'Password is required';
          }
          if (!formData.database_name) {
            newErrors.database_name = 'Database name is required';
          }
          break;
      }
    }

    if (formData.max_checkout_duration < 1) {
      newErrors.max_checkout_duration = 'Max checkout duration must be at least 1 minute';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    const submitData: Partial<typeof formData> = { ...formData };

    // Only include secret fields if they have values
    if (!submitData.password) delete (submitData as any).password;
    if (!submitData.ssh_key) delete (submitData as any).ssh_key;
    if (!submitData.ssh_key_passphrase) delete (submitData as any).ssh_key_passphrase;
    if (!submitData.api_key) delete (submitData as any).api_key;

    if (isEditing) {
      updateMutation.mutate({ id: id!, data: submitData });
    } else {
      createMutation.mutate(submitData as any);
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

  // Generate a secure password
  const generatePassword = () => {
    const length = 20;
    const charset =
      'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;:,.<>?';
    let password = '';
    const array = new Uint32Array(length);
    crypto.getRandomValues(array);
    for (let i = 0; i < length; i++) {
      password += charset[array[i] % charset.length];
    }
    setFormData({ ...formData, password });
  };

  if (isEditing && isLoadingCredential) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  const renderSecretFields = () => {
    switch (formData.type) {
      case 'password':
        return (
          <div className="space-y-4">
            <div className="flex items-center gap-2">
              <div className="flex-1">
                <Input
                  label="Password"
                  type="password"
                  placeholder="Enter password..."
                  value={formData.password}
                  onChange={(e) => setFormData({ ...formData, password: e.target.value })}
                  error={errors.password}
                  required={!isEditing}
                />
              </div>
              <Button
                type="button"
                variant="secondary"
                onClick={generatePassword}
                className="mt-6"
                leftIcon={<RefreshCw className="h-4 w-4" />}
              >
                Generate
              </Button>
            </div>
          </div>
        );

      case 'ssh_key':
        return (
          <div className="space-y-4">
            <Textarea
              label="SSH Private Key"
              placeholder="-----BEGIN RSA PRIVATE KEY-----..."
              value={formData.ssh_key}
              onChange={(e) => setFormData({ ...formData, ssh_key: e.target.value })}
              error={errors.ssh_key}
              rows={8}
              required={!isEditing}
              className="font-mono text-xs"
            />
            <Input
              label="Key Passphrase (Optional)"
              type="password"
              placeholder="Passphrase for the private key..."
              value={formData.ssh_key_passphrase}
              onChange={(e) =>
                setFormData({ ...formData, ssh_key_passphrase: e.target.value })
              }
            />
          </div>
        );

      case 'api_key':
        return (
          <div>
            <Input
              label="API Key"
              placeholder="Enter API key..."
              value={formData.api_key}
              onChange={(e) => setFormData({ ...formData, api_key: e.target.value })}
              error={errors.api_key}
              type={showApiKey ? 'text' : 'password'}
              required={!isEditing}
              rightIcon={
                <button
                  type="button"
                  onClick={() => setShowApiKey(!showApiKey)}
                  className="text-gray-400 hover:text-gray-300"
                >
                  {showApiKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              }
            />
          </div>
        );

      case 'database':
        return (
          <div className="space-y-4">
            <div>
              <Input
                label="Database Name"
                placeholder="e.g., production_db"
                value={formData.database_name}
                onChange={(e) => setFormData({ ...formData, database_name: e.target.value })}
                error={errors.database_name}
                required={!isEditing}
              />
            </div>
            <div>
              <Select
                label="Database Type"
                options={databaseTypes}
                value={formData.database_type}
                onChange={(e) =>
                  setFormData({ ...formData, database_type: e.target.value as any })
                }
              />
            </div>
            <Input
              label="Database Password"
              type="password"
              placeholder="Enter database password..."
              value={formData.password}
              onChange={(e) => setFormData({ ...formData, password: e.target.value })}
              error={errors.password}
              required={!isEditing}
            />
          </div>
        );

      case 'certificate':
      case 'service_account':
      default:
        return (
          <div className="p-4 bg-info-400/10 border border-info-400/30 rounded-md">
            <p className="text-sm text-info-300">
              {formData.type === 'certificate'
                ? 'Certificate management requires uploading certificate files. This feature will be available soon.'
                : 'Service account credentials require JSON key file upload. This feature will be available soon.'}
            </p>
          </div>
        );
    }
  };

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/credentials">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600/20 text-primary-400">
            <KeyIcon className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">
              {isEditing ? 'Edit Credential' : 'Add New Credential'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              {isEditing
                ? 'Update credential configuration and rotation settings'
                : 'Securely store a new credential in the vault'}
            </p>
          </div>
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
                  label="Credential Name"
                  placeholder="e.g., Production DB Admin"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div>
                <Select
                  label="Credential Type"
                  options={credentialTypes}
                  value={formData.type}
                  onChange={(e) => {
                    setFormData({ ...formData, type: e.target.value as CredentialType });
                  }}
                  disabled={isEditing}
                />
              </div>

              <div>
                <Select
                  label="Associated Target"
                  options={[
                    { value: '', label: 'No Target' },
                    ...(targets?.data.map((t) => ({ value: t.id, label: t.name })) || []),
                  ]}
                  value={formData.target_id}
                  onChange={(e) => setFormData({ ...formData, target_id: e.target.value })}
                />
              </div>

              <div className="md:col-span-2">
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

              <div className="md:col-span-2">
                <Textarea
                  label="Description"
                  placeholder="Optional description of this credential..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Credential Details */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Credential Details</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <Input
                  label="Username / Account"
                  placeholder="e.g., admin, root, service-account"
                  value={formData.username}
                  onChange={(e) => setFormData({ ...formData, username: e.target.value })}
                  error={errors.username}
                  required
                />
              </div>

              {renderSecretFields()}
            </div>
          </div>
        </Card>

        {/* Rotation Policy */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Rotation Policy</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Select
                  label="Rotation Policy"
                  options={rotationPolicies}
                  value={formData.rotation_policy}
                  onChange={(e) =>
                    setFormData({ ...formData, rotation_policy: e.target.value as RotationPolicy })
                  }
                />
              </div>

              {formData.rotation_policy !== 'manual' && (
                <div>
                  <Input
                    label="Rotation Schedule (Cron)"
                    placeholder="e.g., 0 2 * * *"
                    value={formData.rotation_schedule}
                    onChange={(e) =>
                      setFormData({ ...formData, rotation_schedule: e.target.value })
                    }
                  />
                  <p className="mt-1 text-xs text-gray-400">
                    Cron expression for automated rotation
                  </p>
                </div>
              )}
            </div>
          </div>
        </Card>

        {/* Checkout Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Checkout Settings</h3>
            <div className="space-y-4">
              <Toggle
                label="Enable Checkout"
                description="Allow users to check out this credential"
                checked={formData.checkout_enabled}
                onChange={(e) =>
                  setFormData({ ...formData, checkout_enabled: e.target.checked })
                }
              />

              {formData.checkout_enabled && (
                <>
                  <div>
                    <Input
                      label="Max Checkout Duration (minutes)"
                      type="number"
                      min="1"
                      value={formData.max_checkout_duration}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          max_checkout_duration: parseInt(e.target.value) || 60,
                        })
                      }
                      error={errors.max_checkout_duration}
                    />
                  </div>

                  <Toggle
                    label="Auto Check-in"
                    description="Automatically check in credentials when session ends"
                    checked={formData.auto_checkin}
                    onChange={(e) =>
                      setFormData({ ...formData, auto_checkin: e.target.checked })
                    }
                  />

                  <Toggle
                    label="Require Approval"
                    description="Checkout requests require approval"
                    checked={formData.require_approval}
                    onChange={(e) =>
                      setFormData({ ...formData, require_approval: e.target.checked })
                    }
                  />
                </>
              )}
            </div>

            {/* Approvers */}
            {formData.checkout_enabled && formData.require_approval && (
              <div className="mt-6">
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  Approvers
                </label>
                <div className="border border-gray-700 rounded-md p-3 max-h-48 overflow-y-auto">
                  {users?.data && users.data.length > 0 ? (
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

        {/* Security Notice */}
        {!isEditing && (
          <Card>
            <div className="card-body bg-warning-400/10 border border-warning-400/30">
              <div className="flex gap-3">
                <KeyIcon className="h-5 w-5 text-warning-400 flex-shrink-0 mt-0.5" />
                <div>
                  <h4 className="text-sm font-semibold text-warning-300">Security Notice</h4>
                  <p className="text-sm text-warning-200/80 mt-1">
                    Credentials will be encrypted using AES-256-GCM envelope encryption before
                    storage. The secret value is only shown once during creation and cannot be
                    retrieved later.
                  </p>
                </div>
              </div>
            </div>
          </Card>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/credentials">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending || updateMutation.isPending}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditing ? 'Save Changes' : 'Create Credential'}
          </Button>
        </div>
      </form>
    </div>
  );
};

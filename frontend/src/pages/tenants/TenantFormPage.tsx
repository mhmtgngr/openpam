import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Palette, Building2, Loader2 } from 'lucide-react';
import { tenantsApi } from '@/api/tenants';
import { Button, Input, Textarea, Toggle, Card, Select } from '@/components/common';
import type { Tenant } from '@/types';
import toast from 'react-hot-toast';

const mfaMethods = [
  { value: 'totp', label: 'TOTP (Authenticator App)' },
  { value: 'webauthn', label: 'WebAuthn (Hardware Key)' },
];

export const TenantFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    logo_url: '',
    primary_color: '#6366f1',
    enforce_mfa: true,
    mfa_methods: ['totp'] as ('totp' | 'webauthn')[],
    session_timeout_minutes: 60,
    audit_retention_days: 90,
    recording_retention_days: 30,
    ip_whitelist: [] as string[],
    max_users: 100,
    max_targets: 500,
  });

  const [ipWhitelistInput, setIpWhitelistInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing tenant for editing
  const { data: existingTenant, isLoading: isLoadingTenant } = useQuery({
    queryKey: ['tenant', id],
    queryFn: () => tenantsApi.get(id!),
    enabled: isEditing,
  });

  // Fetch tenant stats for editing
  const { data: tenantStats } = useQuery({
    queryKey: ['tenant', id, 'stats'],
    queryFn: () => tenantsApi.getStats(id!),
    enabled: isEditing,
  });

  // Populate form when editing
  useEffect(() => {
    if (existingTenant) {
      setFormData({
        name: existingTenant.name || '',
        slug: existingTenant.slug || '',
        logo_url: existingTenant.logo_url || '',
        primary_color: existingTenant.primary_color || '#6366f1',
        enforce_mfa: existingTenant.settings?.enforce_mfa ?? true,
        mfa_methods: existingTenant.settings?.mfa_methods || ['totp'],
        session_timeout_minutes: existingTenant.settings?.session_timeout_minutes || 60,
        audit_retention_days: existingTenant.settings?.audit_retention_days || 90,
        recording_retention_days: existingTenant.settings?.recording_retention_days || 30,
        ip_whitelist: existingTenant.settings?.ip_whitelist || [],
        max_users: existingTenant.max_users || 100,
        max_targets: existingTenant.max_targets || 500,
      });
    }
  }, [existingTenant]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: tenantsApi.create,
    onSuccess: (data) => {
      toast.success('Tenant created successfully');
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      navigate(`/tenants/${data.id}`);
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: typeof formData }) =>
      tenantsApi.update(id, data),
    onSuccess: () => {
      toast.success('Tenant updated successfully');
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      queryClient.invalidateQueries({ queryKey: ['tenant', id] });
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (!formData.slug.trim()) {
      newErrors.slug = 'Slug is required';
    }
    if (!/^[a-z0-9-]+$/.test(formData.slug)) {
      newErrors.slug = 'Slug must contain only lowercase letters, numbers, and hyphens';
    }
    if (formData.max_users < 1) {
      newErrors.max_users = 'Max users must be at least 1';
    }
    if (formData.max_targets < 1) {
      newErrors.max_targets = 'Max targets must be at least 1';
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
      ...formData,
      settings: {
        enforce_mfa: formData.enforce_mfa,
        mfa_methods: formData.mfa_methods,
        session_timeout_minutes: formData.session_timeout_minutes,
        audit_retention_days: formData.audit_retention_days,
        recording_retention_days: formData.recording_retention_days,
        ip_whitelist: formData.ip_whitelist.length > 0 ? formData.ip_whitelist : undefined,
      },
    };

    if (isEditing) {
      updateMutation.mutate({ id: id!, data: submitData });
    } else {
      createMutation.mutate(submitData);
    }
  };

  const handleAddIp = () => {
    const ip = ipWhitelistInput.trim();
    if (ip && !formData.ip_whitelist.includes(ip)) {
      setFormData({
        ...formData,
        ip_whitelist: [...formData.ip_whitelist, ip],
      });
      setIpWhitelistInput('');
    }
  };

  const handleRemoveIp = (ipToRemove: string) => {
    setFormData({
      ...formData,
      ip_whitelist: formData.ip_whitelist.filter((ip) => ip !== ipToRemove),
    });
  };

  const toggleMfaMethod = (method: 'totp' | 'webauthn') => {
    setFormData({
      ...formData,
      mfa_methods: formData.mfa_methods.includes(method)
        ? formData.mfa_methods.filter((m) => m !== method)
        : [...formData.mfa_methods, method],
    });
  };

  const generateSlug = () => {
    const slug = formData.name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, '')
      .trim()
      .replace(/\s+/g, '-');
    setFormData({ ...formData, slug });
  };

  if (isEditing && isLoadingTenant) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/tenants">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg text-white font-semibold"
            style={{ backgroundColor: formData.primary_color }}>
            {formData.name.charAt(0).toUpperCase() || 'T'}
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">
              {isEditing ? 'Edit Tenant' : 'Create Tenant'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              {isEditing
                ? 'Update tenant configuration and settings'
                : 'Configure a new tenant for the platform'}
            </p>
          </div>
        </div>
      </div>

      {/* Usage Stats for Editing */}
      {isEditing && tenantStats && (
        <Card>
          <div className="card-body bg-gray-800">
            <h3 className="text-sm font-semibold text-white mb-3">Current Usage</h3>
            <div className="grid grid-cols-4 gap-4 text-center">
              <div>
                <p className="text-2xl font-bold text-white">{tenantStats.users}</p>
                <p className="text-xs text-gray-400">Users</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-white">{tenantStats.targets}</p>
                <p className="text-xs text-gray-400">Targets</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-white">{tenantStats.credentials}</p>
                <p className="text-xs text-gray-400">Credentials</p>
              </div>
              <div>
                <p className="text-2xl font-bold text-white">{tenantStats.sessions}</p>
                <p className="text-xs text-gray-400">Active Sessions</p>
              </div>
            </div>
          </div>
        </Card>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Basic Information</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Input
                  label="Tenant Name"
                  placeholder="e.g., Acme Corporation"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div>
                <div className="flex items-center gap-2">
                  <div className="flex-1">
                    <Input
                      label="Slug"
                      placeholder="e.g., acme-corp"
                      value={formData.slug}
                      onChange={(e) => setFormData({ ...formData, slug: e.target.value })}
                      error={errors.slug}
                      required
                      readOnly={isEditing}
                    />
                  </div>
                  {!isEditing && (
                    <button
                      type="button"
                      onClick={generateSlug}
                      className="mt-6 px-3 py-2 text-sm bg-gray-700 text-white rounded hover:bg-gray-600"
                    >
                      Auto-generate
                    </button>
                  )}
                </div>
              </div>

              <div>
                <Input
                  label="Logo URL"
                  placeholder="https://example.com/logo.png"
                  value={formData.logo_url}
                  onChange={(e) => setFormData({ ...formData, logo_url: e.target.value })}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Primary Color
                </label>
                <div className="flex gap-2">
                  <input
                    type="color"
                    value={formData.primary_color}
                    onChange={(e) => setFormData({ ...formData, primary_color: e.target.value })}
                    className="h-10 w-14 rounded cursor-pointer"
                  />
                  <Input
                    placeholder="#6366f1"
                    value={formData.primary_color}
                    onChange={(e) => setFormData({ ...formData, primary_color: e.target.value })}
                    className="flex-1"
                  />
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* Limits */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Limits & Quotas</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Input
                  label="Max Users"
                  type="number"
                  min="1"
                  value={formData.max_users}
                  onChange={(e) =>
                    setFormData({ ...formData, max_users: parseInt(e.target.value) || 1 })
                  }
                  error={errors.max_users}
                />
              </div>

              <div>
                <Input
                  label="Max Targets"
                  type="number"
                  min="1"
                  value={formData.max_targets}
                  onChange={(e) =>
                    setFormData({ ...formData, max_targets: parseInt(e.target.value) || 1 })
                  }
                  error={errors.max_targets}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Security Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Security Settings</h3>
            <div className="space-y-4">
              <Toggle
                label="Enforce MFA"
                description="Require all users to enable multi-factor authentication"
                checked={formData.enforce_mfa}
                onChange={(e) => setFormData({ ...formData, enforce_mfa: e.target.checked })}
              />

              {formData.enforce_mfa && (
                <div>
                  <label className="block text-sm font-medium text-gray-300 mb-2">
                    Allowed MFA Methods
                  </label>
                  <div className="space-y-2">
                    {mfaMethods.map((method) => (
                      <label key={method.value} className="flex items-center gap-3 p-3 bg-gray-800 rounded-lg cursor-pointer hover:bg-gray-750">
                        <input
                          type="checkbox"
                          checked={formData.mfa_methods.includes(method.value as any)}
                          onChange={() => toggleMfaMethod(method.value as any)}
                          className="h-4 w-4 rounded border-gray-600 bg-gray-700 text-primary-600 focus:ring-primary-500"
                        />
                        <span className="text-sm text-white">{method.label}</span>
                      </label>
                    ))}
                  </div>
                </div>
              )}

              <div>
                <Input
                  label="Session Timeout (minutes)"
                  type="number"
                  min="5"
                  value={formData.session_timeout_minutes}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      session_timeout_minutes: parseInt(e.target.value) || 60,
                    })
                  }
                />
                <p className="mt-1 text-xs text-gray-400">
                  Users will be logged out after this period of inactivity
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-2">
                  IP Whitelist
                </label>
                <div className="flex gap-2 mb-2">
                  <Input
                    placeholder="Add IP address or CIDR range..."
                    value={ipWhitelistInput}
                    onChange={(e) => setIpWhitelistInput(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddIp())}
                  />
                  <Button type="button" onClick={handleAddIp}>
                    Add
                  </Button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {formData.ip_whitelist.map((ip) => (
                    <span
                      key={ip}
                      className="inline-flex items-center gap-1 rounded-full bg-gray-700 px-3 py-1 text-sm text-white"
                    >
                      {ip}
                      <button
                        type="button"
                        onClick={() => handleRemoveIp(ip)}
                        className="hover:text-danger-400"
                      >
                        ×
                      </button>
                    </span>
                  ))}
                  {formData.ip_whitelist.length === 0 && (
                    <p className="text-sm text-gray-400">No IP restrictions (all IPs allowed)</p>
                  )}
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* Retention Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Data Retention</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Input
                  label="Audit Log Retention (days)"
                  type="number"
                  min="1"
                  value={formData.audit_retention_days}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      audit_retention_days: parseInt(e.target.value) || 90,
                    })
                  }
                />
                <p className="mt-1 text-xs text-gray-400">
                  Audit logs older than this will be automatically deleted
                </p>
              </div>

              <div>
                <Input
                  label="Recording Retention (days)"
                  type="number"
                  min="1"
                  value={formData.recording_retention_days}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      recording_retention_days: parseInt(e.target.value) || 30,
                    })
                  }
                />
                <p className="mt-1 text-xs text-gray-400">
                  Session recordings older than this will be automatically deleted
                </p>
              </div>
            </div>
          </div>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/tenants">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending || updateMutation.isPending}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditing ? 'Save Changes' : 'Create Tenant'}
          </Button>
        </div>
      </form>
    </div>
  );
};

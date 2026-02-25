import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  Save,
  Shield,
  Loader2,
  CheckCircle,
  XCircle,
} from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Textarea, Card, Select, Badge, Toggle } from '@/components/common';
import { PolicyRuleBuilder } from '@/components/policies/PolicyRuleBuilder';
import type { AccessPolicy, PolicyStatus, CreateAccessPolicyData } from '@/types/policy';
import toast from 'react-hot-toast';

type ConflictResolution = 'deny_overrides' | 'allow_overrides' | 'first_applicable';

const CONFLICT_RESOLUTION_OPTIONS: { value: ConflictResolution; label: string; description: string }[] = [
  {
    value: 'deny_overrides',
    label: 'Deny Overrides',
    description: 'Deny decisions take precedence (recommended for security)',
  },
  {
    value: 'allow_overrides',
    label: 'Allow Overrides',
    description: 'Allow decisions take precedence',
  },
  {
    value: 'first_applicable',
    label: 'First Applicable',
    description: 'First matching rule wins (evaluate by priority)',
  },
];

const STATUS_OPTIONS: { value: PolicyStatus; label: string }[] = [
  { value: 'draft', label: 'Draft' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

export const AccessPolicyFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState<CreateAccessPolicyData>({
    name: '',
    description: '',
    status: 'draft',
    priority: 100,
    rules: [],
    conflict_resolution: 'deny_overrides',
    tags: [],
  });

  const [tagInput, setTagInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  // Fetch existing policy for editing
  const { data: existingPolicy, isLoading: isLoadingPolicy } = useQuery({
    queryKey: ['accessPolicy', id],
    queryFn: () => policiesApi.access.get(id!),
    enabled: isEditing,
  });

  // Populate form when editing
  useEffect(() => {
    if (existingPolicy) {
      setFormData({
        name: existingPolicy.name,
        description: existingPolicy.description || '',
        status: existingPolicy.status,
        priority: existingPolicy.priority,
        rules: existingPolicy.rules,
        conflict_resolution: existingPolicy.conflict_resolution,
        tags: existingPolicy.tags || [],
      });
    }
  }, [existingPolicy]);

  // Validate mutation
  const validateMutation = useMutation({
    mutationFn: (data: CreateAccessPolicyData) => policiesApi.access.validate(data),
    onSuccess: (result) => {
      if (result.valid) {
        setValidationErrors([]);
        if (isEditing) {
          updateMutation.mutate({ id: id!, data: formData });
        } else {
          createMutation.mutate(formData);
        }
      } else {
        const allErrors = [
          ...result.errors.map((e: any) => e.message),
          ...result.warnings.map((w: any) => w.message),
        ];
        setValidationErrors(allErrors);
        toast.error('Policy validation failed. Please fix the errors.');
      }
    },
  });

  // Create mutation
  const createMutation = useMutation({
    mutationFn: policiesApi.access.create,
    onSuccess: () => {
      toast.success('Access policy created');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
      navigate('/policies/access');
    },
    onError: (error: any) => {
      if (error.response?.data?.error?.message) {
        toast.error(error.response.data.error.message);
      }
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateAccessPolicyData }) =>
      policiesApi.access.update(id, data),
    onSuccess: () => {
      toast.success('Access policy updated');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
      navigate('/policies/access');
    },
    onError: (error: any) => {
      if (error.response?.data?.error?.message) {
        toast.error(error.response.data.error.message);
      }
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (formData.rules.length === 0) {
      newErrors.rules = 'At least one rule is required';
    }
    if (formData.priority < 1) {
      newErrors.priority = 'Priority must be at least 1';
    }

    // Validate each rule
    formData.rules.forEach((rule, index) => {
      if (!rule.name.trim()) {
        newErrors[`rule_${index}_name`] = `Rule ${index + 1} name is required`;
      }
      if ((!rule.conditions || rule.conditions.length === 0) &&
          (!rule.resources || rule.resources.length === 0) &&
          (!rule.roles || rule.roles.length === 0)) {
        newErrors[`rule_${index}_conditions`] = `Rule ${index + 1} must have conditions, resources, or roles`;
      }
    });

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    // Validate with backend
    validateMutation.mutate(formData);
  };

  const addTag = () => {
    const tag = tagInput.trim().toLowerCase().replace(/[^a-z0-9-]/g, '');
    if (tag && !formData.tags.includes(tag)) {
      setFormData({ ...formData, tags: [...formData.tags, tag] });
      setTagInput('');
    }
  };

  const removeTag = (tag: string) => {
    setFormData({
      ...formData,
      tags: formData.tags.filter((t) => t !== tag),
    });
  };

  const isLoading = validateMutation.isPending || createMutation.isPending || updateMutation.isPending;

  if (isEditing && isLoadingPolicy) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return (
    <div className="max-w-5xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/policies/access">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600/20 text-primary-400">
            <Shield className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">
              {isEditing ? 'Edit Access Policy' : 'Create Access Policy'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              Define rules for controlling resource access
            </p>
          </div>
        </div>
      </div>

      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <Card className="border-danger-600/50 bg-danger-900/10">
          <div className="card-body">
            <div className="flex items-center gap-2 mb-2">
              <XCircle className="h-5 w-5 text-danger-400" />
              <h3 className="font-semibold text-danger-400">Validation Errors</h3>
            </div>
            <ul className="list-disc list-inside space-y-1 text-sm text-danger-300">
              {validationErrors.map((error, index) => (
                <li key={index}>{error}</li>
              ))}
            </ul>
          </div>
        </Card>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Settings */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Basic Settings</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <Input
                  label="Policy Name"
                  placeholder="e.g., Production Database Access"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div className="md:col-span-2">
                <Textarea
                  label="Description"
                  placeholder="Describe what this policy controls..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={2}
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Status
                </label>
                <Select
                  value={formData.status}
                  onChange={(e) => setFormData({ ...formData, status: e.target.value as PolicyStatus })}
                  options={STATUS_OPTIONS}
                />
                <p className="mt-1 text-xs text-gray-500">
                  Only active policies are evaluated during access checks
                </p>
              </div>

              <div>
                <Input
                  label="Priority"
                  type="number"
                  min="1"
                  value={formData.priority}
                  onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 1 })}
                  error={errors.priority}
                />
                <p className="mt-1 text-xs text-gray-500">
                  Higher numbers = higher priority (evaluated first)
                </p>
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Conflict Resolution
                </label>
                <Select
                  value={formData.conflict_resolution}
                  onChange={(e) =>
                    setFormData({ ...formData, conflict_resolution: e.target.value as ConflictResolution })
                  }
                  options={CONFLICT_RESOLUTION_OPTIONS.map((o) => ({
                    value: o.value,
                    label: `${o.label} - ${o.description}`,
                  }))}
                />
                <p className="mt-1 text-xs text-gray-500">
                  How to resolve when multiple rules match
                </p>
              </div>

              {/* Tags */}
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Tags
                </label>
                <div className="flex gap-2 mb-2">
                  <Input
                    placeholder="Add a tag..."
                    value={tagInput}
                    onChange={(e) => setTagInput(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())}
                  />
                  <Button type="button" onClick={addTag}>
                    Add
                  </Button>
                </div>
                <div className="flex flex-wrap gap-2">
                  {formData.tags.map((tag) => (
                    <Badge key={tag} variant="neutral" size="sm">
                      #{tag}
                      <button
                        type="button"
                        onClick={() => removeTag(tag)}
                        className="ml-1 hover:text-danger-400"
                      >
                        ×
                      </button>
                    </Badge>
                  ))}
                  {formData.tags.length === 0 && (
                    <span className="text-sm text-gray-500">No tags</span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </Card>

        {/* Rules */}
        <Card>
          <div className="card-body">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-semibold text-white">Rules</h3>
                <p className="text-sm text-gray-400">
                  Define access rules with conditions
                </p>
              </div>
              {errors.rules && (
                <span className="text-sm text-danger-400">{errors.rules}</span>
              )}
            </div>
            <PolicyRuleBuilder
              rules={formData.rules}
              onChange={(rules) => setFormData({ ...formData, rules })}
            />
          </div>
        </Card>

        {/* Effective Period (Optional) */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Effective Period (Optional)</h3>
            <p className="text-sm text-gray-400 mb-4">
              Set a time range for when this policy should be active
            </p>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Effective From
                </label>
                <Input
                  type="datetime-local"
                  value={
                    formData.effective_from
                      ? new Date(formData.effective_from).toISOString().slice(0, 16)
                      : ''
                  }
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      effective_from: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                    })
                  }
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-300 mb-1">
                  Effective Until
                </label>
                <Input
                  type="datetime-local"
                  value={
                    formData.effective_until
                      ? new Date(formData.effective_until).toISOString().slice(0, 16)
                      : ''
                  }
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      effective_until: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                    })
                  }
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Actions */}
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2 text-sm text-gray-400">
            {formData.rules.length > 0 && (
              <>
                {formData.rules.filter((r) => r.effect === 'allow').length} allow,
                {formData.rules.filter((r) => r.effect === 'deny').length} deny rules
              </>
            )}
          </div>
          <div className="flex justify-end gap-3">
            <Link to="/policies/access">
              <Button variant="secondary" type="button">
                Cancel
              </Button>
            </Link>
            <Button
              type="submit"
              isLoading={isLoading}
              leftIcon={<Save className="h-4 w-4" />}
            >
              {isEditing ? 'Save Changes' : 'Create Policy'}
            </Button>
          </div>
        </div>
      </form>
    </div>
  );
};

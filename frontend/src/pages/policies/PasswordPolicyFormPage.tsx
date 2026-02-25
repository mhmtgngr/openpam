import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Key, Loader2 } from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Textarea, Card, Toggle } from '@/components/common';
import type { PasswordPolicy } from '@/types';
import toast from 'react-hot-toast';

export const PasswordPolicyFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState({
    name: '',
    min_length: 12,
    max_length: 128,
    require_uppercase: true,
    require_lowercase: true,
    require_numbers: true,
    require_special: true,
    forbidden_passwords: [] as string[],
    expiration_days: 90,
    history_count: 5,
  });

  const [forbiddenPasswordInput, setForbiddenPasswordInput] = useState('');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing policy for editing
  const { data: existingPolicy, isLoading: isLoadingPolicy } = useQuery({
    queryKey: ['passwordPolicy', id],
    queryFn: () => policiesApi.password.get(id!),
    enabled: isEditing,
  });

  // Populate form when editing
  useEffect(() => {
    if (existingPolicy) {
      setFormData({
        name: existingPolicy.name || '',
        min_length: existingPolicy.min_length || 12,
        max_length: existingPolicy.max_length || 128,
        require_uppercase: existingPolicy.require_uppercase ?? true,
        require_lowercase: existingPolicy.require_lowercase ?? true,
        require_numbers: existingPolicy.require_numbers ?? true,
        require_special: existingPolicy.require_special ?? true,
        forbidden_passwords: existingPolicy.forbidden_passwords || [],
        expiration_days: existingPolicy.expiration_days || 90,
        history_count: existingPolicy.history_count || 5,
      });
    }
  }, [existingPolicy]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: policiesApi.password.create,
    onSuccess: () => {
      toast.success('Password policy created');
      queryClient.invalidateQueries({ queryKey: ['passwordPolicies'] });
      navigate('/policies/password');
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: typeof formData }) =>
      policiesApi.password.update(id, data),
    onSuccess: () => {
      toast.success('Password policy updated');
      queryClient.invalidateQueries({ queryKey: ['passwordPolicies'] });
      navigate('/policies/password');
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (formData.min_length < 4) {
      newErrors.min_length = 'Minimum length must be at least 4';
    }
    if (formData.max_length < formData.min_length) {
      newErrors.max_length = 'Maximum length must be greater than minimum length';
    }
    if (formData.expiration_days < 1) {
      newErrors.expiration_days = 'Expiration must be at least 1 day';
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
      ...(formData.forbidden_passwords.length > 0 && { forbidden_passwords: formData.forbidden_passwords }),
    };

    if (isEditing) {
      updateMutation.mutate({ id: id!, data: submitData });
    } else {
      createMutation.mutate(submitData);
    }
  };

  const addForbiddenPassword = () => {
    const password = forbiddenPasswordInput.trim();
    if (password && !formData.forbidden_passwords.includes(password)) {
      setFormData({
        ...formData,
        forbidden_passwords: [...formData.forbidden_passwords, password],
      });
      setForbiddenPasswordInput('');
    }
  };

  const removeForbiddenPassword = (password: string) => {
    setFormData({
      ...formData,
      forbidden_passwords: formData.forbidden_passwords.filter((p) => p !== password),
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
        <Link to="/policies/password">
          <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
            Back
          </Button>
        </Link>
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600/20 text-primary-400">
            <Key className="h-5 w-5" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-white">
              {isEditing ? 'Edit Password Policy' : 'Create Password Policy'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              Define password complexity requirements
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
                  placeholder="e.g., Standard Password Policy"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>

              <div>
                <Input
                  label="Minimum Length"
                  type="number"
                  min="4"
                  value={formData.min_length}
                  onChange={(e) =>
                    setFormData({ ...formData, min_length: parseInt(e.target.value) || 4 })
                  }
                  error={errors.min_length}
                />
              </div>

              <div>
                <Input
                  label="Maximum Length"
                  type="number"
                  min="4"
                  value={formData.max_length}
                  onChange={(e) =>
                    setFormData({ ...formData, max_length: parseInt(e.target.value) || 128 })
                  }
                  error={errors.max_length}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Character Requirements */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Character Requirements</h3>
            <div className="space-y-4">
              <Toggle
                label="Require Uppercase Letters"
                description="Passwords must contain at least one uppercase letter (A-Z)"
                checked={formData.require_uppercase}
                onChange={(e) => setFormData({ ...formData, require_uppercase: e.target.checked })}
              />

              <Toggle
                label="Require Lowercase Letters"
                description="Passwords must contain at least one lowercase letter (a-z)"
                checked={formData.require_lowercase}
                onChange={(e) => setFormData({ ...formData, require_lowercase: e.target.checked })}
              />

              <Toggle
                label="Require Numbers"
                description="Passwords must contain at least one number (0-9)"
                checked={formData.require_numbers}
                onChange={(e) => setFormData({ ...formData, require_numbers: e.target.checked })}
              />

              <Toggle
                label="Require Special Characters"
                description="Passwords must contain at least one special character (!@#$%^&*)"
                checked={formData.require_special}
                onChange={(e) => setFormData({ ...formData, require_special: e.target.checked })}
              />
            </div>
          </div>
        </Card>

        {/* Expiration & History */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Expiration & History</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <Input
                  label="Password Expiration (days)"
                  type="number"
                  min="0"
                  value={formData.expiration_days}
                  onChange={(e) =>
                    setFormData({ ...formData, expiration_days: parseInt(e.target.value) || 0 })
                  }
                  error={errors.expiration_days}
                />
                <p className="mt-1 text-xs text-gray-400">
                  Set to 0 for passwords that never expire
                </p>
              </div>

              <div>
                <Input
                  label="Password History Count"
                  type="number"
                  min="0"
                  value={formData.history_count}
                  onChange={(e) =>
                    setFormData({ ...formData, history_count: parseInt(e.target.value) || 0 })
                  }
                />
                <p className="mt-1 text-xs text-gray-400">
                  Number of previous passwords that cannot be reused
                </p>
              </div>
            </div>
          </div>
        </Card>

        {/* Forbidden Passwords */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Forbidden Passwords</h3>
            <p className="text-sm text-gray-400 mb-3">
              Common passwords that users are not allowed to use
            </p>
            <div className="flex gap-2 mb-3">
              <Input
                placeholder="Add a forbidden password..."
                value={forbiddenPasswordInput}
                onChange={(e) => setForbiddenPasswordInput(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addForbiddenPassword())}
                type="password"
              />
              <Button type="button" onClick={addForbiddenPassword}>
                Add
              </Button>
            </div>
            <div className="flex flex-wrap gap-2">
              {formData.forbidden_passwords.map((password) => (
                <span
                  key={password}
                  className="inline-flex items-center gap-1 rounded-full bg-danger-600/20 px-3 py-1 text-sm text-danger-400"
                >
                  ••••••
                  <button
                    type="button"
                    onClick={() => removeForbiddenPassword(password)}
                    className="hover:text-danger-300"
                  >
                    ×
                  </button>
                </span>
              ))}
              {formData.forbidden_passwords.length === 0 && (
                <p className="text-sm text-gray-400">No forbidden passwords</p>
              )}
            </div>
          </div>
        </Card>

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/policies/password">
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

import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save, Shield, Check, Loader2 } from 'lucide-react';
import { rolesApi } from '@/api/roles';
import { Button, Input, Textarea, Card, Badge } from '@/components/common';
import type { Role, Permission } from '@/types';
import toast from 'react-hot-toast';

const permissionResources = [
  'users',
  'roles',
  'targets',
  'credentials',
  'requests',
  'sessions',
  'audits',
  'vaults',
  'policies',
  'tenants',
];

const permissionActions = [
  { value: 'create', label: 'Create' },
  { value: 'read', label: 'View' },
  { value: 'update', label: 'Update' },
  { value: 'delete', label: 'Delete' },
  { value: 'approve', label: 'Approve' },
  { value: 'checkout', label: 'Checkout' },
  { value: 'checkin', label: 'Check-in' },
  { value: 'terminate', label: 'Terminate' },
  { value: 'export', label: 'Export' },
];

export const RoleFormPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEditing = Boolean(id);

  const [formData, setFormData] = useState({
    name: '',
    description: '',
  });

  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set());
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Fetch existing role for editing
  const { data: existingRole, isLoading: isLoadingRole } = useQuery({
    queryKey: ['role', id],
    queryFn: () => rolesApi.get(id!),
    enabled: isEditing,
  });

  // Fetch all available permissions
  const { data: allPermissions, isLoading: isLoadingPermissions } = useQuery({
    queryKey: ['permissions'],
    queryFn: () => rolesApi.listPermissions(),
  });

  // Populate form when editing
  useEffect(() => {
    if (existingRole) {
      setFormData({
        name: existingRole.name || '',
        description: existingRole.description || '',
      });
      setSelectedPermissions(
        new Set(existingRole.permissions?.map((p) => p.id) || [])
      );
    }
  }, [existingRole]);

  // Create mutation
  const createMutation = useMutation({
    mutationFn: (data: typeof formData & { permission_ids: string[] }) =>
      rolesApi.create(data),
    onSuccess: () => {
      toast.success('Role created successfully');
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      navigate('/roles');
    },
  });

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: (data: typeof formData & { permission_ids: string[] }) =>
      rolesApi.update(id!, data),
    onSuccess: () => {
      toast.success('Role updated successfully');
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      queryClient.invalidateQueries({ queryKey: ['role', id] });
    },
  });

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Name is required';
    }
    if (formData.name.length < 2) {
      newErrors.name = 'Name must be at least 2 characters';
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
      permission_ids: Array.from(selectedPermissions),
    };

    if (isEditing) {
      updateMutation.mutate(submitData);
    } else {
      createMutation.mutate(submitData);
    }
  };

  const togglePermission = (permissionId: string) => {
    const newSet = new Set(selectedPermissions);
    if (newSet.has(permissionId)) {
      newSet.delete(permissionId);
    } else {
      newSet.add(permissionId);
    }
    setSelectedPermissions(newSet);
  };

  const toggleAllForResource = (resource: string) => {
    const resourcePermissions = allPermissions?.filter((p) => p.resource === resource) || [];
    const allSelected = resourcePermissions.every((p) => selectedPermissions.has(p.id));

    const newSet = new Set(selectedPermissions);
    if (allSelected) {
      resourcePermissions.forEach((p) => newSet.delete(p.id));
    } else {
      resourcePermissions.forEach((p) => newSet.add(p.id));
    }
    setSelectedPermissions(newSet);
  };

  const getPermissionsForResource = (resource: string) => {
    return allPermissions?.filter((p) => p.resource === resource) || [];
  };

  const isAllSelectedForResource = (resource: string) => {
    const resourcePermissions = getPermissionsForResource(resource);
    if (resourcePermissions.length === 0) return false;
    return resourcePermissions.every((p) => selectedPermissions.has(p.id));
  };

  const isSomeSelectedForResource = (resource: string) => {
    const resourcePermissions = getPermissionsForResource(resource);
    return resourcePermissions.some((p) => selectedPermissions.has(p.id));
  };

  if (isEditing && isLoadingRole) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  if (isLoadingPermissions) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  // Don't allow editing system roles
  if (isEditing && existingRole?.is_system) {
    return (
      <div className="max-w-2xl mx-auto">
        <Card>
          <div className="card-body text-center py-12">
            <Shield className="h-12 w-12 text-warning-400 mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-white mb-2">System Role</h2>
            <p className="text-gray-400 mb-6">
              System roles cannot be modified. These roles are built into the platform.
            </p>
            <Link to="/roles">
              <Button>Back to Roles</Button>
            </Link>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Link to="/roles">
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
              {isEditing ? 'Edit Role' : 'Create Role'}
            </h1>
            <p className="mt-1 text-sm text-gray-400">
              {isEditing
                ? 'Update role permissions and settings'
                : 'Define a new role with specific permissions'}
            </p>
          </div>
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <Card>
          <div className="card-body">
            <h3 className="text-lg font-semibold text-white mb-4">Basic Information</h3>
            <div className="space-y-4">
              <div>
                <Input
                  label="Role Name"
                  placeholder="e.g., Database Administrator"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  error={errors.name}
                  required
                />
              </div>
              <div>
                <Textarea
                  label="Description"
                  placeholder="Optional description of this role..."
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Permissions */}
        <Card>
          <div className="card-body">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Permissions</h3>
              <span className="text-sm text-gray-400">
                {selectedPermissions.size} permission{selectedPermissions.size !== 1 ? 's' : ''} selected
              </span>
            </div>

            <div className="space-y-6">
              {permissionResources.map((resource) => {
                const resourcePermissions = getPermissionsForResource(resource);

                if (resourcePermissions.length === 0) {
                  return null;
                }

                const allSelected = isAllSelectedForResource(resource);
                const someSelected = isSomeSelectedForResource(resource);

                return (
                  <div key={resource} className="border border-gray-700 rounded-lg overflow-hidden">
                    {/* Resource Header */}
                    <button
                      type="button"
                      onClick={() => toggleAllForResource(resource)}
                      className="w-full flex items-center justify-between px-4 py-3 bg-gray-800 hover:bg-gray-750 transition-colors"
                    >
                      <div className="flex items-center gap-3">
                        <div
                          className={`h-5 w-5 rounded border flex items-center justify-center transition-colors ${
                            allSelected
                              ? 'bg-primary-600 border-primary-600'
                              : 'border-gray-600'
                          }`}
                        >
                          {allSelected && <Check className="h-3 w-3 text-white" />}
                        </div>
                        <span className="font-medium text-white capitalize">
                          {resource.replace('_', ' ')}
                        </span>
                      </div>
                      {someSelected && !allSelected && (
                        <Badge variant="neutral" size="sm">
                          Partial
                        </Badge>
                      )}
                    </button>

                    {/* Permission Actions Grid */}
                    <div className="p-4 bg-gray-900/50 grid grid-cols-3 sm:grid-cols-5 gap-2">
                      {permissionActions.map((action) => {
                        const permission = resourcePermissions.find(
                          (p) => p.action === action.value
                        );

                        if (!permission) {
                          return (
                            <div
                              key={`${resource}-${action.value}`}
                              className="p-2 opacity-30 cursor-not-allowed"
                            >
                              <span className="text-sm text-gray-400">{action.label}</span>
                            </div>
                          );
                        }

                        const isSelected = selectedPermissions.has(permission.id);

                        return (
                          <button
                            key={permission.id}
                            type="button"
                            onClick={() => togglePermission(permission.id)}
                            className={`p-2 rounded-md text-sm font-medium transition-colors ${
                              isSelected
                                ? 'bg-primary-600 text-white'
                                : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                            }`}
                          >
                            {action.label}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>

            {allPermissions && allPermissions.length === 0 && (
              <div className="text-center py-8">
                <p className="text-gray-400">No permissions available</p>
              </div>
            )}
          </div>
        </Card>

        {/* Summary */}
        {selectedPermissions.size > 0 && (
          <Card>
            <div className="card-body bg-primary-400/5 border border-primary-500/20">
              <h3 className="text-sm font-semibold text-primary-300 mb-2">
                Permission Summary
              </h3>
              <div className="flex flex-wrap gap-2">
                {Array.from(selectedPermissions)
                  .map((id) => allPermissions?.find((p) => p.id === id))
                  .filter(Boolean)
                  .map((permission) => (
                    <Badge key={permission!.id} variant="neutral" size="sm">
                      {permission!.resource}:{permission!.action}
                    </Badge>
                  ))}
              </div>
            </div>
          </Card>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-3">
          <Link to="/roles">
            <Button variant="secondary" type="button">
              Cancel
            </Button>
          </Link>
          <Button
            type="submit"
            isLoading={createMutation.isPending || updateMutation.isPending}
            leftIcon={<Save className="h-4 w-4" />}
          >
            {isEditing ? 'Save Changes' : 'Create Role'}
          </Button>
        </div>
      </form>
    </div>
  );
};

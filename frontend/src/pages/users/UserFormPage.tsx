import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Save } from 'lucide-react';
import { usersApi } from '@/api/users';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import type { UserRole, UserStatus } from '@/types';
import toast from 'react-hot-toast';

const roles: { value: UserRole; label: string }[] = [
  { value: 'super_admin', label: 'Super Admin' },
  { value: 'admin', label: 'Admin' },
  { value: 'operator', label: 'Operator' },
  { value: 'auditor', label: 'Auditor' },
  { value: 'requester', label: 'Requester' },
  { value: 'user', label: 'User' },
];

const statuses: { value: UserStatus; label: string }[] = [
  { value: 'active', label: 'Active' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'locked', label: 'Locked' },
  { value: 'pending', label: 'Pending' },
];

interface UserFormData {
  email: string;
  first_name: string;
  last_name: string;
  role: UserRole;
  status: UserStatus;
  password?: string;
  send_invite: boolean;
}

export const UserFormPage: React.FC = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const isEdit = !!id;

  const [formData, setFormData] = useState<UserFormData>({
    email: '',
    first_name: '',
    last_name: '',
    role: 'user',
    status: 'active',
    password: '',
    send_invite: true,
  });

  const [errors, setErrors] = useState<Partial<Record<keyof UserFormData, string>>>({});

  const { data: user, isLoading: isLoadingUser } = useQuery({
    queryKey: ['users', id],
    queryFn: () => usersApi.get(id!),
    enabled: isEdit,
  });

  useEffect(() => {
    if (user) {
      setFormData({
        email: user.email,
        first_name: user.first_name,
        last_name: user.last_name,
        role: user.role,
        status: user.status,
        password: '',
        send_invite: false,
      });
    }
  }, [user]);

  const createMutation = useMutation({
    mutationFn: (data: UserFormData) =>
      usersApi.create({
        email: data.email,
        first_name: data.first_name,
        last_name: data.last_name,
        role: data.role,
        password: data.password,
        send_invite: data.send_invite,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User created successfully');
      navigate('/users');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<UserFormData> }) =>
      usersApi.update(id, {
        first_name: data.first_name,
        last_name: data.last_name,
        role: data.role,
        status: data.status,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User updated successfully');
      navigate('/users');
    },
  });

  const validate = (): boolean => {
    const newErrors: Partial<Record<keyof UserFormData, string>> = {};

    if (!formData.email) {
      newErrors.email = 'Email is required';
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Invalid email format';
    }

    if (!formData.first_name) {
      newErrors.first_name = 'First name is required';
    }

    if (!formData.last_name) {
      newErrors.last_name = 'Last name is required';
    }

    if (!isEdit && !formData.password && !formData.send_invite) {
      newErrors.password = 'Password is required when not sending invite';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validate()) return;

    if (isEdit) {
      updateMutation.mutate({ id: id!, data: formData });
    } else {
      createMutation.mutate(formData);
    }
  };

  const handleChange = (field: keyof UserFormData) => (value: string | boolean) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  if (isEdit && isLoadingUser) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary-500 border-t-transparent" />
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-6 flex items-center gap-4">
        <button
          onClick={() => navigate('/users')}
          className="rounded p-2 text-gray-400 hover:bg-gray-800 hover:text-white"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-white">
            {isEdit ? 'Edit User' : 'Add New User'}
          </h1>
          <p className="mt-1 text-sm text-gray-400">
            {isEdit ? 'Update user information and permissions' : 'Create a new user account'}
          </p>
        </div>
      </div>

      <div className="card">
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Personal Information */}
          <div>
            <h3 className="text-lg font-medium text-white">Personal Information</h3>
            <p className="mt-1 text-sm text-gray-400">
              Basic information about the user
            </p>
          </div>

          <div className="grid gap-6 sm:grid-cols-2">
            <Input
              label="First Name"
              value={formData.first_name}
              onChange={(e) => handleChange('first_name')(e.target.value)}
              error={errors.first_name}
              required
            />

            <Input
              label="Last Name"
              value={formData.last_name}
              onChange={(e) => handleChange('last_name')(e.target.value)}
              error={errors.last_name}
              required
            />
          </div>

          <Input
            label="Email Address"
            type="email"
            value={formData.email}
            onChange={(e) => handleChange('email')(e.target.value)}
            error={errors.email}
            required
            disabled={isEdit}
          />

          {!isEdit && (
            <>
              <Input
                label="Password"
                type="password"
                value={formData.password}
                onChange={(e) => handleChange('password')(e.target.value)}
                error={errors.password}
                helperText="Minimum 8 characters. Leave empty to send invitation email."
              />

              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={formData.send_invite}
                  onChange={(e) => handleChange('send_invite')(e.target.checked)}
                  className="h-4 w-4 rounded border-gray-700 bg-gray-800 text-primary-600 focus:ring-primary-500"
                />
                <span className="text-sm text-gray-300">
                  Send invitation email to user
                </span>
              </label>
            </>
          )}

          {/* Role & Status */}
          <div className="border-t border-gray-700 pt-6">
            <h3 className="text-lg font-medium text-white">Role & Status</h3>
            <p className="mt-1 text-sm text-gray-400">
              Define user permissions and account status
            </p>
          </div>

          <Select
            label="Role"
            options={roles}
            value={formData.role}
            onChange={(e) => handleChange('role')(e.target.value)}
            placeholder="Select a role"
          />

          {isEdit && (
            <Select
              label="Status"
              options={statuses}
              value={formData.status}
              onChange={(e) => handleChange('status')(e.target.value as UserStatus)}
              placeholder="Select status"
            />
          )}

          {/* Actions */}
          <div className="flex justify-end gap-3 border-t border-gray-700 pt-6">
            <Button
              type="button"
              variant="secondary"
              onClick={() => navigate('/users')}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              isLoading={createMutation.isPending || updateMutation.isPending}
              leftIcon={<Save className="h-4 w-4" />}
            >
              {isEdit ? 'Save Changes' : 'Create User'}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
};

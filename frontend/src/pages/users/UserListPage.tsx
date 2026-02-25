import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, MoreVertical, UserPlus, Lock, Unlock, Trash2 } from 'lucide-react';
import { usersApi } from '@/api/users';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Table, Column } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { Modal } from '@/components/common';
import { Pagination } from '@/components/common';
import type { User } from '@/types';
import { toast } from 'react-hot-toast';
import clsx from 'clsx';

export const UserListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [roleFilter, setRoleFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [showActionMenu, setShowActionMenu] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ['users', { search, role: roleFilter, status: statusFilter, offset, limit }],
    queryFn: () =>
      usersApi.list({ search, role: roleFilter, status: statusFilter, offset, limit }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => usersApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User deleted successfully');
      setShowDeleteModal(false);
      setSelectedUser(null);
    },
  });

  const unlockMutation = useMutation({
    mutationFn: (id: string) => usersApi.unlock(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
      toast.success('User unlocked successfully');
    },
  });

  const resetPasswordMutation = useMutation({
    mutationFn: (id: string) => usersApi.resetPassword(id),
    onSuccess: () => {
      toast.success('Password reset email sent');
    },
  });

  const handleDelete = () => {
    if (selectedUser) {
      deleteMutation.mutate(selectedUser.id);
    }
  };

  const handleUnlock = (id: string) => {
    unlockMutation.mutate(id);
  };

  const handleResetPassword = (id: string) => {
    resetPasswordMutation.mutate(id);
  };

  const columns: Column<User>[] = [
    {
      key: 'display_name',
      header: 'User',
      render: (_value, row) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-sm font-medium text-white">
            {row.first_name[0]}{row.last_name[0]}
          </div>
          <div>
            <p className="font-medium text-white">
              {row.first_name} {row.last_name}
            </p>
            <p className="text-xs text-gray-400">{row.email}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      render: (value) => (
        <span className="inline-flex items-center rounded-md bg-gray-700 px-2 py-1 text-xs font-medium text-gray-300">
          {String(value).replace('_', ' ')}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (value) => <StatusBadge status={String(value)} />,
    },
    {
      key: 'mfa_enabled',
      header: 'MFA',
      render: (value) => (
        <span className={clsx('text-xs', value ? 'text-success-400' : 'text-warning-400')}>
          {value ? 'Enabled' : 'Disabled'}
        </span>
      ),
    },
    {
      key: 'last_login_at',
      header: 'Last Login',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {value ? new Date(String(value)).toLocaleDateString() : 'Never'}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="relative">
          <button
            onClick={() => setShowActionMenu(showActionMenu === row.id ? null : row.id)}
            className="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white"
          >
            <MoreVertical className="h-4 w-4" />
          </button>
          {showActionMenu === row.id && (
            <div className="dropdown-menu right-0 w-48">
              <Link
                to={`/users/${row.id}`}
                onClick={() => setShowActionMenu(null)}
                className="dropdown-item"
              >
                View Details
              </Link>
              <Link
                to={`/users/${row.id}/edit`}
                onClick={() => setShowActionMenu(null)}
                className="dropdown-item"
              >
                Edit
              </Link>
              {row.status === 'locked' && (
                <button
                  onClick={() => {
                    handleUnlock(row.id);
                    setShowActionMenu(null);
                  }}
                  className="dropdown-item w-full text-left"
                >
                  <Unlock className="mr-2 inline h-4 w-4" />
                  Unlock
                </button>
              )}
              <button
                onClick={() => {
                  handleResetPassword(row.id);
                  setShowActionMenu(null);
                }}
                className="dropdown-item w-full text-left"
              >
                Reset Password
              </button>
              <div className="border-t border-gray-700">
                <button
                  onClick={() => {
                    setSelectedUser(row);
                    setShowDeleteModal(true);
                    setShowActionMenu(null);
                  }}
                  className="dropdown-item w-full text-left text-danger-400 hover:text-danger-300"
                >
                  <Trash2 className="mr-2 inline h-4 w-4" />
                  Delete
                </button>
              </div>
            </div>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Users</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage user accounts and permissions
          </p>
        </div>
        <Link to="/users/new">
          <Button leftIcon={<UserPlus className="h-4 w-4" />}>
            Add User
          </Button>
        </Link>
      </div>

      {/* Filters */}
      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search users..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <select
              className="select w-auto"
              value={roleFilter}
              onChange={(e) => {
                setRoleFilter(e.target.value);
                setOffset(0);
              }}
            >
              <option value="">All Roles</option>
              <option value="super_admin">Super Admin</option>
              <option value="admin">Admin</option>
              <option value="operator">Operator</option>
              <option value="auditor">Auditor</option>
              <option value="requester">Requester</option>
              <option value="user">User</option>
            </select>
            <select
              className="select w-auto"
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setOffset(0);
              }}
            >
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="suspended">Suspended</option>
              <option value="locked">Locked</option>
              <option value="pending">Pending</option>
            </select>
          </div>
        </div>
      </div>

      {/* Table */}
      <div className="card">
        <Table
          data={data?.data || []}
          columns={columns}
          keyField="id"
          isLoading={isLoading}
          emptyMessage="No users found"
        />

        {data && data.pagination.total > limit && (
          <div className="card-footer">
            <Pagination
              total={data.pagination.total}
              limit={limit}
              offset={offset}
              onPageChange={setOffset}
            />
          </div>
        )}
      </div>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => {
          setShowDeleteModal(false);
          setSelectedUser(null);
        }}
        title="Delete User"
        size="sm"
      >
        <p className="text-gray-300">
          Are you sure you want to delete <strong>{selectedUser?.email}</strong>? This action
          cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <Button
            variant="secondary"
            onClick={() => {
              setShowDeleteModal(false);
              setSelectedUser(null);
            }}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={handleDelete}
            isLoading={deleteMutation.isPending}
          >
            Delete User
          </Button>
        </div>
      </Modal>
    </div>
  );
};

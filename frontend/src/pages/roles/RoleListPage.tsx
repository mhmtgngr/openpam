import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Shield, MoreVertical, Pencil, Trash2 } from 'lucide-react';
import { rolesApi } from '@/api/roles';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Table, Column } from '@/components/common';
import { Modal } from '@/components/common';
import { Pagination } from '@/components/common';
import type { Role } from '@/types';
import toast from 'react-hot-toast';

export const RoleListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;
  const [selectedRole, setSelectedRole] = useState<Role | null>(null);
  const [showDeleteModal, setShowDeleteModal] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ['roles', { search, offset, limit }],
    queryFn: () =>
      rolesApi.list({ search, offset, limit }).then((res) => res.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => rolesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['roles'] });
      toast.success('Role deleted successfully');
      setShowDeleteModal(false);
    },
  });

  const columns: Column<Role>[] = [
    {
      key: 'name',
      header: 'Role',
      render: (_value, row) => (
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600/20">
            <Shield className="h-5 w-5 text-primary-400" />
          </div>
          <div>
            <p className="font-medium text-white">{row.name}</p>
            <p className="text-xs text-gray-400">{row.description || 'No description'}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'is_system',
      header: 'Type',
      render: (value) => (
        <span className={`text-xs ${value ? 'text-gray-400' : 'text-primary-400'}`}>
          {value ? 'System' : 'Custom'}
        </span>
      ),
    },
    {
      key: 'permissions',
      header: 'Permissions',
      render: (value) => (
        <span className="text-sm text-gray-400">{Array.isArray(value) ? value.length : 0} permissions</span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          <Link to={`/roles/${row.id}`}>
            <Button variant="ghost" size="sm" leftIcon={<Pencil className="h-4 w-4" />}>
              Edit
            </Button>
          </Link>
          {!row.is_system && (
            <Button
              variant="ghost"
              size="sm"
              leftIcon={<Trash2 className="h-4 w-4" />}
              onClick={() => {
                setSelectedRole(row);
                setShowDeleteModal(true);
              }}
              className="text-danger-400 hover:text-danger-300"
            >
              Delete
            </Button>
          )}
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Roles & Permissions</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage user roles and their associated permissions
          </p>
        </div>
        <Link to="/roles/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Role
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <Input
            placeholder="Search roles..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setOffset(0);
            }}
          />
        </div>
      </div>

      <div className="card">
        <Table
          data={data?.data || []}
          columns={columns}
          keyField="id"
          isLoading={isLoading}
          emptyMessage="No roles found"
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

      <Modal
        isOpen={showDeleteModal}
        onClose={() => {
          setShowDeleteModal(false);
          setSelectedRole(null);
        }}
        title="Delete Role"
        size="sm"
      >
        <p className="text-gray-300">
          Are you sure you want to delete <strong>{selectedRole?.name}</strong>? This action
          cannot be undone.
        </p>
        <div className="flex justify-end gap-3 mt-6">
          <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={() => selectedRole && deleteMutation.mutate(selectedRole.id)}
            isLoading={deleteMutation.isPending}
          >
            Delete Role
          </Button>
        </div>
      </Modal>
    </div>
  );
};

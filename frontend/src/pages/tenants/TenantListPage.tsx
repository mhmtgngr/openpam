import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Building2, MoreVertical, Ban, Play, Edit, Trash2 } from 'lucide-react';
import { tenantsApi } from '@/api/tenants';
import { Button, Input, Select, Table, Column, StatusBadge, Modal, Textarea } from '@/components/common';
import type { Tenant } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const tenantStatuses = [
  { value: '', label: 'All Status' },
  { value: 'active', label: 'Active' },
  { value: 'suspended', label: 'Suspended' },
  { value: 'trial', label: 'Trial' },
];

export const TenantListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(null);
  const [showSuspendModal, setShowSuspendModal] = useState(false);
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [suspendReason, setSuspendReason] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['tenants', { search, status: statusFilter, offset, limit }],
    queryFn: () =>
      tenantsApi.list({ search, status: statusFilter, offset, limit }).then((res) => res.data),
  });

  const suspendMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      tenantsApi.suspend(id, reason),
    onSuccess: () => {
      toast.success('Tenant suspended');
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      setShowSuspendModal(false);
      setSuspendReason('');
    },
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) => tenantsApi.activate(id),
    onSuccess: () => {
      toast.success('Tenant activated');
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tenantsApi.delete(id),
    onSuccess: () => {
      toast.success('Tenant deleted');
      queryClient.invalidateQueries({ queryKey: ['tenants'] });
      setShowDeleteModal(false);
    },
  });

  const handleSuspend = () => {
    if (selectedTenant) {
      suspendMutation.mutate({ id: selectedTenant.id, reason: suspendReason });
    }
  };

  const handleDelete = () => {
    if (selectedTenant && confirm('Are you sure you want to delete this tenant? This action cannot be undone.')) {
      deleteMutation.mutate(selectedTenant.id);
    }
  };

  const columns: Column<Tenant>[] = [
    {
      key: 'name',
      header: 'Tenant',
      render: (_value, row) => (
        <div className="flex items-center gap-3">
          <div
            className="flex h-10 w-10 items-center justify-center rounded-lg text-white font-semibold"
            style={{ backgroundColor: row.primary_color || '#6366f1' }}
          >
            {row.name.charAt(0).toUpperCase()}
          </div>
          <div>
            <p className="font-medium text-white">{row.name}</p>
            <p className="text-xs text-gray-400">{row.slug}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (value) => {
        const status = String(value);
        const variantMap: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'neutral'> = {
          active: 'success',
          suspended: 'danger',
          trial: 'info',
        };
        return <StatusBadge status={status} className={variantMap[status] ? `badge-${variantMap[status]}` : ''} />;
      },
    },
    {
      key: 'max_users',
      header: 'Users',
      render: (_value, row) => (
        <span className="text-sm text-gray-300">
          {row.max_users} max
        </span>
      ),
    },
    {
      key: 'max_targets',
      header: 'Targets',
      render: (_value, row) => (
        <span className="text-sm text-gray-300">
          {row.max_targets} max
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {new Date(String(value)).toLocaleDateString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          {row.status === 'active' ? (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSelectedTenant(row);
                setShowSuspendModal(true);
              }}
              leftIcon={<Ban className="h-4 w-4" />}
            >
              Suspend
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => activateMutation.mutate(row.id)}
              isLoading={activateMutation.isPending}
              leftIcon={<Play className="h-4 w-4" />}
            >
              Activate
            </Button>
          )}
          <Link to={`/tenants/${row.id}`}>
            <Button variant="ghost" size="sm">
              Manage
            </Button>
          </Link>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Tenants</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage platform tenants and their configurations
          </p>
        </div>
        <Link to="/tenants/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Tenant
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search tenants..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={tenantStatuses}
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value);
                setOffset(0);
              }}
            />
          </div>
        </div>
      </div>

      <div className="card">
        <Table
          data={data?.data || []}
          columns={columns}
          keyField="id"
          isLoading={isLoading}
          emptyMessage="No tenants found"
        />

        {data && data.pagination.total > limit && (
          <div className="card-footer">
            <div className="flex items-center justify-between">
              <span className="text-sm text-gray-400">
                Showing {offset + 1} to {Math.min(offset + limit, data.pagination.total)} of{' '}
                {data.pagination.total} tenants
              </span>
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOffset(Math.max(0, offset - limit))}
                  disabled={offset === 0}
                >
                  Previous
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setOffset(offset + limit)}
                  disabled={offset + limit >= data.pagination.total}
                >
                  Next
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Suspend Modal */}
      {showSuspendModal && selectedTenant && (
        <Modal
          isOpen={showSuspendModal}
          onClose={() => {
            setShowSuspendModal(false);
            setSelectedTenant(null);
            setSuspendReason('');
          }}
          title="Suspend Tenant"
          size="sm"
        >
          <p className="mb-4 text-gray-300">
            Are you sure you want to suspend <strong>{selectedTenant.name}</strong>? Users will not be
            able to access the platform.
          </p>
          <Textarea
            label="Reason for suspension"
            placeholder="Provide a reason..."
            value={suspendReason}
            onChange={(e) => setSuspendReason(e.target.value)}
            rows={3}
          />
          <div className="flex justify-end gap-3 mt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setShowSuspendModal(false);
                setSuspendReason('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleSuspend}
              isLoading={suspendMutation.isPending}
            >
              Suspend Tenant
            </Button>
          </div>
        </Modal>
      )}

      {/* Delete Modal */}
      {showDeleteModal && selectedTenant && (
        <Modal
          isOpen={showDeleteModal}
          onClose={() => {
            setShowDeleteModal(false);
            setSelectedTenant(null);
          }}
          title="Delete Tenant"
          size="sm"
        >
          <p className="mb-4 text-gray-300">
            Are you sure you want to delete <strong>{selectedTenant.name}</strong>? This action cannot
            be undone and all data will be permanently lost.
          </p>
          <div className="flex justify-end gap-3 mt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setShowDeleteModal(false);
                setSelectedTenant(null);
              }}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleDelete}
              isLoading={deleteMutation.isPending}
            >
              Delete Tenant
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
};

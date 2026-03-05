import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Key, RefreshCw, Eye, EyeOff } from 'lucide-react';
import { credentialsApi } from '@/api/credentials';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Table, Column } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { Modal } from '@/components/common';
import { Pagination } from '@/components/common';
import type { Credential } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const credentialTypes = [
  { value: '', label: 'All Types' },
  { value: 'password', label: 'Password' },
  { value: 'ssh_key', label: 'SSH Key' },
  { value: 'api_key', label: 'API Key' },
  { value: 'certificate', label: 'Certificate' },
  { value: 'database', label: 'Database' },
  { value: 'service_account', label: 'Service Account' },
];

const rotationPolicies = [
  { value: '', label: 'All Policies' },
  { value: 'manual', label: 'Manual' },
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
  { value: 'monthly', label: 'Monthly' },
  { value: 'on_checkin', label: 'On Check-in' },
  { value: 'on_expiry', label: 'On Expiry' },
];

export const CredentialListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['credentials', { search, type: typeFilter, status: statusFilter, offset, limit }],
    queryFn: () =>
      credentialsApi.list({ search, type: typeFilter, status: statusFilter, offset, limit }),
  });

  const rotateMutation = useMutation({
    mutationFn: (id: string) => credentialsApi.rotate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['credentials'] });
      toast.success('Credential rotated successfully');
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string; code?: string } } } };
      const code = err?.response?.data?.error?.code;
      if (code === 'MFA_REQUIRED') {
        toast.error('MFA verification required to rotate credentials. Please re-authenticate.');
      } else {
        toast.error(err?.response?.data?.error?.message || 'Failed to rotate credential. Please try again.');
      }
    },
  });

  const columns: Column<Credential>[] = [
    {
      key: 'name',
      header: 'Credential',
      render: (_value, row) => (
        <div className="flex items-center gap-3">
          <div className={clsx(
            'flex h-10 w-10 items-center justify-center rounded-lg',
            row.status === 'active' ? 'bg-success-400/20 text-success-400' :
            row.status === 'expiring' ? 'bg-warning-400/20 text-warning-400' :
            'bg-gray-700 text-gray-400'
          )}>
            <Key className="h-5 w-5" />
          </div>
          <div>
            <p className="font-medium text-white">{row.name}</p>
            <p className="text-xs text-gray-400">{row.username}@{row.target?.host || 'Unknown'}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      render: (value) => (
        <span className="inline-flex items-center rounded bg-gray-700 px-2 py-1 text-xs font-medium text-gray-300">
          {String(value).replace('_', ' ')}
        </span>
      ),
    },
    {
      key: 'rotation_policy',
      header: 'Rotation',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {String(value).replace('_', ' ')}
        </span>
      ),
    },
    {
      key: 'last_rotated_at',
      header: 'Last Rotated',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {value ? new Date(String(value)).toLocaleDateString() : 'Never'}
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (value) => <StatusBadge status={String(value)} />,
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => rotateMutation.mutate(row.id)}
            isLoading={rotateMutation.isPending}
            leftIcon={<RefreshCw className="h-4 w-4" />}
          >
            Rotate
          </Button>
          <Link to={`/credentials/${row.id}`}>
            <Button variant="ghost" size="sm">
              View
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
          <h1 className="text-2xl font-bold text-white">Credential Vault</h1>
          <p className="mt-1 text-sm text-gray-400">
            Securely store and manage credentials
          </p>
        </div>
        <Link to="/credentials/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Credential
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search credentials..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={credentialTypes}
              value={typeFilter}
              onChange={(e) => {
                setTypeFilter(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={[
                { value: '', label: 'All Status' },
                { value: 'active', label: 'Active' },
                { value: 'expiring', label: 'Expiring Soon' },
                { value: 'expired', label: 'Expired' },
              ]}
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
          emptyMessage="No credentials found"
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
    </div>
  );
};

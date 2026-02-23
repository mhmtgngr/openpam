import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Server, Play, Trash2, MoreVertical } from 'lucide-react';
import { targetsApi } from '@/api/targets';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Table, Column } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { Modal } from '@/components/common';
import { Pagination } from '@/components/common';
import type { Target } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const targetTypes = [
  { value: '', label: 'All Types' },
  { value: 'ssh', label: 'SSH' },
  { value: 'rdp', label: 'RDP' },
  { value: 'database', label: 'Database' },
  { value: 'kubernetes', label: 'Kubernetes' },
  { value: 'web', label: 'Web' },
  { value: 'api', label: 'API' },
];

const environments = [
  { value: '', label: 'All Environments' },
  { value: 'production', label: 'Production' },
  { value: 'staging', label: 'Staging' },
  { value: 'development', label: 'Development' },
  { value: 'test', label: 'Test' },
];

export const TargetListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [envFilter, setEnvFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;

  const { data, isLoading } = useQuery({
    queryKey: ['targets', { search, type: typeFilter, environment: envFilter, offset, limit }],
    queryFn: () =>
      targetsApi.list({ search, type: typeFilter, environment: envFilter, offset, limit }).then((res) => res.data),
  });

  const testConnectionMutation = useMutation({
    mutationFn: (id: string) => targetsApi.testConnection(id),
    onSuccess: (data) => {
      if (data.data.status === 'online') {
        toast.success(`Connection successful! Latency: ${data.data.latency_ms}ms`);
      } else {
        toast.error('Connection failed');
      }
    },
  });

  const getTypeIcon = (type: string) => {
    return <Server className="h-4 w-4" />;
  };

  const columns: Column<Target>[] = [
    {
      key: 'name',
      header: 'Target',
      render: (_value, row) => (
        <div className="flex items-center gap-3">
          <div className={clsx(
            'flex h-10 w-10 items-center justify-center rounded-lg',
            row.status === 'online' ? 'bg-success-400/20 text-success-400' : 'bg-gray-700 text-gray-400'
          )}>
            {getTypeIcon(row.type)}
          </div>
          <div>
            <p className="font-medium text-white">{row.name}</p>
            <p className="text-xs text-gray-400">{row.host}:{row.port}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'type',
      header: 'Type',
      render: (value) => (
        <span className="inline-flex items-center rounded bg-gray-700 px-2 py-1 text-xs font-medium text-gray-300">
          {String(value).toUpperCase()}
        </span>
      ),
    },
    {
      key: 'environment',
      header: 'Environment',
      render: (value) => {
        const envColors: Record<string, string> = {
          production: 'badge-danger',
          staging: 'badge-warning',
          development: 'badge-info',
          test: 'badge-neutral',
        };
        return <span className={clsx('badge', envColors[String(value)] || 'badge-neutral')}>
          {String(value)}
        </span>;
      },
    },
    {
      key: 'status',
      header: 'Status',
      render: (value) => <StatusBadge status={String(value)} />,
    },
    {
      key: 'sensitivity',
      header: 'Sensitivity',
      render: (value) => {
        const colors: Record<string, string> = {
          high: 'text-danger-400',
          medium: 'text-warning-400',
          low: 'text-success-400',
        };
        return <span className={clsx('text-sm capitalize', colors[String(value)])}>
          {String(value)}
        </span>;
      },
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => testConnectionMutation.mutate(row.id)}
            isLoading={testConnectionMutation.isPending}
            leftIcon={<Play className="h-4 w-4" />}
          >
            Test
          </Button>
          <Link to={`/targets/${row.id}`}>
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
          <h1 className="text-2xl font-bold text-white">Targets</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage target systems for privileged access
          </p>
        </div>
        <Link to="/targets/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Target
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search targets..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={targetTypes}
              value={typeFilter}
              onChange={(e) => {
                setTypeFilter(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={environments}
              value={envFilter}
              onChange={(e) => {
                setEnvFilter(e.target.value);
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
          emptyMessage="No targets found"
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

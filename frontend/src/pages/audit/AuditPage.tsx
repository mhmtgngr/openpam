import React, { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Download, Search, Filter, FileText } from 'lucide-react';
import { auditApi } from '@/api/audit';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Table, Column } from '@/components/common';
import { Badge } from '@/components/common';
import { Pagination } from '@/components/common';
import type { AuditEvent } from '@/types';
import { format } from 'date-fns';
import clsx from 'clsx';

const outcomes = [
  { value: '', label: 'All Outcomes' },
  { value: 'success', label: 'Success' },
  { value: 'failure', label: 'Failure' },
  { value: 'denied', label: 'Denied' },
];

const resourceTypes = [
  { value: '', label: 'All Resources' },
  { value: 'users', label: 'Users' },
  { value: 'targets', label: 'Targets' },
  { value: 'credentials', label: 'Credentials' },
  { value: 'requests', label: 'Requests' },
  { value: 'sessions', label: 'Sessions' },
];

export const AuditPage: React.FC = () => {
  const [search, setSearch] = useState('');
  const [outcomeFilter, setOutcomeFilter] = useState('');
  const [resourceFilter, setResourceFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 50;

  const { data, isLoading } = useQuery({
    queryKey: ['audit', { search, outcome: outcomeFilter, resource_type: resourceFilter, offset, limit }],
    queryFn: () =>
      auditApi.list({ search, outcome: outcomeFilter, resource_type: resourceFilter, offset, limit }).then((res) => res.data),
  });

  const handleExport = async () => {
    try {
      const response = await auditApi.export({ search, outcome: outcomeFilter, resource_type: resourceFilter, format: 'csv' });
      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;
      link.setAttribute('download', `audit-log-${format(new Date(), 'yyyy-MM-dd')}.csv`);
      document.body.appendChild(link);
      link.click();
      link.remove();
    } catch {
      // Error handled by interceptor
    }
  };

  const columns: Column<AuditEvent>[] = [
    {
      key: 'created_at',
      header: 'Timestamp',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {format(new Date(String(value)), 'yyyy-MM-dd HH:mm:ss')}
        </span>
      ),
    },
    {
      key: 'actor',
      header: 'Actor',
      render: (_value, row) => (
        <div>
          <p className="text-sm text-white">{row.actor?.email || 'System'}</p>
          <p className="text-xs text-gray-500">{row.actor_ip}</p>
        </div>
      ),
    },
    {
      key: 'action',
      header: 'Action',
      render: (value) => (
        <span className="text-sm font-mono text-gray-300">
          {String(value).replace(/_/g, ' ')}
        </span>
      ),
    },
    {
      key: 'resource_type',
      header: 'Resource',
      render: (_value, row) => (
        <div>
          <p className="text-sm text-white capitalize">{row.resource_type}</p>
          <p className="text-xs text-gray-500">{row.resource_name || row.resource_id}</p>
        </div>
      ),
    },
    {
      key: 'outcome',
      header: 'Outcome',
      render: (value) => {
        const outcome = String(value);
        const variants: Record<string, 'success' | 'danger' | 'warning'> = {
          success: 'success',
          failure: 'danger',
          denied: 'danger',
          partial: 'warning',
        };
        return (
          <Badge variant={variants[outcome] || 'neutral'}>
            {outcome}
          </Badge>
        );
      },
    },
    {
      key: 'details',
      header: 'Details',
      render: (value) => {
        if (!value || typeof value !== 'object') return '-';
        const details = value as Record<string, unknown>;
        const entries = Object.entries(details).slice(0, 2);
        return (
          <div className="max-w-xs truncate text-xs text-gray-500">
            {entries.map(([k, v]) => `${k}=${v}`).join(', ')}
          </div>
        );
      },
    },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Audit Logs</h1>
          <p className="mt-1 text-sm text-gray-400">
            Complete audit trail of all system activities
          </p>
        </div>
        <Button
          variant="secondary"
          leftIcon={<Download className="h-4 w-4" />}
          onClick={handleExport}
        >
          Export
        </Button>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search audit logs..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={resourceTypes}
              value={resourceFilter}
              onChange={(e) => {
                setResourceFilter(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={outcomes}
              value={outcomeFilter}
              onChange={(e) => {
                setOutcomeFilter(e.target.value);
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
          emptyMessage="No audit events found"
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

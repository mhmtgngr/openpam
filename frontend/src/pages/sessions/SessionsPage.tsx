import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Search, Activity, Ban, Play, Eye } from 'lucide-react';
import { sessionsApi } from '@/api/sessions';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Table, Column } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { Modal } from '@/components/common';
import { Textarea } from '@/components/common';
import { Pagination } from '@/components/common';
import type { Session } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const sessionStatuses = [
  { value: '', label: 'All Status' },
  { value: 'active', label: 'Active' },
  { value: 'ended', label: 'Ended' },
  { value: 'terminated', label: 'Terminated' },
];

const sessionTypes = [
  { value: '', label: 'All Types' },
  { value: 'ssh', label: 'SSH' },
  { value: 'rdp', label: 'RDP' },
  { value: 'database', label: 'Database' },
  { value: 'kubernetes', label: 'Kubernetes' },
];

export const SessionsPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [showTerminateModal, setShowTerminateModal] = useState(false);
  const [terminateReason, setTerminateReason] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['sessions', { search, status: statusFilter, type: typeFilter, offset, limit }],
    queryFn: () =>
      sessionsApi.list({ search, status: statusFilter, type: typeFilter, offset, limit }),
    refetchInterval: (query) => {
      // Refetch more frequently when there are active sessions
      const hasActive = query.state.data?.data?.some((s: Session) => s.status === 'active');
      return hasActive ? 10000 : false; // 10 seconds if active, otherwise no refetch
    },
  });

  const terminateMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      sessionsApi.terminate(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sessions'] });
      toast.success('Session terminated successfully');
      setShowTerminateModal(false);
      setTerminateReason('');
      setSelectedSession(null);
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { error?: { message?: string; code?: string } } } };
      const code = err?.response?.data?.error?.code;
      if (code === 'MFA_REQUIRED') {
        toast.error('MFA verification required to terminate sessions.');
      } else {
        toast.error(err?.response?.data?.error?.message || 'Failed to terminate session. Please try again.');
      }
    },
  });

  const columns: Column<Session>[] = [
    {
      key: 'user',
      header: 'User',
      render: (_value, row) => (
        <div>
          <p className="font-medium text-white">
            {row.user?.first_name} {row.user?.last_name}
          </p>
          <p className="text-xs text-gray-400">{row.user?.email}</p>
        </div>
      ),
    },
    {
      key: 'target',
      header: 'Target',
      render: (_value, row) => (
        <div>
          <p className="text-sm text-white">{row.target?.name || 'Unknown'}</p>
          <p className="text-xs text-gray-400">{row.type}</p>
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
      key: 'status',
      header: 'Status',
      render: (value) => {
        const status = String(value);
        if (status === 'active') {
          return (
            <div className="flex items-center gap-2">
              <span className="relative flex h-2 w-2">
                <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-success-400 opacity-75"></span>
                <span className="relative inline-flex rounded-full h-2 w-2 bg-success-500"></span>
              </span>
              <StatusBadge status={status} />
            </div>
          );
        }
        return <StatusBadge status={status} />;
      },
    },
    {
      key: 'started_at',
      header: 'Started',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {new Date(String(value)).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'duration_seconds',
      header: 'Duration',
      render: (value, row) => {
        if (row.status === 'active') {
          const start = new Date(row.started_at).getTime();
          const now = Date.now();
          const seconds = Math.floor((now - start) / 1000);
          const mins = Math.floor(seconds / 60);
          const secs = seconds % 60;
          return <span className="text-sm text-gray-400">{mins}m {secs}s</span>;
        }
        return (
          <span className="text-sm text-gray-400">
            {value ? `${Math.floor(Number(value) / 60)}m` : '-'}
          </span>
        );
      },
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          {row.status === 'active' && row.can_terminate && (
            <Button
              variant="danger"
              size="sm"
              onClick={() => {
                setSelectedSession(row);
                setShowTerminateModal(true);
              }}
              leftIcon={<Ban className="h-4 w-4" />}
            >
              Terminate
            </Button>
          )}
          <Link to={`/sessions/${row.id}`}>
            <Button variant="ghost" size="sm" leftIcon={<Eye className="h-4 w-4" />}>
              {row.status === 'active' ? 'Monitor' : 'View'}
            </Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleTerminate = () => {
    if (selectedSession) {
      terminateMutation.mutate({
        id: selectedSession.id,
        reason: terminateReason || undefined,
      });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Active Sessions</h1>
          <p className="mt-1 text-sm text-gray-400">
            Monitor and manage privileged access sessions
          </p>
        </div>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search sessions..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={sessionTypes}
              value={typeFilter}
              onChange={(e) => {
                setTypeFilter(e.target.value);
                setOffset(0);
              }}
            />
            <Select
              options={sessionStatuses}
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
          emptyMessage="No sessions found"
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
        isOpen={showTerminateModal}
        onClose={() => {
          setShowTerminateModal(false);
          setSelectedSession(null);
          setTerminateReason('');
        }}
        title="Terminate Session"
        size="sm"
      >
        <p className="mb-4 text-gray-300">
          Are you sure you want to terminate this session? This action cannot be undone.
        </p>
        <Textarea
          label="Reason (optional)"
          placeholder="Provide a reason for terminating this session..."
          value={terminateReason}
          onChange={(e) => setTerminateReason(e.target.value)}
          rows={3}
        />
        <div className="flex justify-end gap-3 mt-4">
          <Button
            variant="secondary"
            onClick={() => {
              setShowTerminateModal(false);
              setTerminateReason('');
            }}
          >
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={handleTerminate}
            isLoading={terminateMutation.isPending}
          >
            Terminate Session
          </Button>
        </div>
      </Modal>
    </div>
  );
};

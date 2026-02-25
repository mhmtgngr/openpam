import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Clock, Check, X, FileText } from 'lucide-react';
import { requestsApi } from '@/api/requests';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Table, Column } from '@/components/common';
import { StatusBadge } from '@/components/common';
import { Modal } from '@/components/common';
import { Pagination } from '@/components/common';
import type { AccessRequest } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const requestStatuses = [
  { value: '', label: 'All Status' },
  { value: 'pending', label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'denied', label: 'Denied' },
  { value: 'active', label: 'Active' },
  { value: 'completed', label: 'Completed' },
  { value: 'cancelled', label: 'Cancelled' },
];

export const MyRequestsPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [offset, setOffset] = useState(0);
  const limit = 20;
  const [selectedRequest, setSelectedRequest] = useState<AccessRequest | null>(null);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['my-requests', { status: statusFilter, offset, limit }],
    queryFn: () =>
      requestsApi.myRequests({ status: statusFilter, offset, limit }),
  });

  const cancelMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      requestsApi.cancel(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['my-requests'] });
      toast.success('Request cancelled successfully');
      setShowCancelModal(false);
      setCancelReason('');
    },
  });

  const columns: Column<AccessRequest>[] = [
    {
      key: 'target',
      header: 'Target',
      render: (_value, row) => (
        <div>
          <p className="font-medium text-white">{row.target?.name || 'Unknown'}</p>
          <p className="text-xs text-gray-400">{row.type.replace('_', ' ')}</p>
        </div>
      ),
    },
    {
      key: 'reason',
      header: 'Reason',
      render: (value) => (
        <p className="max-w-xs truncate text-sm text-gray-300" title={String(value)}>
          {String(value)}
        </p>
      ),
    },
    {
      key: 'duration_minutes',
      header: 'Duration',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {Number(value)} minutes
        </span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      render: (value) => <StatusBadge status={String(value)} />,
    },
    {
      key: 'created_at',
      header: 'Created',
      render: (value) => (
        <span className="text-sm text-gray-400">
          {new Date(String(value)).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_value, row) => (
        <div className="flex justify-end gap-2">
          {row.status === 'pending' && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSelectedRequest(row);
                setShowCancelModal(true);
              }}
              className="text-danger-400 hover:text-danger-300"
            >
              Cancel
            </Button>
          )}
          {row.status === 'approved' && !row.expires_at && (
            <Link to={`/sessions/new?request=${row.id}`}>
              <Button variant="primary" size="sm">
                Start Session
              </Button>
            </Link>
          )}
        </div>
      ),
    },
  ];

  const handleCancel = () => {
    if (selectedRequest && cancelReason.trim()) {
      cancelMutation.mutate({ id: selectedRequest.id, reason: cancelReason });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">My Requests</h1>
          <p className="mt-1 text-sm text-gray-400">
            View and manage your access requests
          </p>
        </div>
        <Link to="/requests/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            New Request
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <div className="flex flex-wrap gap-4">
            <div className="flex-1 min-w-[200px]">
              <Input
                placeholder="Search requests..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => {
                  setSearch(e.target.value);
                  setOffset(0);
                }}
              />
            </div>
            <Select
              options={requestStatuses}
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
          emptyMessage="No requests found"
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
        isOpen={showCancelModal}
        onClose={() => {
          setShowCancelModal(false);
          setSelectedRequest(null);
          setCancelReason('');
        }}
        title="Cancel Request"
        size="sm"
      >
        <p className="mb-4 text-gray-300">
          Are you sure you want to cancel this request? Please provide a reason.
        </p>
        <textarea
          className="textarea mb-4"
          placeholder="Reason for cancellation..."
          value={cancelReason}
          onChange={(e) => setCancelReason(e.target.value)}
          rows={3}
        />
        <div className="flex justify-end gap-3">
          <Button
            variant="secondary"
            onClick={() => {
              setShowCancelModal(false);
              setCancelReason('');
            }}
          >
            Keep Request
          </Button>
          <Button
            variant="danger"
            onClick={handleCancel}
            isLoading={cancelMutation.isPending}
            disabled={!cancelReason.trim()}
          >
            Cancel Request
          </Button>
        </div>
      </Modal>
    </div>
  );
};

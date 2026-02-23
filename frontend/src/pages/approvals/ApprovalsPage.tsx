import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Search, Check, X, Eye } from 'lucide-react';
import { requestsApi } from '@/api/requests';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Table, Column, Modal, Textarea } from '@/components/common';
import type { AccessRequest } from '@/types';
import toast from 'react-hot-toast';

export const ApprovalsPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [selectedRequest, setSelectedRequest] = useState<AccessRequest | null>(null);
  const [showActionModal, setShowActionModal] = useState<'approve' | 'deny' | false>(false);
  const [comment, setComment] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['pending-approvals'],
    queryFn: () =>
      requestsApi.pendingApprovals().then((res) => res.data),
    refetchInterval: 30000, // Poll every 30 seconds
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, comment }: { id: string; comment?: string }) =>
      requestsApi.approve(id, comment),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pending-approvals'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'pending'] });
      toast.success('Request approved');
      setShowActionModal(false);
      setComment('');
    },
  });

  const denyMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      requestsApi.deny(id, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['pending-approvals'] });
      queryClient.invalidateQueries({ queryKey: ['dashboard', 'pending'] });
      toast.success('Request denied');
      setShowActionModal(false);
      setComment('');
    },
  });

  const columns: Column<AccessRequest>[] = [
    {
      key: 'user',
      header: 'Requester',
      render: (_value: unknown, row: AccessRequest) => (
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
      render: (_value: unknown, row: AccessRequest) => (
        <div>
          <p className="text-sm text-white">{row.target?.name || 'Unknown'}</p>
          <p className="text-xs text-gray-400">{row.type.replace('_', ' ')}</p>
        </div>
      ),
    },
    {
      key: 'reason',
      header: 'Reason',
      render: (value: unknown) => (
        <p className="max-w-xs truncate text-sm text-gray-300" title={String(value)}>
          {String(value)}
        </p>
      ),
    },
    {
      key: 'duration_minutes',
      header: 'Duration',
      render: (value: unknown) => (
        <span className="text-sm text-gray-400">
          {Number(value)} minutes
        </span>
      ),
    },
    {
      key: 'created_at',
      header: 'Requested',
      render: (value: unknown) => (
        <span className="text-sm text-gray-400">
          {new Date(String(value)).toLocaleString()}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_value: unknown, row: AccessRequest) => (
        <div className="flex justify-end gap-2">
          <Button
            variant="success"
            size="sm"
            onClick={() => {
              setSelectedRequest(row);
              setShowActionModal('approve');
            }}
            leftIcon={<Check className="h-4 w-4" />}
          >
            Approve
          </Button>
          <Button
            variant="danger"
            size="sm"
            onClick={() => {
              setSelectedRequest(row);
              setShowActionModal('deny');
            }}
            leftIcon={<X className="h-4 w-4" />}
          >
            Deny
          </Button>
          <Link to={`/requests/${row.id}`}>
            <Button variant="ghost" size="sm" leftIcon={<Eye className="h-4 w-4" />}>
              Details
            </Button>
          </Link>
        </div>
      ),
    },
  ];

  const handleAction = () => {
    if (!selectedRequest) return;

    if (showActionModal === 'approve') {
      approveMutation.mutate({ id: selectedRequest.id, comment: comment || undefined });
    } else {
      if (!comment.trim()) {
        toast.error('Please provide a reason for denial');
        return;
      }
      denyMutation.mutate({ id: selectedRequest.id, reason: comment });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Pending Approvals</h1>
          <p className="mt-1 text-sm text-gray-400">
            Review and respond to access requests
          </p>
        </div>
      </div>

      <div className="card">
        <div className="card-body">
          <Input
            placeholder="Search requests..."
            leftIcon={<Search className="h-4 w-4 text-gray-400" />}
          />
        </div>
      </div>

      <div className="card">
        <Table
          data={data?.data || []}
          columns={columns}
          keyField="id"
          isLoading={isLoading}
          emptyMessage="No pending approvals"
        />
      </div>

      <Modal
        isOpen={showActionModal !== false}
        onClose={() => {
          setShowActionModal(false);
          setSelectedRequest(null);
          setComment('');
        }}
        title={showActionModal === 'approve' ? 'Approve Request' : 'Deny Request'}
        size="sm"
      >
        <div className="space-y-4">
          <div>
            <p className="text-sm text-gray-400">Requester</p>
            <p className="text-white">
              {selectedRequest?.user?.first_name} {selectedRequest?.user?.last_name}
            </p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Target</p>
            <p className="text-white">{selectedRequest?.target?.name}</p>
          </div>
          <div>
            <p className="text-sm text-gray-400">Reason</p>
            <p className="text-white">{selectedRequest?.reason}</p>
          </div>

          <Textarea
            label={showActionModal === 'approve' ? 'Comment (optional)' : 'Reason for denial'}
            placeholder={showActionModal === 'approve' ? 'Add a note...' : 'Please explain why this request is being denied...'}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            rows={3}
          />

          <div className="flex justify-end gap-3 pt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setShowActionModal(false);
                setComment('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant={showActionModal === 'approve' ? 'success' : 'danger'}
              onClick={handleAction}
              isLoading={approveMutation.isPending || denyMutation.isPending}
            >
              {showActionModal === 'approve' ? 'Approve' : 'Deny'}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

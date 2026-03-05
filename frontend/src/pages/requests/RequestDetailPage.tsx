import React, { useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ArrowLeft, Check, X, Send, Clock, MessageSquare, User, Calendar, AlertCircle } from 'lucide-react';
import { requestsApi } from '@/api/requests';
import { usersApi } from '@/api/users';
import { useAuth } from '@/contexts/AuthContext';
import { canApproveRequests } from '@/utils/permissions';
import { Button, Card, Badge, Textarea, Modal, Table, Column } from '@/components/common';
import type { AccessRequest, RequestComment, User as UserType } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

export const RequestDetailPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { user } = useAuth();

  const [showDenyModal, setShowDenyModal] = useState(false);
  const [denyReason, setDenyReason] = useState('');
  const [showCommentModal, setShowCommentModal] = useState(false);
  const [commentText, setCommentText] = useState('');

  // Fetch request details
  const { data: request, isLoading } = useQuery({
    queryKey: ['request', id],
    queryFn: () => requestsApi.get(id!),
  });

  // Fetch users for delegation
  const { data: users } = useQuery({
    queryKey: ['users'],
    queryFn: () => usersApi.list({ limit: 100 }),
  });

  // Approve mutation
  const approveMutation = useMutation({
    mutationFn: (comment?: string) => requestsApi.approve(id!, comment),
    onSuccess: () => {
      toast.success('Request approved');
      queryClient.invalidateQueries({ queryKey: ['request', id] });
      queryClient.invalidateQueries({ queryKey: ['requests'] });
    },
  });

  // Deny mutation
  const denyMutation = useMutation({
    mutationFn: (reason: string) => requestsApi.deny(id!, reason),
    onSuccess: () => {
      toast.success('Request denied');
      queryClient.invalidateQueries({ queryKey: ['request', id] });
      queryClient.invalidateQueries({ queryKey: ['requests'] });
      setShowDenyModal(false);
      setDenyReason('');
    },
  });

  // Cancel mutation
  const cancelMutation = useMutation({
    mutationFn: (reason: string) => requestsApi.cancel(id!, reason),
    onSuccess: () => {
      toast.success('Request cancelled');
      queryClient.invalidateQueries({ queryKey: ['request', id] });
      queryClient.invalidateQueries({ queryKey: ['requests'] });
      navigate('/requests/my');
    },
  });

  // Delegate mutation
  const delegateMutation = useMutation({
    mutationFn: ({ toUserId, reason }: { toUserId: string; reason: string }) =>
      requestsApi.delegate(id!, toUserId, reason),
    onSuccess: () => {
      toast.success('Request delegated');
      queryClient.invalidateQueries({ queryKey: ['request', id] });
      queryClient.invalidateQueries({ queryKey: ['requests'] });
    },
  });

  // Add comment mutation
  const addCommentMutation = useMutation({
    mutationFn: (content: string) => requestsApi.addComment(id!, content),
    onSuccess: () => {
      toast.success('Comment added');
      queryClient.invalidateQueries({ queryKey: ['request', id] });
      setShowCommentModal(false);
      setCommentText('');
    },
  });

  const handleApprove = () => {
    if (window.confirm('Are you sure you want to approve this request?')) {
      approveMutation.mutate(undefined);
    }
  };

  const handleDeny = () => {
    if (!denyReason.trim()) {
      toast.error('Please provide a reason for denial');
      return;
    }
    denyMutation.mutate(denyReason);
  };

  const handleCancel = () => {
    const reason = prompt('Please provide a reason for cancellation:');
    if (reason) {
      cancelMutation.mutate(reason);
    }
  };

  const handleAddComment = () => {
    if (!commentText.trim()) {
      toast.error('Please enter a comment');
      return;
    }
    addCommentMutation.mutate(commentText);
  };

  const isPending = request?.status === 'pending';
  const canApprove = isPending && canApproveRequests(user);
  const isFinalized = request?.status === 'approved' || request?.status === 'denied' || request?.status === 'cancelled';

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[50vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-500" />
      </div>
    );
  }

  if (!request) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[50vh] gap-4">
        <h2 className="text-xl font-semibold text-white">Request not found</h2>
        <Link to="/requests/my">
          <Button>Back to My Requests</Button>
        </Link>
      </div>
    );
  }

  const statusVariant: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'neutral'> = {
    pending: 'warning',
    approved: 'success',
    denied: 'danger',
    cancelled: 'neutral',
    active: 'success',
    completed: 'neutral',
    expired: 'neutral',
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link to="/requests/my">
            <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
              Back
            </Button>
          </Link>
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-bold text-white">Request {id?.slice(0, 8)}</h1>
              <Badge variant={statusVariant[request.status] || 'neutral'}>
                {request.status}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-gray-400">
              Created {new Date(request.created_at).toLocaleString()}
            </p>
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2">
          {canApprove && (
            <>
              <Button
                variant="danger"
                size="sm"
                leftIcon={<X className="h-4 w-4" />}
                onClick={() => setShowDenyModal(true)}
              >
                Deny
              </Button>
              <Button
                variant="success"
                size="sm"
                leftIcon={<Check className="h-4 w-4" />}
                onClick={handleApprove}
                isLoading={approveMutation.isPending}
              >
                Approve
              </Button>
            </>
          )}
          {isPending && !canApprove && (
            <Button
              variant="secondary"
              size="sm"
              onClick={handleCancel}
              isLoading={cancelMutation.isPending}
            >
              Cancel Request
            </Button>
          )}
          <Button
            variant="ghost"
            size="sm"
            leftIcon={<MessageSquare className="h-4 w-4" />}
            onClick={() => setShowCommentModal(true)}
          >
            Add Comment
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Request Details */}
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Request Details</h3>
              <dl className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm text-gray-400">Request Type</dt>
                  <dd className="mt-1 text-sm text-white capitalize">
                    {request.type.replace('_', ' ')}
                  </dd>
                </div>
                <div>
                  <dt className="text-sm text-gray-400">Duration</dt>
                  <dd className="mt-1 text-sm text-white">{request.duration_minutes} minutes</dd>
                </div>
                <div className="col-span-2">
                  <dt className="text-sm text-gray-400">Target</dt>
                  <dd className="mt-1 text-sm text-white">
                    {request.target?.name}
                    <p className="text-xs text-gray-400">
                      {request.target?.type} - {request.target?.host}:{request.target?.port}
                    </p>
                  </dd>
                </div>
                {request.credential && (
                  <div className="col-span-2">
                    <dt className="text-sm text-gray-400">Credential</dt>
                    <dd className="mt-1 text-sm text-white">
                      {request.credential.name} ({request.credential.username})
                    </dd>
                  </div>
                )}
                {request.scheduled_start_at && (
                  <div>
                    <dt className="text-sm text-gray-400">Scheduled Start</dt>
                    <dd className="mt-1 text-sm text-white">
                      {new Date(request.scheduled_start_at).toLocaleString()}
                    </dd>
                  </div>
                )}
                {request.scheduled_end_at && (
                  <div>
                    <dt className="text-sm text-gray-400">Scheduled End</dt>
                    <dd className="mt-1 text-sm text-white">
                      {new Date(request.scheduled_end_at).toLocaleString()}
                    </dd>
                  </div>
                )}
              </dl>

              <div className="mt-4 p-3 bg-gray-800 rounded-lg">
                <dt className="text-sm text-gray-400">Reason</dt>
                <dd className="mt-1 text-sm text-white">{request.reason}</dd>
              </div>
            </div>
          </Card>

          {/* Comments */}
          {request.comments && request.comments.length > 0 && (
            <Card>
              <div className="card-body">
                <h3 className="text-lg font-semibold text-white mb-4">Comments</h3>
                <div className="space-y-4">
                  {request.comments.map((comment: RequestComment) => (
                    <div key={comment.id} className="flex gap-3">
                      <div className="flex-shrink-0 h-8 w-8 rounded-full bg-primary-600 flex items-center justify-center">
                        {comment.user?.first_name?.[0] || 'U'}
                      </div>
                      <div className="flex-1 bg-gray-800 rounded-lg p-3">
                        <div className="flex items-center justify-between">
                          <p className="text-sm font-medium text-white">
                            {comment.user?.first_name} {comment.user?.last_name}
                          </p>
                          <p className="text-xs text-gray-400">
                            {new Date(comment.created_at).toLocaleString()}
                          </p>
                        </div>
                        <p className="mt-1 text-sm text-gray-300">{comment.content}</p>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </Card>
          )}

          {/* Approval Workflow */}
          {request.approvers && request.approvers.length > 0 && (
            <Card>
              <div className="card-body">
                <h3 className="text-lg font-semibold text-white mb-4">Approval Workflow</h3>
                <div className="space-y-3">
                  {request.approvers.map((approver) => (
                    <div
                      key={approver.id}
                      className={clsx(
                        'flex items-center gap-3 p-3 rounded-lg border',
                        approver.status === 'approved' && 'border-success-500/50 bg-success-500/10',
                        approver.status === 'denied' && 'border-danger-500/50 bg-danger-500/10',
                        approver.status === 'pending' && 'border-gray-700 bg-gray-800'
                      )}
                    >
                      <div className="flex-shrink-0">
                        {approver.status === 'approved' && (
                          <Check className="h-5 w-5 text-success-400" />
                        )}
                        {approver.status === 'denied' && (
                          <X className="h-5 w-5 text-danger-400" />
                        )}
                        {approver.status === 'pending' && (
                          <Clock className="h-5 w-5 text-warning-400" />
                        )}
                      </div>
                      <div className="flex-1">
                        <p className="text-sm font-medium text-white">
                          {approver.user?.first_name} {approver.user?.last_name}
                        </p>
                        <p className="text-xs text-gray-400">Step {approver.step}</p>
                      </div>
                      <Badge
                        variant={
                          approver.status === 'approved'
                            ? 'success'
                            : approver.status === 'denied'
                              ? 'danger'
                              : 'warning'
                        }
                        size="sm"
                      >
                        {approver.status}
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            </Card>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Requester Info */}
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Requester</h3>
              <div className="flex items-center gap-3">
                <div className="h-10 w-10 rounded-full bg-primary-600 flex items-center justify-center">
                  {request.user?.first_name?.[0] || 'U'}
                </div>
                <div>
                  <p className="text-sm font-medium text-white">
                    {request.user?.first_name} {request.user?.last_name}
                  </p>
                  <p className="text-xs text-gray-400">{request.user?.email}</p>
                </div>
              </div>
            </div>
          </Card>

          {/* Status Timeline */}
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Timeline</h3>
              <div className="space-y-3 text-sm">
                <div className="flex items-start gap-2">
                  <Calendar className="h-4 w-4 text-gray-400 mt-0.5" />
                  <div>
                    <p className="text-white">Created</p>
                    <p className="text-xs text-gray-400">
                      {new Date(request.created_at).toLocaleString()}
                    </p>
                  </div>
                </div>
                {request.approved_at && (
                  <div className="flex items-start gap-2">
                    <Check className="h-4 w-4 text-success-400 mt-0.5" />
                    <div>
                      <p className="text-white">Approved</p>
                      <p className="text-xs text-gray-400">
                        {new Date(request.approved_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                )}
                {request.denied_at && (
                  <div className="flex items-start gap-2">
                    <X className="h-4 w-4 text-danger-400 mt-0.5" />
                    <div>
                      <p className="text-white">Denied</p>
                      <p className="text-xs text-gray-400">
                        {new Date(request.denied_at).toLocaleString()}
                      </p>
                      {request.denied_reason && (
                        <p className="text-xs text-danger-400 mt-1">
                          Reason: {request.denied_reason}
                        </p>
                      )}
                    </div>
                  </div>
                )}
                {request.expires_at && (
                  <div className="flex items-start gap-2">
                    <Clock className="h-4 w-4 text-warning-400 mt-0.5" />
                    <div>
                      <p className="text-white">Expires</p>
                      <p className="text-xs text-gray-400">
                        {new Date(request.expires_at).toLocaleString()}
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </Card>

          {/* Actions - Delegate */}
          {canApprove && users?.data && (
            <Card>
              <div className="card-body">
                <h3 className="text-lg font-semibold text-white mb-4">Delegate Approval</h3>
                <p className="text-sm text-gray-400 mb-3">
                  Delegate this approval to another user
                </p>
                <div className="space-y-2 max-h-48 overflow-y-auto">
                  {users.data
                    .filter((u) => u.id !== request.user_id)
                    .map((user) => (
                      <button
                        key={user.id}
                        onClick={() => {
                          const reason = prompt('Reason for delegation:');
                          if (reason) {
                            delegateMutation.mutate({ toUserId: user.id, reason });
                          }
                        }}
                        className="w-full flex items-center gap-3 p-2 rounded hover:bg-gray-800 text-left"
                        disabled={delegateMutation.isPending}
                      >
                        <div className="h-8 w-8 rounded-full bg-gray-700 flex items-center justify-center text-sm">
                          {user.first_name[0]}
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-medium text-white truncate">
                            {user.first_name} {user.last_name}
                          </p>
                          <p className="text-xs text-gray-400 truncate">{user.email}</p>
                        </div>
                      </button>
                    ))}
                </div>
              </div>
            </Card>
          )}
        </div>
      </div>

      {/* Deny Modal */}
      {showDenyModal && (
        <Modal
          isOpen={showDenyModal}
          onClose={() => {
            setShowDenyModal(false);
            setDenyReason('');
          }}
          title="Deny Request"
          size="sm"
        >
          <p className="mb-4 text-gray-300">
            Please provide a reason for denying this request.
          </p>
          <Textarea
            placeholder="Reason for denial..."
            value={denyReason}
            onChange={(e) => setDenyReason(e.target.value)}
            rows={3}
          />
          <div className="flex justify-end gap-3 mt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setShowDenyModal(false);
                setDenyReason('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="danger"
              onClick={handleDeny}
              isLoading={denyMutation.isPending}
            >
              Deny Request
            </Button>
          </div>
        </Modal>
      )}

      {/* Comment Modal */}
      {showCommentModal && (
        <Modal
          isOpen={showCommentModal}
          onClose={() => {
            setShowCommentModal(false);
            setCommentText('');
          }}
          title="Add Comment"
          size="sm"
        >
          <Textarea
            placeholder="Enter your comment..."
            value={commentText}
            onChange={(e) => setCommentText(e.target.value)}
            rows={4}
          />
          <div className="flex justify-end gap-3 mt-4">
            <Button
              variant="secondary"
              onClick={() => {
                setShowCommentModal(false);
                setCommentText('');
              }}
            >
              Cancel
            </Button>
            <Button
              onClick={handleAddComment}
              isLoading={addCommentMutation.isPending}
            >
              Add Comment
            </Button>
          </div>
        </Modal>
      )}
    </div>
  );
};

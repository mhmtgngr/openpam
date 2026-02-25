import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  AlertTriangle,
  Plus,
  Filter,
  Search,
  CheckCircle,
  XCircle,
  Clock,
  Ban,
  Calendar,
  User,
  FileText,
  RefreshCw,
} from 'lucide-react';
import { complianceExceptionsApi } from '@/api/reports';
import { ExceptionRequestDialog } from '@/components/analytics/ExceptionRequestDialog';
import { Card, Badge } from '@/components/common';
import { Button, Input, Select, Toggle } from '@/components/common';
import { LoadingState } from '@/components/common';
import type {
  ComplianceException,
  ExceptionListParams,
  ExceptionStatus,
  ComplianceFramework,
} from '@/types/reports';
import { format, isAfter, isBefore } from 'date-fns';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const frameworks: { value: ComplianceFramework; label: string }[] = [
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'pci_dss', label: 'PCI DSS' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'gdpr', label: 'GDPR' },
  { value: 'nerc_cip', label: 'NERC CIP' },
  { value: 'custom', label: 'Custom' },
];

const statusOptions: { value: ExceptionStatus; label: string }[] = [
  { value: 'pending', label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'denied', label: 'Denied' },
  { value: 'expired', label: 'Expired' },
  { value: 'revoked', label: 'Revoked' },
];

const statusIcons: Record<ExceptionStatus, React.ElementType> = {
  pending: Clock,
  approved: CheckCircle,
  denied: XCircle,
  expired: Ban,
  revoked: Ban,
};

const statusColors: Record<ExceptionStatus, string> = {
  pending: 'text-warning-400 bg-warning-400/10 border-warning-400/30',
  approved: 'text-success-400 bg-success-400/10 border-success-400/30',
  denied: 'text-danger-400 bg-danger-400/10 border-danger-400/30',
  expired: 'text-gray-400 bg-gray-400/10 border-gray-400/30',
  revoked: 'text-gray-400 bg-gray-400/10 border-gray-400/30',
};

const frameworkColors: Record<string, string> = {
  soc2: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
  iso27001: 'bg-green-500/20 text-green-400 border-green-500/30',
  pci_dss: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
  hipaa: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
  gdpr: 'bg-cyan-500/20 text-cyan-400 border-cyan-500/30',
  nerc_cip: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
  custom: 'bg-gray-500/20 text-gray-400 border-gray-500/30',
};

export const ExceptionManagementPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<ExceptionListParams>({
    limit: 20,
    offset: 0,
    status: undefined,
    framework: undefined,
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedException, setSelectedException] = useState<ComplianceException | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [dialogMode, setDialogMode] = useState<'create' | 'edit' | 'approve' | 'deny' | 'revoke'>('create');
  const [showExpiring, setShowExpiring] = useState(false);

  const { data: exceptionsData, isLoading } = useQuery({
    queryKey: ['complianceExceptions', filters],
    queryFn: () => complianceExceptionsApi.list(filters),
  });

  const { data: expiringData } = useQuery({
    queryKey: ['expiringExceptions'],
    queryFn: () => complianceExceptionsApi.getExpiring(30),
    refetchInterval: 5 * 60 * 1000, // Refresh every 5 minutes
  });

  const { data: pendingData } = useQuery({
    queryKey: ['pendingExceptions'],
    queryFn: () => complianceExceptionsApi.getPending(),
    refetchInterval: 2 * 60 * 1000, // Refresh every 2 minutes
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes?: string }) =>
      complianceExceptionsApi.approve(id, notes),
    onSuccess: () => {
      toast.success('Exception approved');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      queryClient.invalidateQueries({ queryKey: ['pendingExceptions'] });
      queryClient.invalidateQueries({ queryKey: ['expiringExceptions'] });
    },
  });

  const denyMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      complianceExceptionsApi.deny(id, reason),
    onSuccess: () => {
      toast.success('Exception denied');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      queryClient.invalidateQueries({ queryKey: ['pendingExceptions'] });
    },
  });

  const revokeMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      complianceExceptionsApi.revoke(id, reason),
    onSuccess: () => {
      toast.success('Exception revoked');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      queryClient.invalidateQueries({ queryKey: ['expiringExceptions'] });
    },
  });

  const handleSearch = (value: string) => {
    setSearchQuery(value);
    setFilters((prev) => ({ ...prev, search: value || undefined, offset: 0 }));
  };

  const handleFilterChange = (key: keyof ExceptionListParams, value: string | undefined) => {
    setFilters((prev) => ({ ...prev, [key]: value || undefined, offset: 0 }));
  };

  const handleQuickApprove = (exception: ComplianceException) => {
    if (confirm(`Approve exception for control "${exception.control_name}"?`)) {
      approveMutation.mutate({ id: exception.id });
    }
  };

  const handleQuickDeny = (exception: ComplianceException) => {
    const reason = prompt('Enter reason for denial:');
    if (reason) {
      denyMutation.mutate({ id: exception.id, reason });
    }
  };

  const handleQuickRevoke = (exception: ComplianceException) => {
    const reason = prompt('Enter reason for revocation:');
    if (reason) {
      revokeMutation.mutate({ id: exception.id, reason });
    }
  };

  const openDialog = (
    mode: 'create' | 'edit' | 'approve' | 'deny' | 'revoke',
    exception?: ComplianceException
  ) => {
    setDialogMode(mode);
    setSelectedException(exception || null);
    setDialogOpen(true);
  };

  const exceptions = exceptionsData?.data || [];
  const pagination = exceptionsData?.pagination;
  const pendingCount = pendingData?.pagination?.total || 0;
  const expiringExceptions = expiringData?.data || [];

  const handleNextPage = () => {
    if (pagination?.has_more) {
      setFilters((prev) => ({ ...prev, offset: (prev.offset || 0) + (prev.limit || 20) }));
    }
  };

  const handlePrevPage = () => {
    if ((filters.offset || 0) > 0) {
      setFilters((prev) => ({ ...prev, offset: Math.max(0, (prev.offset || 0) - (prev.limit || 20)) }));
    }
  };

  const isExpiringSoon = (exception: ComplianceException) => {
    if (!exception.expires_at || exception.status !== 'approved') return false;
    const expiryDate = new Date(exception.expires_at);
    const daysUntilExpiry = Math.ceil((expiryDate.getTime() - Date.now()) / (1000 * 60 * 60 * 24));
    return daysUntilExpiry <= 30 && daysUntilExpiry >= 0;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Compliance Exceptions</h1>
          <p className="mt-1 text-sm text-gray-400">
            Manage and review compliance exception requests
          </p>
        </div>
        <Button
          variant="primary"
          onClick={() => openDialog('create')}
          className="flex items-center gap-2"
        >
          <Plus className="h-4 w-4" />
          Request Exception
        </Button>
      </div>

      {/* Summary Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Pending Requests"
          value={pendingCount}
          icon={Clock}
          color="orange"
        />
        <StatCard
          title="Active Exceptions"
          value={exceptions.filter((e) => e.status === 'approved').length}
          icon={CheckCircle}
          color="green"
        />
        <StatCard
          title="Expiring Soon"
          value={expiringExceptions.length}
          icon={AlertTriangle}
          color="red"
        />
        <StatCard
          title="Total Exceptions"
          value={pagination?.total || 0}
          icon={FileText}
          color="blue"
        />
      </div>

      {/* Expiring Soon Alert */}
      {expiringExceptions.length > 0 && (
        <Card className="border-warning-500/30 bg-warning-500/10">
          <div className="card-body">
            <div className="flex items-start justify-between">
              <div className="flex items-start gap-3">
                <AlertTriangle className="h-5 w-5 text-warning-400 mt-0.5" />
                <div>
                  <h3 className="font-semibold text-white">Exceptions Expiring Soon</h3>
                  <p className="mt-1 text-sm text-gray-300">
                    {expiringExceptions.length} exception{expiringExceptions.length !== 1 ? 's' : ''} expiring within 30 days
                  </p>
                </div>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowExpiring(!showExpiring)}
              >
                {showExpiring ? 'Hide' : 'Show'}
              </Button>
            </div>
            {showExpiring && (
              <div className="mt-4 space-y-2">
                {expiringExceptions.map((exception) => (
                  <div
                    key={exception.id}
                    className="flex items-center justify-between rounded-lg bg-gray-800/50 p-3"
                  >
                    <div>
                      <p className="text-sm font-medium text-white">{exception.control_name}</p>
                      <p className="text-xs text-gray-400">
                        Expires: {exception.expires_at && format(new Date(exception.expires_at), 'MMM dd, yyyy')}
                      </p>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => openDialog('edit', exception)}
                    >
                      Review
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap items-center gap-4">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search exceptions..."
                value={searchQuery}
                onChange={(e) => handleSearch(e.target.value)}
                className="pl-10"
              />
            </div>

            <Select
              value={filters.framework || ''}
              onChange={(e) => handleFilterChange('framework', e.target.value || undefined)}
              className="w-40"
            >
              <option value="">All Frameworks</option>
              {frameworks.map((fw) => (
                <option key={fw.value} value={fw.value}>
                  {fw.label}
                </option>
              ))}
            </Select>

            <Select
              value={filters.status || ''}
              onChange={(e) => handleFilterChange('status', e.target.value || undefined)}
              className="w-40"
            >
              <option value="">All Status</option>
              {statusOptions.map((st) => (
                <option key={st.value} value={st.value}>
                  {st.label}
                </option>
              ))}
            </Select>

            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setFilters({ limit: 20, offset: 0 });
                setSearchQuery('');
              }}
            >
              Clear Filters
            </Button>
          </div>
        </div>
      </Card>

      {/* Exceptions List */}
      <Card>
        <div className="card-body p-0">
          {isLoading ? (
            <div className="py-12">
              <LoadingState message="Loading exceptions..." />
            </div>
          ) : exceptions.length === 0 ? (
            <div className="py-12 text-center">
              <AlertTriangle className="mx-auto h-12 w-12 text-gray-600" />
              <h3 className="mt-4 text-sm font-medium text-white">No exceptions found</h3>
              <p className="mt-2 text-sm text-gray-400">
                {searchQuery || filters.framework || filters.status
                  ? 'Try adjusting your filters'
                  : 'No compliance exceptions have been requested'}
              </p>
            </div>
          ) : (
            <div className="divide-y divide-gray-700">
              {exceptions.map((exception) => {
                const StatusIcon = statusIcons[exception.status];
                const isPending = exception.status === 'pending';
                const isApproved = exception.status === 'approved';
                const isExpiring = isExpiringSoon(exception);

                return (
                  <div
                    key={exception.id}
                    className={clsx(
                      'p-4 transition-colors',
                      isExpiring && 'bg-warning-500/5'
                    )}
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex items-start gap-4 flex-1">
                        <div className={clsx(
                          'mt-1 rounded-lg p-2',
                          statusColors[exception.status] || statusColors.pending
                        )}>
                          <StatusIcon className="h-5 w-5" />
                        </div>

                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 flex-wrap">
                            <h3 className="font-medium text-white">{exception.control_name}</h3>
                            <Badge
                              variant="outline"
                              className={clsx('border', frameworkColors[exception.framework] || frameworkColors.custom)}
                            >
                              {exception.framework.toUpperCase()}
                            </Badge>
                            <Badge
                              variant="outline"
                              className={clsx('border capitalize', statusColors[exception.status])}
                            >
                              {exception.status}
                            </Badge>
                            {isExpiring && (
                              <Badge variant="outline" className="border-warning-500/30 text-warning-400">
                                Expiring Soon
                              </Badge>
                            )}
                          </div>

                          <p className="mt-1 text-sm text-gray-300 line-clamp-2">
                            {exception.reason}
                          </p>

                          <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-2 text-xs text-gray-400">
                            <div className="flex items-center gap-1">
                              <User className="h-3 w-3" />
                              <span>Requested by {exception.requested_by_user?.display_name || exception.requested_by_user?.email || exception.requested_by}</span>
                            </div>
                            <div className="flex items-center gap-1">
                              <Calendar className="h-3 w-3" />
                              <span>
                                {format(new Date(exception.requested_at), 'MMM dd, yyyy')}
                              </span>
                            </div>
                            {exception.expires_at && (
                              <div className="flex items-center gap-1">
                                <Clock className="h-3 w-3" />
                                <span className={isExpiring ? 'text-warning-400' : ''}>
                                  Expires: {format(new Date(exception.expires_at), 'MMM dd, yyyy')}
                                </span>
                              </div>
                            )}
                          </div>

                          {exception.business_justification && (
                            <details className="mt-2 group">
                              <summary className="cursor-pointer text-xs text-primary-400 hover:text-primary-300">
                                View justification
                              </summary>
                              <p className="mt-2 text-sm text-gray-400 pl-4 border-l-2 border-gray-700">
                                {exception.business_justification}
                              </p>
                            </details>
                          )}
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        {isPending && (
                          <>
                            <Button
                              variant="success"
                              size="sm"
                              onClick={() => handleQuickApprove(exception)}
                              loading={approveMutation.isPending}
                            >
                              <CheckCircle className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="danger"
                              size="sm"
                              onClick={() => handleQuickDeny(exception)}
                              loading={denyMutation.isPending}
                            >
                              <XCircle className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="secondary"
                              size="sm"
                              onClick={() => openDialog('approve', exception)}
                            >
                              Review
                            </Button>
                          </>
                        )}

                        {isApproved && (
                          <>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openDialog('edit', exception)}
                            >
                              Edit
                            </Button>
                            <Button
                              variant="warning"
                              size="sm"
                              onClick={() => handleQuickRevoke(exception)}
                              loading={revokeMutation.isPending}
                            >
                              <Ban className="h-4 w-4" />
                            </Button>
                          </>
                        )}

                        {exception.status === 'denied' && (
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => openDialog('edit', exception)}
                          >
                            View Details
                          </Button>
                        )}
                      </div>
                    </div>

                    {exception.denial_reason && (
                      <div className="mt-3 ml-12 rounded-md bg-danger-500/10 p-2 text-sm text-danger-400">
                        <strong>Denial reason:</strong> {exception.denial_reason}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </div>

        {/* Pagination */}
        {pagination && pagination.total > (filters.limit || 20) && (
          <div className="card-body border-t border-gray-700">
            <div className="flex items-center justify-between">
              <p className="text-sm text-gray-400">
                Showing {Math.max(0, filters.offset || 0) + 1} to{' '}
                {Math.min(pagination.total, (filters.offset || 0) + (filters.limit || 20))} of{' '}
                {pagination.total} exceptions
              </p>
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handlePrevPage}
                  disabled={(filters.offset || 0) === 0}
                >
                  Previous
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleNextPage}
                  disabled={!pagination.has_more}
                >
                  Next
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>

      {/* Dialog */}
      <ExceptionRequestDialog
        isOpen={dialogOpen}
        onClose={() => {
          setDialogOpen(false);
          setSelectedException(null);
        }}
        exception={selectedException || undefined}
        mode={dialogMode}
        onSuccess={() => {
          queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
          queryClient.invalidateQueries({ queryKey: ['pendingExceptions'] });
        }}
      />
    </div>
  );
};

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ElementType;
  color: 'blue' | 'green' | 'red' | 'orange';
}

const StatCard: React.FC<StatCardProps> = ({ title, value, icon: Icon, color }) => {
  const colorClasses = {
    blue: 'text-blue-400 bg-blue-400/10',
    green: 'text-success-400 bg-success-400/10',
    red: 'text-danger-400 bg-danger-400/10',
    orange: 'text-warning-400 bg-warning-400/10',
  };

  return (
    <div className="card">
      <div className="card-body">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-sm text-gray-400">{title}</p>
            <p className="mt-2 text-2xl font-bold text-white">{value}</p>
          </div>
          <div className={clsx('rounded-lg p-3', colorClasses[color])}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
      </div>
    </div>
  );
};

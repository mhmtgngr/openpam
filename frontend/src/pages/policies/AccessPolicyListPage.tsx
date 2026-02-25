import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import {
  Plus,
  Search,
  Shield,
  CheckCircle,
  XCircle,
  Clock,
  Copy,
  Trash2,
  Play,
  Loader2,
  ChevronDown,
  ChevronUp,
  Tag,
} from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Card, Badge, Select } from '@/components/common';
import type { AccessPolicy, PolicyStatus } from '@/types/policy';
import toast from 'react-hot-toast';

type StatusFilter = 'all' | PolicyStatus;

export const AccessPolicyListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('all');
  const [expandedPolicies, setExpandedPolicies] = useState<Set<string>>(new Set());

  const { data, isLoading } = useQuery({
    queryKey: ['accessPolicies', { search, status: statusFilter === 'all' ? undefined : statusFilter }],
    queryFn: () =>
      policiesApi.access
        .list({ search, status: statusFilter === 'all' ? undefined : statusFilter })
        ,
  });

  const activateMutation = useMutation({
    mutationFn: (id: string) =>
      policiesApi.access.update(id, { status: 'active' }),
    onSuccess: () => {
      toast.success('Policy activated');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
    },
  });

  const deactivateMutation = useMutation({
    mutationFn: (id: string) =>
      policiesApi.access.update(id, { status: 'inactive' }),
    onSuccess: () => {
      toast.success('Policy deactivated');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => policiesApi.access.delete(id),
    onSuccess: () => {
      toast.success('Policy deleted');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
    },
  });

  const cloneMutation = useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      policiesApi.access.clone(id, name),
    onSuccess: () => {
      toast.success('Policy cloned');
      queryClient.invalidateQueries({ queryKey: ['accessPolicies'] });
    },
  });

  const handleDelete = (id: string, name: string) => {
    if (confirm(`Are you sure you want to delete "${name}"? This action cannot be undone.`)) {
      deleteMutation.mutate(id);
    }
  };

  const handleClone = (id: string, name: string) => {
    const newName = prompt(`Enter name for the cloned policy:`, `${name} (Copy)`);
    if (newName) {
      cloneMutation.mutate({ id, name: newName });
    }
  };

  const toggleExpand = (id: string) => {
    setExpandedPolicies((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const getStatusVariant = (status: PolicyStatus) => {
    switch (status) {
      case 'active':
        return 'success';
      case 'inactive':
        return 'neutral';
      case 'draft':
        return 'warning';
      default:
        return 'neutral';
    }
  };

  const getStatusIcon = (status: PolicyStatus) => {
    switch (status) {
      case 'active':
        return <CheckCircle className="h-4 w-4" />;
      case 'inactive':
        return <XCircle className="h-4 w-4" />;
      case 'draft':
        return <Clock className="h-4 w-4" />;
      default:
        return null;
    }
  };

  const getEffectColor = (effect: 'allow' | 'deny') => {
    return effect === 'allow' ? 'text-success-400' : 'text-danger-400';
  };

  const isExpanded = (id: string) => expandedPolicies.has(id);

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Access Policies</h1>
          <p className="mt-1 text-sm text-gray-400">
            Control who can access what resources and when
          </p>
        </div>
        <div className="flex gap-3">
          <Link to="/policies/access/test">
            <Button variant="secondary" leftIcon={<Play className="h-4 w-4" />}>
              Test Policies
            </Button>
          </Link>
          <Link to="/policies/access/new">
            <Button leftIcon={<Plus className="h-4 w-4" />}>
              New Policy
            </Button>
          </Link>
        </div>
      </div>

      {/* Filters */}
      <div className="card">
        <div className="card-body">
          <div className="flex flex-col md:flex-row gap-4">
            <div className="flex-1">
              <Input
                placeholder="Search policies by name or tag..."
                leftIcon={<Search className="h-4 w-4 text-gray-400" />}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <div className="w-full md:w-48">
              <Select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                options={[
                  { value: 'all', label: 'All Statuses' },
                  { value: 'active', label: 'Active' },
                  { value: 'inactive', label: 'Inactive' },
                  { value: 'draft', label: 'Draft' },
                ]}
              />
            </div>
          </div>
        </div>
      </div>

      {/* Loading State */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      ) : (
        <div className="space-y-4">
          {data?.data.map((policy: AccessPolicy) => (
            <Card key={policy.id} className="overflow-hidden">
              <div className="card-body">
                {/* Policy Header */}
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-4 flex-1">
                    <div
                      className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                        policy.status === 'active'
                          ? 'bg-success-600/20 text-success-400'
                          : policy.status === 'draft'
                          ? 'bg-warning-600/20 text-warning-400'
                          : 'bg-gray-600/20 text-gray-400'
                      }`}
                    >
                      <Shield className="h-5 w-5" />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-3">
                        <h3 className="font-semibold text-white">{policy.name}</h3>
                        <div className="flex items-center gap-1">
                          {getStatusIcon(policy.status)}
                          <Badge variant={getStatusVariant(policy.status)} size="sm">
                            {policy.status}
                          </Badge>
                        </div>
                        {policy.is_default && (
                          <Badge variant="info" size="sm">Default</Badge>
                        )}
                        {policy.is_system && (
                          <Badge variant="neutral" size="sm">System</Badge>
                        )}
                      </div>
                      {policy.description && (
                        <p className="mt-1 text-sm text-gray-400">{policy.description}</p>
                      )}
                      <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                        <span>Priority: {policy.priority}</span>
                        <span>Rules: {policy.rules.length}</span>
                        {policy.tags.length > 0 && (
                          <div className="flex items-center gap-1">
                            <Tag className="h-3 w-3" />
                            {policy.tags.slice(0, 2).map((tag) => (
                              <span key={tag} className="text-gray-400">
                                #{tag}
                              </span>
                            ))}
                            {policy.tags.length > 2 && (
                              <span className="text-gray-400">+{policy.tags.length - 2}</span>
                            )}
                          </div>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => toggleExpand(policy.id)}
                    >
                      {isExpanded(policy.id) ? (
                        <ChevronUp className="h-4 w-4" />
                      ) : (
                        <ChevronDown className="h-4 w-4" />
                      )}
                    </Button>
                    {policy.status !== 'active' && !policy.is_system && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => activateMutation.mutate(policy.id)}
                        isLoading={activateMutation.isPending}
                      >
                        Activate
                      </Button>
                    )}
                    {policy.status === 'active' && !policy.is_system && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => deactivateMutation.mutate(policy.id)}
                        isLoading={deactivateMutation.isPending}
                      >
                        Deactivate
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleClone(policy.id, policy.name)}
                      isLoading={cloneMutation.isPending}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                    <Link to={`/policies/access/${policy.id}`}>
                      <Button variant="ghost" size="sm">Edit</Button>
                    </Link>
                    {!policy.is_system && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleDelete(policy.id, policy.name)}
                        isLoading={deleteMutation.isPending}
                        className="text-danger-400 hover:text-danger-300"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                </div>

                {/* Expanded Rules View */}
                {isExpanded(policy.id) && policy.rules.length > 0 && (
                  <div className="mt-4 pt-4 border-t border-gray-700">
                    <div className="space-y-3">
                      <h4 className="text-sm font-medium text-gray-300">Rules</h4>
                      {policy.rules.map((rule, index) => (
                        <div
                          key={rule.id || index}
                          className="bg-gray-800/50 rounded-lg p-3 border border-gray-700"
                        >
                          <div className="flex items-center justify-between mb-2">
                            <div className="flex items-center gap-2">
                              <span className={`font-medium ${getEffectColor(rule.effect)}`}>
                                {rule.effect.toUpperCase()}
                              </span>
                              <span className="text-white font-medium">{rule.name}</span>
                              {rule.priority !== undefined && (
                                <span className="text-xs text-gray-500">
                                  Priority: {rule.priority}
                                </span>
                              )}
                            </div>
                            <span className="text-xs text-gray-500">
                              {rule.logical_operator} {rule.conditions.length} condition(s)
                            </span>
                          </div>
                          {rule.description && (
                            <p className="text-sm text-gray-400 mb-2">{rule.description}</p>
                          )}
                          <div className="flex flex-wrap gap-2 text-xs">
                            {rule.actions.map((action) => (
                              <Badge key={action} variant="neutral" size="sm">
                                {action}
                              </Badge>
                            ))}
                            {rule.roles.map((role) => (
                              <Badge key={role} variant="info" size="sm">
                                {role}
                              </Badge>
                            ))}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Conflict Resolution */}
                {policy.conflict_resolution && (
                  <div className="mt-3 text-xs text-gray-500">
                    Conflict Resolution: {policy.conflict_resolution.replace(/_/g, ' ')}
                  </div>
                )}
              </div>
            </Card>
          ))}

          {/* Empty State */}
          {data?.data.length === 0 && (
            <div className="text-center py-12">
              <Shield className="h-16 w-16 text-gray-600 mx-auto mb-4" />
              <h3 className="text-lg font-medium text-white mb-2">No access policies found</h3>
              <p className="text-gray-400 mb-4">
                {search || statusFilter !== 'all'
                  ? 'Try adjusting your search or filters'
                  : 'Create your first access policy to control resource access'}
              </p>
              {!search && statusFilter === 'all' && (
                <Link to="/policies/access/new">
                  <Button leftIcon={<Plus className="h-4 w-4" />}>
                    Create Policy
                  </Button>
                </Link>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
};

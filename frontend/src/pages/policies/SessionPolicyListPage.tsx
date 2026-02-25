import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Clock, Shield, Loader2 } from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Card, Badge } from '@/components/common';
import type { SessionPolicy } from '@/types';
import toast from 'react-hot-toast';

export const SessionPolicyListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['sessionPolicies', { search }],
    queryFn: () =>
      policiesApi.session.list({ search }),
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => policiesApi.session.setDefault(id),
    onSuccess: () => {
      toast.success('Default policy updated');
      queryClient.invalidateQueries({ queryKey: ['sessionPolicies'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => policiesApi.session.delete(id),
    onSuccess: () => {
      toast.success('Policy deleted');
      queryClient.invalidateQueries({ queryKey: ['sessionPolicies'] });
    },
  });

  const handleDelete = (id: string, name: string) => {
    if (confirm(`Are you sure you want to delete "${name}"?`)) {
      deleteMutation.mutate(id);
    }
  };

  return (
    <div className="max-w-5xl mx-auto space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Session Policies</h1>
          <p className="mt-1 text-sm text-gray-400">
            Configure session security and monitoring policies
          </p>
        </div>
        <Link to="/policies/session/new">
          <Button leftIcon={<Plus className="h-4 w-4" />}>
            Add Policy
          </Button>
        </Link>
      </div>

      <div className="card">
        <div className="card-body">
          <Input
            placeholder="Search policies..."
            leftIcon={<Search className="h-4 w-4 text-gray-400" />}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {data?.data.map((policy: SessionPolicy) => (
            <Card key={policy.id}>
              <div className="card-body">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-info-600/20 text-info-400">
                      <Clock className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-white">{policy.name}</h3>
                      {(policy as any).is_default && (
                        <Badge variant="success" size="sm">Default</Badge>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    {!(policy as any).is_default && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDefaultMutation.mutate(policy.id)}
                        isLoading={setDefaultMutation.isPending}
                      >
                        Set Default
                      </Button>
                    )}
                    <Link to={`/policies/session/${policy.id}`}>
                      <Button variant="ghost" size="sm">Edit</Button>
                    </Link>
                  </div>
                </div>

                <div className="space-y-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Max Duration</span>
                    <span className="text-white">{policy.max_duration_minutes} minutes</span>
                  </div>

                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Idle Timeout</span>
                    <span className="text-white">{policy.idle_timeout_minutes} minutes</span>
                  </div>

                  <div className="flex items-center gap-4">
                    <span className="text-gray-400">Requirements:</span>
                    <div className="flex flex-wrap gap-2">
                      {policy.require_approval && (
                        <Badge variant="warning" size="sm">Approval</Badge>
                      )}
                      {policy.require_reason && (
                        <Badge variant="neutral" size="sm">Reason</Badge>
                      )}
                      {policy.require_mfa && (
                        <Badge variant="success" size="sm">MFA</Badge>
                      )}
                      {policy.allow_recording && (
                        <Badge variant="info" size="sm">Recording</Badge>
                      )}
                    </div>
                  </div>

                  {policy.monitor_keywords.length > 0 && (
                    <div>
                      <span className="text-gray-400">Monitor Keywords: </span>
                      <span className="text-white">{policy.monitor_keywords.length} defined</span>
                    </div>
                  )}

                  {policy.blocked_commands.length > 0 && (
                    <div>
                      <span className="text-gray-400">Blocked Commands: </span>
                      <span className="text-white">{policy.blocked_commands.length} defined</span>
                    </div>
                  )}
                </div>
              </div>
            </Card>
          ))}

          {data?.data.length === 0 && (
            <div className="col-span-2 text-center py-12">
              <Shield className="h-12 w-12 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400">No session policies found</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

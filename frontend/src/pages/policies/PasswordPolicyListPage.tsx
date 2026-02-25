import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { Plus, Search, Key, Shield, Check, Loader2 } from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Card, Badge, Toggle } from '@/components/common';
import type { PasswordPolicy } from '@/types';
import toast from 'react-hot-toast';

export const PasswordPolicyListPage: React.FC = () => {
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');

  const { data, isLoading } = useQuery({
    queryKey: ['passwordPolicies', { search }],
    queryFn: () =>
      policiesApi.password.list({ search }),
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => policiesApi.password.setDefault(id),
    onSuccess: () => {
      toast.success('Default policy updated');
      queryClient.invalidateQueries({ queryKey: ['passwordPolicies'] });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => policiesApi.password.delete(id),
    onSuccess: () => {
      toast.success('Policy deleted');
      queryClient.invalidateQueries({ queryKey: ['passwordPolicies'] });
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
          <h1 className="text-2xl font-bold text-white">Password Policies</h1>
          <p className="mt-1 text-sm text-gray-400">
            Configure password requirements and rotation policies
          </p>
        </div>
        <Link to="/policies/password/new">
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
          {data?.data.map((policy: PasswordPolicy) => (
            <Card key={policy.id}>
              <div className="card-body">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-600/20 text-primary-400">
                      <Key className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="font-semibold text-white">{policy.name}</h3>
                      {policy.id === 'default' && (
                        <Badge variant="success" size="sm">Default</Badge>
                      )}
                    </div>
                  </div>
                  <div className="flex gap-2">
                    {!(policy as any).is_system && policy.id !== 'default' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setDefaultMutation.mutate(policy.id)}
                        isLoading={setDefaultMutation.isPending}
                      >
                        Set Default
                      </Button>
                    )}
                    <Link to={`/policies/password/${policy.id}`}>
                      <Button variant="ghost" size="sm">Edit</Button>
                    </Link>
                  </div>
                </div>

                <div className="space-y-3 text-sm">
                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">Min Length</span>
                    <span className="text-white">{policy.min_length} characters</span>
                  </div>

                  <div className="flex items-center gap-4">
                    <span className="text-gray-400">Requirements:</span>
                    <div className="flex flex-wrap gap-2">
                      {policy.require_uppercase && (
                        <Badge variant="neutral" size="sm">Uppercase</Badge>
                      )}
                      {policy.require_lowercase && (
                        <Badge variant="neutral" size="sm">Lowercase</Badge>
                      )}
                      {policy.require_numbers && (
                        <Badge variant="neutral" size="sm">Numbers</Badge>
                      )}
                      {policy.require_special && (
                        <Badge variant="neutral" size="sm">Special</Badge>
                      )}
                    </div>
                  </div>

                  {policy.expiration_days && (
                    <div className="flex items-center justify-between">
                      <span className="text-gray-400">Expiration</span>
                      <span className="text-white">{policy.expiration_days} days</span>
                    </div>
                  )}

                  <div className="flex items-center justify-between">
                    <span className="text-gray-400">History</span>
                    <span className="text-white">{policy.history_count} passwords</span>
                  </div>
                </div>
              </div>
            </Card>
          ))}

          {data?.data.length === 0 && (
            <div className="col-span-2 text-center py-12">
              <Shield className="h-12 w-12 text-gray-600 mx-auto mb-4" />
              <p className="text-gray-400">No password policies found</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

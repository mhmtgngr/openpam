import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import {
  ArrowLeft,
  Play,
  Plus,
  Trash2,
  CheckCircle,
  XCircle,
  Loader2,
  User,
  Shield,
  Target,
  Zap,
  Copy,
} from 'lucide-react';
import { policiesApi } from '@/api/policies';
import { Button, Input, Card, Badge, Select, Textarea } from '@/components/common';
import type {
  AccessPolicy,
  TestScenario,
  TestPolicyResult,
  PolicyEvaluationRequest,
  PolicyEvaluationResult,
} from '@/types/policy';
import toast from 'react-hot-toast';

// Common test data
const COMMON_USERS = [
  { id: 'user-1', name: 'Admin User', roles: ['admin', 'operator'] },
  { id: 'user-2', name: 'DevOps Engineer', roles: ['operator'] },
  { id: 'user-3', name: 'Auditor', roles: ['auditor'] },
  { id: 'user-4', name: 'Regular User', roles: ['user'] },
];

const COMMON_RESOURCES = [
  'credential:prod-db-ssh',
  'target:production-server',
  'session:ssh-*',
  'request:access-request',
];

const COMMON_ACTIONS = ['checkout', 'connect', 'approve', 'view', 'delete', 'terminate'];

export const PolicyTestPage: React.FC = () => {
  const navigate = useNavigate();

  const [policyId, setPolicyId] = useState('');
  const [testScenarios, setTestScenarios] = useState<TestScenario[]>([
    {
      name: 'Scenario 1',
      user_id: 'user-1',
      user_roles: ['admin'],
      resource: 'credential:prod-db-ssh',
      action: 'checkout',
      context: { ip_address: '192.168.1.100' },
    },
  ]);
  const [testResults, setTestResults] = useState<TestPolicyResult[]>([]);
  const [selectedScenarioIndex, setSelectedScenarioIndex] = useState<number | null>(null);

  // Fetch policy for testing
  const { data: policy, isLoading: isLoadingPolicy } = useQuery({
    queryKey: ['accessPolicy', policyId],
    queryFn: () => policiesApi.access.get(policyId).then((res) => res.data),
    enabled: !!policyId,
  });

  // Test evaluation mutation (direct evaluation)
  const evaluateMutation = useMutation({
    mutationFn: (request: PolicyEvaluationRequest) =>
      policiesApi.access.evaluate(request),
  });

  // Test policy mutation
  const testMutation = useMutation({
    mutationFn: (data: { policy: AccessPolicy; test_scenarios: TestScenario[] }) =>
      policiesApi.access.test(data),
    onSuccess: (response) => {
      const results = response.data;
      setTestResults(
        results.map((result: any, index: number) => ({
          scenario_name: testScenarios[index]?.name || `Scenario ${index + 1}`,
          result: result.result,
          passed: result.passed,
          expected_allowed: testScenarios[index]?.expected_allowed,
        }))
      );
      toast.success('Policy test completed');
    },
  });

  // Direct evaluate without saving
  const handleEvaluate = async (scenario: TestScenario, index: number) => {
    const request: PolicyEvaluationRequest = {
      user_id: scenario.user_id,
      user_roles: scenario.user_roles,
      user_attributes: {},
      resource: scenario.resource,
      action: scenario.action,
      context: scenario.context,
    };

    try {
      const response = await policiesApi.access.evaluate(request);
      const result = response.data;

      setTestResults((prev) => {
        const newResults = [...prev];
        newResults[index] = {
          scenario_name: scenario.name,
          result,
          passed: scenario.expected_allowed === undefined || result.allowed === scenario.expected_allowed,
          expected_allowed: scenario.expected_allowed,
        };
        return newResults;
      });
    } catch (error) {
      toast.error('Evaluation failed');
    }
  };

  // Run all tests
  const runAllTests = () => {
    if (!policy) {
      toast.error('Please select a policy to test');
      return;
    }

    testMutation.mutate({
      policy,
      test_scenarios: testScenarios,
    });
  };

  const addScenario = () => {
    setTestScenarios([
      ...testScenarios,
      {
        name: `Scenario ${testScenarios.length + 1}`,
        user_id: 'user-1',
        user_roles: ['user'],
        resource: '',
        action: 'view',
        context: {},
      },
    ]);
  };

  const removeScenario = (index: number) => {
    setTestScenarios(testScenarios.filter((_, i) => i !== index));
    setTestResults(testResults.filter((_, i) => i !== index));
  };

  const updateScenario = (index: number, updates: Partial<TestScenario>) => {
    const newScenarios = [...testScenarios];
    newScenarios[index] = { ...newScenarios[index], ...updates };
    setTestScenarios(newScenarios);
  };

  const loadPolicy = () => {
    const id = prompt('Enter Policy ID:');
    if (id) {
      setPolicyId(id);
    }
  };

  const duplicateScenario = (index: number) => {
    const scenario = testScenarios[index];
    setTestScenarios([
      ...testScenarios,
      { ...scenario, name: `${scenario.name} (Copy)` },
    ]);
  };

  return (
    <div className="max-w-6xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link to="/policies/access">
            <Button variant="ghost" size="sm" leftIcon={<ArrowLeft className="h-4 w-4" />}>
              Back
            </Button>
          </Link>
          <div>
            <h1 className="text-2xl font-bold text-white">Policy Testing</h1>
            <p className="mt-1 text-sm text-gray-400">
              Test policy evaluation with different scenarios
            </p>
          </div>
        </div>
        <div className="flex gap-3">
          <Button variant="secondary" onClick={loadPolicy}>
            Load Policy
          </Button>
          {policy && (
            <Button
              onClick={runAllTests}
              isLoading={testMutation.isPending}
              leftIcon={<Play className="h-4 w-4" />}
            >
              Run All Tests
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Test Scenarios */}
        <div className="lg:col-span-2 space-y-4">
          <Card>
            <div className="card-body">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-white">Test Scenarios</h3>
                <Button variant="secondary" size="sm" onClick={addScenario} leftIcon={<Plus className="h-4 w-4" />}>
                  Add Scenario
                </Button>
              </div>

              {/* Policy Info */}
              {policy && (
                <div className="mb-4 p-3 bg-primary-900/20 border border-primary-600/30 rounded-lg">
                  <div className="flex items-center gap-2 text-sm">
                    <Shield className="h-4 w-4 text-primary-400" />
                    <span className="text-white font-medium">{policy.name}</span>
                    <Badge variant={policy.status === 'active' ? 'success' : 'neutral'} size="sm">
                      {policy.status}
                    </Badge>
                  </div>
                  <p className="text-xs text-gray-400 mt-1">{policy.description}</p>
                </div>
              )}

              {/* Scenarios List */}
              <div className="space-y-4">
                {testScenarios.map((scenario, index) => {
                  const result = testResults[index];
                  const isSelected = selectedScenarioIndex === index;

                  return (
                    <div
                      key={index}
                      className={`border rounded-lg overflow-hidden ${
                        result
                          ? result.passed
                            ? 'border-success-600/50 bg-success-900/10'
                            : 'border-danger-600/50 bg-danger-900/10'
                          : 'border-gray-700 bg-gray-800/30'
                      }`}
                    >
                      {/* Scenario Header */}
                      <div className="flex items-center justify-between p-3 bg-gray-800/50">
                        <div className="flex items-center gap-3">
                          {result ? (
                            result.result.allowed ? (
                              <CheckCircle className="h-5 w-5 text-success-400" />
                            ) : (
                              <XCircle className="h-5 w-5 text-danger-400" />
                            )
                          ) : (
                            <div className="h-5 w-5 rounded-full border-2 border-gray-600" />
                          )}
                          <Input
                            value={scenario.name}
                            onChange={(e) => updateScenario(index, { name: e.target.value })}
                            className="font-medium text-white"
                          />
                        </div>
                        <div className="flex items-center gap-2">
                          {result && (
                            <Badge
                              variant={result.passed ? 'success' : 'danger'}
                              size="sm"
                            >
                              {result.result.allowed ? 'ALLOW' : 'DENY'}
                            </Badge>
                          )}
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setSelectedScenarioIndex(isSelected ? null : index)}
                          >
                            {isSelected ? <span>−</span> : <span>+</span>}
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => duplicateScenario(index)}
                          >
                            <Copy className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleEvaluate(scenario, index)}
                            isLoading={evaluateMutation.isPending}
                            disabled={!policy}
                          >
                            <Play className="h-4 w-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => removeScenario(index)}
                            className="text-danger-400"
                          >
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>

                      {/* Expanded Scenario Details */}
                      {isSelected && (
                        <div className="p-4 space-y-4">
                          {/* User & Roles */}
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <Input
                                label="User ID"
                                value={scenario.user_id}
                                onChange={(e) => updateScenario(index, { user_id: e.target.value })}
                                placeholder="user-123"
                                leftIcon={<User className="h-4 w-4 text-gray-400" />}
                              />
                            </div>
                            <div>
                              <Input
                                label="Roles (comma-separated)"
                                value={scenario.user_roles.join(',')}
                                onChange={(e) =>
                                  updateScenario(index, {
                                    user_roles: e.target.value.split(',').map((s) => s.trim()),
                                  })
                                }
                                placeholder="admin, operator"
                              />
                            </div>
                          </div>

                          {/* Resource & Action */}
                          <div className="grid grid-cols-2 gap-4">
                            <div>
                              <Select
                                label="Resource"
                                value={scenario.resource}
                                onChange={(e) => updateScenario(index, { resource: e.target.value })}
                                options={COMMON_RESOURCES.map((r) => ({ value: r, label: r }))}
                                placeholder="Select or enter resource..."
                              />
                            </div>
                            <div>
                              <Select
                                label="Action"
                                value={scenario.action}
                                onChange={(e) => updateScenario(index, { action: e.target.value })}
                                options={COMMON_ACTIONS.map((a) => ({ value: a, label: a }))}
                              />
                            </div>
                          </div>

                          {/* Context */}
                          <Textarea
                            label="Context (JSON)"
                            value={JSON.stringify(scenario.context, null, 2)}
                            onChange={(e) => {
                              try {
                                const context = JSON.parse(e.target.value);
                                updateScenario(index, { context });
                              } catch {
                                // Invalid JSON, ignore
                              }
                            }}
                            rows={3}
                            placeholder='{"ip_address": "192.168.1.100"}'
                          />

                          {/* Expected Result */}
                          <Select
                            label="Expected Result (Optional)"
                            value={scenario.expected_allowed === undefined ? '' : String(scenario.expected_allowed)}
                            onChange={(e) =>
                              updateScenario(index, {
                                expected_allowed: e.target.value === '' ? undefined : e.target.value === 'true',
                              })
                            }
                            options={[
                              { value: '', label: 'Any' },
                              { value: 'true', label: 'Allow' },
                              { value: 'false', label: 'Deny' },
                            ]}
                          />

                          {/* Result Details */}
                          {result && (
                            <div className="mt-4 p-3 bg-gray-900 rounded-lg">
                              <h4 className="text-sm font-medium text-gray-300 mb-2">Evaluation Result</h4>
                              <div className="space-y-2 text-sm">
                                <div className="flex justify-between">
                                  <span className="text-gray-400">Decision:</span>
                                  <span className={result.result.allowed ? 'text-success-400' : 'text-danger-400'}>
                                    {result.result.allowed ? 'ALLOW' : 'DENY'}
                                  </span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-400">Matched Policy:</span>
                                  <span className="text-white">{result.result.matched_policy_id || 'N/A'}</span>
                                </div>
                                <div className="flex justify-between">
                                  <span className="text-gray-400">Reason:</span>
                                  <span className="text-white">{result.result.reason}</span>
                                </div>
                                {scenario.expected_allowed !== undefined && (
                                  <div className="flex justify-between">
                                    <span className="text-gray-400">Test Result:</span>
                                    <span className={result.passed ? 'text-success-400' : 'text-danger-400'}>
                                      {result.passed ? 'PASSED' : 'FAILED'}
                                    </span>
                                  </div>
                                )}
                              </div>

                              {/* Evaluation Details */}
                              {result.result.details && result.result.details.length > 0 && (
                                <div className="mt-3 pt-3 border-t border-gray-700">
                                  <h5 className="text-xs font-medium text-gray-400 mb-2">
                                    Evaluated Policies
                                  </h5>
                                  <div className="space-y-2">
                                    {result.result.details.map((detail, idx) => (
                                      <div
                                        key={idx}
                                        className={`p-2 rounded ${
                                          detail.matched ? 'bg-primary-900/20' : 'bg-gray-800'
                                        }`}
                                      >
                                        <div className="flex items-center justify-between">
                                          <span className="text-white text-xs">{detail.policy_name}</span>
                                          {detail.matched && (
                                            <Badge
                                              variant={detail.effect === 'allow' ? 'success' : 'danger'}
                                              size="sm"
                                            >
                                              {detail.effect.toUpperCase()}
                                            </Badge>
                                          )}
                                        </div>
                                        {detail.rule_name && (
                                          <span className="text-gray-500 text-xs">
                                            Rule: {detail.rule_name}
                                          </span>
                                        )}
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            </div>
          </Card>
        </div>

        {/* Summary Sidebar */}
        <div className="space-y-4">
          {/* Test Summary */}
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Test Summary</h3>
              <div className="space-y-3">
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Total Scenarios</span>
                  <span className="text-white font-medium">{testScenarios.length}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Executed</span>
                  <span className="text-white font-medium">{testResults.length}</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Passed</span>
                  <span className="text-success-400 font-medium">
                    {testResults.filter((r) => r?.passed).length}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Failed</span>
                  <span className="text-danger-400 font-medium">
                    {testResults.filter((r) => r && !r.passed).length}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Allowed</span>
                  <span className="text-success-400 font-medium">
                    {testResults.filter((r) => r?.result.allowed).length}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-gray-400">Denied</span>
                  <span className="text-danger-400 font-medium">
                    {testResults.filter((r) => r && !r.result.allowed).length}
                  </span>
                </div>
              </div>
            </div>
          </Card>

          {/* Quick Actions */}
          <Card>
            <div className="card-body">
              <h3 className="text-lg font-semibold text-white mb-4">Quick Actions</h3>
              <div className="space-y-2">
                <Button
                  variant="secondary"
                  size="sm"
                  fullWidth
                  onClick={() =>
                    setTestScenarios([
                      {
                        name: 'Admin Access',
                        user_id: 'user-1',
                        user_roles: ['admin'],
                        resource: 'credential:prod-db',
                        action: 'checkout',
                        context: { ip_address: '10.0.0.1' },
                        expected_allowed: true,
                      },
                      {
                        name: 'User Denied',
                        user_id: 'user-2',
                        user_roles: ['user'],
                        resource: 'credential:prod-db',
                        action: 'checkout',
                        context: { ip_address: '10.0.0.2' },
                        expected_allowed: false,
                      },
                    ])
                  }
                >
                  Load Sample Scenarios
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  fullWidth
                  onClick={() => {
                    setTestResults([]);
                    toast.success('Results cleared');
                  }}
                >
                  Clear Results
                </Button>
              </div>
            </div>
          </Card>

          {/* Policy Info */}
          {policy && (
            <Card>
              <div className="card-body">
                <h3 className="text-lg font-semibold text-white mb-4">Policy Details</h3>
                <div className="space-y-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-gray-400">Name</span>
                    <span className="text-white">{policy.name}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Status</span>
                    <Badge variant={policy.status === 'active' ? 'success' : 'neutral'} size="sm">
                      {policy.status}
                    </Badge>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Rules</span>
                    <span className="text-white">{policy.rules.length}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-gray-400">Priority</span>
                    <span className="text-white">{policy.priority}</span>
                  </div>
                </div>
              </div>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
};

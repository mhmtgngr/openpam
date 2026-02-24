import React, { useState } from 'react';
import {
  GripVertical,
  Plus,
  Trash2,
  CheckCircle,
  XCircle,
  ChevronDown,
  ChevronUp,
} from 'lucide-react';
import { Button, Input, Card, Select, Badge, Toggle, Textarea } from '@/components/common';
import { ConditionBuilder } from './ConditionBuilder';
import type { PolicyRule, PolicyEffect, LogicalOperator } from '@/types/policy';

interface PolicyRuleBuilderProps {
  rules: PolicyRule[];
  onChange: (rules: PolicyRule[]) => void;
  readOnly?: boolean;
  className?: string;
}

const EFFECT_OPTIONS: { value: PolicyEffect; label: string; color: string }[] = [
  { value: 'allow', label: 'Allow', color: 'text-success-400' },
  { value: 'deny', label: 'Deny', color: 'text-danger-400' },
];

const LOGICAL_OPERATORS: { value: LogicalOperator; label: string; description: string }[] = [
  { value: 'AND', label: 'AND', description: 'All conditions must be true' },
  { value: 'OR', label: 'OR', description: 'At least one condition must be true' },
];

const COMMON_ACTIONS = [
  'checkout',
  'connect',
  'approve',
  'deny',
  'view',
  'edit',
  'delete',
  'export',
  'rotate',
  'terminate',
];

const COMMON_RESOURCES = [
  'credential:*',
  'target:*',
  'session:*',
  'request:*',
  'policy:*',
  'user:*',
];

export const PolicyRuleBuilder: React.FC<PolicyRuleBuilderProps> = ({
  rules,
  onChange,
  readOnly = false,
  className = '',
}) => {
  const [expandedRules, setExpandedRules] = useState<Set<number>>(new Set([0]));

  const addRule = () => {
    const newRule: PolicyRule = {
      name: `Rule ${rules.length + 1}`,
      effect: 'allow',
      conditions: [],
      logical_operator: 'AND',
      priority: rules.length + 1,
      resources: [],
      actions: [],
      roles: [],
      users: [],
    };
    onChange([...rules, newRule]);
    setExpandedRules(new Set([...expandedRules, rules.length]));
  };

  const updateRule = (index: number, updates: Partial<PolicyRule>) => {
    const newRules = [...rules];
    newRules[index] = { ...newRules[index], ...updates };
    onChange(newRules);
  };

  const removeRule = (index: number) => {
    onChange(rules.filter((_, i) => i !== index));
    setExpandedRules(
      new Set(
        [...expandedRules].filter((i) => i !== index).map((i) => (i > index ? i - 1 : i))
      )
    );
  };

  const duplicateRule = (index: number) => {
    const rule = rules[index];
    const newRule = {
      ...rule,
      name: `${rule.name} (Copy)`,
      priority: rules.length + 1,
    };
    onChange([...rules, newRule]);
    setExpandedRules(new Set([...expandedRules, rules.length]));
  };

  const toggleExpand = (index: number) => {
    setExpandedRules((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const addTag = (index: number, field: 'resources' | 'actions' | 'roles' | 'users', value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return;

    const rule = rules[index];
    const currentValues = rule[field] || [];
    if (!currentValues.includes(trimmed)) {
      updateRule(index, { [field]: [...currentValues, trimmed] });
    }
  };

  const removeTag = (index: number, field: 'resources' | 'actions' | 'roles' | 'users', value: string) => {
    const rule = rules[index];
    updateRule(index, {
      [field]: (rule[field] || []).filter((v) => v !== value),
    });
  };

  return (
    <div className={`space-y-4 ${className}`}>
      {rules.length === 0 ? (
        <div className="text-center py-8 border border-dashed border-gray-700 rounded-lg">
          <Plus className="h-10 w-10 text-gray-600 mx-auto mb-2" />
          <p className="text-sm text-gray-400">No rules defined</p>
          <p className="text-xs text-gray-500 mt-1">
            Rules are evaluated in priority order to determine access
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {rules.map((rule, index) => {
            const isExpanded = expandedRules.has(index);
            const effectConfig = EFFECT_OPTIONS.find((e) => e.value === rule.effect);

            return (
              <div
                key={rule.id || index}
                className={`bg-gray-800/50 rounded-lg border-2 ${
                  rule.effect === 'deny' ? 'border-danger-600/50' : 'border-gray-700'
                } overflow-hidden`}
              >
                {/* Rule Header */}
                <div className="flex items-center gap-3 p-4 bg-gray-800/80">
                  {rule.effect === 'deny' ? (
                    <XCircle className="h-5 w-5 text-danger-400 flex-shrink-0" />
                  ) : (
                    <CheckCircle className="h-5 w-5 text-success-400 flex-shrink-0" />
                  )}

                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      <Input
                        value={rule.name}
                        onChange={(e) => updateRule(index, { name: e.target.value })}
                        className="font-semibold text-white"
                        disabled={readOnly}
                      />
                      <Badge
                        variant={rule.effect === 'deny' ? 'danger' : 'success'}
                        size="sm"
                        className={effectConfig?.color}
                      >
                        {rule.effect.toUpperCase()}
                      </Badge>
                      <Badge variant="neutral" size="sm">
                        Priority: {rule.priority}
                      </Badge>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <span className="text-xs text-gray-500">
                      {rule.conditions.length} condition(s)
                    </span>
                    {!readOnly && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => toggleExpand(index)}
                      >
                        {isExpanded ? (
                          <ChevronUp className="h-4 w-4" />
                        ) : (
                          <ChevronDown className="h-4 w-4" />
                        )}
                      </Button>
                    )}
                    {!readOnly && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => duplicateRule(index)}
                        title="Duplicate rule"
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    )}
                    {!readOnly && rules.length > 1 && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeRule(index)}
                        className="text-danger-400 hover:text-danger-300"
                        title="Remove rule"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                </div>

                {/* Expanded Rule Content */}
                {isExpanded && (
                  <div className="p-4 space-y-4">
                    {/* Description */}
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">
                        Description
                      </label>
                      <Textarea
                        value={rule.description || ''}
                        onChange={(e) => updateRule(index, { description: e.target.value })}
                        placeholder="Describe what this rule does..."
                        rows={2}
                        readOnly={readOnly}
                      />
                    </div>

                    {/* Effect and Logical Operator */}
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Effect
                        </label>
                        <Select
                          value={rule.effect}
                          onChange={(e) => updateRule(index, { effect: e.target.value as PolicyEffect })}
                          options={EFFECT_OPTIONS.map((e) => ({ value: e.value, label: e.label }))}
                          disabled={readOnly}
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Logical Operator
                        </label>
                        <Select
                          value={rule.logical_operator}
                          onChange={(e) =>
                            updateRule(index, { logical_operator: e.target.value as LogicalOperator })
                          }
                          options={LOGICAL_OPERATORS.map((o) => ({ value: o.value, label: `${o.value} - ${o.description}` }))}
                          disabled={readOnly}
                        />
                      </div>
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Priority
                        </label>
                        <Input
                          type="number"
                          min="1"
                          value={rule.priority}
                          onChange={(e) => updateRule(index, { priority: parseInt(e.target.value) || 1 })}
                          disabled={readOnly}
                        />
                      </div>
                    </div>

                    {/* Resources */}
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">
                        Resources
                      </label>
                      <div className="space-y-2">
                        <div className="flex gap-2">
                          <Select
                            placeholder="Select or type a resource..."
                            onChange={(e) => {
                              if (e.target.value) {
                                addTag(index, 'resources', e.target.value);
                                e.target.value = '';
                              }
                            }}
                            options={COMMON_RESOURCES.map((r) => ({ value: r, label: r }))}
                            disabled={readOnly}
                          />
                          <Input
                            placeholder="Or type custom resource..."
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                const target = e.target as HTMLInputElement;
                                addTag(index, 'resources', target.value);
                                target.value = '';
                              }
                            }}
                            disabled={readOnly}
                          />
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {(rule.resources || []).map((resource) => (
                            <Badge key={resource} variant="neutral" size="sm">
                              {resource}
                              {!readOnly && (
                                <button
                                  type="button"
                                  onClick={() => removeTag(index, 'resources', resource)}
                                  className="ml-1 hover:text-danger-400"
                                >
                                  ×
                                </button>
                              )}
                            </Badge>
                          ))}
                          {(!rule.resources || rule.resources.length === 0) && (
                            <span className="text-sm text-gray-500">No resources specified (matches all)</span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Actions */}
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">
                        Actions
                      </label>
                      <div className="space-y-2">
                        <div className="flex gap-2">
                          <Select
                            placeholder="Select or type an action..."
                            onChange={(e) => {
                              if (e.target.value) {
                                addTag(index, 'actions', e.target.value);
                                e.target.value = '';
                              }
                            }}
                            options={COMMON_ACTIONS.map((a) => ({ value: a, label: a }))}
                            disabled={readOnly}
                          />
                          <Input
                            placeholder="Or type custom action..."
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') {
                                const target = e.target as HTMLInputElement;
                                addTag(index, 'actions', target.value);
                                target.value = '';
                              }
                            }}
                            disabled={readOnly}
                          />
                        </div>
                        <div className="flex flex-wrap gap-2">
                          {(rule.actions || []).map((action) => (
                            <Badge key={action} variant="info" size="sm">
                              {action}
                              {!readOnly && (
                                <button
                                  type="button"
                                  onClick={() => removeTag(index, 'actions', action)}
                                  className="ml-1 hover:text-danger-400"
                                >
                                  ×
                                </button>
                              )}
                            </Badge>
                          ))}
                          {(!rule.actions || rule.actions.length === 0) && (
                            <span className="text-sm text-gray-500">No actions specified (matches all)</span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Roles */}
                    <div>
                      <label className="block text-sm font-medium text-gray-300 mb-1">
                        Roles
                      </label>
                      <div className="space-y-2">
                        <Input
                          placeholder="Enter roles (comma-separated)..."
                          defaultValue={(rule.roles || []).join(',')}
                          onKeyDown={(e) => {
                            if (e.key === 'Enter') {
                              const target = e.target as HTMLInputElement;
                              const values = target.value.split(',').map((s) => s.trim()).filter(Boolean);
                              values.forEach((v) => addTag(index, 'roles', v));
                              target.value = '';
                            }
                          }}
                          onBlur={(e) => {
                            const values = e.target.value.split(',').map((s) => s.trim()).filter(Boolean);
                            if (values.length > 0) {
                              updateRule(index, { roles: values });
                            }
                          }}
                          disabled={readOnly}
                        />
                        <div className="flex flex-wrap gap-2">
                          {(rule.roles || []).map((role) => (
                            <Badge key={role} variant="success" size="sm">
                              {role}
                              {!readOnly && (
                                <button
                                  type="button"
                                  onClick={() => removeTag(index, 'roles', role)}
                                  className="ml-1 hover:text-danger-400"
                                >
                                  ×
                                </button>
                              )}
                            </Badge>
                          ))}
                          {(!rule.roles || rule.roles.length === 0) && (
                            <span className="text-sm text-gray-500">No roles specified (matches all)</span>
                          )}
                        </div>
                      </div>
                    </div>

                    {/* Conditions */}
                    <div>
                      <div className="flex items-center justify-between mb-2">
                        <label className="block text-sm font-medium text-gray-300">
                          Conditions
                        </label>
                        <span className="text-xs text-gray-500">
                          {rule.logical_operator} - All conditions must be met
                        </span>
                      </div>
                      <ConditionBuilder
                        conditions={rule.conditions || []}
                        onChange={(conditions) => updateRule(index, { conditions })}
                        readOnly={readOnly}
                      />
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Add Rule Button */}
      {!readOnly && (
        <Button
          variant="secondary"
          onClick={addRule}
          leftIcon={<Plus className="h-4 w-4" />}
          fullWidth
        >
          Add Rule
        </Button>
      )}
    </div>
  );
};

const Copy = ({ className }: { className?: string }) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
    className={className}
  >
    <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
    <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
  </svg>
);

export default PolicyRuleBuilder;

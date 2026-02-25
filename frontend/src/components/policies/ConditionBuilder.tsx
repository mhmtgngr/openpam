import React, { useState } from 'react';
import { X, Plus, Info, Clock, Globe, Shield, User } from 'lucide-react';
import { Input, Select, Button, Badge, Card, Toggle } from '@/components/common';
import type {
  PolicyCondition,
  ConditionType,
  ConditionOperator,
} from '@/types/policy';

interface ConditionBuilderProps {
  conditions: PolicyCondition[];
  onChange: (conditions: PolicyCondition[]) => void;
  readOnly?: boolean;
  className?: string;
}

// Condition type definitions
const CONDITION_TYPES: {
  value: ConditionType;
  label: string;
  icon: React.ReactNode;
  description: string;
  fields: { name: string; label: string; type: string; placeholder: string }[];
}[] = [
  {
    value: 'time_range',
    label: 'Time Range',
    icon: <Clock className="h-4 w-4" />,
    description: 'Restrict access to specific time periods',
    fields: [
      { name: 'start_time', label: 'Start Time', type: 'time', placeholder: '09:00' },
      { name: 'end_time', label: 'End Time', type: 'time', placeholder: '17:00' },
      { name: 'days', label: 'Days of Week', type: 'text', placeholder: 'Mon,Tue,Wed,Thu,Fri' },
      { name: 'timezone', label: 'Timezone', type: 'text', placeholder: 'UTC' },
    ],
  },
  {
    value: 'ip_range',
    label: 'IP Range',
    icon: <Globe className="h-4 w-4" />,
    description: 'Restrict access from specific IP addresses',
    fields: [
      { name: 'ips', label: 'IP Addresses/CIDRs', type: 'text', placeholder: '192.168.1.0/24,10.0.0.1' },
    ],
  },
  {
    value: 'role',
    label: 'Role',
    icon: <Shield className="h-4 w-4" />,
    description: 'Match based on user roles',
    fields: [
      { name: 'roles', label: 'Roles', type: 'text', placeholder: 'admin,operator,auditor' },
    ],
  },
  {
    value: 'user_attribute',
    label: 'User Attribute',
    icon: <User className="h-4 w-4" />,
    description: 'Match based on user attributes',
    fields: [
      { name: 'attribute', label: 'Attribute Name', type: 'text', placeholder: 'department,location,level' },
      { name: 'value', label: 'Attribute Value', type: 'text', placeholder: 'engineering,remote,senior' },
    ],
  },
  {
    value: 'resource',
    label: 'Resource',
    icon: <Shield className="h-4 w-4" />,
    description: 'Match based on resource properties',
    fields: [
      { name: 'property', label: 'Resource Property', type: 'text', placeholder: 'type,environment,sensitivity' },
      { name: 'value', label: 'Property Value', type: 'text', placeholder: 'ssh,production,high' },
    ],
  },
  {
    value: 'custom',
    label: 'Custom Expression',
    icon: <Info className="h-4 w-4" />,
    description: 'Define a custom condition expression',
    fields: [
      { name: 'expression', label: 'Expression', type: 'text', placeholder: 'user.department == "engineering"' },
    ],
  },
];

// Operators based on condition type
const OPERATORS_BY_TYPE: Record<ConditionType, { value: ConditionOperator; label: string }[]> = {
  time_range: [
    { value: 'time_between', label: 'Between' },
    { value: 'day_of_week', label: 'Day of Week' },
  ],
  ip_range: [
    { value: 'ip_in_range', label: 'In Range' },
    { value: 'equals', label: 'Equals' },
    { value: 'in', label: 'In List' },
  ],
  role: [
    { value: 'equals', label: 'Equals' },
    { value: 'in', label: 'In List' },
    { value: 'not_in', label: 'Not In List' },
  ],
  user_attribute: [
    { value: 'equals', label: 'Equals' },
    { value: 'not_equals', label: 'Not Equals' },
    { value: 'contains', label: 'Contains' },
    { value: 'in', label: 'In List' },
    { value: 'not_in', label: 'Not In List' },
  ],
  resource: [
    { value: 'equals', label: 'Equals' },
    { value: 'not_equals', label: 'Not Equals' },
    { value: 'contains', label: 'Contains' },
    { value: 'starts_with', label: 'Starts With' },
    { value: 'in', label: 'In List' },
  ],
  custom: [
    { value: 'regex', label: 'Regex Match' },
  ],
};

export const ConditionBuilder: React.FC<ConditionBuilderProps> = ({
  conditions,
  onChange,
  readOnly = false,
  className = '',
}) => {
  const [expandedCondition, setExpandedCondition] = useState<string | null>(null);

  const addCondition = () => {
    const newCondition: PolicyCondition = {
      id: `cond_${Date.now()}`,
      type: 'time_range',
      field: 'time',
      operator: 'time_between',
      value: '',
      negated: false,
    };
    onChange([...conditions, newCondition]);
    setExpandedCondition(newCondition.id ?? null);
  };

  const updateCondition = (index: number, updates: Partial<PolicyCondition>) => {
    const newConditions = [...conditions];
    newConditions[index] = { ...newConditions[index], ...updates };
    onChange(newConditions);
  };

  const removeCondition = (index: number) => {
    onChange(conditions.filter((_, i) => i !== index));
  };

  const getConditionLabel = (condition: PolicyCondition): string => {
    const typeInfo = CONDITION_TYPES.find((t) => t.value === condition.type);
    if (!typeInfo) return condition.type;

    const valueStr = Array.isArray(condition.value)
      ? condition.value.join(', ')
      : String(condition.value ?? '');

    return `${condition.negated ? 'NOT ' : ''}${typeInfo.label}: ${valueStr || '(not set)'}`;
  };

  const getConditionTypeConfig = (type: ConditionType) => {
    return CONDITION_TYPES.find((t) => t.value === type);
  };

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Conditions List */}
      {conditions.length === 0 ? (
        <div className="text-center py-8 border border-dashed border-gray-700 rounded-lg">
          <Shield className="h-10 w-10 text-gray-600 mx-auto mb-2" />
          <p className="text-sm text-gray-400">No conditions defined</p>
          <p className="text-xs text-gray-500 mt-1">
            Add conditions to control when this rule applies
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {conditions.map((condition, index) => {
            const typeConfig = getConditionTypeConfig(condition.type);
            const isExpanded = expandedCondition === condition.id;

            return (
              <div
                key={condition.id || index}
                className={`bg-gray-800/50 rounded-lg border ${
                  condition.negated ? 'border-warning-600/50' : 'border-gray-700'
                }`}
              >
                {/* Condition Header */}
                <div className="flex items-center gap-3 p-3">
                  {condition.negated && (
                    <Badge variant="warning" size="sm">NOT</Badge>
                  )}
                  <div className="flex items-center gap-2 flex-1">
                    {typeConfig?.icon}
                    <span className="text-sm font-medium text-white">
                      {getConditionLabel(condition)}
                    </span>
                  </div>
                  {!readOnly && (
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() =>
                          setExpandedCondition(isExpanded ? null : (condition.id || null))
                        }
                      >
                        {isExpanded ? '−' : '+'}
                      </Button>
                      <Toggle
                        checked={condition.negated}
                        onChange={(checked) => updateCondition(index, { negated: checked })}
                        className="mr-2"
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeCondition(index)}
                        className="text-danger-400 hover:text-danger-300"
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                </div>

                {/* Expanded Condition Editor */}
                {isExpanded && typeConfig && !readOnly && (
                  <div className="px-3 pb-3 border-t border-gray-700 pt-3">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      {/* Condition Type */}
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Condition Type
                        </label>
                        <Select
                          value={condition.type}
                          onChange={(e) =>
                            updateCondition(index, {
                              type: e.target.value as ConditionType,
                              operator: OPERATORS_BY_TYPE[e.target.value as ConditionType][0].value,
                            })
                          }
                          options={CONDITION_TYPES.map((t) => ({
                            value: t.value,
                            label: t.label,
                          }))}
                        />
                      </div>

                      {/* Operator */}
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Operator
                        </label>
                        <Select
                          value={condition.operator}
                          onChange={(e) =>
                            updateCondition(index, { operator: e.target.value as ConditionOperator })
                          }
                          options={OPERATORS_BY_TYPE[condition.type] || []}
                        />
                      </div>

                      {/* Field Name */}
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Field Name
                        </label>
                        <Input
                          value={condition.field}
                          onChange={(e) => updateCondition(index, { field: e.target.value })}
                          placeholder="Enter field name"
                        />
                      </div>

                      {/* Value Input(s) */}
                      <div>
                        <label className="block text-sm font-medium text-gray-300 mb-1">
                          Value
                        </label>
                        {condition.type === 'time_range' ? (
                          <div className="flex gap-2">
                            <Input
                              type="time"
                              value={
                                typeof condition.value === 'object' && condition.value
                                  ? (condition.value as any).start_time || ''
                                  : ''
                              }
                              onChange={(e) =>
                                updateCondition(index, {
                                  value: { ...(condition.value as any), start_time: e.target.value },
                                })
                              }
                            />
                            <Input
                              type="time"
                              value={
                                typeof condition.value === 'object' && condition.value
                                  ? (condition.value as any).end_time || ''
                                  : ''
                              }
                              onChange={(e) =>
                                updateCondition(index, {
                                  value: { ...(condition.value as any), end_time: e.target.value },
                                })
                              }
                            />
                          </div>
                        ) : Array.isArray(condition.value) ? (
                          <Input
                            value={condition.value.join(',')}
                            onChange={(e) =>
                              updateCondition(index, {
                                value: e.target.value.split(',').map((s) => s.trim()),
                              })
                            }
                            placeholder="Comma-separated values"
                          />
                        ) : (
                          <Input
                            value={String(condition.value || '')}
                            onChange={(e) =>
                              updateCondition(index, {
                                value:
                                  condition.type === 'custom' ||
                                  condition.operator === 'regex' ||
                                  condition.type === 'user_attribute'
                                    ? e.target.value
                                    : e.target.value,
                              })
                            }
                            placeholder="Enter value"
                          />
                        )}
                      </div>
                    </div>

                    {/* Type-specific help text */}
                    <div className="mt-3 text-xs text-gray-500">
                      {typeConfig.description}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      )}

      {/* Add Condition Button */}
      {!readOnly && (
        <Button
          variant="secondary"
          size="sm"
          onClick={addCondition}
          leftIcon={<Plus className="h-4 w-4" />}
          fullWidth
        >
          Add Condition
        </Button>
      )}
    </div>
  );
};

export default ConditionBuilder;

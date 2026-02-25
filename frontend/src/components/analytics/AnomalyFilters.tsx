import React from 'react';
import { Search, Filter, X } from 'lucide-react';
import { Input, Select, Button } from '@/components/common';
import type { AnomalySeverity, AnomalyType } from '@/types';
import clsx from 'clsx';

export interface AnomalyFilterValues {
  search?: string;
  severity?: AnomalySeverity | '';
  status?: string;
  type?: AnomalyType | '';
  start_date?: string;
  end_date?: string;
  period?: number;
}

const severityOptions: Array<{ value: AnomalySeverity | ''; label: string }> = [
  { value: '', label: 'All Severities' },
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
];

const statusOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Statuses' },
  { value: 'open', label: 'Open' },
  { value: 'investigating', label: 'Investigating' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'false_positive', label: 'False Positive' },
];

const typeOptions: Array<{ value: AnomalyType | ''; label: string }> = [
  { value: '', label: 'All Types' },
  { value: 'unusual_access_time', label: 'Unusual Access Time' },
  { value: 'unusual_location', label: 'Unusual Location' },
  { value: 'privileged_escalation', label: 'Privileged Escalation' },
  { value: 'bulk_data_access', label: 'Bulk Data Access' },
  { value: 'command_injection', label: 'Command Injection' },
  { value: 'ransomware_indicators', label: 'Ransomware Indicators' },
  { value: 'impossible_travel', label: 'Impossible Travel' },
  { value: 'account_takeover', label: 'Account Takeover' },
  { value: 'credential_theft', label: 'Credential Theft' },
  { value: 'excessive_failed_logins', label: 'Excessive Failed Logins' },
];

const periodOptions: Array<{ value: number; label: string }> = [
  { value: 7, label: 'Last 7 days' },
  { value: 14, label: 'Last 14 days' },
  { value: 30, label: 'Last 30 days' },
  { value: 90, label: 'Last 90 days' },
];

interface AnomalyFiltersProps {
  filters: AnomalyFilterValues;
  onChange: (filters: AnomalyFilterValues) => void;
  onReset: () => void;
  isLoading?: boolean;
  showPeriodSelector?: boolean;
}

export const AnomalyFilters: React.FC<AnomalyFiltersProps> = ({
  filters,
  onChange,
  onReset,
  isLoading = false,
  showPeriodSelector = true,
}) => {
  const updateFilter = <K extends keyof AnomalyFilterValues>(key: K, value: AnomalyFilterValues[K]) => {
    onChange({ ...filters, [key]: value });
  };

  const hasActiveFilters = Boolean(
    filters.search ||
    filters.severity ||
    filters.status ||
    filters.type
  );

  const handleReset = () => {
    onReset();
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-2">
          <Filter className="h-4 w-4 text-gray-400" />
          <span className="text-sm font-medium text-gray-300">Filters</span>
        </div>

        {hasActiveFilters && (
          <Button
            variant="secondary"
            size="sm"
            onClick={handleReset}
            disabled={isLoading}
            leftIcon={<X className="h-4 w-4" />}
          >
            Clear Filters
          </Button>
        )}
      </div>

      <div className="flex flex-wrap gap-3">
        {/* Period Selector */}
        {showPeriodSelector && (
          <div className="w-full sm:w-auto">
            <Select
              options={periodOptions.map(p => ({ value: String(p.value), label: p.label }))}
              value={String(filters.period || 30)}
              onChange={(e) => updateFilter('period', Number(e.target.value))}
              className="w-full sm:w-40"
              disabled={isLoading}
            />
          </div>
        )}

        {/* Search */}
        <div className="flex-1 min-w-[200px]">
          <Input
            placeholder="Search anomalies..."
            leftIcon={<Search className="h-4 w-4 text-gray-400" />}
            value={filters.search || ''}
            onChange={(e) => updateFilter('search', e.target.value)}
            disabled={isLoading}
          />
        </div>

        {/* Severity Filter */}
        <div className="w-full sm:w-40">
          <Select
            options={severityOptions}
            value={filters.severity || ''}
            onChange={(e) => updateFilter('severity', e.target.value as AnomalySeverity | '')}
            disabled={isLoading}
          />
        </div>

        {/* Status Filter */}
        <div className="w-full sm:w-40">
          <Select
            options={statusOptions}
            value={filters.status || ''}
            onChange={(e) => updateFilter('status', e.target.value)}
            disabled={isLoading}
          />
        </div>

        {/* Type Filter */}
        <div className="w-full sm:w-56">
          <Select
            options={typeOptions}
            value={filters.type || ''}
            onChange={(e) => updateFilter('type', e.target.value as AnomalyType | '')}
            disabled={isLoading}
          />
        </div>
      </div>

      {/* Active Filters Display */}
      {hasActiveFilters && (
        <div className="flex flex-wrap gap-2">
          {filters.search && (
            <FilterBadge label="Search" value={filters.search} onRemove={() => updateFilter('search', '')} />
          )}
          {filters.severity && (
            <FilterBadge
              label="Severity"
              value={severityOptions.find(s => s.value === filters.severity)?.label || filters.severity}
              onRemove={() => updateFilter('severity', '')}
            />
          )}
          {filters.status && (
            <FilterBadge
              label="Status"
              value={statusOptions.find(s => s.value === filters.status)?.label || filters.status}
              onRemove={() => updateFilter('status', '')}
            />
          )}
          {filters.type && (
            <FilterBadge
              label="Type"
              value={typeOptions.find(t => t.value === filters.type)?.label || filters.type}
              onRemove={() => updateFilter('type', '')}
            />
          )}
        </div>
      )}
    </div>
  );
};

interface FilterBadgeProps {
  label: string;
  value: string;
  onRemove: () => void;
}

const FilterBadge: React.FC<FilterBadgeProps> = ({ label, value, onRemove }) => {
  // Truncate long values
  const displayValue = value.length > 20 ? `${value.substring(0, 20)}...` : value;

  return (
    <span className="inline-flex items-center gap-1 rounded-full bg-primary-500/20 px-3 py-1 text-xs">
      <span className="font-medium text-primary-300">{label}:</span>
      <span className="text-white">{displayValue}</span>
      <button
        type="button"
        onClick={onRemove}
        className="ml-1 text-gray-400 hover:text-white transition-colors"
      >
        <X className="h-3 w-3" />
      </button>
    </span>
  );
};

export default AnomalyFilters;

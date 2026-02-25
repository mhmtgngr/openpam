import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  ArrowLeft,
  FileText,
  Calendar,
  Settings,
  Filter,
  CheckCircle,
  Sparkles,
  Info,
  AlertCircle,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card, Badge, Button } from '@/components/common';
import { Input, Select, Textarea, Label, Toggle, DatePicker } from '@/components/common';
import { LoadingState } from '@/components/common';
import type {
  ComplianceFramework,
  GenerateReportRequest,
  ReportSection,
  FrameworkMetadata,
  ReportFilter,
} from '@/types/reports';
import { format, subMonths, subYears } from 'date-fns';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const frameworks: { value: ComplianceFramework; label: string; description: string }[] = [
  { value: 'soc2', label: 'SOC 2', description: 'Service Organization Control 2' },
  { value: 'iso27001', label: 'ISO 27001', description: 'Information Security Management' },
  { value: 'pci_dss', label: 'PCI DSS', description: 'Payment Card Industry Data Security' },
  { value: 'hipaa', label: 'HIPAA', description: 'Health Insurance Portability' },
  { value: 'gdpr', label: 'GDPR', description: 'General Data Protection Regulation' },
  { value: 'nerc_cip', label: 'NERC CIP', description: 'Critical Infrastructure Protection' },
  { value: 'custom', label: 'Custom', description: 'Custom compliance framework' },
];

const reportFormats = [
  { value: 'pdf', label: 'PDF', description: 'Best for sharing and printing' },
  { value: 'html', label: 'HTML', description: 'Interactive web format' },
  { value: 'json', label: 'JSON', description: 'Machine-readable format' },
  { value: 'csv', label: 'CSV', description: 'Spreadsheet compatible' },
  { value: 'xlsx', label: 'Excel', description: 'Microsoft Excel format' },
];

const defaultSections: ReportSection[] = [
  { id: 'summary', name: 'Executive Summary', enabled: true },
  { id: 'controls', name: 'Control Assessment', enabled: true },
  { id: 'findings', name: 'Findings and Recommendations', enabled: true },
  { id: 'metrics', name: 'Metrics and Statistics', enabled: true },
  { id: 'evidence', name: 'Evidence Documentation', enabled: false },
  { id: 'exceptions', name: 'Compliance Exceptions', enabled: true },
];

const periodPresets = [
  { label: 'Last Month', value: 1, unit: 'months' },
  { label: 'Last Quarter', value: 3, unit: 'months' },
  { label: 'Last 6 Months', value: 6, unit: 'months' },
  { label: 'Last Year', value: 1, unit: 'years' },
];

type FilterState = {
  environments: string[];
  sessionTypes: string[];
  riskLevels: string[];
};

export const ReportGeneratorPage: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [reportName, setReportName] = useState('');
  const [selectedFramework, setSelectedFramework] = useState<ComplianceFramework>('soc2');
  const [period, setPeriod] = useState({
    start: format(subMonths(new Date(), 1), 'yyyy-MM-dd'),
    end: format(new Date(), 'yyyy-MM-dd'),
  });
  const [selectedFormat, setSelectedFormat] = useState('pdf');
  const [selectedSections, setSelectedSections] = useState<string[]>([
    'summary',
    'controls',
    'findings',
    'exceptions',
  ]);
  const [filters, setFilters] = useState<FilterState>({
    environments: [],
    sessionTypes: [],
    riskLevels: [],
  });
  const [options, setOptions] = useState({
    includePii: false,
    includeSessionLogs: true,
    includeCommandHistory: true,
    redactSensitiveData: true,
    aggregateResults: true,
    compareWithPrevious: false,
  });
  const [scheduleReport, setScheduleReport] = useState(false);

  // Fetch framework metadata for available controls
  const { data: frameworkMetadata, isLoading: loadingMetadata } = useQuery({
    queryKey: ['frameworkMetadata', selectedFramework],
    queryFn: () => reportsApi.getFrameworkMetadata(selectedFramework),
    enabled: step === 2,
  });

  const generateMutation = useMutation({
    mutationFn: (request: GenerateReportRequest) => reportsApi.generate(request),
    onSuccess: (data) => {
      if (data.status === 'completed' && data.report_id) {
        toast.success('Report generated successfully!');
        navigate(`/reports/${data.report_id}`);
      } else if (data.job_id) {
        toast.success('Report generation started!');
        navigate(`/reports?job=${data.job_id}`);
      } else {
        toast.success('Report request submitted!');
        navigate('/reports');
      }
      queryClient.invalidateQueries({ queryKey: ['reports'] });
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to generate report');
    },
  });

  const handlePeriodPreset = (value: number, unit: 'months' | 'years') => {
    const end = new Date();
    const start = unit === 'months' ? subMonths(end, value) : subYears(end, value);
    setPeriod({
      start: format(start, 'yyyy-MM-dd'),
      end: format(end, 'yyyy-MM-dd'),
    });
  };

  const toggleSection = (sectionId: string) => {
    setSelectedSections((prev) =>
      prev.includes(sectionId)
        ? prev.filter((id) => id !== sectionId)
        : [...prev, sectionId]
    );
  };

  const toggleFilter = (category: keyof FilterState, value: string) => {
    setFilters((prev) => ({
      ...prev,
      [category]: prev[category].includes(value)
        ? prev[category].filter((v) => v !== value)
        : [...prev[category], value],
    }));
  };

  const handleNext = () => {
    if (step === 1) {
      if (!reportName.trim()) {
        toast.error('Please enter a report name');
        return;
      }
      setStep(2);
    } else if (step === 2) {
      setStep(3);
    }
  };

  const handleBack = () => {
    if (step > 1) setStep((step - 1) as 1 | 2 | 3);
  };

  const handleGenerate = () => {
    const request: GenerateReportRequest = {
      framework: selectedFramework,
      period_start: period.start,
      period_end: period.end,
      include_sections: selectedSections,
      format: selectedFormat as any,
      filters: {
        environments: filters.environments.length > 0 ? filters.environments : undefined,
        session_types: filters.sessionTypes.length > 0 ? filters.sessionTypes : undefined,
        risk_levels: filters.riskLevels.length > 0 ? filters.riskLevels : undefined,
      },
      options: {
        include_pii: options.includePii,
        include_session_logs: options.includeSessionLogs,
        include_command_history: options.includeCommandHistory,
        redact_sensitive_data: options.redactSensitiveData,
        aggregate_results: options.aggregateResults,
        comparison_period: options.compareWithPrevious
          ? {
              start: format(
                subMonths(new Date(period.start), 1),
                'yyyy-MM-dd'
              ),
              end: format(
                subMonths(new Date(period.end), 1),
                'yyyy-MM-dd'
              ),
            }
          : undefined,
      },
    };

    generateMutation.mutate(request);
  };

  const environmentOptions = ['production', 'staging', 'development', 'test'];
  const sessionTypeOptions = ['ssh', 'rdp', 'database', 'kubernetes', 'web', 'api'];
  const riskLevelOptions = ['critical', 'high', 'medium', 'low'];

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <button
          onClick={() => navigate('/reports')}
          className="rounded-lg p-2 text-gray-400 hover:bg-gray-800 hover:text-white"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-white">Generate Compliance Report</h1>
          <p className="mt-1 text-sm text-gray-400">
            Configure and generate a new compliance report
          </p>
        </div>
      </div>

      {/* Progress Steps */}
      <div className="flex items-center justify-center">
        {[
          { num: 1, label: 'Basic Info', icon: FileText },
          { num: 2, label: 'Content', icon: Settings },
          { num: 3, label: 'Review', icon: CheckCircle },
        ].map((s) => (
          <React.Fragment key={s.num}>
            <div className="flex items-center">
              <div
                className={clsx(
                  'flex h-10 w-10 items-center justify-center rounded-full border-2 transition-colors',
                  step >= s.num
                    ? 'border-primary-500 bg-primary-500 text-white'
                    : 'border-gray-700 text-gray-500'
                )}
              >
                {step > s.num ? <CheckCircle className="h-5 w-5" /> : <s.icon className="h-5 w-5" />}
              </div>
              <span
                className={clsx(
                  'ml-2 text-sm font-medium',
                  step >= s.num ? 'text-white' : 'text-gray-500'
                )}
              >
                {s.label}
              </span>
            </div>
            {s.num < 3 && (
              <div className="mx-4 h-0.5 w-16 bg-gray-700">
                <div
                  className={clsx(
                    'h-full transition-all duration-300',
                    step > s.num ? 'w-full bg-primary-500' : 'w-0 bg-primary-500'
                  )}
                />
              </div>
            )}
          </React.Fragment>
        ))}
      </div>

      {/* Step 1: Basic Info */}
      {step === 1 && (
        <Card>
          <div className="card-body space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-white">Report Details</h3>
              <p className="mt-1 text-sm text-gray-400">
                Provide basic information about the report you want to generate.
              </p>
            </div>

            <div className="space-y-4">
              <div>
                <Label htmlFor="report-name">Report Name *</Label>
                <Input
                  id="report-name"
                  value={reportName}
                  onChange={(e) => setReportName(e.target.value)}
                  placeholder="e.g., Q1 2024 SOC 2 Compliance Report"
                />
              </div>

              <div>
                <Label htmlFor="framework">Compliance Framework *</Label>
                <div className="mt-2 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  {frameworks.map((fw) => (
                    <button
                      key={fw.value}
                      type="button"
                      onClick={() => setSelectedFramework(fw.value as ComplianceFramework)}
                      className={clsx(
                        'flex flex-col items-start rounded-lg border p-4 text-left transition-colors',
                        selectedFramework === fw.value
                          ? 'border-primary-500 bg-primary-500/10'
                          : 'border-gray-700 hover:bg-gray-800'
                      )}
                    >
                      <span className="font-medium text-white">{fw.label}</span>
                      <span className="text-sm text-gray-400">{fw.description}</span>
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <Label>Reporting Period *</Label>
                <div className="mt-2 flex flex-wrap gap-2 mb-3">
                  {periodPresets.map((preset) => (
                    <Button
                      key={preset.label}
                      variant="ghost"
                      size="sm"
                      onClick={() => handlePeriodPreset(preset.value, preset.unit as 'months' | 'years')}
                    >
                      {preset.label}
                    </Button>
                  ))}
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <Label htmlFor="period-start">Start Date</Label>
                    <Input
                      id="period-start"
                      type="date"
                      value={period.start}
                      onChange={(e) => setPeriod((prev) => ({ ...prev, start: e.target.value }))}
                    />
                  </div>
                  <div>
                    <Label htmlFor="period-end">End Date</Label>
                    <Input
                      id="period-end"
                      type="date"
                      value={period.end}
                      onChange={(e) => setPeriod((prev) => ({ ...prev, end: e.target.value }))}
                    />
                  </div>
                </div>
              </div>

              <div>
                <Label htmlFor="format">Output Format</Label>
                <select
                  id="format"
                  value={selectedFormat}
                  onChange={(e) => setSelectedFormat(e.target.value)}
                  className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                >
                  {reportFormats.map((fmt) => (
                    <option key={fmt.value} value={fmt.value}>
                      {fmt.label} - {fmt.description}
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex items-center justify-between rounded-lg border border-gray-700 p-4">
                <div>
                  <span className="text-sm font-medium text-white">Schedule this report</span>
                  <p className="text-xs text-gray-400">Set up automatic recurring generation</p>
                </div>
                <Toggle
                  checked={scheduleReport}
                  onChange={(e) => setScheduleReport((e.target as HTMLInputElement).checked)}
                />
              </div>
            </div>

            <div className="flex justify-end gap-3">
              <Button variant="secondary" onClick={() => navigate('/reports')}>
                Cancel
              </Button>
              <Button variant="primary" onClick={handleNext}>
                Next: Configure Content
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Step 2: Content & Filters */}
      {step === 2 && (
        <Card>
          <div className="card-body space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-white">Report Content</h3>
              <p className="mt-1 text-sm text-gray-400">
                Select which sections to include and apply filters.
              </p>
            </div>

            {/* Report Sections */}
            <div>
              <Label className="mb-3">Include Sections</Label>
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                {defaultSections.map((section) => (
                  <label
                    key={section.id}
                    className={clsx(
                      'flex items-start gap-3 rounded-lg border p-4 cursor-pointer transition-colors',
                      selectedSections.includes(section.id)
                        ? 'border-primary-500 bg-primary-500/10'
                        : 'border-gray-700 hover:bg-gray-800'
                    )}
                  >
                    <input
                      type="checkbox"
                      checked={selectedSections.includes(section.id)}
                      onChange={() => toggleSection(section.id)}
                      className="mt-1"
                    />
                    <div>
                      <span className="font-medium text-white">{section.name}</span>
                    </div>
                  </label>
                ))}
              </div>
            </div>

            {/* Filters */}
            <div>
              <Label className="mb-3 flex items-center gap-2">
                <Filter className="h-4 w-4" />
                Filters (Optional)
              </Label>
              <div className="space-y-4 rounded-lg border border-gray-700 bg-gray-800/50 p-4">
                {/* Environments */}
                <div>
                  <p className="text-sm font-medium text-white mb-2">Environments</p>
                  <div className="flex flex-wrap gap-2">
                    {environmentOptions.map((env) => (
                      <button
                        key={env}
                        type="button"
                        onClick={() => toggleFilter('environments', env)}
                        className={clsx(
                          'rounded-full px-3 py-1 text-sm capitalize transition-colors',
                          filters.environments.includes(env)
                            ? 'bg-primary-500 text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        )}
                      >
                        {env}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Session Types */}
                <div>
                  <p className="text-sm font-medium text-white mb-2">Session Types</p>
                  <div className="flex flex-wrap gap-2">
                    {sessionTypeOptions.map((type) => (
                      <button
                        key={type}
                        type="button"
                        onClick={() => toggleFilter('sessionTypes', type)}
                        className={clsx(
                          'rounded-full px-3 py-1 text-sm capitalize transition-colors',
                          filters.sessionTypes.includes(type)
                            ? 'bg-primary-500 text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        )}
                      >
                        {type}
                      </button>
                    ))}
                  </div>
                </div>

                {/* Risk Levels */}
                <div>
                  <p className="text-sm font-medium text-white mb-2">Risk Levels</p>
                  <div className="flex flex-wrap gap-2">
                    {riskLevelOptions.map((level) => (
                      <button
                        key={level}
                        type="button"
                        onClick={() => toggleFilter('riskLevels', level)}
                        className={clsx(
                          'rounded-full px-3 py-1 text-sm capitalize transition-colors',
                          filters.riskLevels.includes(level)
                            ? 'bg-danger-500 text-white'
                            : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                        )}
                      >
                        {level}
                      </button>
                    ))}
                  </div>
                </div>
              </div>
            </div>

            {/* Options */}
            <div>
              <Label className="mb-3">Options</Label>
              <div className="space-y-3 rounded-lg border border-gray-700 bg-gray-800/50 p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-white">Redact Sensitive Data</span>
                    <p className="text-xs text-gray-400">Mask passwords, tokens, and other sensitive info</p>
                  </div>
                  <Toggle
                    checked={options.redactSensitiveData}
                    onChange={(e) => setOptions((prev) => ({ ...prev, redactSensitiveData: (e.target as HTMLInputElement).checked }))}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-white">Include Session Logs</span>
                    <p className="text-xs text-gray-400">Attach detailed session activity logs</p>
                  </div>
                  <Toggle
                    checked={options.includeSessionLogs}
                    onChange={(e) => setOptions((prev) => ({ ...prev, includeSessionLogs: (e.target as HTMLInputElement).checked }))}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-white">Include Command History</span>
                    <p className="text-xs text-gray-400">Include executed commands in the report</p>
                  </div>
                  <Toggle
                    checked={options.includeCommandHistory}
                    onChange={(e) => setOptions((prev) => ({ ...prev, includeCommandHistory: (e.target as HTMLInputElement).checked }))}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-white">Compare with Previous Period</span>
                    <p className="text-xs text-gray-400">Show trends and changes from previous period</p>
                  </div>
                  <Toggle
                    checked={options.compareWithPrevious}
                    onChange={(e) => setOptions((prev) => ({ ...prev, compareWithPrevious: (e.target as HTMLInputElement).checked }))}
                  />
                </div>

                <div className="flex items-center justify-between">
                  <div>
                    <span className="text-sm font-medium text-white">Include PII Data</span>
                    <p className="text-xs text-gray-400 text-warning-400">Not recommended for external reports</p>
                  </div>
                  <Toggle
                    checked={options.includePii}
                    onChange={(e) => setOptions((prev) => ({ ...prev, includePii: (e.target as HTMLInputElement).checked }))}
                  />
                </div>
              </div>
            </div>

            <div className="flex justify-between">
              <Button variant="secondary" onClick={handleBack}>
                Back
              </Button>
              <Button variant="primary" onClick={handleNext}>
                Next: Review & Generate
              </Button>
            </div>
          </div>
        </Card>
      )}

      {/* Step 3: Review */}
      {step === 3 && (
        <Card>
          <div className="card-body space-y-6">
            <div>
              <h3 className="text-lg font-semibold text-white flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-primary-400" />
                Review Report Configuration
              </h3>
              <p className="mt-1 text-sm text-gray-400">
                Review your report settings before generating.
              </p>
            </div>

            {/* Summary */}
            <div className="space-y-4 rounded-lg border border-gray-700 bg-gray-800/50 p-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <span className="text-sm text-gray-400">Report Name</span>
                  <p className="font-medium text-white">{reportName}</p>
                </div>
                <div>
                  <span className="text-sm text-gray-400">Framework</span>
                  <p className="font-medium text-white">
                    {frameworks.find((f) => f.value === selectedFramework)?.label}
                  </p>
                </div>
                <div>
                  <span className="text-sm text-gray-400">Period</span>
                  <p className="font-medium text-white">
                    {format(new Date(period.start), 'MMM dd, yyyy')} —{' '}
                    {format(new Date(period.end), 'MMM dd, yyyy')}
                  </p>
                </div>
                <div>
                  <span className="text-sm text-gray-400">Format</span>
                  <p className="font-medium text-white uppercase">{selectedFormat}</p>
                </div>
              </div>

              {frameworkMetadata && (
                <div className="pt-4 border-t border-gray-700">
                  <span className="text-sm text-gray-400">
                    {frameworkMetadata.controls.length} controls will be assessed
                  </span>
                </div>
              )}
            </div>

            {/* Selected Sections */}
            <div>
              <Label className="mb-3">Selected Sections ({selectedSections.length})</Label>
              <div className="flex flex-wrap gap-2">
                {selectedSections.map((sectionId) => {
                  const section = defaultSections.find((s) => s.id === sectionId);
                  return (
                    <Badge key={sectionId} variant="neutral">
                      {section?.name}
                    </Badge>
                  );
                })}
              </div>
            </div>

            {/* Active Filters */}
            {(filters.environments.length > 0 ||
              filters.sessionTypes.length > 0 ||
              filters.riskLevels.length > 0) && (
              <div>
                <Label className="mb-3">Active Filters</Label>
                <div className="space-y-2">
                  {filters.environments.length > 0 && (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-400">Environments:</span>
                      <div className="flex flex-wrap gap-1">
                        {filters.environments.map((env) => (
                          <Badge key={env} variant="neutral">
                            {env}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  {filters.sessionTypes.length > 0 && (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-400">Session Types:</span>
                      <div className="flex flex-wrap gap-1">
                        {filters.sessionTypes.map((type) => (
                          <Badge key={type} variant="neutral">
                            {type}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                  {filters.riskLevels.length > 0 && (
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-gray-400">Risk Levels:</span>
                      <div className="flex flex-wrap gap-1">
                        {filters.riskLevels.map((level) => (
                          <Badge key={level} variant="danger" className="border-danger-500/30">
                            {level}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            {/* Warning for PII */}
            {options.includePii && (
              <div className="flex gap-2 rounded-lg border border-warning-500/30 bg-warning-500/10 p-3">
                <AlertCircle className="h-4 w-4 text-warning-400 mt-0.5 flex-shrink-0" />
                <p className="text-xs text-gray-300">
                  You have chosen to include PII data in this report. Ensure proper handling and
                  distribution of sensitive information.
                </p>
              </div>
            )}

            {/* Info Box */}
            <div className="flex gap-2 rounded-lg border border-primary-500/30 bg-primary-500/10 p-3">
              <Info className="h-4 w-4 text-primary-400 mt-0.5 flex-shrink-0" />
              <p className="text-xs text-gray-300">
                Report generation may take several minutes depending on the selected period and
                data volume. You will be notified when the report is ready.
              </p>
            </div>

            <div className="flex justify-between">
              <Button variant="secondary" onClick={handleBack}>
                Back
              </Button>
              <div className="flex gap-3">
                <Button variant="secondary" onClick={() => navigate('/reports')}>
                  Cancel
                </Button>
                <Button
                  variant="primary"
                  onClick={handleGenerate}
                  disabled={generateMutation.isPending}
                  isLoading={generateMutation.isPending}
                >
                  <Sparkles className="mr-2 h-4 w-4" />
                  Generate Report
                </Button>
              </div>
            </div>
          </div>
        </Card>
      )}
    </div>
  );
};

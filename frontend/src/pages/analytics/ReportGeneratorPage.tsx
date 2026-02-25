/**
 * ReportGeneratorPage - Report generation form with framework selection, date range, and format options
 * Handles both ad-hoc report generation and scheduled report creation
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { format, subDays, subMonths } from 'date-fns';
import {
  ArrowLeft,
  FileText,
  Calendar,
  Settings,
  Play,
  Clock,
  CheckCircle,
  Info,
  Sparkles,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Button } from '@/components/common';
import { Select } from '@/components/common';
import { Input } from '@/components/common';
import { Toggle } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import { ReportScheduleDialog } from '@/components/analytics/ReportScheduleDialog';
import { toast } from 'react-hot-toast';
import type {
  ReportType,
  ReportFormat,
  ReportConfig,
  ReportSchedule,
  ComplianceFramework,
  GenerateReportRequest,
} from '@/types/reports';
import clsx from 'clsx';

const reportTypeOptions: Array<{ value: ReportType; label: string; description: string }> = [
  { value: 'compliance', label: 'Compliance Report', description: 'Generate a framework-specific compliance report' },
  { value: 'session_activity', label: 'Session Activity', description: 'Report on privileged session activity' },
  { value: 'command_analysis', label: 'Command Analysis', description: 'Analyze command execution patterns and risks' },
  { value: 'user_access', label: 'User Access Report', description: 'Summary of user access patterns and permissions' },
  { value: 'anomaly_summary', label: 'Anomaly Summary', description: 'Report detected anomalies and security issues' },
  { value: 'audit_trail', label: 'Audit Trail', description: 'Comprehensive audit log export' },
  { value: 'credential_usage', label: 'Credential Usage', description: 'Report on credential checkout and usage' },
];

const frameworkOptions: Array<{ value: ComplianceFramework; label: string }> = [
  { value: 'soc2', label: 'SOC 2 Type II' },
  { value: 'iso27001', label: 'ISO 27001:2022' },
  { value: 'pci_dss', label: 'PCI DSS 4.0' },
  { value: 'hipaa', label: 'HIPAA Security Rule' },
  { value: 'gdpr', label: 'GDPR Compliance' },
  { value: 'custom', label: 'Custom Framework' },
];

const formatOptions: Array<{ value: ReportFormat; label: string; description: string }> = [
  { value: 'pdf', label: 'PDF', description: 'Formatted document with charts and tables' },
  { value: 'xlsx', label: 'Excel', description: 'Spreadsheet with multiple sheets' },
  { value: 'csv', label: 'CSV', description: 'Comma-separated values for data analysis' },
  { value: 'json', label: 'JSON', description: 'Structured data format' },
  { value: 'html', label: 'HTML', description: 'Interactive web document' },
];

const dateRangePresets: Array<{ label: string; getValue: () => { start: string; end: string } }> = [
  {
    label: 'Last 7 Days',
    getValue: () => ({
      start: format(subDays(new Date(), 7), 'yyyy-MM-dd'),
      end: format(new Date(), 'yyyy-MM-dd'),
    }),
  },
  {
    label: 'Last 30 Days',
    getValue: () => ({
      start: format(subDays(new Date(), 30), 'yyyy-MM-dd'),
      end: format(new Date(), 'yyyy-MM-dd'),
    }),
  },
  {
    label: 'Last 90 Days',
    getValue: () => ({
      start: format(subDays(new Date(), 90), 'yyyy-MM-dd'),
      end: format(new Date(), 'yyyy-MM-dd'),
    }),
  },
  {
    label: 'This Month',
    getValue: () => ({
      start: format(new Date(new Date().getFullYear(), new Date().getMonth(), 1), 'yyyy-MM-dd'),
      end: format(new Date(), 'yyyy-MM-dd'),
    }),
  },
  {
    label: 'Last Month',
    getValue: () => ({
      start: format(new Date(new Date().getFullYear(), new Date().getMonth() - 1, 1), 'yyyy-MM-dd'),
      end: format(new Date(new Date().getFullYear(), new Date().getMonth(), 0), 'yyyy-MM-dd'),
    }),
  },
  {
    label: 'This Quarter',
    getValue: () => {
      const quarter = Math.floor(new Date().getMonth() / 3);
      return {
        start: format(new Date(new Date().getFullYear(), quarter * 3, 1), 'yyyy-MM-dd'),
        end: format(new Date(), 'yyyy-MM-dd'),
      };
    },
  },
  {
    label: 'Last Quarter',
    getValue: () => {
      const quarter = Math.floor(new Date().getMonth() / 3) - 1;
      return {
        start: format(new Date(new Date().getFullYear(), quarter * 3, 1), 'yyyy-MM-dd'),
        end: format(new Date(new Date().getFullYear(), (quarter + 1) * 3, 0), 'yyyy-MM-dd'),
      };
    },
  },
  {
    label: 'This Year (YTD)',
    getValue: () => ({
      start: format(new Date(new Date().getFullYear(), 0, 1), 'yyyy-MM-dd'),
      end: format(new Date(), 'yyyy-MM-dd'),
    }),
  },
];

export const ReportGeneratorPage: React.FC = () => {
  const navigate = useNavigate();

  // State
  const [step, setStep] = useState(1); // 1: Type & Framework, 2: Configuration, 3: Review
  const [reportType, setReportType] = useState<ReportType>('compliance');
  const [framework, setFramework] = useState<ComplianceFramework>('soc2');
  const [reportFormat, setReportFormat] = useState<ReportFormat>('pdf');
  const [periodStart, setPeriodStart] = useState(format(subDays(new Date(), 90), 'yyyy-MM-dd'));
  const [periodEnd, setPeriodEnd] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [reportName, setReportName] = useState('');
  const [includeRawData, setIncludeRawData] = useState(false);
  const [includeCharts, setIncludeCharts] = useState(true);
  const [selectedSections, setSelectedSections] = useState<string[]>([]);
  const [showScheduleDialog, setShowScheduleDialog] = useState(false);
  const [schedule, setSchedule] = useState<ReportSchedule | undefined>();
  const [saveAsTemplate, setSaveAsTemplate] = useState(false);

  // Mutation: Generate report
  const generateMutation = useMutation({
    mutationFn: reportsApi.generate,
    onSuccess: (response) => {
      toast.success('Report generation started successfully');
      navigate(`/analytics/reports/${response.data?.snapshot_id || response.snapshot_id || response.job_id}`);
    },
    onError: (error: Error) => {
      toast.error(`Failed to generate report: ${error.message}`);
    },
  });

  // Validation
  const isValid = () => {
    if (!reportType) return false;
    if (reportType === 'compliance' && !framework) return false;
    if (!periodStart || !periodEnd) return false;
    if (new Date(periodStart) > new Date(periodEnd)) return false;
    if (reportType === 'compliance' && selectedSections.length === 0) return false;
    return true;
  };

  const handleDatePreset = (preset: typeof dateRangePresets[0]) => {
    const { start, end } = preset.getValue();
    setPeriodStart(start);
    setPeriodEnd(end);
  };

  const handleContinue = () => {
    if (step === 1) {
      if (!isValid()) {
        toast.error('Please complete all required fields');
        return;
      }
      setStep(2);
    } else if (step === 2) {
      setStep(3);
    }
  };

  const handleBack = () => {
    if (step > 1) {
      setStep(step - 1);
    } else {
      navigate('/analytics/reports');
    }
  };

  const handleGenerate = () => {
    const config: ReportConfig = {
      period_start: periodStart,
      period_end: periodEnd,
      include_sections: selectedSections.length > 0 ? selectedSections : undefined,
      filters: {},
      format_options: {
        include_charts: includeCharts,
        include_raw_data: includeRawData,
      },
    };

    const data: GenerateReportRequest = {
      type: reportType,
      framework: reportType === 'compliance' ? framework : undefined,
      format: reportFormat,
      period_start: periodStart,
      period_end: periodEnd,
      include_sections: selectedSections.length > 0 ? selectedSections : undefined,
      filters: {},
      format_options: {
        include_charts: includeCharts,
        include_raw_data: includeRawData,
      },
    };

    generateMutation.mutate(data);
  };

  const handleSchedule = (newSchedule: ReportSchedule) => {
    setSchedule(newSchedule);
    setShowScheduleDialog(false);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/analytics/reports')}
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Reports
        </Button>
        <div>
          <h1 className="text-2xl font-bold text-white">Generate Report</h1>
          <p className="mt-1 text-sm text-gray-400">
            {step === 1 && 'Select report type and framework'}
            {step === 2 && 'Configure report parameters'}
            {step === 3 && 'Review and generate'}
          </p>
        </div>
      </div>

      {/* Progress Indicator */}
      <div className="flex items-center gap-2">
        {[
          { num: 1, label: 'Type' },
          { num: 2, label: 'Configure' },
          { num: 3, label: 'Review' },
        ].map((item, idx, arr) => (
          <React.Fragment key={item.num}>
            <button
              onClick={() => step >= item.num && setStep(item.num)}
              disabled={step < item.num}
              className={clsx(
                'flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium transition-colors',
                {
                  'bg-primary-600 text-white': step === item.num,
                  'bg-success-600 text-white': step > item.num,
                  'bg-gray-800 text-gray-400 cursor-not-allowed': step < item.num,
                  'cursor-pointer': step >= item.num,
                }
              )}
            >
              {step > item.num ? <CheckCircle className="h-4 w-4" /> : <span className="flex h-5 w-5 items-center justify-center rounded-full bg-current text-xs">{item.num}</span>}
              {item.label}
            </button>
            {idx < arr.length - 1 && (
              <div className={clsx('h-0.5 w-8', {
                'bg-success-600': step > item.num,
                'bg-gray-700': step <= item.num,
              })} />
            )}
          </React.Fragment>
        ))}
      </div>

      {/* Step 1: Type & Framework */}
      {step === 1 && (
        <div className="space-y-6">
          <Card>
            <CardHeader
              title="Report Type"
              subtitle="Select the type of report you want to generate"
              icon={<FileText className="h-5 w-5" />}
            />
            <div className="card-body">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {reportTypeOptions.map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => setReportType(option.value)}
                    className={clsx(
                      'flex flex-col items-start rounded-lg border p-4 text-left transition-colors',
                      reportType === option.value
                        ? 'border-primary-500 bg-primary-500/10'
                        : 'border-gray-800 hover:bg-gray-800/50'
                    )}
                  >
                    <div className="flex w-full items-center justify-between">
                      <span className="font-medium text-white">{option.label}</span>
                      <input
                        type="radio"
                        checked={reportType === option.value}
                        onChange={() => setReportType(option.value)}
                        className="h-4 w-4 border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                      />
                    </div>
                    <p className="mt-2 text-sm text-gray-400">{option.description}</p>
                  </button>
                ))}
              </div>
            </div>
          </Card>

          {/* Framework Selection (for compliance reports) */}
          {reportType === 'compliance' && (
            <Card>
              <CardHeader
                title="Compliance Framework"
                subtitle="Select the compliance framework for this report"
                icon={<Settings className="h-5 w-5" />}
              />
              <div className="card-body">
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {frameworkOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => setFramework(option.value)}
                      className={clsx(
                        'flex items-center justify-between rounded-lg border p-4 text-left transition-colors',
                        framework === option.value
                          ? 'border-primary-500 bg-primary-500/10'
                          : 'border-gray-800 hover:bg-gray-800/50'
                      )}
                    >
                      <span className="font-medium text-white">{option.label}</span>
                      <input
                        type="radio"
                        checked={framework === option.value}
                        onChange={() => setFramework(option.value)}
                        className="h-4 w-4 border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                      />
                    </button>
                  ))}
                </div>
              </div>
            </Card>
          )}
        </div>
      )}

      {/* Step 2: Configuration */}
      {step === 2 && (
        <div className="space-y-6">
          {/* Date Range */}
          <Card>
            <CardHeader
              title="Report Period"
              subtitle="Select the date range for the report"
              icon={<Calendar className="h-5 w-5" />}
            />
            <div className="card-body space-y-4">
              {/* Quick Presets */}
              <div>
                <label className="mb-2 block text-sm font-medium text-gray-300">Quick Select</label>
                <div className="flex flex-wrap gap-2">
                  {dateRangePresets.map((preset) => (
                    <Button
                      key={preset.label}
                      variant="secondary"
                      size="sm"
                      onClick={() => handleDatePreset(preset)}
                    >
                      {preset.label}
                    </Button>
                  ))}
                </div>
              </div>

              {/* Custom Date Range */}
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Start Date</label>
                  <Input
                    type="date"
                    value={periodStart}
                    onChange={(e) => setPeriodStart(e.target.value)}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">End Date</label>
                  <Input
                    type="date"
                    value={periodEnd}
                    onChange={(e) => setPeriodEnd(e.target.value)}
                  />
                </div>
              </div>

              {periodStart && periodEnd && new Date(periodStart) > new Date(periodEnd) && (
                <div className="rounded-lg bg-danger-500/10 p-3 text-sm text-danger-400">
                  <Info className="mr-1 inline h-4 w-4" />
                  Start date must be before end date
                </div>
              )}
            </div>
          </Card>

          {/* Format Options */}
          <Card>
            <CardHeader
              title="Format Options"
              subtitle="Choose the output format and additional options"
              icon={<Settings className="h-5 w-5" />}
            />
            <div className="card-body space-y-4">
              <div>
                <label className="mb-2 block text-sm font-medium text-gray-300">Output Format</label>
                <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {formatOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => setReportFormat(option.value)}
                      className={clsx(
                        'flex flex-col rounded-lg border p-4 text-left transition-colors',
                        reportFormat === option.value
                          ? 'border-primary-500 bg-primary-500/10'
                          : 'border-gray-800 hover:bg-gray-800/50'
                      )}
                    >
                      <div className="flex items-center justify-between">
                        <span className="font-medium text-white">{option.label}</span>
                        <input
                          type="radio"
                          checked={reportFormat === option.value}
                          onChange={() => setReportFormat(option.value)}
                          className="h-4 w-4 border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                        />
                      </div>
                      <p className="mt-1 text-xs text-gray-400">{option.description}</p>
                    </button>
                  ))}
                </div>
              </div>

              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-white">Include Charts</p>
                    <p className="text-xs text-gray-400">Add visual charts and graphs (PDF/HTML only)</p>
                  </div>
                  <Toggle checked={includeCharts} onChange={setIncludeCharts} />
                </div>
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm font-medium text-white">Include Raw Data</p>
                    <p className="text-xs text-gray-400">Attach raw data tables</p>
                  </div>
                  <Toggle checked={includeRawData} onChange={setIncludeRawData} />
                </div>
              </div>
            </div>
          </Card>

          {/* Report Sections (for compliance reports) */}
          {reportType === 'compliance' && (
            <Card>
              <CardHeader
                title="Report Sections"
                subtitle="Select which sections to include in the report"
                icon={<FileText className="h-5 w-5" />}
              />
              <div className="card-body">
                <div className="space-y-2">
                  {[
                    { id: 'summary', label: 'Executive Summary', description: 'High-level overview and key findings' },
                    { id: 'controls', label: 'Control Assessment', description: 'Detailed control compliance status' },
                    { id: 'findings', label: 'Findings and Recommendations', description: 'Issues found and remediation steps' },
                    { id: 'metrics', label: 'Metrics and Statistics', description: 'Quantitative compliance metrics' },
                    { id: 'exceptions', label: 'Compliance Exceptions', description: 'Active exception requests and status' },
                    { id: 'evidence', label: 'Evidence Documentation', description: 'Supporting evidence and artifacts' },
                  ].map((section) => (
                    <label
                      key={section.id}
                      className={clsx(
                        'flex cursor-pointer items-start gap-3 rounded-lg border p-4 transition-colors',
                        selectedSections.includes(section.id)
                          ? 'border-primary-500 bg-primary-500/10'
                          : 'border-gray-800 hover:bg-gray-800/50'
                      )}
                    >
                      <input
                        type="checkbox"
                        checked={selectedSections.includes(section.id)}
                        onChange={() => {
                          setSelectedSections((prev) =>
                            prev.includes(section.id)
                              ? prev.filter((id) => id !== section.id)
                              : [...prev, section.id]
                          );
                        }}
                        className="mt-1 h-4 w-4 border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                      />
                      <div>
                        <p className="text-sm font-medium text-white">{section.label}</p>
                        <p className="text-xs text-gray-400">{section.description}</p>
                      </div>
                    </label>
                  ))}
                </div>
              </div>
            </Card>
          )}
        </div>
      )}

      {/* Step 3: Review */}
      {step === 3 && (
        <div className="space-y-6">
          <Card>
            <CardHeader
              title="Review Report Configuration"
              subtitle="Verify all settings before generating"
              icon={<CheckCircle className="h-5 w-5" />}
            />
            <div className="card-body space-y-6">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div>
                  <p className="text-xs text-gray-500">Report Type</p>
                  <p className="text-sm font-medium text-white">
                    {reportTypeOptions.find((t) => t.value === reportType)?.label}
                  </p>
                </div>
                {reportType === 'compliance' && (
                  <div>
                    <p className="text-xs text-gray-500">Framework</p>
                    <p className="text-sm font-medium text-white">
                      {frameworkOptions.find((f) => f.value === framework)?.label}
                    </p>
                  </div>
                )}
                <div>
                  <p className="text-xs text-gray-500">Format</p>
                  <p className="text-sm font-medium text-white">
                    {formatOptions.find((f) => f.value === reportFormat)?.label}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-gray-500">Period</p>
                  <p className="text-sm font-medium text-white">
                    {format(new Date(periodStart), 'MMM d, yyyy')} - {format(new Date(periodEnd), 'MMM d, yyyy')}
                  </p>
                </div>
              </div>

              {selectedSections.length > 0 && (
                <div>
                  <p className="mb-2 text-xs text-gray-500">Included Sections</p>
                  <div className="flex flex-wrap gap-2">
                    {selectedSections.map((section) => (
                      <Badge key={section} variant="neutral" className="capitalize">
                        {section.replace('_', ' ')}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              <div className="rounded-lg bg-gray-900/50 p-4">
                <div className="flex items-center gap-2">
                  <Info className="h-4 w-4 text-primary-400" />
                  <p className="text-sm text-gray-400">
                    The report will be generated asynchronously. You will be redirected to the reports page
                    where you can monitor progress and download the completed report.
                  </p>
                </div>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Footer Actions */}
      <Card>
        <CardFooter className="flex items-center justify-between">
          <Button
            variant="secondary"
            onClick={handleBack}
            disabled={generateMutation.isPending}
          >
            {step === 1 ? 'Cancel' : 'Back'}
          </Button>

          <div className="flex items-center gap-3">
            {step === 2 && (
              <Button
                variant="secondary"
                leftIcon={<Clock className="h-4 w-4" />}
                onClick={() => setShowScheduleDialog(true)}
              >
                Schedule Report
              </Button>
            )}
            {step === 3 ? (
              <Button
                variant="primary"
                leftIcon={<Play className="h-4 w-4" />}
                onClick={handleGenerate}
                isLoading={generateMutation.isPending}
              >
                Generate Report
              </Button>
            ) : (
              <Button
                variant="primary"
                onClick={handleContinue}
              >
                Continue
              </Button>
            )}
          </div>
        </CardFooter>
      </Card>

      {/* Schedule Dialog */}
      <ReportScheduleDialog
        isOpen={showScheduleDialog}
        onClose={() => setShowScheduleDialog(false)}
        onSchedule={handleSchedule}
        reportType={reportType}
        framework={reportType === 'compliance' ? framework : undefined}
      />
    </div>
  );
};

export default ReportGeneratorPage;

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
import { ComplianceReportTemplate } from '@/components/analytics/ComplianceReportTemplate';
import { useReports } from '@/contexts/ReportContext';
import { toast } from 'react-hot-toast';
import type { ReportType, ReportFormat, ReportConfig, ReportSchedule, ComplianceFramework } from '@/types/reports';
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
  const { templates } = useReports();

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
      navigate(`/analytics/reports/${response.data.snapshot_id}`);
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

    generateMutation.mutate({
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
    });
  };

  const handleSchedule = (newSchedule: ReportSchedule) => {
    setSchedule(newSchedule);
    setShowScheduleDialog(false);
  };

  const getConfig = (): ReportConfig => ({
    period_start: periodStart,
    period_end: periodEnd,
    include_sections: selectedSections,
    filters: {},
    format_options: {
      include_charts: includeCharts,
      include_raw_data: includeRawData,
    },
  });

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={handleBack}
          leftIcon={<ArrowLeft className="h-4 w-4" />}
        >
          Back
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
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">Start Date</label>
                  <Input
                    type="date"
                    value={periodStart}
                    onChange={(e) => setPeriodStart(e.target.value)}
                    max={periodEnd}
                  />
                </div>
                <div>
                  <label className="mb-1 block text-sm font-medium text-gray-300">End Date</label>
                  <Input
                    type="date"
                    value={periodEnd}
                    onChange={(e) => setPeriodEnd(e.target.value)}
                    min={periodStart}
                  />
                </div>
              </div>

              {periodStart && periodEnd && new Date(periodStart) > new Date(periodEnd) && (
                <p className="text-sm text-danger-400">Start date must be before end date</p>
              )}
            </div>
          </Card>

          {/* Format Options */}
          <Card>
            <CardHeader
              title="Output Format"
              subtitle="Select the format for the generated report"
            />
            <div className="card-body">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {formatOptions.map((option) => (
                  <button
                    key={option.value}
                    type="button"
                    onClick={() => setReportFormat(option.value)}
                    className={clsx(
                      'flex flex-col items-start rounded-lg border p-4 text-left transition-colors',
                      reportFormat === option.value
                        ? 'border-primary-500 bg-primary-500/10'
                        : 'border-gray-800 hover:bg-gray-800/50'
                    )}
                  >
                    <div className="flex w-full items-center justify-between">
                      <div>
                        <span className="font-medium text-white">{option.label}</span>
                        <p className="mt-1 text-sm text-gray-400">{option.description}</p>
                      </div>
                      <input
                        type="radio"
                        checked={reportFormat === option.value}
                        onChange={() => setReportFormat(option.value)}
                        className="h-4 w-4 border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                      />
                    </div>
                  </button>
                ))}
              </div>
            </div>
          </Card>

          {/* Additional Options */}
          <Card>
            <CardHeader title="Additional Options" />
            <div className="card-body space-y-4">
              <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
                <div>
                  <p className="font-medium text-white">Include Charts</p>
                  <p className="mt-1 text-sm text-gray-400">Add visual charts and graphs to the report</p>
                </div>
                <Toggle checked={includeCharts} onChange={setIncludeCharts} />
              </div>
              <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
                <div>
                  <p className="font-medium text-white">Include Raw Data</p>
                  <p className="mt-1 text-sm text-gray-400">Append raw data tables to the report</p>
                </div>
                <Toggle checked={includeRawData} onChange={setIncludeRawData} />
              </div>
            </div>
          </Card>

          {/* Report Sections (for compliance reports) */}
          {reportType === 'compliance' && (
            <ComplianceReportTemplate
              framework={framework}
              config={getConfig()}
              onConfigChange={(config) => setSelectedSections(config.include_sections)}
            />
          )}
        </div>
      )}

      {/* Step 3: Review */}
      {step === 3 && (
        <div className="space-y-6">
          <Card>
            <CardHeader title="Review Report Configuration" />
            <div className="card-body space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <div>
                  <p className="text-sm text-gray-400">Report Type</p>
                  <p className="text-white capitalize">{reportType.replace('_', ' ')}</p>
                </div>
                {framework && (
                  <div>
                    <p className="text-sm text-gray-400">Framework</p>
                    <p className="text-white uppercase">{framework}</p>
                  </div>
                )}
                <div>
                  <p className="text-sm text-gray-400">Format</p>
                  <p className="text-white uppercase">{reportFormat}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Period</p>
                  <p className="text-white">
                    {format(new Date(periodStart), 'MMM d, yyyy')} - {format(new Date(periodEnd), 'MMM d, yyyy')}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Include Charts</p>
                  <p className="text-white">{includeCharts ? 'Yes' : 'No'}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-400">Include Raw Data</p>
                  <p className="text-white">{includeRawData ? 'Yes' : 'No'}</p>
                </div>
              </div>

              {selectedSections.length > 0 && (
                <div>
                  <p className="text-sm text-gray-400">Sections</p>
                  <div className="mt-2 flex flex-wrap gap-2">
                    {selectedSections.map((section) => (
                      <Badge key={section} variant="neutral">
                        {section.replace(/_/g, ' ').replace(/\b\w/g, (l) => l.toUpperCase())}
                      </Badge>
                    ))}
                  </div>
                </div>
              )}

              {schedule && (
                <div>
                  <p className="text-sm text-gray-400">Schedule</p>
                  <div className="mt-2 flex items-center gap-2">
                    <Badge variant="success">
                      <Clock className="mr-1 h-3 w-3" />
                      {schedule.frequency}
                    </Badge>
                    {schedule.time && <span className="text-white">at {schedule.time}</span>}
                    {schedule.timezone && <span className="text-gray-400">{schedule.timezone}</span>}
                  </div>
                </div>
              )}
            </div>
          </Card>

          {/* Schedule Option */}
          <Card>
            <div className="card-body">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="font-medium text-white">Schedule Recurring Report</h3>
                  <p className="mt-1 text-sm text-gray-400">
                    Set up automatic generation of this report on a schedule
                  </p>
                </div>
                <Button
                  variant="secondary"
                  onClick={() => setShowScheduleDialog(true)}
                  leftIcon={<Clock className="h-4 w-4" />}
                >
                  {schedule ? 'Edit Schedule' : 'Add Schedule'}
                </Button>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Actions */}
      <Card>
        <CardFooter className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            {schedule && (
              <Badge variant="success">
                <Clock className="mr-1 h-3 w-3" />
                Scheduled: {schedule.frequency}
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-3">
            <Button variant="secondary" onClick={handleBack}>
              {step === 1 ? 'Cancel' : 'Back'}
            </Button>
            {step < 3 ? (
              <Button variant="primary" onClick={handleContinue} disabled={!isValid()}>
                Continue
              </Button>
            ) : (
              <Button
                variant="primary"
                onClick={handleGenerate}
                isLoading={generateMutation.isPending}
                disabled={!isValid()}
                leftIcon={<Play className="h-4 w-4" />}
              >
                Generate Report
              </Button>
            )}
          </div>
        </CardFooter>
      </Card>

      {/* Schedule Dialog */}
      <ReportScheduleDialog
        isOpen={showScheduleDialog}
        onClose={() => setShowScheduleDialog(false)}
        onSave={handleSchedule}
        existingSchedule={schedule}
        reportName={reportName || `${reportType} Report`}
      />
    </div>
  );
};

export default ReportGeneratorPage;

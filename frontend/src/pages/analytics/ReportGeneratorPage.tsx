/**
 * ReportGeneratorPage - Report generation form with framework selection, date range, and format options
 * Handles both ad-hoc report generation and scheduled report creation
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useMutation } from '@tanstack/react-query';
import { format, subDays } from 'date-fns';
import {
  ArrowLeft,
  FileText,
  Calendar,
  Settings,
  CheckCircle,
  Sparkles,
  Info,
  AlertCircle,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card, Badge, Button } from '@/components/common';
import { Input, Select, Textarea, Label } from '@/components/common';
import { LoadingState } from '@/components/common';
import type {
  ReportType,
  ReportFormat,
  GenerateReportRequest,
} from '@/types/reports';
import type { ComplianceFramework } from '@/types';
import toast from 'react-hot-toast';
import clsx from 'clsx';

const reportTypes: { value: ReportType; label: string; description: string }[] = [
  { value: 'compliance', label: 'Compliance Report', description: 'Full compliance assessment with controls and findings' },
  { value: 'session_activity', label: 'Session Activity', description: 'User session activity and metrics' },
  { value: 'command_analysis', label: 'Command Analysis', description: 'Command execution analysis and risk assessment' },
  { value: 'user_access', label: 'User Access', description: 'User access patterns and privileges' },
  { value: 'anomaly_summary', label: 'Anomaly Summary', description: 'Security anomalies and detected threats' },
  { value: 'audit_trail', label: 'Audit Trail', description: 'Comprehensive audit log report' },
  { value: 'credential_usage', label: 'Credential Usage', description: 'Credential checkout and usage patterns' },
];

const formats: { value: ReportFormat; label: string; extension: string }[] = [
  { value: 'pdf', label: 'PDF Document', extension: '.pdf' },
  { value: 'html', label: 'HTML Report', extension: '.html' },
  { value: 'xlsx', label: 'Excel Spreadsheet', extension: '.xlsx' },
  { value: 'csv', label: 'CSV Data', extension: '.csv' },
  { value: 'json', label: 'JSON Data', extension: '.json' },
];

const periodPresets = [
  { label: 'Last 7 days', days: 7 },
  { label: 'Last 30 days', days: 30 },
  { label: 'Last 90 days', days: 90 },
  { label: 'This month', days: 'month' },
  { label: 'Last 3 months', days: 90 },
  { label: 'Last 12 months', days: 365 },
];

const sectionOptions = [
  { id: 'summary', label: 'Executive Summary', description: 'High-level overview and key metrics' },
  { id: 'controls', label: 'Control Assessment', description: 'Detailed control status and evidence' },
  { id: 'findings', label: 'Findings & Recommendations', description: 'Identified issues and remediation steps' },
  { id: 'metrics', label: 'Metrics & Statistics', description: 'Detailed quantitative analysis' },
  { id: 'evidence', label: 'Evidence Attachments', description: 'Supporting documentation and screenshots' },
];

export const ReportGeneratorPage: React.FC = () => {
  const navigate = useNavigate();
  const [reportName, setReportName] = useState('');
  const [reportType, setReportType] = useState<ReportType>('compliance');
  const [framework, setFramework] = useState<ComplianceFramework>('soc2');
  const [reportFormat, setReportFormat] = useState<ReportFormat>('pdf');
  const [description, setDescription] = useState('');
  const [periodPreset, setPeriodPreset] = useState<string>('30');
  const [startDate, setStartDate] = useState(format(subDays(new Date(), 30), 'yyyy-MM-dd'));
  const [endDate, setEndDate] = useState(format(new Date(), 'yyyy-MM-dd'));
  const [selectedSections, setSelectedSections] = useState<string[]>(['summary', 'controls']);

  const generateMutation = useMutation({
    mutationFn: (data: GenerateReportRequest) => reportsApi.generate(data),
    onSuccess: (response) => {
      toast.success(`Report generation started (Job ID: ${response.job_id})`);
      navigate('/reports');
    },
    onError: (error: Error) => {
      toast.error(`Failed to generate report: ${error.message}`);
    },
  });

  const handlePresetChange = (preset: string) => {
    setPeriodPreset(preset);
    const end = new Date();
    setEndDate(format(end, 'yyyy-MM-dd'));

    if (preset === 'month') {
      const start = new Date(end.getFullYear(), end.getMonth(), 1);
      setStartDate(format(start, 'yyyy-MM-dd'));
    } else {
      const days = parseInt(preset, 10);
      const start = subDays(end, days);
      setStartDate(format(start, 'yyyy-MM-dd'));
    }
  };

  const toggleSection = (sectionId: string) => {
    setSelectedSections((prev) =>
      prev.includes(sectionId)
        ? prev.filter((id) => id !== sectionId)
        : [...prev, sectionId]
    );
  };

  const handleGenerate = () => {
    if (!reportName.trim()) {
      toast.error('Please enter a report name');
      return;
    }

    if (!startDate || !endDate) {
      toast.error('Please select a valid date range');
      return;
    }

    const data: GenerateReportRequest = {
      type: reportType,
      framework,
      format: reportFormat,
      period_start: startDate,
      period_end: endDate,
      include_sections: selectedSections,
      filters: {
        environments: [],
        risk_levels: [],
      },
    };

    generateMutation.mutate(data);
  };

  const selectedFormat = formats.find((f) => f.value === reportFormat);
  const selectedType = reportTypes.find((t) => t.value === reportType);

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/reports')}
          className="flex items-center gap-2"
        >
          <ArrowLeft className="h-4 w-4" />
          Back to Reports
        </Button>
      </div>

      <div>
        <h1 className="text-2xl font-bold text-white">Generate Report</h1>
        <p className="mt-1 text-sm text-gray-400">
          Create a new compliance or analytics report with custom parameters
        </p>
      </div>

      {/* Report Configuration */}
      <Card>
        <div className="card-body space-y-6">
          {/* Report Name */}
          <div>
            <Label htmlFor="reportName">Report Name</Label>
            <Input
              id="reportName"
              placeholder="e.g., Q4 2024 SOC 2 Compliance Report"
              value={reportName}
              onChange={(e) => setReportName(e.target.value)}
              className="mt-1"
            />
          </div>

          {/* Report Type */}
          <div>
            <Label>Report Type</Label>
            <p className="mt-1 text-xs text-gray-400">Select the type of report you want to generate</p>
            <div className="mt-3 grid gap-3 sm:grid-cols-2">
              {reportTypes.map((type) => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => setReportType(type.value)}
                  className={clsx(
                    'flex items-start gap-3 rounded-lg border p-4 text-left transition-colors',
                    reportType === type.value
                      ? 'border-primary-500 bg-primary-500/10'
                      : 'border-gray-700 hover:bg-gray-800'
                  )}
                >
                  <div className={clsx(
                    'mt-1 rounded-lg p-2',
                    reportType === type.value ? 'bg-primary-500/20 text-primary-400' : 'bg-gray-800 text-gray-400'
                  )}>
                    <FileText className="h-4 w-4" />
                  </div>
                  <div className="flex-1">
                    <h4 className="font-medium text-white">{type.label}</h4>
                    <p className="mt-1 text-xs text-gray-400">{type.description}</p>
                  </div>
                  {reportType === type.value && (
                    <CheckCircle className="h-5 w-5 text-primary-400" />
                  )}
                </button>
              ))}
            </div>
          </div>

          {/* Framework (for compliance reports) */}
          {reportType === 'compliance' && (
            <div>
              <Label>Compliance Framework</Label>
              <div className="mt-3 grid gap-2 sm:grid-cols-3">
                {[
                  { value: 'soc2' as ComplianceFramework, label: 'SOC 2' },
                  { value: 'iso27001' as ComplianceFramework, label: 'ISO 27001' },
                  { value: 'pci_dss' as ComplianceFramework, label: 'PCI DSS' },
                  { value: 'hipaa' as ComplianceFramework, label: 'HIPAA' },
                  { value: 'gdpr' as ComplianceFramework, label: 'GDPR' },
                  { value: 'nerc_cip' as ComplianceFramework, label: 'NERC CIP' },
                  { value: 'custom' as ComplianceFramework, label: 'Custom' },
                ].map((fw) => (
                  <button
                    key={fw.value}
                    type="button"
                    onClick={() => setFramework(fw.value)}
                    className={clsx(
                      'rounded-lg border px-4 py-3 text-left transition-colors',
                      framework === fw.value
                        ? 'border-primary-500 bg-primary-500/10'
                        : 'border-gray-700 hover:bg-gray-800'
                    )}
                  >
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-white">{fw.label}</span>
                      {framework === fw.value && <CheckCircle className="h-4 w-4 text-primary-400" />}
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}

          {/* Format Selection */}
          <div>
            <Label>Output Format</Label>
            <div className="mt-3 flex flex-wrap gap-3">
              {formats.map((fmt) => (
                <button
                  key={fmt.value}
                  type="button"
                  onClick={() => setReportFormat(fmt.value)}
                  className={clsx(
                    'flex items-center gap-2 rounded-lg border px-4 py-2 transition-colors',
                    reportFormat === fmt.value
                      ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                      : 'border-gray-700 text-gray-300 hover:bg-gray-800'
                  )}
                >
                  <span className="font-medium">{fmt.label}</span>
                  <span className="text-xs text-gray-500">({fmt.extension})</span>
                </button>
              ))}
            </div>
          </div>

          {/* Date Range */}
          <div>
            <Label>Report Period</Label>
            <p className="mt-1 text-xs text-gray-400">Select the time period for the report data</p>

            <div className="mt-3 flex flex-wrap gap-2">
              {periodPresets.map((preset) => (
                <button
                  key={preset.label}
                  type="button"
                  onClick={() => handlePresetChange(String(preset.days))}
                  className={clsx(
                    'rounded-lg border px-3 py-1.5 text-sm transition-colors',
                    periodPreset === String(preset.days)
                      ? 'border-primary-500 bg-primary-500/10 text-primary-400'
                      : 'border-gray-700 text-gray-300 hover:bg-gray-800'
                  )}
                >
                  {preset.label}
                </button>
              ))}
            </div>

            <div className="mt-4 grid grid-cols-2 gap-4">
              <div>
                <Label htmlFor="startDate">Start Date</Label>
                <Input
                  id="startDate"
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <Label htmlFor="endDate">End Date</Label>
                <Input
                  id="endDate"
                  type="date"
                  value={endDate}
                  onChange={(e) => setEndDate(e.target.value)}
                  className="mt-1"
                />
              </div>
            </div>
          </div>

          {/* Include Sections */}
          <div>
            <Label>Report Sections</Label>
            <p className="mt-1 text-xs text-gray-400">Select which sections to include in the report</p>
            <div className="mt-3 space-y-2">
              {sectionOptions.map((section) => (
                <label
                  key={section.id}
                  className="flex cursor-pointer items-start gap-3 rounded-lg border border-gray-700 p-3 transition-colors hover:bg-gray-800"
                >
                  <input
                    type="checkbox"
                    checked={selectedSections.includes(section.id)}
                    onChange={() => toggleSection(section.id)}
                    className="mt-1 h-4 w-4 rounded border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                  />
                  <div>
                    <h4 className="font-medium text-white">{section.label}</h4>
                    <p className="text-xs text-gray-400">{section.description}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>

          {/* Description (Optional) */}
          <div>
            <Label htmlFor="description">Description (Optional)</Label>
            <Textarea
              id="description"
              placeholder="Add a description for this report..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              className="mt-1"
            />
          </div>
        </div>
      </Card>

      {/* Summary */}
      <Card className="border-primary-500/30 bg-primary-500/5">
        <div className="card-body">
          <div className="flex items-start gap-3">
            <Sparkles className="h-5 w-5 text-primary-400 mt-0.5" />
            <div className="flex-1">
              <h3 className="font-medium text-white">Report Summary</h3>
              <div className="mt-2 grid gap-2 text-sm sm:grid-cols-2">
                <div className="flex items-center gap-2">
                  <span className="text-gray-400">Type:</span>
                  <span className="font-medium text-white">{selectedType?.label}</span>
                </div>
                {framework && (
                  <div className="flex items-center gap-2">
                    <span className="text-gray-400">Framework:</span>
                    <span className="font-medium text-white uppercase">{framework}</span>
                  </div>
                )}
                <div className="flex items-center gap-2">
                  <span className="text-gray-400">Format:</span>
                  <span className="font-medium text-white">{selectedFormat?.label}</span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-gray-400">Period:</span>
                  <span className="font-medium text-white">
                    {format(new Date(startDate), 'MMM d')} - {format(new Date(endDate), 'MMM d, yyyy')}
                  </span>
                </div>
              </div>
              <div className="mt-2 flex flex-wrap gap-1">
                {selectedSections.map((section) => {
                  const sectionDef = sectionOptions.find((s) => s.id === section);
                  return (
                    <Badge key={section} variant="neutral" className="text-xs">
                      {sectionDef?.label}
                    </Badge>
                  );
                })}
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-between">
        <Button
          variant="secondary"
          onClick={() => navigate('/reports')}
        >
          Cancel
        </Button>
        <Button
          variant="primary"
          onClick={handleGenerate}
          isLoading={generateMutation.isPending}
          className="flex items-center gap-2"
        >
          <FileText className="h-4 w-4" />
          Generate Report
        </Button>
      </div>
    </div>
  );
};

export default ReportGeneratorPage;

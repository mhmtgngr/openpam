<<<<<<< HEAD
import React, { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Calendar, Clock, Mail, Save, X, Globe, Info } from 'lucide-react';
import { Modal } from '@/components/common';
import { Input, Textarea, Label, Toggle, Button } from '@/components/common';
import { reportSchedulesApi } from '@/api/reports';
import type {
  ReportSchedule,
  CreateReportScheduleData,
  ScheduleFrequency,
  ComplianceFramework,
  DistributionMethod,
  ReportConfig,
  DistributionConfig,
} from '@/types/reports';
import { format } from 'date-fns';
import toast from 'react-hot-toast';
=======
/**
 * ReportScheduleDialog - Dialog for scheduling recurring report generation
 * Supports one-time, daily, weekly, monthly, and quarterly schedules
 */

import React, { useState, useEffect } from 'react';
import { X, Calendar, Clock, Info } from 'lucide-react';
import { Modal } from '@/components/common';
import { Button } from '@/components/common';
import { Select } from '@/components/common';
import { Input } from '@/components/common';
import { Toggle } from '@/components/common';
import { Badge } from '@/components/common';
import clsx from 'clsx';
import type { ReportSchedule, ReportScheduleFrequency } from '@/types/reports';
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192

interface ReportScheduleDialogProps {
  isOpen: boolean;
  onClose: () => void;
<<<<<<< HEAD
  schedule?: ReportSchedule;
  onSuccess?: () => void;
  framework?: ComplianceFramework;
  defaultPeriod?: { start: string; end: string };
}

interface FormData {
  name: string;
  description: string;
  framework: ComplianceFramework;
  frequency: ScheduleFrequency;
  timezone: string;
  isActive: boolean;
  recipients: string;
  distributionEmail: boolean;
  distributionWebhook: boolean;
  webhookUrl: string;
  reportFormat: string;
}

const commonTimezones = [
  'UTC',
  'America/New_York',
  'America/Chicago',
  'America/Denver',
  'America/Los_Angeles',
  'Europe/London',
  'Europe/Paris',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Shanghai',
  'Australia/Sydney',
];

const frameworks: { value: ComplianceFramework; label: string; description: string }[] = [
  { value: 'soc2', label: 'SOC 2', description: 'Service Organization Control 2' },
  { value: 'iso27001', label: 'ISO 27001', description: 'Information Security Management' },
  { value: 'pci_dss', label: 'PCI DSS', description: 'Payment Card Industry Data Security' },
  { value: 'hipaa', label: 'HIPAA', description: 'Health Insurance Portability' },
  { value: 'gdpr', label: 'GDPR', description: 'General Data Protection Regulation' },
  { value: 'nerc_cip', label: 'NERC CIP', description: 'Critical Infrastructure Protection' },
  { value: 'custom', label: 'Custom', description: 'Custom compliance framework' },
];

const frequencies: { value: ScheduleFrequency; label: string; description: string }[] = [
  { value: 'once', label: 'One Time', description: 'Run once at specified time' },
  { value: 'daily', label: 'Daily', description: 'Run every day at specified time' },
  { value: 'weekly', label: 'Weekly', description: 'Run every week on specified day' },
  { value: 'monthly', label: 'Monthly', description: 'Run every month on specified date' },
  { value: 'quarterly', label: 'Quarterly', description: 'Run every quarter' },
  { value: 'yearly', label: 'Yearly', description: 'Run every year' },
];

const reportFormats = [
  { value: 'pdf', label: 'PDF' },
  { value: 'html', label: 'HTML' },
  { value: 'csv', label: 'CSV' },
  { value: 'xlsx', label: 'Excel' },
  { value: 'json', label: 'JSON' },
];

const reportSections = [
  { id: 'summary', name: 'Executive Summary', description: 'High-level overview and scores' },
  { id: 'controls', name: 'Control Assessment', description: 'Detailed control compliance status' },
  { id: 'findings', name: 'Findings and Recommendations', description: 'Issues and remediation steps' },
  { id: 'metrics', name: 'Metrics and Statistics', description: 'Quantitative measurements' },
  { id: 'evidence', name: 'Evidence Documentation', description: 'Supporting evidence and logs' },
  { id: 'exceptions', name: 'Compliance Exceptions', description: 'Active and past exceptions' },
=======
  onSave: (schedule: ReportSchedule) => void;
  existingSchedule?: ReportSchedule;
  reportName?: string;
}

const frequencyOptions: Array<{ value: ReportScheduleFrequency; label: string; description: string }> = [
  { value: 'once', label: 'One-time', description: 'Generate the report once immediately' },
  { value: 'daily', label: 'Daily', description: 'Generate every day at a specified time' },
  { value: 'weekly', label: 'Weekly', description: 'Generate on a specific day of the week' },
  { value: 'monthly', label: 'Monthly', description: 'Generate on a specific day of the month' },
  { value: 'quarterly', label: 'Quarterly', description: 'Generate at the end of each quarter' },
];

const daysOfWeek = [
  { value: 0, label: 'Sunday' },
  { value: 1, label: 'Monday' },
  { value: 2, label: 'Tuesday' },
  { value: 3, label: 'Wednesday' },
  { value: 4, label: 'Thursday' },
  { value: 5, label: 'Friday' },
  { value: 6, label: 'Saturday' },
];

const timezones = [
  { value: 'UTC', label: 'UTC (Coordinated Universal Time)' },
  { value: 'America/New_York', label: 'Eastern Time (ET)' },
  { value: 'America/Chicago', label: 'Central Time (CT)' },
  { value: 'America/Denver', label: 'Mountain Time (MT)' },
  { value: 'America/Los_Angeles', label: 'Pacific Time (PT)' },
  { value: 'Europe/London', label: 'London (GMT/BST)' },
  { value: 'Europe/Paris', label: 'Central European (CET)' },
  { value: 'Asia/Tokyo', label: 'Japan (JST)' },
  { value: 'Australia/Sydney', label: 'Sydney (AEST)' },
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
];

export const ReportScheduleDialog: React.FC<ReportScheduleDialogProps> = ({
  isOpen,
  onClose,
<<<<<<< HEAD
  schedule,
  onSuccess,
  framework: defaultFramework,
  defaultPeriod,
}) => {
  const queryClient = useQueryQueryClient();
  const [formData, setFormData] = useState<FormData>({
    name: '',
    description: '',
    framework: defaultFramework || 'soc2',
    frequency: 'monthly',
    timezone: 'UTC',
    isActive: true,
    recipients: '',
    distributionEmail: true,
    distributionWebhook: false,
    webhookUrl: '',
    reportFormat: 'pdf',
  });

  const [scheduleTime, setScheduleTime] = useState('00:00');
  const [scheduleDay, setScheduleDay] = useState('1');
  const [scheduleWeekday, setScheduleWeekday] = useState('monday');
  const [selectedSections, setSelectedSections] = useState<string[]>([
    'summary',
    'controls',
    'findings',
  ]);

  const isEditing = !!schedule;

  useEffect(() => {
    if (schedule) {
      setFormData({
        name: schedule.name,
        description: schedule.description || '',
        framework: schedule.framework,
        frequency: schedule.frequency,
        timezone: schedule.timezone,
        isActive: schedule.is_active,
        recipients: schedule.recipients.join(', '),
        distributionEmail: schedule.distribution_config.methods.some((m) => m.type === 'email' && m.enabled),
        distributionWebhook: schedule.distribution_config.methods.some((m) => m.type === 'webhook' && m.enabled),
        webhookUrl: schedule.distribution_config.methods.find((m) => m.type === 'webhook')?.config?.url as string || '',
        reportFormat: schedule.report_config.format || 'pdf',
      });
    } else if (defaultFramework) {
      setFormData((prev) => ({ ...prev, framework: defaultFramework }));
    }
  }, [schedule, defaultFramework]);

  const createMutation = useMutation({
    mutationFn: (data: CreateReportScheduleData) => reportSchedulesApi.create(data),
    onSuccess: () => {
      toast.success(isEditing ? 'Schedule updated successfully' : 'Schedule created successfully');
      queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to save schedule');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateReportScheduleData }) =>
      reportSchedulesApi.update(id, data),
    onSuccess: () => {
      toast.success('Schedule updated successfully');
      queryClient.invalidateQueries({ queryKey: ['reportSchedules'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to update schedule');
    },
  });

  const handleInputChange = (field: keyof FormData, value: string | boolean) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const toggleSection = (sectionId: string) => {
    setSelectedSections((prev) =>
      prev.includes(sectionId) ? prev.filter((id) => id !== sectionId) : [...prev, sectionId]
    );
  };

  const buildCronExpression = (): string => {
    const [hours, minutes] = scheduleTime.split(':').map(Number);

    switch (formData.frequency) {
      case 'daily':
        return `${minutes} ${hours} * * *`;
      case 'weekly':
        const weekdayMap: Record<string, number> = {
          sunday: 0,
          monday: 1,
          tuesday: 2,
          wednesday: 3,
          thursday: 4,
          friday: 5,
          saturday: 6,
        };
        return `${minutes} ${hours} * * ${weekdayMap[scheduleWeekday]}`;
      case 'monthly':
        return `${minutes} ${hours} ${scheduleDay} * *`;
      case 'quarterly':
        // First day of quarter
        return `${minutes} ${hours} 1 1,4,7,10 *`;
      case 'yearly':
        return `${minutes} ${hours} 1 1 *`;
      default:
        return `${minutes} ${hours} * * *`;
    }
  };

  const buildDistributionConfig = (): DistributionConfig => {
    const methods: DistributionMethod[] = [];

    if (formData.distributionEmail && formData.recipients) {
      methods.push({
        type: 'email',
        enabled: true,
        config: {
          to: formData.recipients.split(',').map((e) => e.trim()).filter(Boolean),
          include_attachments: true,
          formats: [formData.reportFormat as any],
        },
      });
    }

    if (formData.distributionWebhook && formData.webhookUrl) {
      methods.push({
        type: 'webhook',
        enabled: true,
        config: {
          url: formData.webhookUrl,
          retry_count: 3,
          timeout_seconds: 30,
          verify_ssl: true,
        },
      });
    }

    return {
      enabled: methods.length > 0,
      methods,
    };
  };

  const buildReportConfig = (): ReportConfig => {
    const now = new Date();
    const periodStart = defaultPeriod?.start || format(now, 'yyyy-MM-dd');
    const periodEnd = defaultPeriod?.end || format(new Date(now.setMonth(now.getMonth() - 1)), 'yyyy-MM-dd');

    return {
      framework: formData.framework,
      period_start: periodStart,
      period_end: periodEnd,
      include_sections: selectedSections.map((id) => ({
        id,
        name: reportSections.find((s) => s.id === id)?.name || id,
        enabled: true,
      })),
      format: formData.reportFormat as any,
      options: {
        include_pii: false,
        include_session_logs: true,
        include_command_history: true,
        redact_sensitive_data: true,
        aggregate_results: true,
      },
    };
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!formData.name.trim()) {
      toast.error('Please enter a schedule name');
      return;
    }

    if (formData.distributionEmail && !formData.recipients.trim()) {
      toast.error('Please enter recipient email addresses');
      return;
    }

    if (formData.distributionWebhook && !formData.webhookUrl.trim()) {
      toast.error('Please enter webhook URL');
      return;
    }

    const data: CreateReportScheduleData = {
      name: formData.name,
      description: formData.description || undefined,
      framework: formData.framework,
      frequency: formData.frequency,
      cron_expression: buildCronExpression(),
      timezone: formData.timezone,
      is_active: formData.isActive,
      recipients: formData.recipients.split(',').map((e) => e.trim()).filter(Boolean),
      distribution_config: buildDistributionConfig(),
      report_config: buildReportConfig(),
    };

    if (isEditing && schedule) {
      updateMutation.mutate({ id: schedule.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const weekdayOptions = [
    { value: 'sunday', label: 'Sunday' },
    { value: 'monday', label: 'Monday' },
    { value: 'tuesday', label: 'Tuesday' },
    { value: 'wednesday', label: 'Wednesday' },
    { value: 'thursday', label: 'Thursday' },
    { value: 'friday', label: 'Friday' },
    { value: 'saturday', label: 'Saturday' },
  ];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={isEditing ? 'Edit Report Schedule' : 'Create Report Schedule'}
      size="xl"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={createMutation.isPending || updateMutation.isPending}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleSubmit}
            disabled={createMutation.isPending || updateMutation.isPending}
            isLoading={createMutation.isPending || updateMutation.isPending}
          >
            <Save className="mr-2 h-4 w-4" />
            {isEditing ? 'Update Schedule' : 'Create Schedule'}
          </Button>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Basic Information */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white">Basic Information</h4>

          <div>
            <Label htmlFor="schedule-name">Schedule Name *</Label>
            <Input
              id="schedule-name"
              value={formData.name}
              onChange={(e) => handleInputChange('name', e.target.value)}
              placeholder="Monthly SOC 2 Report"
              required
            />
          </div>

          <div>
            <Label htmlFor="schedule-description">Description</Label>
            <Textarea
              id="schedule-description"
              value={formData.description}
              onChange={(e) => handleInputChange('description', e.target.value)}
              placeholder="Optional description of this scheduled report"
              rows={2}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="schedule-framework">Compliance Framework *</Label>
              <select
                id="schedule-framework"
                value={formData.framework}
                onChange={(e) => handleInputChange('framework', e.target.value)}
                disabled={isEditing}
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
              >
                {frameworks.map((fw) => (
                  <option key={fw.value} value={fw.value}>
                    {fw.label} - {fw.description}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <Label htmlFor="schedule-frequency">Frequency *</Label>
              <select
                id="schedule-frequency"
                value={formData.frequency}
                onChange={(e) => handleInputChange('frequency', e.target.value as ScheduleFrequency)}
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
              >
                {frequencies.map((freq) => (
                  <option key={freq.value} value={freq.value}>
                    {freq.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Schedule Details */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white flex items-center gap-2">
            <Clock className="h-4 w-4" />
            Schedule Details
          </h4>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <Label htmlFor="schedule-time">Time (UTC)</Label>
              <Input
                id="schedule-time"
                type="time"
                value={scheduleTime}
                onChange={(e) => setScheduleTime(e.target.value)}
              />
            </div>

            <div>
              <Label htmlFor="schedule-timezone">Timezone</Label>
              <select
                id="schedule-timezone"
                value={formData.timezone}
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => handleInputChange('timezone', e.target.value)}
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
              >
                {commonTimezones.map((tz) => (
                  <option key={tz} value={tz}>
                    {tz}
                  </option>
                ))}
              </select>
            </div>

            {formData.frequency === 'monthly' && (
              <div>
                <Label htmlFor="schedule-day">Day of Month</Label>
                <select
                  id="schedule-day"
                  value={scheduleDay}
                  onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setScheduleDay(e.target.value)}
                  className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                >
                  {Array.from({ length: 28 }, (_, i) => (
                    <option key={i + 1} value={String(i + 1)}>
                      {i + 1}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {formData.frequency === 'weekly' && (
              <div>
                <Label htmlFor="schedule-weekday">Day of Week</Label>
                <select
                  id="schedule-weekday"
                  value={scheduleWeekday}
                  onChange={(e: React.ChangeEvent<HTMLSelectElement>) => setScheduleWeekday(e.target.value)}
                  className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                >
                  {weekdayOptions.map((day) => (
                    <option key={day.value} value={day.value}>
                      {day.label}
                    </option>
                  ))}
                </select>
              </div>
            )}
          </div>

          <div className="flex items-center gap-2">
            <Toggle
              checked={formData.isActive}
              onChange={(e) => handleInputChange('isActive', (e.target as HTMLInputElement).checked)}
            />
            <Label htmlFor="active-toggle" className="mb-0">
              Schedule is active
            </Label>
          </div>
        </div>

        {/* Report Content */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white">Report Content</h4>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="report-format">Output Format</Label>
              <select
                id="report-format"
                value={formData.reportFormat}
                onChange={(e: React.ChangeEvent<HTMLSelectElement>) => handleInputChange('reportFormat', e.target.value)}
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
              >
                {reportFormats.map((fmt) => (
                  <option key={fmt.value} value={fmt.value}>
                    {fmt.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div>
            <Label>Include Sections</Label>
            <div className="mt-2 grid grid-cols-2 gap-2">
              {reportSections.map((section) => (
                <label
                  key={section.id}
                  className="flex items-start gap-2 rounded border border-gray-700 p-3 cursor-pointer hover:bg-gray-800 transition-colors"
                >
                  <input
                    type="checkbox"
                    checked={selectedSections.includes(section.id)}
                    onChange={() => toggleSection(section.id)}
                    className="mt-1"
                  />
                  <div>
                    <span className="text-sm font-medium text-white">{section.name}</span>
                    <p className="text-xs text-gray-400">{section.description}</p>
                  </div>
                </label>
              ))}
            </div>
          </div>
        </div>

        {/* Distribution */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white flex items-center gap-2">
            <Mail className="h-4 w-4" />
            Distribution
          </h4>

          <div className="space-y-4 rounded-lg border border-gray-700 p-4">
            <div className="flex items-center justify-between">
              <div>
                <span className="text-sm font-medium text-white">Email Distribution</span>
                <p className="text-xs text-gray-400">Send report via email</p>
              </div>
              <Toggle
                checked={formData.distributionEmail}
                onChange={(e) => handleInputChange('distributionEmail', (e.target as HTMLInputElement).checked)}
              />
            </div>

            {formData.distributionEmail && (
              <div>
                <Label htmlFor="recipients">Recipients (comma-separated)</Label>
                <Input
                  id="recipients"
                  type="email"
                  value={formData.recipients}
                  onChange={(e) => handleInputChange('recipients', e.target.value)}
                  placeholder="admin@example.com, compliance@example.com"
                />
              </div>
            )}

            <div className="flex items-center justify-between">
              <div>
                <span className="text-sm font-medium text-white">Webhook Distribution</span>
                <p className="text-xs text-gray-400">POST report to webhook URL</p>
              </div>
              <Toggle
                checked={formData.distributionWebhook}
                onChange={(e) => handleInputChange('distributionWebhook', (e.target as HTMLInputElement).checked)}
              />
            </div>

            {formData.distributionWebhook && (
              <div>
                <Label htmlFor="webhook-url">Webhook URL</Label>
                <Input
                  id="webhook-url"
                  type="url"
                  value={formData.webhookUrl}
                  onChange={(e) => handleInputChange('webhookUrl', e.target.value)}
                  placeholder="https://your-server.com/webhook"
                />
              </div>
            )}
          </div>
        </div>

        {/* Info Box */}
        <div className="flex gap-2 rounded-lg border border-primary-500/30 bg-primary-500/10 p-3">
          <Info className="h-4 w-4 text-primary-400 mt-0.5 flex-shrink-0" />
          <p className="text-xs text-gray-300">
            Scheduled reports will be generated automatically based on the configured frequency.
            Reports will be distributed to the configured recipients. You can view all scheduled
            reports and their run history in the Reports page.
          </p>
        </div>
      </form>
=======
  onSave,
  existingSchedule,
  reportName = 'Report',
}) => {
  const [frequency, setFrequency] = useState<ReportScheduleFrequency>(
    existingSchedule?.frequency || 'once'
  );
  const [enabled, setEnabled] = useState(existingSchedule?.enabled ?? true);
  const [time, setTime] = useState(existingSchedule?.time || '09:00');
  const [timezone, setTimezone] = useState(existingSchedule?.timezone || 'UTC');
  const [dayOfWeek, setDayOfWeek] = useState(existingSchedule?.day_of_week ?? 1);
  const [dayOfMonth, setDayOfMonth] = useState(existingSchedule?.day_of_month ?? 1);
  const [recipients, setRecipients] = useState(existingSchedule?.recipients?.join(', ') || '');

  // Reset form when dialog opens/closes
  useEffect(() => {
    if (isOpen) {
      setFrequency(existingSchedule?.frequency || 'once');
      setEnabled(existingSchedule?.enabled ?? true);
      setTime(existingSchedule?.time || '09:00');
      setTimezone(existingSchedule?.timezone || 'UTC');
      setDayOfWeek(existingSchedule?.day_of_week ?? 1);
      setDayOfMonth(existingSchedule?.day_of_month ?? 1);
      setRecipients(existingSchedule?.recipients?.join(', ') || '');
    }
  }, [isOpen, existingSchedule]);

  const getScheduleSummary = (): string => {
    if (frequency === 'once') {
      return 'Will run once immediately when saved';
    }

    const timeStr = `${time} ${timezone.replace('_', ' ')}`;
    const recipientStr = recipients ? `, sent to ${recipients}` : '';

    switch (frequency) {
      case 'daily':
        return `Runs every day at ${timeStr}${recipientStr}`;
      case 'weekly':
        return `Runs every ${daysOfWeek.find((d) => d.value === dayOfWeek)?.label} at ${timeStr}${recipientStr}`;
      case 'monthly':
        return `Runs on the ${dayOfMonth}${getOrdinalSuffix(dayOfMonth)} of each month at ${timeStr}${recipientStr}`;
      case 'quarterly':
        return `Runs on the last day of each quarter (Mar 31, Jun 30, Sep 30, Dec 31) at ${timeStr}${recipientStr}`;
    }
  };

  const getOrdinalSuffix = (day: number): string => {
    if (day > 3 && day < 21) return 'th';
    switch (day % 10) {
      case 1:
        return 'st';
      case 2:
        return 'nd';
      case 3:
        return 'rd';
      default:
        return 'th';
    }
  };

  const handleSave = () => {
    const schedule: ReportSchedule = {
      frequency,
      enabled,
      time,
      timezone,
      recipients: recipients
        ? recipients.split(',').map((r) => r.trim()).filter(Boolean)
        : undefined,
    };

    if (frequency === 'weekly') {
      schedule.day_of_week = dayOfWeek;
    } else if (frequency === 'monthly') {
      schedule.day_of_month = dayOfMonth;
    }

    onSave(schedule);
  };

  const isFormValid = () => {
    if (frequency === 'once') return true;
    if (!time) return false;
    if (frequency === 'weekly' && dayOfWeek === undefined) return false;
    if (frequency === 'monthly' && (dayOfMonth < 1 || dayOfMonth > 31)) return false;
    return true;
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Schedule Report Generation">
      <div className="space-y-5">
        {/* Report Name */}
        <div>
          <p className="text-sm text-gray-400">
            Scheduling: <span className="font-medium text-white">{reportName}</span>
          </p>
        </div>

        {/* Enable/Disable Schedule */}
        <div className="flex items-center justify-between rounded-lg border border-gray-800 p-4">
          <div>
            <h4 className="font-medium text-white">Enable Schedule</h4>
            <p className="mt-1 text-sm text-gray-400">
              Turn off to pause automatic generation without deleting the schedule
            </p>
          </div>
          <Toggle checked={enabled} onChange={setEnabled} />
        </div>

        {/* Frequency */}
        <div>
          <label className="mb-2 block text-sm font-medium text-gray-300">Frequency</label>
          <div className="grid gap-3">
            {frequencyOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={() => setFrequency(option.value)}
                className={clsx(
                  'flex items-start gap-3 rounded-lg border p-4 text-left transition-colors',
                  frequency === option.value
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-800 hover:bg-gray-800/50'
                )}
              >
                <div className="mt-0.5">
                  <div
                    className={clsx(
                      'h-4 w-4 rounded-full border-2',
                      frequency === option.value
                        ? 'border-primary-500 bg-primary-500'
                        : 'border-gray-600'
                    )}
                  >
                    {frequency === option.value && (
                      <div className="flex h-full w-full items-center justify-center">
                        <div className="h-1.5 w-1.5 rounded-full bg-white" />
                      </div>
                    )}
                  </div>
                </div>
                <div>
                  <p className="font-medium text-white">{option.label}</p>
                  <p className="mt-1 text-sm text-gray-400">{option.description}</p>
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Time Configuration (not for one-time) */}
        {frequency !== 'once' && (
          <div className="grid gap-4 sm:grid-cols-2">
            {/* Time */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-300">Time</label>
              <Input
                type="time"
                value={time}
                onChange={(e) => setTime(e.target.value)}
              />
            </div>

            {/* Timezone */}
            <div>
              <label className="mb-1 block text-sm font-medium text-gray-300">Timezone</label>
              <Select
                options={timezones}
                value={timezone}
                onChange={(e) => setTimezone(e.target.value)}
              />
            </div>
          </div>
        )}

        {/* Day of Week (for weekly) */}
        {frequency === 'weekly' && (
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Day of Week</label>
            <Select
              options={daysOfWeek}
              value={dayOfWeek.toString()}
              onChange={(e) => setDayOfWeek(parseInt(e.target.value, 10))}
            />
          </div>
        )}

        {/* Day of Month (for monthly) */}
        {frequency === 'monthly' && (
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Day of Month</label>
            <Select
              options={Array.from({ length: 31 }, (_, i) => ({
                value: (i + 1).toString(),
                label: `${i + 1}${getOrdinalSuffix(i + 1)}`,
              }))}
              value={dayOfMonth.toString()}
              onChange={(e) => setDayOfMonth(parseInt(e.target.value, 10))}
            />
            <p className="mt-1 text-xs text-gray-500">
              Note: For months with fewer days, the report will run on the last day of the month.
            </p>
          </div>
        )}

        {/* Email Recipients */}
        <div>
          <label className="mb-1 block text-sm font-medium text-gray-300">
            Email Recipients <span className="text-gray-500">(optional)</span>
          </label>
          <Input
            type="text"
            placeholder="user1@example.com, user2@example.com"
            value={recipients}
            onChange={(e) => setRecipients(e.target.value)}
          />
          <p className="mt-1 text-xs text-gray-500">
            Comma-separated email addresses to receive the generated report
          </p>
        </div>

        {/* Schedule Summary */}
        <div className="rounded-lg bg-gray-900 p-4">
          <div className="flex items-start gap-3">
            <Info className="mt-0.5 h-5 w-5 text-primary-400" />
            <div>
              <p className="text-sm font-medium text-white">Schedule Summary</p>
              <p className="mt-1 text-sm text-gray-400">{getScheduleSummary()}</p>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-3 border-t border-gray-800 pt-4">
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={!isFormValid() || !enabled}>
            Save Schedule
          </Button>
        </div>
      </div>
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192
    </Modal>
  );
};

<<<<<<< HEAD
// Helper function wrapper
function useQueryQueryClient() {
  return useQueryClient();
}
=======
export default ReportScheduleDialog;
>>>>>>> team/complete-todo-items-in-internalpamanalyt-1772007192

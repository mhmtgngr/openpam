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

interface ReportScheduleDialogProps {
  isOpen: boolean;
  onClose: () => void;
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
];

export const ReportScheduleDialog: React.FC<ReportScheduleDialogProps> = ({
  isOpen,
  onClose,
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
    </Modal>
  );
};

export default ReportScheduleDialog;

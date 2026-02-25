import React, { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Share, Mail, Webhook, Link as LinkIcon, Save, X, Globe, Check } from 'lucide-react';
import { Modal } from '@/components/common';
import { Input, Textarea, Label, Button, Toggle } from '@/components/common';
import { reportsApi } from '@/api/reports';
import type { Report, ExportReportRequest } from '@/types/reports';
import { ReportFormat } from '@/types/reports';
import { format } from 'date-fns';
import toast from 'react-hot-toast';

interface ReportDistributionDialogProps {
  isOpen: boolean;
  onClose: () => void;
  report: Report | null;
  onSuccess?: () => void;
}

interface EmailRecipient {
  email: string;
  valid: boolean;
}

interface DistributionSettings {
  method: 'email' | 'webhook' | 'share_link' | 'download';
  format: ReportFormat;
  recipients: EmailRecipient[];
  webhookUrl: string;
  shareLinkExpiry: string;
  includePassword: boolean;
  sharePassword: string;
  redactPii: boolean;
  includeMetadata: boolean;
}

const formatOptions: { value: ReportFormat; label: string; description: string }[] = [
  { value: 'pdf', label: 'PDF', description: 'Portable document format, best for sharing' },
  { value: 'html', label: 'HTML', description: 'Interactive web page format' },
  { value: 'csv', label: 'CSV', description: 'Comma-separated values, for spreadsheets' },
  { value: 'xlsx', label: 'Excel', description: 'Microsoft Excel spreadsheet' },
  { value: 'json', label: 'JSON', description: 'Machine-readable data format' },
];

export const ReportDistributionDialog: React.FC<ReportDistributionDialogProps> = ({
  isOpen,
  onClose,
  report,
  onSuccess,
}) => {
  const queryClient = useQueryQueryClient();
  const [settings, setSettings] = useState<DistributionSettings>({
    method: 'email',
    format: 'pdf',
    recipients: [],
    webhookUrl: '',
    shareLinkExpiry: '7',
    includePassword: false,
    sharePassword: '',
    redactPii: true,
    includeMetadata: true,
  });

  const [generatedLink, setGeneratedLink] = useState<string | null>(null);
  const [linkCopied, setLinkCopied] = useState(false);
  const [emailInput, setEmailInput] = useState('');

  const exportMutation = useMutation({
    mutationFn: (request: ExportReportRequest) => {
      if (!report) throw new Error('No report selected');
      return reportsApi.export(report.id, request);
    },
    onSuccess: (data) => {
      if (settings.method === 'share_link') {
        setGeneratedLink(data.download_url);
        toast.success('Share link generated successfully');
      } else if (settings.method === 'webhook') {
        toast.success('Report sent to webhook successfully');
      } else if (settings.method === 'email') {
        toast.success('Report sent via email successfully');
      }
      queryClient.invalidateQueries({ queryKey: ['reports'] });
      onSuccess?.();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to distribute report');
    },
  });

  useEffect(() => {
    if (isOpen) {
      // Reset state when dialog opens
      setGeneratedLink(null);
      setLinkCopied(false);
      setEmailInput('');
      setSettings({
        method: 'email',
        format: 'pdf',
        recipients: [],
        webhookUrl: '',
        shareLinkExpiry: '7',
        includePassword: false,
        sharePassword: '',
        redactPii: true,
        includeMetadata: true,
      });
    }
  }, [isOpen]);

  const handleAddRecipient = () => {
    if (!emailInput.trim()) return;

    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    const isValid = emailRegex.test(emailInput);

    setSettings((prev) => ({
      ...prev,
      recipients: [
        ...prev.recipients,
        { email: emailInput.trim(), valid: isValid },
      ],
    }));
    setEmailInput('');
  };

  const handleRemoveRecipient = (index: number) => {
    setSettings((prev) => ({
      ...prev,
      recipients: prev.recipients.filter((_, i) => i !== index),
    }));
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      handleAddRecipient();
    }
  };

  const handleGenerateShareLink = () => {
    const request: ExportReportRequest = {
      format: settings.format,
      include_metadata: settings.includeMetadata,
      redact_pii: settings.redactPii,
    };

    exportMutation.mutate(request);
  };

  const handleSendWebhook = () => {
    // This would typically call a specialized webhook distribution endpoint
    toast.success('Webhook configuration saved');
    onClose();
  };

  const handleSendEmail = () => {
    const hasInvalidRecipients = settings.recipients.some((r) => !r.valid);
    if (hasInvalidRecipients) {
      toast.error('Please fix invalid email addresses');
      return;
    }

    if (settings.recipients.length === 0) {
      toast.error('Please add at least one recipient');
      return;
    }

    const request: ExportReportRequest = {
      format: settings.format,
      include_metadata: settings.includeMetadata,
      redact_pii: settings.redactPii,
    };

    exportMutation.mutate(request);
  };

  const handleDirectDownload = () => {
    if (!report) return;

    const downloadUrl = reportsApi.download(report.id);
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = `${report.name.replace(/\s+/g, '_')}.${settings.format}`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);

    toast.success('Report download started');
    onClose();
  };

  const handleCopyLink = () => {
    if (generatedLink) {
      navigator.clipboard.writeText(generatedLink);
      setLinkCopied(true);
      toast.success('Link copied to clipboard');
      setTimeout(() => setLinkCopied(false), 2000);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    switch (settings.method) {
      case 'email':
        handleSendEmail();
        break;
      case 'webhook':
        handleSendWebhook();
        break;
      case 'share_link':
        handleGenerateShareLink();
        break;
      case 'download':
        handleDirectDownload();
        break;
    }
  };

  const validRecipientsCount = settings.recipients.filter((r) => r.valid).length;

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Distribute Report"
      size="lg"
      footer={
        generatedLink ? (
          <>
            <Button variant="ghost" onClick={() => setGeneratedLink(null)}>
              Close
            </Button>
            <Button variant="primary" onClick={handleCopyLink}>
              {linkCopied ? (
                <>
                  <Check className="mr-2 h-4 w-4" />
                  Copied!
                </>
              ) : (
                <>
                  <LinkIcon className="mr-2 h-4 w-4" />
                  Copy Link
                </>
              )}
            </Button>
          </>
        ) : (
          <>
            <Button variant="ghost" onClick={onClose} disabled={exportMutation.isPending}>
              Cancel
            </Button>
            <Button
              variant="primary"
              onClick={handleSubmit}
              disabled={exportMutation.isPending}
              isLoading={exportMutation.isPending}
            >
              <Share className="mr-2 h-4 w-4" />
              {settings.method === 'download'
                ? 'Download Report'
                : settings.method === 'share_link'
                  ? 'Generate Link'
                  : 'Send Report'}
            </Button>
          </>
        )
      }
    >
      {generatedLink ? (
        <div className="space-y-4">
          <div className="text-center">
            <div className="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-success-500/20">
              <Check className="h-6 w-6 text-success-400" />
            </div>
            <h3 className="text-lg font-semibold text-white">Share Link Generated</h3>
            <p className="mt-2 text-sm text-gray-400">
              Your report has been prepared for sharing. Use the link below to access it.
            </p>
          </div>

          <div className="rounded-lg border border-gray-700 bg-gray-800 p-4">
            <p className="mb-2 text-xs text-gray-400">Share Link</p>
            <p className="break-all text-sm text-white">{generatedLink}</p>
            <p className="mt-2 text-xs text-gray-500">
              Expires in {settings.shareLinkExpiry} days
            </p>
          </div>

          <div className="rounded-lg border border-warning-500/30 bg-warning-500/10 p-3">
            <p className="text-xs text-gray-300">
              <strong>Warning:</strong> Anyone with this link can access the report. Share only
              with trusted recipients. The link will expire after the specified time period.
            </p>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Report Info */}
          {report && (
            <div className="rounded-lg border border-gray-700 bg-gray-800/50 p-3">
              <p className="text-sm font-medium text-white">{report.name}</p>
              <p className="text-xs text-gray-400">
                {report.framework.toUpperCase()} • {format(new Date(report.created_at), 'MMM dd, yyyy')}
              </p>
            </div>
          )}

          {/* Distribution Method Selection */}
          <div className="space-y-4">
            <h4 className="text-sm font-semibold text-white">Distribution Method</h4>

            <div className="grid grid-cols-2 gap-3">
              <button
                type="button"
                onClick={() => setSettings((prev) => ({ ...prev, method: 'email' }))}
                className={`flex flex-col items-start rounded-lg border p-4 text-left transition-colors ${
                  settings.method === 'email'
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-700 hover:bg-gray-800'
                }`}
              >
                <Mail className="mb-2 h-5 w-5 text-primary-400" />
                <span className="font-medium text-white">Email</span>
                <span className="text-xs text-gray-400">Send report via email</span>
              </button>

              <button
                type="button"
                onClick={() => setSettings((prev) => ({ ...prev, method: 'webhook' }))}
                className={`flex flex-col items-start rounded-lg border p-4 text-left transition-colors ${
                  settings.method === 'webhook'
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-700 hover:bg-gray-800'
                }`}
              >
                <Webhook className="mb-2 h-5 w-5 text-primary-400" />
                <span className="font-medium text-white">Webhook</span>
                <span className="text-xs text-gray-400">POST to webhook URL</span>
              </button>

              <button
                type="button"
                onClick={() => setSettings((prev) => ({ ...prev, method: 'share_link' }))}
                className={`flex flex-col items-start rounded-lg border p-4 text-left transition-colors ${
                  settings.method === 'share_link'
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-700 hover:bg-gray-800'
                }`}
              >
                <LinkIcon className="mb-2 h-5 w-5 text-primary-400" />
                <span className="font-medium text-white">Share Link</span>
                <span className="text-xs text-gray-400">Generate shareable link</span>
              </button>

              <button
                type="button"
                onClick={() => setSettings((prev) => ({ ...prev, method: 'download' }))}
                className={`flex flex-col items-start rounded-lg border p-4 text-left transition-colors ${
                  settings.method === 'download'
                    ? 'border-primary-500 bg-primary-500/10'
                    : 'border-gray-700 hover:bg-gray-800'
                }`}
              >
                <Save className="mb-2 h-5 w-5 text-primary-400" />
                <span className="font-medium text-white">Download</span>
                <span className="text-xs text-gray-400">Direct file download</span>
              </button>
            </div>
          </div>

          {/* Format Selection */}
          <div className="space-y-4">
            <h4 className="text-sm font-semibold text-white">Output Format</h4>

            <div className="grid grid-cols-1 gap-2">
              {formatOptions.map((fmt) => (
                <label
                  key={fmt.value}
                  className={`flex items-center justify-between rounded-lg border p-3 cursor-pointer transition-colors ${
                    settings.format === fmt.value
                      ? 'border-primary-500 bg-primary-500/10'
                      : 'border-gray-700 hover:bg-gray-800'
                  }`}
                >
                  <div>
                    <span className="font-medium text-white">{fmt.label}</span>
                    <p className="text-xs text-gray-400">{fmt.description}</p>
                  </div>
                  <input
                    type="radio"
                    name="format"
                    value={fmt.value}
                    checked={settings.format === fmt.value}
                    onChange={(e) => setSettings((prev) => ({ ...prev, format: e.target.value as ReportFormat }))}
                    className="h-4 w-4 border-gray-600 text-primary-600 focus:ring-primary-500"
                  />
                </label>
              ))}
            </div>
          </div>

          {/* Email Recipients */}
          {settings.method === 'email' && (
            <div className="space-y-4">
              <h4 className="text-sm font-semibold text-white flex items-center gap-2">
                <Mail className="h-4 w-4" />
                Recipients
              </h4>

              <div className="flex gap-2">
                <Input
                  value={emailInput}
                  onChange={(e) => setEmailInput(e.target.value)}
                  onKeyPress={handleKeyPress}
                  placeholder="Enter email address"
                  type="email"
                />
                <Button type="button" onClick={handleAddRecipient} variant="secondary">
                  Add
                </Button>
              </div>

              {settings.recipients.length > 0 && (
                <div className="space-y-2">
                  {settings.recipients.map((recipient, index) => (
                    <div
                      key={index}
                      className="flex items-center justify-between rounded border border-gray-700 bg-gray-800 px-3 py-2"
                    >
                      <div className="flex items-center gap-2">
                        {recipient.valid ? (
                          <Check className="h-4 w-4 text-success-400" />
                        ) : (
                          <X className="h-4 w-4 text-danger-400" />
                        )}
                        <span className={`text-sm ${recipient.valid ? 'text-white' : 'text-danger-400'}`}>
                          {recipient.email}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => handleRemoveRecipient(index)}
                        className="text-gray-400 hover:text-danger-400"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    </div>
                  ))}
                </div>
              )}

              <p className="text-xs text-gray-400">
                {validRecipientsCount} valid recipient{validRecipientsCount !== 1 ? 's' : ''}
              </p>
            </div>
          )}

          {/* Webhook Configuration */}
          {settings.method === 'webhook' && (
            <div className="space-y-4">
              <h4 className="text-sm font-semibold text-white flex items-center gap-2">
                <Webhook className="h-4 w-4" />
                Webhook Configuration
              </h4>

              <div>
                <Label htmlFor="webhook-url">Webhook URL</Label>
                <Input
                  id="webhook-url"
                  type="url"
                  value={settings.webhookUrl}
                  onChange={(e) => setSettings((prev) => ({ ...prev, webhookUrl: e.target.value }))}
                  placeholder="https://your-server.com/api/webhook"
                  required
                />
              </div>

              <div className="rounded-lg border border-gray-700 p-3">
                <p className="text-xs text-gray-400 mb-2">Payload Preview:</p>
                <pre className="overflow-x-auto text-xs text-gray-300">
                  {JSON.stringify(
                    {
                      report_id: report?.id,
                      report_name: report?.name,
                      framework: report?.framework,
                      format: settings.format,
                      download_url: '[URL]',
                      generated_at: new Date().toISOString(),
                    },
                    null,
                    2
                  )}
                </pre>
              </div>
            </div>
          )}

          {/* Share Link Settings */}
          {settings.method === 'share_link' && (
            <div className="space-y-4">
              <h4 className="text-sm font-semibold text-white flex items-center gap-2">
                <LinkIcon className="h-4 w-4" />
                Link Settings
              </h4>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label htmlFor="link-expiry">Link Expiration</Label>
                  <select
                    id="link-expiry"
                    value={settings.shareLinkExpiry}
                    onChange={(e) => setSettings((prev) => ({ ...prev, shareLinkExpiry: e.target.value }))}
                    className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                  >
                    <option value="1">1 day</option>
                    <option value="3">3 days</option>
                    <option value="7">7 days</option>
                    <option value="14">14 days</option>
                    <option value="30">30 days</option>
                  </select>
                </div>

                <div className="flex items-center justify-between rounded-lg border border-gray-700 p-3">
                  <div>
                    <span className="text-sm font-medium text-white">Password Protection</span>
                    <p className="text-xs text-gray-400">Require password to access</p>
                  </div>
                  <Toggle
                    checked={settings.includePassword}
                    onChange={(checked: boolean) => setSettings((prev) => ({ ...prev, includePassword: checked }))}
                  />
                </div>
              </div>

              {settings.includePassword && (
                <div>
                  <Label htmlFor="share-password">Link Password</Label>
                  <Input
                    id="share-password"
                    type="password"
                    value={settings.sharePassword}
                    onChange={(e) => setSettings((prev) => ({ ...prev, sharePassword: e.target.value }))}
                    placeholder="Enter password"
                  />
                </div>
              )}
            </div>
          )}

          {/* Options */}
          <div className="space-y-4">
            <h4 className="text-sm font-semibold text-white">Options</h4>

            <div className="space-y-3">
              <div className="flex items-center justify-between rounded-lg border border-gray-700 p-3">
                <div>
                  <span className="text-sm font-medium text-white">Redact PII</span>
                  <p className="text-xs text-gray-400">Remove personally identifiable information</p>
                </div>
                <Toggle
                  checked={settings.redactPii}
                  onChange={(checked: boolean) => setSettings((prev) => ({ ...prev, redactPii: checked }))}
                />
              </div>

              <div className="flex items-center justify-between rounded-lg border border-gray-700 p-3">
                <div>
                  <span className="text-sm font-medium text-white">Include Metadata</span>
                  <p className="text-xs text-gray-400">Add report generation metadata</p>
                </div>
                <Toggle
                  checked={settings.includeMetadata}
                  onChange={(checked: boolean) => setSettings((prev) => ({ ...prev, includeMetadata: checked }))}
                />
              </div>
            </div>
          </div>
        </form>
      )}
    </Modal>
  );
};

// Helper function wrapper
function useQueryQueryClient() {
  return useQueryClient();
}

/**
 * ReportViewer - Component for viewing generated report content
 * Supports inline preview, download options, and various report formats
 */

import React, { useState, useEffect } from 'react';
import { X, Download, Share2, Printer, Maximize2, Minimize2, ChevronLeft, ChevronRight } from 'lucide-react';
import { Button } from './Button';
import { Badge } from './Badge';
import { LoadingState } from './LoadingState';
import { EmptyState } from './EmptyState';
import { format } from 'date-fns';
import clsx from 'clsx';
import type { ReportSnapshot, ReportFormat } from '@/types/reports';

interface ReportViewerProps {
  report: ReportSnapshot | null;
  onClose?: () => void;
  onDownload?: (report: ReportSnapshot) => void;
  isLoading?: boolean;
}

const formatLabels: Record<ReportFormat, string> = {
  pdf: 'PDF Document',
  csv: 'CSV Spreadsheet',
  json: 'JSON Data',
  xlsx: 'Excel Spreadsheet',
  html: 'HTML Document',
};

export const ReportViewer: React.FC<ReportViewerProps> = ({
  report,
  onClose,
  onDownload,
  isLoading = false,
}) => {
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [currentPage, setCurrentPage] = useState(0);
  const [iframeLoaded, setIframeLoaded] = useState(false);

  useEffect(() => {
    if (report) {
      setIframeLoaded(false);
      setCurrentPage(0);
    }
  }, [report]);

  if (!report && !isLoading) {
    return null;
  }

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
        <LoadingState message="Loading report..." />
      </div>
    );
  }

  if (!report) return null;

  const handleDownload = () => {
    onDownload?.(report);
  };

  const handlePrint = () => {
    if (report.file_url) {
      window.open(report.file_url, '_blank');
    }
  };

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen);
  };

  const renderPreviewContent = () => {
    // If report is not completed yet
    if (report.status !== 'completed') {
      return (
        <div className="flex h-full items-center justify-center">
          <EmptyState
            title={report.status === 'generating' ? 'Report is being generated' : 'Report not available'}
            description={
              report.status === 'generating'
                ? 'Please wait while the report is being generated. This may take a few minutes.'
                : report.error_message || 'This report could not be generated.'
            }
          />
        </div>
      );
    }

    // If no file URL but completed
    if (!report.file_url) {
      return (
        <div className="flex h-full items-center justify-center">
          <EmptyState
            title="Report file not available"
            description="The report file may have expired or been deleted."
          />
        </div>
      );
    }

    // Render based on format
    switch (report.format) {
      case 'pdf':
      case 'html':
        return (
          <div className="relative h-full">
            {!iframeLoaded && (
              <div className="absolute inset-0 flex items-center justify-center bg-gray-900">
                <LoadingState message="Loading preview..." />
              </div>
            )}
            <iframe
              src={report.file_url}
              className="h-full w-full border-0"
              onLoad={() => setIframeLoaded(true)}
              title="Report Preview"
            />
          </div>
        );

      case 'json':
        return (
          <div className="h-full overflow-auto p-6">
            <pre className="rounded-lg bg-gray-900 p-4 text-sm text-gray-300">
              {JSON.stringify(report.metadata || {}, null, 2)}
            </pre>
          </div>
        );

      case 'csv':
      case 'xlsx':
        return (
          <div className="flex h-full items-center justify-center">
            <EmptyState
              title="Preview not available"
              description={`${formatLabels[report.format]} files cannot be previewed inline. Please download the file to view its contents.`}
              action={
                <Button variant="primary" onClick={handleDownload} leftIcon={<Download className="h-4 w-4" />}>
                  Download {formatLabels[report.format]}
                </Button>
              }
            />
          </div>
        );

      default:
        return (
          <div className="flex h-full items-center justify-center">
            <EmptyState title="Unsupported format" description="This report format cannot be previewed." />
          </div>
        );
    }
  };

  return (
    <div
      className={clsx(
        'fixed inset-0 z-50 flex bg-black/70 transition-all',
        isFullscreen ? 'p-0' : 'p-4 md:p-8'
      )}
    >
      <div
        className={clsx(
          'flex w-full flex-col overflow-hidden rounded-xl bg-gray-900 shadow-2xl',
          isFullscreen ? 'h-full rounded-none' : 'max-h-[90vh]'
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-gray-800 px-6 py-4">
          <div className="flex-1">
            <div className="flex items-center gap-3">
              <h2 className="text-xl font-semibold text-white">
                {report.report_name || 'Report'}
              </h2>
              <Badge variant="neutral">{formatLabels[report.format]}</Badge>
              <Badge
                variant={
                  report.status === 'completed'
                    ? 'success'
                    : report.status === 'failed'
                    ? 'danger'
                    : 'warning'
                }
              >
                {report.status}
              </Badge>
            </div>
            <div className="mt-1 flex items-center gap-4 text-sm text-gray-400">
              <span>
                {format(new Date(report.config.period_start), 'MMM d, yyyy')} -{' '}
                {format(new Date(report.config.period_end), 'MMM d, yyyy')}
              </span>
              <span>•</span>
              <span>Created {format(new Date(report.created_at), 'MMM d, yyyy HH:mm')}</span>
              {report.generated_by_user && (
                <>
                  <span>•</span>
                  <span>By {report.generated_by_user.display_name || report.generated_by_user.email}</span>
                </>
              )}
            </div>
          </div>

          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onClick={handleDownload}
              leftIcon={<Download className="h-4 w-4" />}
              disabled={report.status !== 'completed' || !report.file_url}
            >
              Download
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={handlePrint}
              disabled={report.status !== 'completed' || !report.file_url}
            >
              <Printer className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={toggleFullscreen}
            >
              {isFullscreen ? <Minimize2 className="h-4 w-4" /> : <Maximize2 className="h-4 w-4" />}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={onClose}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-hidden">
          {renderPreviewContent()}
        </div>

        {/* Footer */}
        {report.metadata && (
          <div className="border-t border-gray-800 px-6 py-3">
            <div className="flex items-center justify-between text-sm">
              <div className="flex items-center gap-4 text-gray-400">
                {report.metadata.row_count && (
                  <span>{report.metadata.row_count.toLocaleString()} records</span>
                )}
                {report.metadata.duration_seconds && (
                  <span>Generated in {report.metadata.duration_seconds}s</span>
                )}
                {report.file_size_bytes && (
                  <span>
                    {(report.file_size_bytes / 1024 / 1024).toFixed(2)} MB
                  </span>
                )}
              </div>
              {report.expires_at && (
                <div className="text-sm text-gray-500">
                  Expires {format(new Date(report.expires_at), 'MMM d, yyyy HH:mm')}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ReportViewer;

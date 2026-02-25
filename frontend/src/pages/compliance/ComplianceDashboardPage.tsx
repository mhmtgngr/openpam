/**
 * ComplianceDashboardPage Component
 *
 * Main compliance dashboard showing overall compliance scores,
 * control status, exceptions, and remediation recommendations.
 */

import React, { useState, useCallback, useEffect, useMemo } from 'react';
import {
  Shield,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  Filter,
  Download,
  RefreshCw,
  TrendingUp,
  TrendingDown,
  Minus,
} from 'lucide-react';
import { complianceApi } from '@/api/compliance';
import { useAnalytics } from '@/contexts/AnalyticsContext';
import { ComplianceGauge, ComplianceScoreCard, ComplianceMeter } from '@/components/analytics/ComplianceGauge';
import type { ComplianceFramework, ComplianceControl, ComplianceException } from '@/types';

interface TimeRangeOption {
  value: string;
  label: string;
}

const TIME_RANGES: TimeRangeOption[] = [
  { value: '30d', label: 'Last 30 Days' },
  { value: '90d', label: 'Last 90 Days' },
  { value: '180d', label: 'Last 6 Months' },
  { value: '365d', label: 'Last Year' },
];

const FRAMEWORKS: Array<{ value: ComplianceFramework; label: string; description: string }> = [
  { value: 'soc2', label: 'SOC 2', description: 'Service Organization Control 2' },
  { value: 'iso27001', label: 'ISO 27001', description: 'Information Security Management' },
  { value: 'pci_dss', label: 'PCI DSS', description: 'Payment Card Industry Data Security' },
  { value: 'hipaa', label: 'HIPAA', description: 'Health Insurance Portability and Accountability' },
  { value: 'gdpr', label: 'GDPR', description: 'General Data Protection Regulation' },
  { value: 'nerc_cip', label: 'NERC CIP', description: 'Critical Infrastructure Protection' },
];

export const ComplianceDashboardPage: React.FC = () => {
  const { complianceDashboard, selectedFramework, setSelectedFramework, refreshComplianceDashboard } = useAnalytics();

  const [loading, setLoading] = useState(false);
  const [showFrameworkSelector, setShowFrameworkSelector] = useState(false);

  // Handle framework change
  const handleFrameworkChange = useCallback((framework: ComplianceFramework) => {
    setSelectedFramework(framework);
    setShowFrameworkSelector(false);
    setLoading(true);
    setTimeout(() => setLoading(false), 1000);
  }, [setSelectedFramework]);

  // Handle refresh
  const handleRefresh = useCallback(async () => {
    setLoading(true);
    try {
      await refreshComplianceDashboard();
    } finally {
      setLoading(false);
    }
  }, [refreshComplianceDashboard]);

  // Calculate stats from dashboard data
  const stats = useMemo(() => {
    if (!complianceDashboard) {
      return {
        overallScore: 0,
        compliantCount: 0,
        nonCompliantCount: 0,
        partialCount: 0,
        notApplicableCount: 0,
        criticalFindings: 0,
        highFindings: 0,
        exceptionsActive: 0,
        trends: { score: 0, controls: 0 },
      };
    }

    const controls = complianceDashboard.controls || [];
    const exceptions = complianceDashboard.exceptions || [];

    const compliantCount = controls.filter((c) => c.status === 'compliant').length;
    const nonCompliantCount = controls.filter((c) => c.status === 'non_compliant').length;
    const partialCount = controls.filter((c) => c.status === 'partial').length;
    const notApplicableCount = controls.filter((c) => c.status === 'not_applicable').length;

    const criticalFindings = controls.filter((c) => c.status === 'non_compliant').length;

    const highFindings = controls.filter((c) => c.status === 'partial').length;

    const exceptionsActive = exceptions.filter((e) => e.status === 'active').length;

    return {
      overallScore: complianceDashboard.overall_score || 0,
      compliantCount,
      nonCompliantCount,
      partialCount,
      notApplicableCount,
      criticalFindings,
      highFindings,
      exceptionsActive,
      trends: { score: 5, controls: 3 }, // Mock trend data
    };
  }, [complianceDashboard]);

  // Group controls by category
  const controlsByCategory = useMemo(() => {
    if (!complianceDashboard?.controls) return [];

    const grouped = new Map<string, ComplianceControl[]>();
    complianceDashboard.controls.forEach((control) => {
      const category = control.category || 'Other';
      if (!grouped.has(category)) {
        grouped.set(category, []);
      }
      grouped.get(category)!.push(control);
    });

    return Array.from(grouped.entries()).map(([category, controls]) => {
      const avgScore = controls.reduce((sum, c) => sum + c.score, 0) / controls.length;
      const compliantCount = controls.filter((c) => c.status === 'compliant').length;
      return {
        category,
        avgScore: Math.round(avgScore),
        controlCount: controls.length,
        compliantCount,
        controls,
      };
    }).sort((a, b) => a.avgScore - b.avgScore);
  }, [complianceDashboard]);

  const currentFramework = FRAMEWORKS.find((f) => f.value === selectedFramework) || FRAMEWORKS[0];

  return (
    <div className="compliance-dashboard-page">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-3">
            <Shield className="text-blue-600" size={32} />
            <div>
              <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
                Compliance Dashboard
              </h1>
              <p className="text-gray-600 dark:text-gray-400">
                Track and manage compliance posture across frameworks
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Framework Selector */}
          <div className="relative">
            <button
              onClick={() => setShowFrameworkSelector(!showFrameworkSelector)}
              className="px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg flex items-center gap-2 hover:bg-gray-50 dark:hover:bg-gray-700"
            >
              <Shield size={18} />
              {currentFramework.label}
            </button>

            {showFrameworkSelector && (
              <div className="absolute right-0 mt-2 w-72 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-10">
                <div className="p-2">
                  {FRAMEWORKS.map((fw) => (
                    <button
                      key={fw.value}
                      onClick={() => handleFrameworkChange(fw.value)}
                      className={`w-full text-left px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 ${
                        fw.value === selectedFramework ? 'bg-blue-50 dark:bg-blue-900/20 text-blue-600' : ''
                      }`}
                    >
                      <div className="font-medium">{fw.label}</div>
                      <div className="text-xs text-gray-500">{fw.description}</div>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>

          <button
            onClick={handleRefresh}
            disabled={loading}
            className="px-4 py-2 bg-white dark:bg-gray-800 border border-gray-300 dark:border-gray-600 rounded-lg flex items-center gap-2 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50"
          >
            <RefreshCw size={18} className={loading ? 'animate-spin' : ''} />
            Refresh
          </button>

          <button
            onClick={() => {/* TODO: Implement export */}}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <Download size={18} />
            Export Report
          </button>
        </div>
      </div>

      {loading && !complianceDashboard ? (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
        </div>
      ) : (
        <div className="space-y-6">
          {/* Score Cards */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Overall Score */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Overall Score</h3>
                {stats.trends.score > 0 ? (
                  <TrendingUp size={16} className="text-green-500" />
                ) : stats.trends.score < 0 ? (
                  <TrendingDown size={16} className="text-red-500" />
                ) : (
                  <Minus size={16} className="text-gray-500" />
                )}
              </div>
              <div className="flex items-center justify-center">
                <ComplianceGauge score={stats.overallScore} size="lg" />
              </div>
            </div>

            {/* Compliant Controls */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Compliant</h3>
                <CheckCircle size={18} className="text-green-500" />
              </div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {stats.compliantCount}
              </div>
              <div className="text-sm text-gray-500 mt-1">
                of {complianceDashboard?.controls?.length || 0} controls
              </div>
              <ComplianceMeter score={
                complianceDashboard?.controls?.length
                  ? (stats.compliantCount / complianceDashboard.controls.length) * 100
                  : 0
              } className="mt-3" />
            </div>

            {/* Non-Compliant Controls */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Non-Compliant</h3>
                <XCircle size={18} className="text-red-500" />
              </div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {stats.nonCompliantCount}
              </div>
              <div className="text-sm text-gray-500 mt-1">
                controls require attention
              </div>
              {stats.nonCompliantCount > 0 && (
                <button className="text-sm text-blue-600 hover:text-blue-700 mt-3">
                  View Details →
                </button>
              )}
            </div>

            {/* Active Exceptions */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <div className="flex items-center justify-between mb-2">
                <h3 className="text-sm font-medium text-gray-600 dark:text-gray-400">Exceptions</h3>
                <AlertTriangle size={18} className="text-yellow-500" />
              </div>
              <div className="text-3xl font-bold text-gray-900 dark:text-white">
                {stats.exceptionsActive}
              </div>
              <div className="text-sm text-gray-500 mt-1">
                active exceptions
              </div>
              {stats.exceptionsActive > 0 && (
                <button className="text-sm text-blue-600 hover:text-blue-700 mt-3">
                  Manage →
                </button>
              )}
            </div>
          </div>

          {/* Findings Summary */}
          {(stats.criticalFindings > 0 || stats.highFindings > 0) && (
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
              <div className="flex items-center gap-3">
                <AlertTriangle size={24} className="text-red-600 dark:text-red-400" />
                <div className="flex-1">
                  <h3 className="font-semibold text-red-900 dark:text-red-200">
                    Action Required: {stats.criticalFindings + stats.highFindings} High-Priority Findings
                  </h3>
                  <p className="text-sm text-red-700 dark:text-red-300">
                    {stats.criticalFindings} critical, {stats.highFindings} high severity findings need remediation
                  </p>
                </div>
                <button className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700">
                  Review Findings
                </button>
              </div>
            </div>
          )}

          {/* Category Breakdown */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            {/* Scores by Category */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Scores by Category
              </h3>
              <div className="space-y-4">
                {controlsByCategory.map((cat) => (
                  <div key={cat.category} className="flex items-center gap-4">
                    <div className="flex-1">
                      <div className="flex justify-between text-sm mb-1">
                        <span className="text-gray-700 dark:text-gray-300">{cat.category}</span>
                        <span className="font-medium text-gray-900 dark:text-white">
                          {cat.avgScore}%
                        </span>
                      </div>
                      <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                        <div
                          className={`h-2 rounded-full transition-all ${
                            cat.avgScore >= 90 ? 'bg-green-500' : cat.avgScore >= 70 ? 'bg-yellow-500' : 'bg-red-500'
                          }`}
                          style={{ width: `${cat.avgScore}%` }}
                        />
                      </div>
                      <div className="text-xs text-gray-500 mt-1">
                        {cat.compliantCount} of {cat.controlCount} compliant
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Recent Activity */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Recent Activity
              </h3>
              <div className="space-y-3">
                <div className="flex items-start gap-3">
                  <CheckCircle size={16} className="text-green-500 mt-1" />
                  <div>
                    <p className="text-sm text-gray-900 dark:text-white">
                      Control AC-001 marked as compliant
                    </p>
                    <p className="text-xs text-gray-500">2 hours ago</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <AlertTriangle size={16} className="text-yellow-500 mt-1" />
                  <div>
                    <p className="text-sm text-gray-900 dark:text-white">
                      New finding detected for Control AC-003
                    </p>
                    <p className="text-xs text-gray-500">5 hours ago</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <Clock size={16} className="text-blue-500 mt-1" />
                  <div>
                    <p className="text-sm text-gray-900 dark:text-white">
                      Exception request submitted for Control AC-007
                    </p>
                    <p className="text-xs text-gray-500">1 day ago</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <RefreshCw size={16} className="text-gray-500 mt-1" />
                  <div>
                    <p className="text-sm text-gray-900 dark:text-white">
                      Assessment completed for {currentFramework.label}
                    </p>
                    <p className="text-xs text-gray-500">2 days ago</p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Control List */}
          <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
            <div className="p-6 border-b border-gray-200 dark:border-gray-700">
              <div className="flex items-center justify-between">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  Controls
                </h3>
                <div className="flex items-center gap-2">
                  <button className="px-3 py-1 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 rounded-lg flex items-center gap-1">
                    <Filter size={14} />
                    Filter
                  </button>
                </div>
              </div>
            </div>

            <div className="divide-y divide-gray-200 dark:divide-gray-700">
              {complianceDashboard?.controls?.slice(0, 10).map((control) => (
                <ControlRow key={control.id} control={control} />
              ))}
            </div>

            {complianceDashboard?.controls && complianceDashboard.controls.length > 10 && (
              <div className="p-4 text-center">
                <button className="text-blue-600 hover:text-blue-700 text-sm">
                  View all {complianceDashboard.controls.length} controls →
                </button>
              </div>
            )}
          </div>

          {/* Active Exceptions */}
          {stats.exceptionsActive > 0 && (
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6">
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">
                Active Exceptions
              </h3>
              <div className="space-y-3">
                {complianceDashboard?.exceptions
                  ?.filter((e) => e.status === 'active')
                  .slice(0, 5)
                  .map((exception) => (
                    <ExceptionRow key={exception.id} exception={exception} />
                  ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

interface ControlRowProps {
  control: ComplianceControl;
}

const ControlRow: React.FC<ControlRowProps> = ({ control }) => {
  const statusConfig = {
    compliant: { icon: CheckCircle, color: 'text-green-500', bg: 'bg-green-50 dark:bg-green-900/20' },
    non_compliant: { icon: XCircle, color: 'text-red-500', bg: 'bg-red-50 dark:bg-red-900/20' },
    partial: { icon: Minus, color: 'text-yellow-500', bg: 'bg-yellow-50 dark:bg-yellow-900/20' },
    not_applicable: { icon: Minus, color: 'text-gray-500', bg: 'bg-gray-50 dark:bg-gray-900/20' },
  };

  const config = statusConfig[control.status];
  const StatusIcon = config.icon;

  return (
    <div className="p-4 hover:bg-gray-50 dark:hover:bg-gray-750 flex items-center gap-4">
      <StatusIcon size={20} className={config.color} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <h4 className="font-medium text-gray-900 dark:text-white">
            {control.name}
          </h4>
          <span className={`text-xs px-2 py-0.5 rounded ${config.bg} ${config.color} capitalize`}>
            {control.status.replace('_', ' ')}
          </span>
        </div>
        {control.description && (
          <p className="text-sm text-gray-500 truncate">{control.description}</p>
        )}
        {control.category && (
          <span className="text-xs text-gray-400">{control.category}</span>
        )}
      </div>
      <div className="text-right">
        <div className="text-lg font-semibold text-gray-900 dark:text-white">
          {control.score}%
        </div>
        <div className="text-xs text-gray-500">{control.evidence_count || 0} evidence</div>
      </div>
    </div>
  );
};

interface ExceptionRowProps {
  exception: ComplianceException;
}

const ExceptionRow: React.FC<ExceptionRowProps> = ({ exception }) => {
  const isExpiringSoon = exception.expires_at
    ? new Date(exception.expires_at) < new Date(Date.now() + 30 * 24 * 60 * 60 * 1000)
    : false;

  return (
    <div className={`p-4 rounded-lg border ${
      isExpiringSoon
        ? 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800'
        : 'bg-gray-50 dark:bg-gray-800/50 border-gray-200 dark:border-gray-700'
    }`}>
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h4 className="font-medium text-gray-900 dark:text-white">
              {exception.control_name}
            </h4>
            {isExpiringSoon && (
              <span className="text-xs px-2 py-0.5 bg-yellow-200 dark:bg-yellow-800 text-yellow-800 dark:text-yellow-200 rounded">
                Expiring Soon
              </span>
            )}
          </div>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
            {exception.reason}
          </p>
          {exception.expires_at && (
            <p className="text-xs text-gray-500 mt-2">
              Expires: {new Date(exception.expires_at).toLocaleDateString()}
            </p>
          )}
        </div>
        <button className="text-blue-600 hover:text-blue-700 text-sm">
          View Details
        </button>
      </div>
    </div>
  );
};

export default ComplianceDashboardPage;

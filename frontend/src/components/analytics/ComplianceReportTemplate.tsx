/**
 * ComplianceReportTemplate - Template-based report generation for different compliance frameworks
 * Supports SOC2, ISO27001, PCI DSS, HIPAA, and GDPR frameworks
 */

import React, { useState } from 'react';
import { Shield, CheckCircle, XCircle, AlertTriangle, HelpCircle } from 'lucide-react';
import { Card, CardHeader, CardFooter } from '@/components/common';
import { Badge } from '@/components/common';
import { Button } from '@/components/common';
import { Select } from '@/components/common';
import { Toggle } from '@/components/common';
import { Input } from '@/components/common';
import { Textarea } from '@/components/common';
import { DatePicker } from '@/components/common';
import clsx from 'clsx';
import type { ComplianceFramework, ReportFormat } from '@/types';
import type { ReportConfig, ReportTemplateSection } from '@/types/reports';

interface ComplianceReportTemplateProps {
  framework: ComplianceFramework;
  config: ReportConfig;
  onConfigChange: (config: ReportConfig) => void;
  onGenerate?: () => void;
  isGenerating?: boolean;
}

interface FrameworkConfig {
  name: string;
  description: string;
  icon: React.ElementType;
  color: string;
  sections: ReportTemplateSection[];
  defaultIncludeSections: string[];
}

const frameworkConfigs: Record<ComplianceFramework, FrameworkConfig> = {
  soc2: {
    name: 'SOC 2 Type II',
    description: 'Service Organization Control 2 - Trust Services Criteria',
    icon: Shield,
    color: 'text-blue-400',
    sections: [
      { id: 'executive_summary', name: 'executive_summary', title: 'Executive Summary', type: 'summary', required: true, order: 1, config: {} },
      { id: 'security', name: 'security', title: 'Security Controls', type: 'table', required: true, order: 2, config: {} },
      { id: 'availability', name: 'availability', title: 'Availability Monitoring', type: 'table', required: true, order: 3, config: {} },
      { id: 'processing_integrity', name: 'processing_integrity', title: 'Processing Integrity', type: 'table', required: false, order: 4, config: {} },
      { id: 'confidentiality', name: 'confidentiality', title: 'Confidentiality Measures', type: 'table', required: true, order: 5, config: {} },
      { id: 'privacy', name: 'privacy', title: 'Privacy Practices', type: 'table', required: false, order: 6, config: {} },
      { id: 'incident_log', name: 'incident_log', title: 'Security Incidents', type: 'table', required: true, order: 7, config: {} },
      { id: 'change_log', name: 'change_log', title: 'System Changes', type: 'table', required: true, order: 8, config: {} },
      { id: 'access_review', name: 'access_review', title: 'Access Reviews', type: 'table', required: true, order: 9, config: {} },
      { id: 'evidence_appendix', name: 'evidence_appendix', title: 'Evidence Appendix', type: 'text', required: false, order: 10, config: {} },
    ],
    defaultIncludeSections: ['executive_summary', 'security', 'availability', 'confidentiality', 'incident_log', 'change_log', 'access_review'],
  },
  iso27001: {
    name: 'ISO 27001:2022',
    description: 'Information Security Management System',
    icon: Shield,
    color: 'text-green-400',
    sections: [
      { id: 'executive_summary', name: 'executive_summary', title: 'Executive Summary', type: 'summary', required: true, order: 1, config: {} },
      { id: 'scope', name: 'scope', title: 'ISMS Scope', type: 'text', required: true, order: 2, config: {} },
      { id: 'info_security_policy', name: 'info_security_policy', title: 'Information Security Policy', type: 'text', required: true, order: 3, config: {} },
      { id: 'risk_assessment', name: 'risk_assessment', title: 'Risk Assessment', type: 'table', required: true, order: 4, config: {} },
      { id: 'asset_management', name: 'asset_management', title: 'Asset Management', type: 'table', required: true, order: 5, config: {} },
      { id: 'access_control', name: 'access_control', title: 'Access Control', type: 'table', required: true, order: 6, config: {} },
      { id: 'cryptography', name: 'cryptography', title: 'Cryptography', type: 'table', required: true, order: 7, config: {} },
      { id: 'physical_security', name: 'physical_security', title: 'Physical Security', type: 'table', required: true, order: 8, config: {} },
      { id: 'operations_security', name: 'operations_security', title: 'Operations Security', type: 'table', required: true, order: 9, config: {} },
      { id: 'compliance', name: 'compliance', title: 'Compliance Evaluation', type: 'table', required: true, order: 10, config: {} },
    ],
    defaultIncludeSections: ['executive_summary', 'scope', 'info_security_policy', 'risk_assessment', 'access_control', 'compliance'],
  },
  pci_dss: {
    name: 'PCI DSS 4.0',
    description: 'Payment Card Industry Data Security Standard',
    icon: Shield,
    color: 'text-purple-400',
    sections: [
      { id: 'executive_summary', name: 'executive_summary', title: 'Executive Summary', type: 'summary', required: true, order: 1, config: {} },
      { id: 'network_security', name: 'network_security', title: 'Network Security', type: 'table', required: true, order: 2, config: {} },
      { id: 'data_protection', name: 'data_protection', title: 'Data Protection', type: 'table', required: true, order: 3, config: {} },
      { id: 'vulnerability_management', name: 'vulnerability_management', title: 'Vulnerability Management', type: 'table', required: true, order: 4, config: {} },
      { id: 'access_control', name: 'access_control', title: 'Access Control', type: 'table', required: true, order: 5, config: {} },
      { id: 'monitoring', name: 'monitoring', title: 'Monitoring & Testing', type: 'table', required: true, order: 6, config: {} },
      { id: 'policy', name: 'policy', title: 'Information Security Policy', type: 'text', required: true, order: 7, config: {} },
    ],
    defaultIncludeSections: ['executive_summary', 'network_security', 'data_protection', 'access_control', 'monitoring'],
  },
  hipaa: {
    name: 'HIPAA Security Rule',
    description: 'Health Insurance Portability and Accountability Act',
    icon: Shield,
    color: 'text-red-400',
    sections: [
      { id: 'executive_summary', name: 'executive_summary', title: 'Executive Summary', type: 'summary', required: true, order: 1, config: {} },
      { id: 'administrative_safeguards', name: 'administrative_safeguards', title: 'Administrative Safeguards', type: 'table', required: true, order: 2, config: {} },
      { id: 'physical_safeguards', name: 'physical_safeguards', title: 'Physical Safeguards', type: 'table', required: true, order: 3, config: {} },
      { id: 'technical_safeguards', name: 'technical_safeguards', title: 'Technical Safeguards', type: 'table', required: true, order: 4, config: {} },
      { id: 'phi_inventory', name: 'phi_inventory', title: 'PHI Inventory', type: 'table', required: true, order: 5, config: {} },
      { id: 'baa_tracking', name: 'baa_tracking', title: 'BAA Tracking', type: 'table', required: true, order: 6, config: {} },
      { id: 'incident_breach', name: 'incident_breach', title: 'Incidents & Breaches', type: 'table', required: true, order: 7, config: {} },
    ],
    defaultIncludeSections: ['executive_summary', 'administrative_safeguards', 'physical_safeguards', 'technical_safeguards'],
  },
  gdpr: {
    name: 'GDPR Compliance',
    description: 'General Data Protection Regulation',
    icon: Shield,
    color: 'text-yellow-400',
    sections: [
      { id: 'executive_summary', name: 'executive_summary', title: 'Executive Summary', type: 'summary', required: true, order: 1, config: {} },
      { id: 'data_inventory', name: 'data_inventory', title: 'Data Inventory', type: 'table', required: true, order: 2, config: {} },
      { id: 'lawful_basis', name: 'lawful_basis', title: 'Lawful Basis for Processing', type: 'table', required: true, order: 3, config: {} },
      { id: 'data_subject_rights', name: 'data_subject_rights', title: 'Data Subject Rights', type: 'table', required: true, order: 4, config: {} },
      { id: 'data_transfers', name: 'data_transfers', title: 'International Data Transfers', type: 'table', required: false, order: 5, config: {} },
      { id: 'dpia_records', name: 'dpia_records', title: 'DPIA Records', type: 'table', required: false, order: 6, config: {} },
      { id: 'breach_log', name: 'breach_log', title: 'Personal Data Breaches', type: 'table', required: true, order: 7, config: {} },
      { id: 'processors', name: 'processors', title: 'Data Processors', type: 'table', required: true, order: 8, config: {} },
    ],
    defaultIncludeSections: ['executive_summary', 'data_inventory', 'lawful_basis', 'data_subject_rights', 'breach_log'],
  },
  custom: {
    name: 'Custom Framework',
    description: 'Build your own compliance framework report',
    icon: Shield,
    color: 'text-gray-400',
    sections: [],
    defaultIncludeSections: [],
  },
};

interface ControlStatus {
  id: string;
  name: string;
  status: 'compliant' | 'non_compliant' | 'partial' | 'not_applicable';
  evidence_count: number;
  last_tested: string;
}

const mockControls: Record<ComplianceFramework, ControlStatus[]> = {
  soc2: [
    { id: 'CC1.1', name: 'Control Environment', status: 'compliant', evidence_count: 12, last_tested: '2024-01-15' },
    { id: 'CC2.1', name: 'Communication & Information', status: 'compliant', evidence_count: 8, last_tested: '2024-01-15' },
    { id: 'CC3.1', name: 'Risk Assessment', status: 'partial', evidence_count: 15, last_tested: '2024-01-10' },
    { id: 'CC4.1', name: 'Monitoring Activities', status: 'compliant', evidence_count: 20, last_tested: '2024-01-15' },
    { id: 'CC5.1', name: 'Control Activities', status: 'compliant', evidence_count: 18, last_tested: '2024-01-15' },
    { id: 'CC6.1', name: 'Logical Access', status: 'partial', evidence_count: 25, last_tested: '2024-01-12' },
  ],
  iso27001: [
    { id: 'A.5.1', name: 'Policies for Information Security', status: 'compliant', evidence_count: 5, last_tested: '2024-01-15' },
    { id: 'A.8.2', name: 'Privileged Access Rights', status: 'partial', evidence_count: 12, last_tested: '2024-01-10' },
    { id: 'A.12.3', name: 'Backup', status: 'compliant', evidence_count: 8, last_tested: '2024-01-15' },
    { id: 'A.14.2', name: 'Secure Development', status: 'non_compliant', evidence_count: 3, last_tested: '2024-01-08' },
  ],
  pci_dss: [
    { id: '1.1', name: 'Firewall Configuration', status: 'compliant', evidence_count: 6, last_tested: '2024-01-15' },
    { id: '3.1', name: 'Cardholder Data Protection', status: 'compliant', evidence_count: 10, last_tested: '2024-01-15' },
    { id: '4.1', name: 'Encryption', status: 'compliant', evidence_count: 8, last_tested: '2024-01-15' },
    { id: '7.1', name: 'Access Control', status: 'partial', evidence_count: 15, last_tested: '2024-01-12' },
  ],
  hipaa: [
    { id: '164.308(a)', name: 'Security Management Process', status: 'compliant', evidence_count: 9, last_tested: '2024-01-15' },
    { id: '164.310(d)', name: 'Workstation Security', status: 'compliant', evidence_count: 6, last_tested: '2024-01-15' },
    { id: '164.312(b)', name: 'Audit Controls', status: 'partial', evidence_count: 11, last_tested: '2024-01-10' },
  ],
  gdpr: [
    { id: 'Art.32', name: 'Security of Processing', status: 'compliant', evidence_count: 14, last_tested: '2024-01-15' },
    { id: 'Art.30', name: 'Records of Processing', status: 'partial', evidence_count: 7, last_tested: '2024-01-12' },
    { id: 'Art.33', name: 'Breach Notification', status: 'compliant', evidence_count: 5, last_tested: '2024-01-15' },
  ],
  custom: [],
};

const statusIcons = {
  compliant: CheckCircle,
  non_compliant: XCircle,
  partial: AlertTriangle,
  not_applicable: HelpCircle,
};

const statusColors = {
  compliant: 'text-success-400 bg-success-500/20',
  non_compliant: 'text-danger-400 bg-danger-500/20',
  partial: 'text-warning-400 bg-warning-500/20',
  not_applicable: 'text-gray-400 bg-gray-500/20',
};

export const ComplianceReportTemplate: React.FC<ComplianceReportTemplateProps> = ({
  framework,
  config,
  onConfigChange,
  onGenerate,
  isGenerating = false,
}) => {
  const frameworkConfig = frameworkConfigs[framework] || frameworkConfigs.custom;
  const FrameworkIcon = frameworkConfig.icon;
  const controls = mockControls[framework] || [];

  const [selectedSections, setSelectedSections] = useState<string[]>(
    config.include_sections || frameworkConfig.defaultIncludeSections
  );

  const handleToggleSection = (sectionId: string) => {
    const newSections = selectedSections.includes(sectionId)
      ? selectedSections.filter((id) => id !== sectionId)
      : [...selectedSections, sectionId];

    setSelectedSections(newSections);
    onConfigChange({
      ...config,
      include_sections: newSections,
    });
  };

  const handleSelectAll = () => {
    const allSectionIds = frameworkConfig.sections.map((s) => s.id);
    setSelectedSections(allSectionIds);
    onConfigChange({
      ...config,
      include_sections: allSectionIds,
    });
  };

  const handleSelectRequired = () => {
    const requiredIds = frameworkConfig.sections.filter((s) => s.required).map((s) => s.id);
    setSelectedSections(requiredIds);
    onConfigChange({
      ...config,
      include_sections: requiredIds,
    });
  };

  const handleClearAll = () => {
    setSelectedSections([]);
    onConfigChange({
      ...config,
      include_sections: [],
    });
  };

  const overallScore = controls.length > 0
    ? Math.round(
        (controls.filter((c) => c.status === 'compliant').length / controls.length) * 100
      )
    : 0;

  return (
    <div className="space-y-6">
      {/* Framework Header */}
      <Card>
        <div className="card-body">
          <div className="flex items-start gap-4">
            <div className={clsx('rounded-lg p-3', frameworkConfig.color.replace('text-', 'bg-').replace('-400', '-500/20'))}>
              <FrameworkIcon className={clsx('h-8 w-8', frameworkConfig.color)} />
            </div>
            <div className="flex-1">
              <h3 className="text-xl font-semibold text-white">{frameworkConfig.name}</h3>
              <p className="mt-1 text-sm text-gray-400">{frameworkConfig.description}</p>
              <div className="mt-3 flex items-center gap-4">
                <div className="flex items-center gap-2">
                  <span className="text-sm text-gray-400">Compliance Score:</span>
                  <Badge variant={overallScore >= 80 ? 'success' : overallScore >= 60 ? 'warning' : 'danger'}>
                    {overallScore}%
                  </Badge>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-sm text-gray-400">Controls:</span>
                  <span className="text-sm text-white">{controls.length}</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* Control Status Summary */}
      {controls.length > 0 && (
        <Card>
          <CardHeader title="Control Status" subtitle="Compliance status for framework controls" />
          <div className="card-body">
            <div className="space-y-2">
              {controls.map((control) => {
                const StatusIcon = statusIcons[control.status];
                return (
                  <div
                    key={control.id}
                    className="flex items-center justify-between rounded-lg border border-gray-800 p-3 hover:bg-gray-800/50"
                  >
                    <div className="flex items-center gap-3">
                      <div className={clsx('rounded-md p-2', statusColors[control.status])}>
                        <StatusIcon className="h-4 w-4" />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-white">{control.id}</span>
                          <span className="text-sm text-gray-400">{control.name}</span>
                        </div>
                        <div className="mt-1 text-xs text-gray-500">
                          {control.evidence_count} evidence items • Last tested {control.last_tested}
                        </div>
                      </div>
                    </div>
                    <Badge
                      variant={
                        control.status === 'compliant'
                          ? 'success'
                          : control.status === 'non_compliant'
                          ? 'danger'
                          : 'warning'
                      }
                    >
                      {control.status.replace('_', ' ')}
                    </Badge>
                  </div>
                );
              })}
            </div>
          </div>
        </Card>
      )}

      {/* Report Sections */}
      {frameworkConfig.sections.length > 0 && (
        <Card>
          <CardHeader
            title="Report Sections"
            subtitle="Select which sections to include in the generated report"
            action={
              <div className="flex items-center gap-2">
                <Button variant="secondary" size="sm" onClick={handleSelectRequired}>
                  Required
                </Button>
                <Button variant="secondary" size="sm" onClick={handleSelectAll}>
                  Select All
                </Button>
                <Button variant="ghost" size="sm" onClick={handleClearAll}>
                  Clear
                </Button>
              </div>
            }
          />
          <div className="card-body">
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {frameworkConfig.sections.map((section) => {
                const isSelected = selectedSections.includes(section.id);
                return (
                  <button
                    key={section.id}
                    type="button"
                    onClick={() => handleToggleSection(section.id)}
                    className={clsx(
                      'flex items-start gap-3 rounded-lg border p-4 text-left transition-colors',
                      isSelected
                        ? 'border-primary-500 bg-primary-500/10'
                        : 'border-gray-800 hover:bg-gray-800/50'
                    )}
                  >
                    <div className="mt-0.5">
                      <input
                        type="checkbox"
                        checked={isSelected}
                        onChange={() => {}}
                        className="h-4 w-4 rounded border-gray-600 bg-gray-800 text-primary-500 focus:ring-primary-500"
                      />
                    </div>
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-white">{section.title}</span>
                        {section.required && (
                          <Badge variant="neutral" size="sm">Required</Badge>
                        )}
                      </div>
                      <p className="mt-1 text-xs text-gray-500 capitalize">{section.type}</p>
                    </div>
                  </button>
                );
              })}
            </div>
          </div>
          <CardFooter className="flex items-center justify-between">
            <span className="text-sm text-gray-400">
              {selectedSections.length} of {frameworkConfig.sections.length} sections selected
            </span>
          </CardFooter>
        </Card>
      )}

      {/* Generate Button */}
      {onGenerate && (
        <div className="flex justify-end">
          <Button
            variant="primary"
            onClick={onGenerate}
            isLoading={isGenerating}
            disabled={selectedSections.length === 0}
          >
            Generate Report
          </Button>
        </div>
      )}
    </div>
  );
};

export default ComplianceReportTemplate;

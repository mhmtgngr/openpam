/**
 * TemplateEditor Component
 *
 * Visual editor for creating and modifying report templates.
 * Supports drag-and-drop section ordering, section configuration, and preview.
 */

import React, { useState, useCallback, useMemo } from 'react';
import { GripVertical, Plus, Trash2, Eye, Settings, ChevronDown, ChevronUp } from 'lucide-react';
import type { ReportTemplate, ReportTemplateSection, ReportFormat, ComplianceFramework } from '@/types/reports';
import type { ReportType } from '@/types';

interface TemplateEditorProps {
  template?: Partial<ReportTemplate>;
  onSave: (template: Partial<ReportTemplate>) => void;
  onCancel?: () => void;
  readOnly?: boolean;
  className?: string;
}

interface SectionConfig {
  id: string;
  name: string;
  title: string;
  type: 'table' | 'chart' | 'summary' | 'text' | 'heatmap';
  required: boolean;
  config: Record<string, unknown>;
  order: number;
  expanded: boolean;
}

// Available section types with default configurations
const AVAILABLE_SECTIONS: Omit<SectionConfig, 'id' | 'order' | 'expanded'>[] = [
  {
    name: 'executive_summary',
    title: 'Executive Summary',
    type: 'summary',
    required: true,
    config: {
      include_score: true,
      include_findings: true,
      include_recommendations: true,
    },
  },
  {
    name: 'compliance_overview',
    title: 'Compliance Overview',
    type: 'chart',
    required: true,
    config: {
      chart_type: 'gauge',
      include_trend: true,
    },
  },
  {
    name: 'control_status',
    title: 'Control Status',
    type: 'table',
    required: true,
    config: {
      columns: ['name', 'status', 'score', 'last_assessed'],
      group_by_category: true,
    },
  },
  {
    name: 'findings_detail',
    title: 'Findings Detail',
    type: 'table',
    required: false,
    config: {
      columns: ['severity', 'title', 'description', 'affected_resources', 'status'],
      group_by_severity: true,
    },
  },
  {
    name: 'session_activity',
    title: 'Session Activity',
    type: 'chart',
    required: false,
    config: {
      chart_type: 'line',
      time_grouping: 'day',
    },
  },
  {
    name: 'user_activity',
    title: 'User Activity Heatmap',
    type: 'heatmap',
    required: false,
    config: {
      metric: 'session_count',
    },
  },
  {
    name: 'command_analysis',
    title: 'Command Analysis',
    type: 'table',
    required: false,
    config: {
      include_risk_level: true,
      top_n: 20,
    },
  },
  {
    name: 'anomaly_summary',
    title: 'Anomaly Summary',
    type: 'table',
    required: false,
    config: {
      group_by_type: true,
      include_status: true,
    },
  },
  {
    name: 'exceptions',
    title: 'Compliance Exceptions',
    type: 'table',
    required: false,
    config: {
      include_status: true,
      include_expiration: true,
    },
  },
  {
    name: 'recommendations',
    title: 'Recommendations',
    type: 'text',
    required: false,
    config: {
      format: 'bullet',
      max_items: 10,
    },
  },
  {
    name: 'audit_trail',
    title: 'Audit Trail',
    type: 'table',
    required: false,
    config: {
      columns: ['timestamp', 'actor', 'action', 'resource', 'outcome'],
      limit: 100,
    },
  },
];

export const TemplateEditor: React.FC<TemplateEditorProps> = ({
  template,
  onSave,
  onCancel,
  readOnly = false,
  className = '',
}) => {
  const [name, setName] = useState(template?.name || '');
  const [description, setDescription] = useState(template?.description || '');
  const [reportType, setReportType] = useState<ReportType>(template?.type || 'compliance');
  const [framework, setFramework] = useState<ComplianceFramework>(template?.framework || 'soc2');
  const [sections, setSections] = useState<SectionConfig[]>(() => {
    const existingSections = template?.sections || [];
    return AVAILABLE_SECTIONS.map((section, index) => {
      const existing = existingSections.find((s) => s.name === section.name);
      return {
        ...section,
        id: existing?.id || `section_${index}`,
        order: existing?.order ?? index,
        expanded: false,
        config: existing?.config || section.config,
      };
    });
  });
  const [showPreview, setShowPreview] = useState(false);

  // Filter sections to only show selected ones
  const selectedSections = useMemo(() => {
    return sections.filter((s) => {
      const existing = template?.sections?.find((es) => es.name === s.name);
      return s.required || existing !== undefined;
    });
  }, [sections, template]);

  // Toggle section expansion
  const toggleExpanded = useCallback((id: string) => {
    setSections((prev) =>
      prev.map((s) => (s.id === id ? { ...s, expanded: !s.expanded } : s))
    );
  }, []);

  // Move section up
  const moveUp = useCallback((id: string) => {
    setSections((prev) => {
      const index = prev.findIndex((s) => s.id === id);
      if (index <= 0) return prev;
      const newSections = [...prev];
      [newSections[index - 1], newSections[index]] = [newSections[index], newSections[index - 1]];
      return newSections.map((s, i) => ({ ...s, order: i }));
    });
  }, []);

  // Move section down
  const moveDown = useCallback((id: string) => {
    setSections((prev) => {
      const index = prev.findIndex((s) => s.id === id);
      if (index >= prev.length - 1) return prev;
      const newSections = [...prev];
      [newSections[index], newSections[index + 1]] = [newSections[index + 1], newSections[index]];
      return newSections.map((s, i) => ({ ...s, order: i }));
    });
  }, []);

  // Remove section (if not required)
  const removeSection = useCallback((id: string) => {
    setSections((prev) =>
      prev.map((s) => (s.id === id && !s.required ? { ...s, order: -1 } : s))
    );
  }, []);

  // Add section back
  const addSection = useCallback((name: string) => {
    setSections((prev) => {
      const maxOrder = Math.max(...prev.filter((s) => s.order >= 0).map((s) => s.order), -1);
      return prev.map((s) => (s.name === name ? { ...s, order: maxOrder + 1 } : s));
    });
  }, []);

  // Update section config
  const updateSectionConfig = useCallback((id: string, config: Record<string, unknown>) => {
    setSections((prev) =>
      prev.map((s) => (s.id === id ? { ...s, config: { ...s.config, ...config } } : s))
    );
  }, []);

  // Handle save
  const handleSave = useCallback(() => {
    const templateSections: ReportTemplateSection[] = sections
      .filter((s) => s.order >= 0)
      .map(({ id, name, title, type, required, config, order }) => ({
        id,
        name,
        title,
        type,
        required,
        config,
        order,
      }))
      .sort((a, b) => a.order - b.order);

    onSave({
      name,
      description,
      type: reportType,
      framework,
      sections: templateSections,
      config: template?.config || {
        period_start: '',
        period_end: '',
        include_sections: templateSections.map((s) => s.name),
        filters: {},
      },
    });
  }, [name, description, reportType, framework, sections, template, onSave]);

  return (
    <div className={`template-editor ${className}`}>
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
          {template?.id ? 'Edit Template' : 'Create Template'}
        </h2>
        <div className="flex gap-2">
          <button
            onClick={() => setShowPreview(!showPreview)}
            className="px-3 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 rounded-lg flex items-center gap-2"
          >
            <Eye size={16} />
            {showPreview ? 'Edit' : 'Preview'}
          </button>
          {onCancel && (
            <button
              onClick={onCancel}
              className="px-4 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-700 rounded-lg"
            >
              Cancel
            </button>
          )}
          {!readOnly && (
            <button
              onClick={handleSave}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Save Template
            </button>
          )}
        </div>
      </div>

      {showPreview ? (
        <TemplatePreview
          name={name}
          description={description}
          reportType={reportType}
          framework={framework}
          sections={selectedSections.sort((a, b) => a.order - b.order)}
        />
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Template Settings */}
          <div className="lg:col-span-1 space-y-6">
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
              <h3 className="font-medium text-gray-900 dark:text-white mb-4">Template Settings</h3>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Template Name
                  </label>
                  <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    disabled={readOnly}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    placeholder="My Compliance Report"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Description
                  </label>
                  <textarea
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    disabled={readOnly}
                    rows={3}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    placeholder="Describe what this report is for..."
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Report Type
                  </label>
                  <select
                    value={reportType}
                    onChange={(e) => setReportType(e.target.value as ReportType)}
                    disabled={readOnly}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  >
                    <option value="compliance">Compliance Report</option>
                    <option value="session_activity">Session Activity</option>
                    <option value="command_analysis">Command Analysis</option>
                    <option value="user_access">User Access</option>
                    <option value="anomaly_summary">Anomaly Summary</option>
                    <option value="audit_trail">Audit Trail</option>
                    <option value="credential_usage">Credential Usage</option>
                    <option value="custom">Custom</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                    Framework
                  </label>
                  <select
                    value={framework}
                    onChange={(e) => setFramework(e.target.value as ComplianceFramework)}
                    disabled={readOnly}
                    className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  >
                    <option value="soc2">SOC 2</option>
                    <option value="iso27001">ISO 27001</option>
                    <option value="pci_dss">PCI DSS</option>
                    <option value="hipaa">HIPAA</option>
                    <option value="gdpr">GDPR</option>
                    <option value="nerc_cip">NERC CIP</option>
                    <option value="custom">Custom</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Available Sections */}
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
              <h3 className="font-medium text-gray-900 dark:text-white mb-4">Available Sections</h3>
              <div className="space-y-2">
                {sections
                  .filter((s) => s.order < 0)
                  .map((section) => (
                    <button
                      key={section.id}
                      onClick={() => addSection(section.name)}
                      disabled={readOnly}
                      className="w-full px-3 py-2 text-left text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg flex items-center gap-2"
                    >
                      <Plus size={16} />
                      <span>{section.title}</span>
                    </button>
                  ))}
              </div>
            </div>
          </div>

          {/* Section Builder */}
          <div className="lg:col-span-2">
            <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
              <h3 className="font-medium text-gray-900 dark:text-white mb-4">
                Report Sections
              </h3>

              <div className="space-y-2">
                {selectedSections.length === 0 ? (
                  <p className="text-gray-500 text-center py-8">
                    No sections selected. Add sections from the available sections.
                  </p>
                ) : (
                  selectedSections.map((section, index) => (
                    <SectionEditor
                      key={section.id}
                      section={section}
                      index={index}
                      total={selectedSections.length}
                      onToggleExpanded={() => toggleExpanded(section.id)}
                      onMoveUp={() => moveUp(section.id)}
                      onMoveDown={() => moveDown(section.id)}
                      onRemove={() => removeSection(section.id)}
                      onUpdateConfig={(config) => updateSectionConfig(section.id, config)}
                      readOnly={readOnly}
                    />
                  ))
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

interface SectionEditorProps {
  section: SectionConfig;
  index: number;
  total: number;
  onToggleExpanded: () => void;
  onMoveUp: () => void;
  onMoveDown: () => void;
  onRemove: () => void;
  onUpdateConfig: (config: Record<string, unknown>) => void;
  readOnly?: boolean;
}

const SectionEditor: React.FC<SectionEditorProps> = ({
  section,
  index,
  total,
  onToggleExpanded,
  onMoveUp,
  onMoveDown,
  onRemove,
  onUpdateConfig,
  readOnly = false,
}) => {
  const typeIcons: Record<string, string> = {
    summary: '📊',
    chart: '📈',
    table: '📋',
    text: '📝',
    heatmap: '🔥',
  };

  return (
    <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      <div className="flex items-center gap-2 px-4 py-3 bg-gray-50 dark:bg-gray-750">
        <GripVertical size={16} className="text-gray-400 cursor-move" />
        <span className="text-lg">{typeIcons[section.type] || '📄'}</span>
        <span className="flex-1 font-medium text-gray-900 dark:text-white">
          {section.title}
        </span>
        {section.required && (
          <span className="text-xs px-2 py-1 bg-blue-100 text-blue-800 rounded-full">
            Required
          </span>
        )}
        <span className="text-xs text-gray-500 capitalize">{section.type}</span>
        <button
          onClick={onToggleExpanded}
          className="p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded"
        >
          {section.expanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </button>
        <div className="flex gap-1">
          <button
            onClick={onMoveUp}
            disabled={index === 0 || readOnly}
            className="p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded disabled:opacity-50"
          >
            <ChevronUp size={16} />
          </button>
          <button
            onClick={onMoveDown}
            disabled={index === total - 1 || readOnly}
            className="p-1 hover:bg-gray-200 dark:hover:bg-gray-600 rounded disabled:opacity-50"
          >
            <ChevronDown size={16} />
          </button>
          {!section.required && !readOnly && (
            <button
              onClick={onRemove}
              className="p-1 hover:bg-red-100 dark:hover:bg-red-900 text-red-600 rounded"
            >
              <Trash2 size={16} />
            </button>
          )}
        </div>
      </div>

      {section.expanded && (
        <div className="p-4 bg-white dark:bg-gray-800">
          <SectionConfigForm
            section={section}
            onUpdateConfig={onUpdateConfig}
            readOnly={readOnly}
          />
        </div>
      )}
    </div>
  );
};

interface SectionConfigFormProps {
  section: SectionConfig;
  onUpdateConfig: (config: Record<string, unknown>) => void;
  readOnly?: boolean;
}

const SectionConfigForm: React.FC<SectionConfigFormProps> = ({
  section,
  onUpdateConfig,
  readOnly = false,
}) => {
  const config = section.config as Record<string, unknown>;

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        {section.type === 'table' && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Columns
              </label>
              <input
                type="text"
                value={(config.columns as string[])?.join(', ') || ''}
                onChange={(e) =>
                  onUpdateConfig({
                    columns: e.target.value.split(',').map((s) => s.trim()),
                  })
                }
                disabled={readOnly}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
                placeholder="name, status, score"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Group By
              </label>
              <select
                value={(config.group_by as string) || ''}
                onChange={(e) => onUpdateConfig({ group_by: e.target.value })}
                disabled={readOnly}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
              >
                <option value="">None</option>
                <option value="category">Category</option>
                <option value="severity">Severity</option>
                <option value="status">Status</option>
              </select>
            </div>
          </>
        )}

        {section.type === 'chart' && (
          <>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Chart Type
              </label>
              <select
                value={(config.chart_type as string) || 'line'}
                onChange={(e) => onUpdateConfig({ chart_type: e.target.value })}
                disabled={readOnly}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
              >
                <option value="line">Line Chart</option>
                <option value="bar">Bar Chart</option>
                <option value="gauge">Gauge</option>
                <option value="pie">Pie Chart</option>
              </select>
            </div>
            <div className="flex items-center gap-2 pt-6">
              <input
                type="checkbox"
                id={`trend-${section.id}`}
                checked={(config.include_trend as boolean) || false}
                onChange={(e) => onUpdateConfig({ include_trend: e.target.checked })}
                disabled={readOnly}
                className="rounded"
              />
              <label htmlFor={`trend-${section.id}`} className="text-sm text-gray-700 dark:text-gray-300">
                Include trend line
              </label>
            </div>
          </>
        )}

        {section.type === 'summary' && (
          <>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id={`score-${section.id}`}
                checked={(config.include_score as boolean) !== false}
                onChange={(e) => onUpdateConfig({ include_score: e.target.checked })}
                disabled={readOnly}
                className="rounded"
              />
              <label htmlFor={`score-${section.id}`} className="text-sm text-gray-700 dark:text-gray-300">
                Include compliance score
              </label>
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id={`findings-${section.id}`}
                checked={(config.include_findings as boolean) !== false}
                onChange={(e) => onUpdateConfig({ include_findings: e.target.checked })}
                disabled={readOnly}
                className="rounded"
              />
              <label htmlFor={`findings-${section.id}`} className="text-sm text-gray-700 dark:text-gray-300">
                Include findings summary
              </label>
            </div>
          </>
        )}

        {section.type === 'heatmap' && (
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Metric
            </label>
            <select
              value={(config.metric as string) || 'session_count'}
              onChange={(e) => onUpdateConfig({ metric: e.target.value })}
              disabled={readOnly}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white text-sm"
            >
              <option value="session_count">Session Count</option>
              <option value="duration">Duration</option>
              <option value="command_count">Command Count</option>
              <option value="failed_auths">Failed Authentications</option>
            </select>
          </div>
        )}
      </div>
    </div>
  );
};

// Preview component
interface TemplatePreviewProps {
  name: string;
  description: string;
  reportType: ReportType;
  framework: ComplianceFramework;
  sections: SectionConfig[];
}

const TemplatePreview: React.FC<TemplatePreviewProps> = ({
  name,
  description,
  reportType,
  framework,
  sections,
}) => {
  return (
    <div className="template-preview border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden">
      {/* Header */}
      <div className="bg-gradient-to-r from-blue-600 to-blue-800 text-white p-6">
        <h2 className="text-2xl font-bold">{name || 'Untitled Report'}</h2>
        {description && <p className="text-blue-100 mt-2">{description}</p>}
        <div className="flex gap-4 mt-4 text-sm">
          <span className="px-2 py-1 bg-white/20 rounded">{reportType}</span>
          <span className="px-2 py-1 bg-white/20 rounded">{framework.toUpperCase()}</span>
          <span className="px-2 py-1 bg-white/20 rounded">{sections.length} sections</span>
        </div>
      </div>

      {/* Table of Contents */}
      <div className="p-6 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
        <h3 className="font-semibold text-gray-900 dark:text-white mb-3">Table of Contents</h3>
        <ol className="space-y-1">
          {sections.map((section, index) => (
            <li key={section.id} className="text-gray-600 dark:text-gray-400">
              <span className="font-medium text-gray-900 dark:text-white mr-2">
                {index + 1}.
              </span>
              {section.title}
              {section.required && (
                <span className="ml-2 text-xs text-blue-600">(required)</span>
              )}
            </li>
          ))}
        </ol>
      </div>

      {/* Section Previews */}
      <div className="p-6 space-y-6">
        {sections.map((section) => (
          <div
            key={section.id}
            className="border border-gray-200 dark:border-gray-700 rounded-lg p-4"
          >
            <h4 className="font-medium text-gray-900 dark:text-white flex items-center gap-2">
              {section.title}
              <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-600 dark:text-gray-400 rounded">
                {section.type}
              </span>
            </h4>
            <div className="mt-4 bg-gray-100 dark:bg-gray-800 rounded p-8 text-center text-gray-400">
              {section.type === 'summary' && '📊 Summary content will appear here'}
              {section.type === 'chart' && '📈 Chart will render here'}
              {section.type === 'table' && '📋 Table data will appear here'}
              {section.type === 'text' && '📝 Text content will appear here'}
              {section.type === 'heatmap' && '🔥 Heatmap visualization will appear here'}
            </div>
          </div>
        ))}
      </div>

      {/* Footer */}
      <div className="p-4 bg-gray-50 dark:bg-gray-800 border-t border-gray-200 dark:border-gray-700 text-center text-sm text-gray-500">
        Generated on {new Date().toLocaleDateString()} by OpenPAM
      </div>
    </div>
  );
};

export default TemplateEditor;

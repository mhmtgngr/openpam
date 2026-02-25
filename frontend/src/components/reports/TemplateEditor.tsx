/**
 * TemplateEditor - Visual template editor for report customization
 * Allows users to create and modify report templates with drag-and-drop sections
 */

import React, { useState } from 'react';
import {
  FileText,
  Plus,
  Trash2,
  GripVertical,
  Eye,
  Settings,
  ChevronDown,
  ChevronUp,
  Type,
  BarChart3,
  Table,
  AlignLeft,
  Calendar,
  Save,
} from 'lucide-react';
import { Card, CardHeader } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Textarea } from '@/components/common';
import { Select } from '@/components/common';
import { Toggle } from '@/components/common';
import { Badge } from '@/components/common';
import { Modal } from '@/components/common';
import type { ReportTemplate, ReportTemplateSection } from '@/types/reports';
import clsx from 'clsx';

export interface TemplateEditorProps {
  template?: ReportTemplate;
  onSave: (template: Omit<ReportTemplate, 'id' | 'created_at' | 'updated_at'>) => void;
  onCancel: () => void;
  readOnly?: boolean;
}

const sectionTypes = [
  { value: 'summary', label: 'Summary', icon: FileText, description: 'Executive summary section' },
  { value: 'table', label: 'Table', icon: Table, description: 'Data table with rows and columns' },
  { value: 'chart', label: 'Chart', icon: BarChart3, description: 'Visual chart or graph' },
  { value: 'text', label: 'Text', icon: AlignLeft, description: 'Free-form text content' },
  { value: 'heatmap', label: 'Heatmap', icon: BarChart3, description: 'Color-coded data visualization' },
];

const defaultSection: Partial<ReportTemplateSection> = {
  enabled: true,
  config: {},
};

export const TemplateEditor: React.FC<TemplateEditorProps> = ({
  template,
  onSave,
  onCancel,
  readOnly = false,
}) => {
  const [name, setName] = useState(template?.name || '');
  const [description, setDescription] = useState(template?.description || '');
  const [sections, setSections] = useState<ReportTemplateSection[]>(
    template?.sections || []
  );
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set());
  const [showPreview, setShowPreview] = useState(false);
  const [showAddSection, setShowAddSection] = useState(false);

  const toggleSectionExpanded = (sectionId: string) => {
    setExpandedSections((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(sectionId)) {
        newSet.delete(sectionId);
      } else {
        newSet.add(sectionId);
      }
      return newSet;
    });
  };

  const addSection = (type: string) => {
    const newSection: ReportTemplateSection = {
      id: `section_${Date.now()}`,
      name: `${type.charAt(0).toUpperCase() + type.slice(1)} Section`,
      title: `${type.charAt(0).toUpperCase() + type.slice(1)}`,
      type: type as ReportTemplateSection['type'],
      required: false,
      enabled: true,
      config: {},
      order: sections.length,
    };
    setSections([...sections, newSection]);
    setExpandedSections(new Set([...expandedSections, newSection.id]));
    setShowAddSection(false);
  };

  const removeSection = (sectionId: string) => {
    setSections(sections.filter((s) => s.id !== sectionId));
    setExpandedSections((prev) => {
      const newSet = new Set(prev);
      newSet.delete(sectionId);
      return newSet;
    });
  };

  const updateSection = (sectionId: string, updates: Partial<ReportTemplateSection>) => {
    setSections(sections.map((s) =>
      s.id === sectionId ? { ...s, ...updates } : s
    ));
  };

  const moveSection = (fromIndex: number, toIndex: number) => {
    const newSections = [...sections];
    const [moved] = newSections.splice(fromIndex, 1);
    newSections.splice(toIndex, 0, moved);
    // Update order values
    newSections.forEach((s, i) => s.order = i);
    setSections(newSections);
  };

  const handleSave = () => {
    if (!name.trim()) {
      return;
    }
    onSave({
      name,
      description,
      type: template?.type || 'custom',
      framework: template?.framework,
      sections: sections.map((s, i) => ({ ...s, order: i })),
      is_system: false,
      config: template?.config || {
        period_start: '',
        period_end: '',
        include_sections: [],
        filters: {},
      },
    });
  };

  const getSectionIcon = (type: ReportTemplateSection['type']) => {
    const found = sectionTypes.find((t) => t.value === type);
    return found?.icon || FileText;
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <Card>
        <CardHeader
          title="Report Template Editor"
          subtitle="Customize your report template by adding and configuring sections"
          icon={<Settings className="h-5 w-5" />}
        />
        <div className="card-body space-y-4">
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Template Name *</label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Monthly Compliance Report"
              disabled={readOnly}
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-gray-300">Description</label>
            <Textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Describe what this template is for..."
              rows={2}
              disabled={readOnly}
            />
          </div>
        </div>
      </Card>

      {/* Sections */}
      <Card>
        <CardHeader
          title="Report Sections"
          subtitle="Drag to reorder. Sections will appear in the report in this order."
          action={
            !readOnly && (
              <Button
                variant="secondary"
                size="sm"
                leftIcon={<Plus className="h-4 w-4" />}
                onClick={() => setShowAddSection(true)}
              >
                Add Section
              </Button>
            )
          }
        />

        <div className="card-body space-y-3">
          {sections.length === 0 ? (
            <div className="py-8 text-center text-gray-500">
              <FileText className="mx-auto h-12 w-12 mb-2 opacity-50" />
              <p>No sections added yet. Click "Add Section" to get started.</p>
            </div>
          ) : (
            sections.map((section, index) => {
              const SectionIcon = getSectionIcon(section.type);
              const isExpanded = expandedSections.has(section.id);

              return (
                <div
                  key={section.id}
                  className={clsx(
                    'rounded-lg border transition-all',
                    section.enabled ? 'border-gray-700 bg-gray-900/30' : 'border-gray-800 bg-gray-900/10 opacity-60'
                  )}
                >
                  {/* Section Header */}
                  <div className="flex items-center gap-3 p-4">
                    {!readOnly && (
                      <div className="cursor-grab text-gray-600 hover:text-gray-400">
                        <GripVertical className="h-5 w-5" />
                      </div>
                    )}
                    <div className={clsx('rounded-lg p-2', {
                      'bg-primary-500/20 text-primary-400': section.enabled,
                      'bg-gray-800 text-gray-500': !section.enabled,
                    })}>
                      <SectionIcon className="h-4 w-4" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <Input
                        value={section.name}
                        onChange={(e) => updateSection(section.id, { name: e.target.value })}
                        className="font-medium"
                        disabled={readOnly}
                      />
                    </div>
                    <Badge variant="neutral" className="capitalize">
                      {section.type}
                    </Badge>
                    {!readOnly && (
                      <Toggle
                        checked={section.enabled}
                        onChange={(checked) => updateSection(section.id, { enabled: checked })}
                      />
                    )}
                    <button
                      onClick={() => toggleSectionExpanded(section.id)}
                      className="text-gray-500 hover:text-white transition-colors"
                    >
                      {isExpanded ? <ChevronUp className="h-5 w-5" /> : <ChevronDown className="h-5 w-5" />}
                    </button>
                    {!readOnly && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeSection(section.id)}
                        className="text-danger-400 hover:text-danger-300"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    )}
                  </div>

                  {/* Section Details */}
                  {isExpanded && (
                    <div className="border-t border-gray-800 p-4 space-y-4">
                      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div>
                          <label className="mb-1 block text-sm font-medium text-gray-300">Display Title</label>
                          <Input
                            value={section.title}
                            onChange={(e) => updateSection(section.id, { title: e.target.value })}
                            placeholder="Section title shown in report"
                            disabled={readOnly}
                          />
                        </div>
                        <div className="flex items-center gap-3">
                          <Toggle
                            checked={section.required}
                            onChange={(checked) => updateSection(section.id, { required: checked })}
                            disabled={readOnly}
                          />
                          <label className="text-sm text-gray-300">Required Section</label>
                        </div>
                      </div>

                      {/* Type-specific configuration */}
                      {section.type === 'chart' && (
                        <div className="space-y-3">
                          <h4 className="text-sm font-medium text-white">Chart Configuration</h4>
                          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                            <div>
                              <label className="mb-1 block text-sm font-medium text-gray-300">Chart Type</label>
                              <Select
                                options={[
                                  { value: 'line', label: 'Line Chart' },
                                  { value: 'bar', label: 'Bar Chart' },
                                  { value: 'pie', label: 'Pie Chart' },
                                  { value: 'area', label: 'Area Chart' },
                                ]}
                                value={(section.config as any)?.chartType || 'line'}
                                onChange={(e) => updateSection(section.id, {
                                  config: { ...section.config, chartType: e.target.value }
                                })}
                                disabled={readOnly}
                              />
                            </div>
                            <div>
                              <label className="mb-1 block text-sm font-medium text-gray-300">Data Source</label>
                              <Select
                                options={[
                                  { value: 'sessions', label: 'Session Data' },
                                  { value: 'users', label: 'User Activity' },
                                  { value: 'commands', label: 'Command Analysis' },
                                  { value: 'compliance', label: 'Compliance Scores' },
                                ]}
                                value={(section.config as any)?.dataSource || ''}
                                onChange={(e) => updateSection(section.id, {
                                  config: { ...section.config, dataSource: e.target.value }
                                })}
                                disabled={readOnly}
                              />
                            </div>
                          </div>
                        </div>
                      )}

                      {section.type === 'table' && (
                        <div className="space-y-3">
                          <h4 className="text-sm font-medium text-white">Table Configuration</h4>
                          <div>
                            <label className="mb-1 block text-sm font-medium text-gray-300">Columns</label>
                            <Input
                              value={(section.config as any)?.columns || ''}
                              onChange={(e) => updateSection(section.id, {
                                config: { ...section.config, columns: e.target.value }
                              })}
                              placeholder="e.g., name, status, date, user"
                              disabled={readOnly}
                            />
                          </div>
                        </div>
                      )}

                      {section.type === 'text' && (
                        <div className="space-y-3">
                          <h4 className="text-sm font-medium text-white">Text Content</h4>
                          <Textarea
                            value={(section.config as any)?.content || ''}
                            onChange={(e) => updateSection(section.id, {
                              config: { ...section.config, content: e.target.value }
                            })}
                            placeholder="Enter the default text content..."
                            rows={4}
                            disabled={readOnly}
                          />
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })
          )}
        </div>
      </Card>

      {/* Actions */}
      <div className="flex items-center justify-end gap-3">
        {!readOnly && (
          <>
            <Button variant="secondary" onClick={onCancel}>
              Cancel
            </Button>
            <Button
              variant="secondary"
              leftIcon={<Eye className="h-4 w-4" />}
              onClick={() => setShowPreview(true)}
            >
              Preview
            </Button>
            <Button
              variant="primary"
              leftIcon={<Save className="h-4 w-4" />}
              onClick={handleSave}
              disabled={!name.trim()}
            >
              Save Template
            </Button>
          </>
        )}
        {readOnly && (
          <Button variant="secondary" onClick={onCancel}>
            Close
          </Button>
        )}
      </div>

      {/* Add Section Modal */}
      <Modal
        isOpen={showAddSection}
        onClose={() => setShowAddSection(false)}
        title="Add Section"
        size="md"
      >
        <div className="space-y-4">
          <p className="text-sm text-gray-400">Select the type of section to add to your template:</p>
          <div className="grid gap-3">
            {sectionTypes.map((type) => {
              const Icon = type.icon;
              return (
                <button
                  key={type.value}
                  onClick={() => addSection(type.value)}
                  className="flex items-start gap-4 rounded-lg border border-gray-700 p-4 text-left transition-colors hover:bg-gray-800"
                >
                  <div className="rounded-lg bg-primary-500/20 p-2 text-primary-400">
                    <Icon className="h-5 w-5" />
                  </div>
                  <div>
                    <p className="font-medium text-white">{type.label}</p>
                    <p className="text-sm text-gray-400">{type.description}</p>
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      </Modal>

      {/* Preview Modal */}
      <Modal
        isOpen={showPreview}
        onClose={() => setShowPreview(false)}
        title="Template Preview"
        size="lg"
      >
        <div className="space-y-4">
          <div className="rounded-lg bg-white p-6 text-black">
            <h2 className="text-xl font-bold mb-4">{name || 'Untitled Template'}</h2>
            {description && <p className="text-gray-600 mb-6">{description}</p>}
            <div className="space-y-4">
              {sections.filter(s => s.enabled).map((section) => (
                <div key={section.id} className="border-b pb-4">
                  <h3 className="font-semibold text-lg">{section.title}</h3>
                  <p className="text-sm text-gray-500 capitalize">{section.type} section</p>
                </div>
              ))}
            </div>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default TemplateEditor;

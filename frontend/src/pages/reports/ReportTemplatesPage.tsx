/**
 * ReportTemplatesPage Component
 *
 * Manages report templates for compliance and analytics reports.
 * Users can view, create, edit, and delete report templates.
 */

import React, { useState, useCallback, useEffect } from 'react';
import { Plus, Edit, Trash2, Search, FileText, Copy, CheckCircle } from 'lucide-react';
import { reportsApi } from '@/api/reports';
import type { ReportTemplate, ReportType, ComplianceFramework } from '@/types/reports';
import { TemplateEditor } from '@/components/reports/TemplateEditor';

const FRAMEWORK_LABELS: Record<ComplianceFramework, string> = {
  soc2: 'SOC 2',
  iso27001: 'ISO 27001',
  pci_dss: 'PCI DSS',
  hipaa: 'HIPAA',
  gdpr: 'GDPR',
  nerc_cip: 'NERC CIP',
  custom: 'Custom',
};

const TYPE_LABELS: Record<ReportType, string> = {
  compliance: 'Compliance',
  session_activity: 'Session Activity',
  command_analysis: 'Command Analysis',
  user_access: 'User Access',
  anomaly_summary: 'Anomaly Summary',
  audit_trail: 'Audit Trail',
  credential_usage: 'Credential Usage',
  custom: 'Custom',
};

type ViewMode = 'list' | 'create' | 'edit';

interface Filters {
  type?: ReportType;
  framework?: ComplianceFramework;
  search: string;
}

export const ReportTemplatesPage: React.FC = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [templates, setTemplates] = useState<ReportTemplate[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<ReportTemplate | undefined>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [filters, setFilters] = useState<Filters>({ search: '' });
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  // Fetch templates
  const fetchTemplates = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await reportsApi.listTemplates({
        type: filters.type,
        framework: filters.framework,
      });
      setTemplates(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load templates');
    } finally {
      setLoading(false);
    }
  }, [filters.type, filters.framework]);

  useEffect(() => {
    fetchTemplates();
  }, [fetchTemplates]);

  // Filter templates
  const filteredTemplates = templates.filter((t) => {
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      return (
        t.name.toLowerCase().includes(searchLower) ||
        t.description?.toLowerCase().includes(searchLower)
      );
    }
    return true;
  });

  // Handle create new
  const handleCreate = useCallback(() => {
    setSelectedTemplate(undefined);
    setViewMode('create');
  }, []);

  // Handle edit
  const handleEdit = useCallback((template: ReportTemplate) => {
    setSelectedTemplate(template);
    setViewMode('edit');
  }, []);

  // Handle duplicate
  const handleDuplicate = useCallback(async (template: ReportTemplate) => {
    try {
      const newTemplate = await reportsApi.create({
        name: `${template.name} (Copy)`,
        description: template.description,
        framework: template.framework,
        type: template.type,
        config: template.config,
        sections: template.sections,
      });
      setTemplates((prev) => [...prev, newTemplate]);
      setSuccessMessage('Template duplicated successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to duplicate template');
    }
  }, []);

  // Handle delete
  const handleDelete = useCallback(async (id: string) => {
    try {
      await reportsApi.delete(id);
      setTemplates((prev) => prev.filter((t) => t.id !== id));
      setDeleteConfirm(null);
      setSuccessMessage('Template deleted successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete template');
    }
  }, []);

  // Handle save
  const handleSave = useCallback(async (data: Partial<ReportTemplate>) => {
    try {
      if (selectedTemplate) {
        // Update existing
        const updated = await reportsApi.update(selectedTemplate.id, data);
        setTemplates((prev) => prev.map((t) => (t.id === selectedTemplate.id ? updated : t)));
      } else {
        // Create new
        const created = await reportsApi.create(data as any);
        setTemplates((prev) => [...prev, created]);
      }
      setViewMode('list');
      setSuccessMessage(selectedTemplate ? 'Template updated successfully' : 'Template created successfully');
      setTimeout(() => setSuccessMessage(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save template');
    }
  }, [selectedTemplate]);

  // Render list view
  if (viewMode === 'list') {
    return (
      <div className="report-templates-page">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 dark:text-white">
              Report Templates
            </h1>
            <p className="text-gray-600 dark:text-gray-400 mt-1">
              Create and manage report templates for compliance and analytics
            </p>
          </div>
          <button
            onClick={handleCreate}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 flex items-center gap-2"
          >
            <Plus size={20} />
            New Template
          </button>
        </div>

        {/* Success Message */}
        {successMessage && (
          <div className="mb-4 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg flex items-center gap-2 text-green-800 dark:text-green-200">
            <CheckCircle size={20} />
            {successMessage}
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg text-red-800 dark:text-red-200">
            {error}
            <button
              onClick={() => setError(null)}
              className="ml-4 underline hover:no-underline"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Filters */}
        <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4 mb-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            {/* Search */}
            <div className="md:col-span-2">
              <div className="relative">
                <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search templates..."
                  value={filters.search}
                  onChange={(e) => setFilters({ ...filters, search: e.target.value })}
                  className="w-full pl-10 pr-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>
            </div>

            {/* Type Filter */}
            <div>
              <select
                value={filters.type || ''}
                onChange={(e) => setFilters({ ...filters, type: e.target.value as ReportType | undefined })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="">All Types</option>
                {Object.entries(TYPE_LABELS).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </div>

            {/* Framework Filter */}
            <div>
              <select
                value={filters.framework || ''}
                onChange={(e) => setFilters({ ...filters, framework: e.target.value as ComplianceFramework | undefined })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="">All Frameworks</option>
                {Object.entries(FRAMEWORK_LABELS).map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        {/* Templates Grid */}
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
          </div>
        ) : filteredTemplates.length === 0 ? (
          <div className="text-center py-12">
            <FileText size={48} className="mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              {filters.search || filters.type || filters.framework ? 'No templates found' : 'No templates yet'}
            </h3>
            <p className="text-gray-500 mb-6">
              {filters.search || filters.type || filters.framework
                ? 'Try adjusting your filters'
                : 'Create your first report template to get started'}
            </p>
            {!filters.search && !filters.type && !filters.framework && (
              <button
                onClick={handleCreate}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 inline-flex items-center gap-2"
              >
                <Plus size={18} />
                Create Template
              </button>
            )}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredTemplates.map((template) => (
              <TemplateCard
                key={template.id}
                template={template}
                onEdit={() => handleEdit(template)}
                onDuplicate={() => handleDuplicate(template)}
                onDelete={() => setDeleteConfirm(template.id)}
                isDeleting={deleteConfirm === template.id}
                onConfirmDelete={() => handleDelete(template.id)}
                onCancelDelete={() => setDeleteConfirm(null)}
              />
            ))}
          </div>
        )}
      </div>
    );
  }

  // Render editor view
  return (
    <div>
      <TemplateEditor
        template={selectedTemplate}
        onSave={handleSave}
        onCancel={() => setViewMode('list')}
      />
    </div>
  );
};

interface TemplateCardProps {
  template: ReportTemplate;
  onEdit: () => void;
  onDuplicate: () => void;
  onDelete: () => void;
  isDeleting: boolean;
  onConfirmDelete: () => void;
  onCancelDelete: () => void;
}

const TemplateCard: React.FC<TemplateCardProps> = ({
  template,
  onEdit,
  onDuplicate,
  onDelete,
  isDeleting,
  onConfirmDelete,
  onCancelDelete,
}) => {
  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden hover:shadow-md transition-shadow">
      {/* Card Header */}
      <div className="p-4 border-b border-gray-200 dark:border-gray-700">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2">
            <FileText size={20} className="text-blue-600" />
            <h3 className="font-semibold text-gray-900 dark:text-white truncate">
              {template.name}
            </h3>
          </div>
          {template.is_system && (
            <span className="text-xs px-2 py-1 bg-blue-100 text-blue-800 rounded-full">
              System
            </span>
          )}
        </div>
        {template.description && (
          <p className="text-sm text-gray-500 mt-2 line-clamp-2">
            {template.description}
          </p>
        )}
      </div>

      {/* Card Body */}
      <div className="p-4">
        <div className="flex gap-2 mb-4">
          <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded">
            {TYPE_LABELS[template.type]}
          </span>
          <span className="text-xs px-2 py-1 bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded">
            {FRAMEWORK_LABELS[template.framework]}
          </span>
        </div>

        <div className="text-sm text-gray-500">
          {template.sections.length} sections
        </div>
      </div>

      {/* Card Footer */}
      <div className="px-4 py-3 bg-gray-50 dark:bg-gray-750 border-t border-gray-200 dark:border-gray-700 flex items-center justify-between">
        <div className="text-xs text-gray-500">
          Updated {new Date(template.updated_at).toLocaleDateString()}
        </div>

        {isDeleting ? (
          <div className="flex items-center gap-2">
            <button
              onClick={onConfirmDelete}
              className="px-2 py-1 bg-red-600 text-white text-xs rounded hover:bg-red-700"
            >
              Confirm
            </button>
            <button
              onClick={onCancelDelete}
              className="px-2 py-1 bg-gray-200 dark:bg-gray-600 text-gray-700 dark:text-gray-300 text-xs rounded hover:bg-gray-300 dark:hover:bg-gray-500"
            >
              Cancel
            </button>
          </div>
        ) : (
          <div className="flex gap-1">
            <button
              onClick={onEdit}
              className="p-1.5 text-gray-600 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-600 rounded"
              title="Edit"
            >
              <Edit size={16} />
            </button>
            <button
              onClick={onDuplicate}
              className="p-1.5 text-gray-600 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-600 rounded"
              title="Duplicate"
            >
              <Copy size={16} />
            </button>
            {!template.is_system && (
              <button
                onClick={onDelete}
                className="p-1.5 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20 rounded"
                title="Delete"
              >
                <Trash2 size={16} />
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default ReportTemplatesPage;

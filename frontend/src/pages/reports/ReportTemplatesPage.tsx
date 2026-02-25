/**
 * ReportTemplates Page - Manage report templates
 * Allows users to create, edit, and manage report templates
 */

import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  FileText,
  Plus,
  Search,
  Edit,
  Trash2,
  Copy,
  Eye,
  Calendar,
  Filter,
} from 'lucide-react';
import { reportsApi } from '@/api/reports';
import { Card, CardHeader } from '@/components/common';
import { Button } from '@/components/common';
import { Input } from '@/components/common';
import { Select } from '@/components/common';
import { Badge } from '@/components/common';
import { LoadingState } from '@/components/common';
import { EmptyState } from '@/components/common';
import { Modal } from '@/components/common';
import { TemplateEditor } from '@/components/reports/TemplateEditor';
import { toast } from 'react-hot-toast';
import type { ReportTemplate, ReportType, ComplianceFramework } from '@/types/reports';
import clsx from 'clsx';

const reportTypeOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Types' },
  { value: 'compliance', label: 'Compliance' },
  { value: 'session_activity', label: 'Session Activity' },
  { value: 'command_analysis', label: 'Command Analysis' },
  { value: 'user_access', label: 'User Access' },
  { value: 'anomaly_summary', label: 'Anomaly Summary' },
  { value: 'audit_trail', label: 'Audit Trail' },
  { value: 'credential_usage', label: 'Credential Usage' },
];

const frameworkOptions: Array<{ value: string; label: string }> = [
  { value: '', label: 'All Frameworks' },
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'pci_dss', label: 'PCI DSS' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'gdpr', label: 'GDPR' },
];

interface TemplateCardProps {
  template: ReportTemplate;
  onEdit: () => void;
  onView: () => void;
  onDelete: () => void;
  onDuplicate: (template: ReportTemplate) => void;
  readOnly?: boolean;
}

const TemplateCard: React.FC<TemplateCardProps> = ({
  template,
  onEdit,
  onView,
  onDelete,
  onDuplicate,
  readOnly = false,
}) => {
  const getTypeColor = (type: string) => {
    switch (type) {
      case 'compliance': return 'text-blue-400 bg-blue-500/20';
      case 'session_activity': return 'text-green-400 bg-green-500/20';
      case 'command_analysis': return 'text-purple-400 bg-purple-500/20';
      case 'anomaly_summary': return 'text-orange-400 bg-orange-500/20';
      default: return 'text-gray-400 bg-gray-500/20';
    }
  };

  return (
    <Card className="group hover:shadow-lg transition-shadow">
      <div className="card-body">
        {/* Header */}
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center gap-2">
            <div className={clsx('rounded-lg p-2', getTypeColor(template.type))}>
              <FileText className="h-4 w-4" />
            </div>
            <div>
              <h3 className="font-semibold text-white">{template.name}</h3>
              {template.is_system && (
                <Badge variant="neutral" className="text-xs mt-1">System</Badge>
              )}
            </div>
          </div>
        </div>

        {/* Description */}
        {template.description && (
          <p className="text-sm text-gray-400 line-clamp-2 mb-3">
            {template.description}
          </p>
        )}

        {/* Metadata */}
        <div className="flex flex-wrap items-center gap-2 mb-4">
          <Badge variant="neutral" className="capitalize">
            {template.type.replace('_', ' ')}
          </Badge>
          {template.framework && (
            <Badge variant="primary">{template.framework.toUpperCase()}</Badge>
          )}
        </div>

        {/* Section Count */}
        <div className="text-xs text-gray-500 mb-4">
          {template.sections.length} section{template.sections.length !== 1 ? 's' : ''}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-end gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
          <Button
            variant="ghost"
            size="sm"
            onClick={onView}
            leftIcon={<Eye className="h-4 w-4" />}
          >
            View
          </Button>
          {!readOnly && (
            <>
              <Button
                variant="ghost"
                size="sm"
                onClick={onEdit}
                leftIcon={<Edit className="h-4 w-4" />}
              >
                Edit
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDuplicate(template)}
                leftIcon={<Copy className="h-4 w-4" />}
              >
                Duplicate
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onDelete()}
                leftIcon={<Trash2 className="h-4 w-4 text-danger-400" />}
              />
            </>
          )}
        </div>
      </div>
    </Card>
  );
};

export const ReportTemplatesPage: React.FC = () => {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [searchQuery, setSearchQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState('');
  const [frameworkFilter, setFrameworkFilter] = useState('');
  const [showEditor, setShowEditor] = useState(false);
  const [editingTemplate, setEditingTemplate] = useState<ReportTemplate | undefined>();
  const [viewingTemplate, setViewingTemplate] = useState<ReportTemplate | undefined>();

  // Query: Templates
  const { data: templates, isLoading } = useQuery({
    queryKey: ['reportTemplates', typeFilter, frameworkFilter],
    queryFn: () => reportsApi.listTemplates({
      type: typeFilter as ReportType | undefined,
      framework: frameworkFilter as ComplianceFramework | undefined,
    }),
  });

  // Mutation: Create/Update Template
  const saveMutation = useMutation({
    mutationFn: (data: { template: Omit<ReportTemplate, 'id' | 'created_at' | 'updated_at'>; isEdit?: boolean }) =>
      data.isEdit && editingTemplate
        ? reportsApi.updateTemplate(editingTemplate.id, data.template)
        : reportsApi.createTemplate(data.template as any),
    onSuccess: () => {
      toast.success(editingTemplate ? 'Template updated successfully' : 'Template created successfully');
      setShowEditor(false);
      setEditingTemplate(undefined);
      queryClient.invalidateQueries({ queryKey: ['reportTemplates'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to save template: ${error.message}`);
    },
  });

  // Mutation: Delete Template
  const deleteMutation = useMutation({
    mutationFn: (id: string) => reportsApi.deleteTemplate(id),
    onSuccess: () => {
      toast.success('Template deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['reportTemplates'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to delete template: ${error.message}`);
    },
  });

  // Mutation: Duplicate Template
  const duplicateMutation = useMutation({
    mutationFn: (template: ReportTemplate) =>
      reportsApi.createTemplate({
        ...template,
        name: `${template.name} (Copy)`,
        is_system: false,
      } as any),
    onSuccess: () => {
      toast.success('Template duplicated successfully');
      queryClient.invalidateQueries({ queryKey: ['reportTemplates'] });
    },
    onError: (error: Error) => {
      toast.error(`Failed to duplicate template: ${error.message}`);
    },
  });

  const handleCreate = () => {
    setEditingTemplate(undefined);
    setShowEditor(true);
  };

  const handleEdit = (template: ReportTemplate) => {
    setEditingTemplate(template);
    setShowEditor(true);
  };

  const handleView = (template: ReportTemplate) => {
    setViewingTemplate(template);
  };

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this template?')) {
      deleteMutation.mutate(id);
    }
  };

  const handleDuplicate = (template: ReportTemplate) => {
    duplicateMutation.mutate(template);
  };

  const handleSave = (template: Omit<ReportTemplate, 'id' | 'created_at' | 'updated_at'>) => {
    saveMutation.mutate({ template, isEdit: !!editingTemplate });
  };

  const filteredTemplates = templates?.data?.filter((template) => {
    if (searchQuery && !template.name.toLowerCase().includes(searchQuery.toLowerCase())) {
      return false;
    }
    return true;
  }) || [];

  const systemTemplates = filteredTemplates.filter((t) => t.is_system);
  const customTemplates = filteredTemplates.filter((t) => !t.is_system);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">Report Templates</h1>
          <p className="mt-1 text-sm text-gray-400">
            Create and manage report templates for consistent reporting
          </p>
        </div>
        <Button
          variant="primary"
          leftIcon={<Plus className="h-4 w-4" />}
          onClick={handleCreate}
        >
          Create Template
        </Button>
      </div>

      {/* Filters */}
      <Card>
        <div className="card-body">
          <div className="flex flex-wrap gap-3">
            <div className="relative flex-1 min-w-[200px]">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
              <Input
                placeholder="Search templates..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select
              options={reportTypeOptions}
              value={typeFilter}
              onChange={(e) => setTypeFilter(e.target.value)}
              className="w-40"
            />
            <Select
              options={frameworkOptions}
              value={frameworkFilter}
              onChange={(e) => setFrameworkFilter(e.target.value)}
              className="w-40"
            />
          </div>
        </div>
      </Card>

      {/* Templates List */}
      {isLoading ? (
        <div className="flex min-h-[400px] items-center justify-center">
          <LoadingState message="Loading templates..." />
        </div>
      ) : filteredTemplates.length === 0 ? (
        <Card>
          <EmptyState
            title="No templates found"
            description={
              searchQuery || typeFilter || frameworkFilter
                ? 'Try adjusting your filters'
                : 'Get started by creating your first template'
            }
            action={
              !searchQuery && !typeFilter && !frameworkFilter ? (
                <Button
                  variant="primary"
                  leftIcon={<Plus className="h-4 w-4" />}
                  onClick={handleCreate}
                >
                  Create Template
                </Button>
              ) : undefined
            }
          />
        </Card>
      ) : (
        <div className="space-y-6">
          {/* System Templates */}
          {systemTemplates.length > 0 && (
            <div>
              <h3 className="text-lg font-semibold text-white mb-3">System Templates</h3>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {systemTemplates.map((template) => (
                  <TemplateCard
                    key={template.id}
                    template={template}
                    onEdit={() => {}}
                    onView={handleView}
                    onDelete={() => {}}
                    onDuplicate={handleDuplicate}
                    readOnly
                  />
                ))}
              </div>
            </div>
          )}

          {/* Custom Templates */}
          {customTemplates.length > 0 && (
            <div>
              <h3 className="text-lg font-semibold text-white mb-3">Custom Templates</h3>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {customTemplates.map((template) => (
                  <TemplateCard
                    key={template.id}
                    template={template}
                    onEdit={handleEdit}
                    onView={handleView}
                    onDelete={handleDelete}
                    onDuplicate={handleDuplicate}
                  />
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* Editor Modal */}
      <Modal
        isOpen={showEditor}
        onClose={() => {
          setShowEditor(false);
          setEditingTemplate(undefined);
        }}
        title={editingTemplate ? 'Edit Template' : 'Create Template'}
        size="xl"
      >
        <TemplateEditor
          template={editingTemplate}
          onSave={handleSave}
          onCancel={() => {
            setShowEditor(false);
            setEditingTemplate(undefined);
          }}
        />
      </Modal>

      {/* View Modal */}
      <Modal
        isOpen={!!viewingTemplate}
        onClose={() => setViewingTemplate(undefined)}
        title={viewingTemplate?.name}
        size="lg"
      >
        {viewingTemplate && (
          <div className="space-y-4">
            <p className="text-sm text-gray-400">{viewingTemplate.description}</p>
            <div className="flex items-center gap-2">
              <Badge variant="neutral">{viewingTemplate.type}</Badge>
              {viewingTemplate.framework && (
                <Badge variant="primary">{viewingTemplate.framework}</Badge>
              )}
            </div>
            <div className="border-t border-gray-800 pt-4">
              <h4 className="text-sm font-medium text-white mb-3">Sections</h4>
              <div className="space-y-2">
                {viewingTemplate.sections.map((section) => (
                  <div
                    key={section.id}
                    className="flex items-center justify-between rounded-lg bg-gray-900/50 p-3"
                  >
                    <div>
                      <p className="text-sm font-medium text-white">{section.title}</p>
                      <p className="text-xs text-gray-500 capitalize">{section.type}</p>
                    </div>
                    {section.required && (
                      <Badge variant="warning" className="text-xs">Required</Badge>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default ReportTemplatesPage;

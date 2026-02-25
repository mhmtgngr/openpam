import React, { useState, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { AlertTriangle, FileText, Upload, X, Calendar, Info } from 'lucide-react';
import { Modal } from '@/components/common';
import { Input, Textarea, Label, Button } from '@/components/common';
import { complianceExceptionsApi } from '@/api/reports';
import type {
  ComplianceException,
  CreateExceptionData,
  ComplianceFramework,
} from '@/types/reports';
import { format } from 'date-fns';
import toast from 'react-hot-toast';

interface ExceptionRequestDialogProps {
  isOpen: boolean;
  onClose: () => void;
  exception?: ComplianceException;
  onSuccess?: () => void;
  controlId?: string;
  controlName?: string;
  framework?: ComplianceFramework;
  mode?: 'create' | 'edit' | 'approve' | 'deny' | 'revoke';
}

const frameworks: { value: ComplianceFramework; label: string }[] = [
  { value: 'soc2', label: 'SOC 2' },
  { value: 'iso27001', label: 'ISO 27001' },
  { value: 'pci_dss', label: 'PCI DSS' },
  { value: 'hipaa', label: 'HIPAA' },
  { value: 'gdpr', label: 'GDPR' },
  { value: 'nerc_cip', label: 'NERC CIP' },
  { value: 'custom', label: 'Custom' },
];

export const ExceptionRequestDialog: React.FC<ExceptionRequestDialogProps> = ({
  isOpen,
  onClose,
  exception,
  onSuccess,
  controlId,
  controlName,
  framework: defaultFramework,
  mode = 'create',
}) => {
  const queryClient = useQueryQueryClient();
  const [formData, setFormData] = useState({
    control_id: controlId || exception?.control_id || '',
    control_name: controlName || exception?.control_name || '',
    framework: defaultFramework || exception?.framework || 'soc2',
    reason: '',
    business_justification: '',
    mitigation_plan: '',
    expires_at: '',
    denial_reason: '',
    review_notes: '',
  });

  const [files, setFiles] = useState<File[]>([]);
  const isEditing = mode === 'edit' || !!exception;

  useEffect(() => {
    if (exception) {
      setFormData({
        control_id: exception.control_id,
        control_name: exception.control_name,
        framework: exception.framework,
        reason: exception.reason,
        business_justification: exception.business_justification,
        mitigation_plan: exception.mitigation_plan || '',
        expires_at: exception.expires_at ? format(new Date(exception.expires_at), 'yyyy-MM-dd') : '',
        denial_reason: exception.denial_reason || '',
        review_notes: exception.review_notes || '',
      });
    } else if (controlId) {
      setFormData((prev) => ({
        ...prev,
        control_id: controlId,
        control_name: controlName || '',
      }));
    }
  }, [exception, controlId, controlName]);

  const createMutation = useMutation({
    mutationFn: (data: CreateExceptionData) => complianceExceptionsApi.create(data),
    onSuccess: () => {
      toast.success('Exception request submitted successfully');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      onSuccess?.();
      onClose();
      resetForm();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to submit exception request');
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: CreateExceptionData }) =>
      complianceExceptionsApi.update(id, data),
    onSuccess: () => {
      toast.success('Exception updated successfully');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to update exception');
    },
  });

  const approveMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes?: string }) =>
      complianceExceptionsApi.approve(id, notes),
    onSuccess: () => {
      toast.success('Exception approved successfully');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to approve exception');
    },
  });

  const denyMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      complianceExceptionsApi.deny(id, reason),
    onSuccess: () => {
      toast.success('Exception denied successfully');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to deny exception');
    },
  });

  const revokeMutation = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      complianceExceptionsApi.revoke(id, reason),
    onSuccess: () => {
      toast.success('Exception revoked successfully');
      queryClient.invalidateQueries({ queryKey: ['complianceExceptions'] });
      onSuccess?.();
      onClose();
    },
    onError: (error: Error) => {
      toast.error(error.message || 'Failed to revoke exception');
    },
  });

  const handleInputChange = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFiles = Array.from(e.target.files || []);
    setFiles((prev) => [...prev, ...selectedFiles]);
  };

  const removeFile = (index: number) => {
    setFiles((prev) => prev.filter((_, i) => i !== index));
  };

  const resetForm = () => {
    setFormData({
      control_id: '',
      control_name: '',
      framework: 'soc2',
      reason: '',
      business_justification: '',
      mitigation_plan: '',
      expires_at: '',
      denial_reason: '',
      review_notes: '',
    });
    setFiles([]);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (mode === 'approve') {
      if (exception) {
        approveMutation.mutate({ id: exception.id, notes: formData.review_notes || undefined });
      }
      return;
    }

    if (mode === 'deny') {
      if (!formData.denial_reason.trim()) {
        toast.error('Please provide a reason for denial');
        return;
      }
      if (exception) {
        denyMutation.mutate({ id: exception.id, reason: formData.denial_reason });
      }
      return;
    }

    if (mode === 'revoke') {
      if (!formData.review_notes.trim()) {
        toast.error('Please provide a reason for revocation');
        return;
      }
      if (exception) {
        revokeMutation.mutate({ id: exception.id, reason: formData.review_notes });
      }
      return;
    }

    // Create or Edit mode
    if (!formData.control_id.trim()) {
      toast.error('Please enter a control ID');
      return;
    }

    if (!formData.control_name.trim()) {
      toast.error('Please enter a control name');
      return;
    }

    if (!formData.reason.trim()) {
      toast.error('Please provide a reason for the exception');
      return;
    }

    if (!formData.business_justification.trim()) {
      toast.error('Please provide business justification');
      return;
    }

    const data: CreateExceptionData = {
      control_id: formData.control_id,
      control_name: formData.control_name,
      framework: formData.framework,
      reason: formData.reason,
      business_justification: formData.business_justification,
      mitigation_plan: formData.mitigation_plan || undefined,
      expires_at: formData.expires_at || undefined,
      documents: files.length > 0 ? files : undefined,
    };

    if (isEditing && exception) {
      updateMutation.mutate({ id: exception.id, data });
    } else {
      createMutation.mutate(data);
    }
  };

  const getDialogTitle = () => {
    switch (mode) {
      case 'approve':
        return 'Approve Exception Request';
      case 'deny':
        return 'Deny Exception Request';
      case 'revoke':
        return 'Revoke Exception';
      case 'edit':
        return 'Edit Exception';
      default:
        return isEditing ? 'Edit Exception Request' : 'Request Compliance Exception';
    }
  };

  const getSubmitButtonText = () => {
    switch (mode) {
      case 'approve':
        return 'Approve';
      case 'deny':
        return 'Deny';
      case 'revoke':
        return 'Revoke';
      default:
        return isEditing ? 'Update Exception' : 'Submit Request';
    }
  };

  const getSubmitButtonVariant = () => {
    switch (mode) {
      case 'deny':
      case 'revoke':
        return 'danger' as const;
      case 'approve':
        return 'success' as const;
      default:
        return 'primary' as const;
    }
  };

  const isLoading =
    createMutation.isPending ||
    updateMutation.isPending ||
    approveMutation.isPending ||
    denyMutation.isPending ||
    revokeMutation.isPending;

  const isReviewMode = mode === 'approve' || mode === 'deny' || mode === 'revoke';

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={getDialogTitle()}
      size="lg"
      footer={
        <>
          <Button variant="ghost" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            variant={getSubmitButtonVariant()}
            onClick={handleSubmit}
            disabled={isLoading}
            isLoading={isLoading}
          >
            {getSubmitButtonText()}
          </Button>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Exception Details */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white flex items-center gap-2">
            <AlertTriangle className="h-4 w-4" />
            Exception Details
          </h4>

          {exception && (
            <div className="rounded-lg border border-gray-700 bg-gray-800/50 p-3">
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>
                  <span className="text-gray-400">Control ID:</span>
                  <span className="ml-2 text-white">{exception.control_id}</span>
                </div>
                <div>
                  <span className="text-gray-400">Status:</span>
                  <span className={`ml-2 capitalize ${
                    exception.status === 'approved' ? 'text-success-400' :
                    exception.status === 'denied' ? 'text-danger-400' :
                    exception.status === 'expired' ? 'text-warning-400' :
                    'text-primary-400'
                  }`}>
                    {exception.status}
                  </span>
                </div>
              </div>
            </div>
          )}

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="control-id">Control ID *</Label>
              <Input
                id="control-id"
                value={formData.control_id}
                onChange={(e) => handleInputChange('control_id', e.target.value)}
                placeholder="AC-001"
                disabled={isReviewMode || !!exception}
                required
              />
            </div>

            <div>
              <Label htmlFor="control-name">Control Name *</Label>
              <Input
                id="control-name"
                value={formData.control_name}
                onChange={(e) => handleInputChange('control_name', e.target.value)}
                placeholder="Access Control Policy"
                disabled={isReviewMode}
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label htmlFor="framework">Compliance Framework *</Label>
              <select
                id="framework"
                value={formData.framework}
                onChange={(e) => handleInputChange('framework', e.target.value)}
                disabled={isReviewMode || !!exception}
                className="w-full rounded-lg border border-gray-700 bg-gray-800 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                required
              >
                {frameworks.map((fw) => (
                  <option key={fw.value} value={fw.value}>
                    {fw.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <Label htmlFor="expires-at">Expiration Date (Optional)</Label>
              <Input
                id="expires-at"
                type="date"
                value={formData.expires_at}
                onChange={(e) => handleInputChange('expires_at', e.target.value)}
                disabled={isReviewMode}
                min={format(new Date(), 'yyyy-MM-dd')}
              />
            </div>
          </div>
        </div>

        {/* Justification */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-white flex items-center gap-2">
            <FileText className="h-4 w-4" />
            Justification
          </h4>

          <div>
            <Label htmlFor="reason">Reason for Exception *</Label>
            <Textarea
              id="reason"
              value={formData.reason}
              onChange={(e) => handleInputChange('reason', e.target.value)}
              placeholder="Briefly describe why this exception is needed"
              rows={2}
              disabled={isReviewMode}
              required
            />
          </div>

          <div>
            <Label htmlFor="business-justification">Business Justification *</Label>
            <Textarea
              id="business-justification"
              value={formData.business_justification}
              onChange={(e) => handleInputChange('business_justification', e.target.value)}
              placeholder="Explain the business case and why compliance cannot be met at this time"
              rows={3}
              disabled={isReviewMode}
              required
            />
          </div>

          <div>
            <Label htmlFor="mitigation-plan">Mitigation Plan (Optional)</Label>
            <Textarea
              id="mitigation-plan"
              value={formData.mitigation_plan}
              onChange={(e) => handleInputChange('mitigation_plan', e.target.value)}
              placeholder="Describe compensating controls or mitigation strategies"
              rows={2}
              disabled={isReviewMode}
            />
          </div>
        </div>

        {/* Document Upload */}
        {!isReviewMode && (
          <div className="space-y-4">
            <h4 className="text-sm font-semibold text-white flex items-center gap-2">
              <Upload className="h-4 w-4" />
              Supporting Documents (Optional)
            </h4>

            <div className="rounded-lg border border-dashed border-gray-700 p-4">
              <input
                type="file"
                id="document-upload"
                multiple
                onChange={handleFileChange}
                className="hidden"
              />
              <label
                htmlFor="document-upload"
                className="flex flex-col items-center justify-center cursor-pointer"
              >
                <Upload className="mb-2 h-8 w-8 text-gray-500" />
                <span className="text-sm text-gray-400">
                  Click to upload or drag and drop
                </span>
                <span className="text-xs text-gray-500">
                  PDF, DOC, DOCX, XLS, XLSX up to 10MB
                </span>
              </label>
            </div>

            {files.length > 0 && (
              <div className="space-y-2">
                {files.map((file, index) => (
                  <div
                    key={index}
                    className="flex items-center justify-between rounded border border-gray-700 bg-gray-800 px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <FileText className="h-4 w-4 text-gray-400" />
                      <span className="text-sm text-white">{file.name}</span>
                      <span className="text-xs text-gray-500">
                        ({(file.size / 1024).toFixed(1)} KB)
                      </span>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeFile(index)}
                      className="text-gray-400 hover:text-danger-400"
                    >
                      <X className="h-4 w-4" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Review Mode - Approval/Denial Notes */}
        {isReviewMode && (
          <div className="space-y-4">
            <h4 className="text-sm font-semibold text-white">Review Decision</h4>

            {mode === 'deny' && (
              <div>
                <Label htmlFor="denial-reason">Reason for Denial *</Label>
                <Textarea
                  id="denial-reason"
                  value={formData.denial_reason}
                  onChange={(e) => handleInputChange('denial_reason', e.target.value)}
                  placeholder="Explain why this exception request is being denied"
                  rows={3}
                  required
                />
              </div>
            )}

            {(mode === 'approve' || mode === 'revoke') && (
              <div>
                <Label htmlFor="review-notes">Review Notes</Label>
                <Textarea
                  id="review-notes"
                  value={formData.review_notes}
                  onChange={(e) => handleInputChange('review_notes', e.target.value)}
                  placeholder="Add any additional notes for the audit trail"
                  rows={2}
                />
              </div>
            )}
          </div>
        )}

        {/* Info Box */}
        <div className="flex gap-2 rounded-lg border border-warning-500/30 bg-warning-500/10 p-3">
          <AlertTriangle className="h-4 w-4 text-warning-400 mt-0.5 flex-shrink-0" />
          <div className="text-xs text-gray-300">
            {mode === 'create' || mode === 'edit' ? (
              <>
                Compliance exceptions are auditable decisions that require business justification.
                All exceptions are subject to review and approval by authorized personnel.
                Active exceptions are visible in compliance reports and assessments.
              </>
            ) : mode === 'approve' ? (
              <>
                Approving this exception will allow the control to be marked as compliant with
                the documented mitigations. This action will be logged in the audit trail.
              </>
            ) : mode === 'deny' ? (
              <>
                Denying this exception will require the control to be fully compliant.
                The requester will be notified of the denial reason.
              </>
            ) : (
              <>
                Revoking this exception will immediately mark the associated control as
                non-compliant. This action cannot be undone and will be logged in the audit trail.
              </>
            )}
          </div>
        </div>
      </form>
    </Modal>
  );
};

// Helper function wrapper
function useQueryQueryClient() {
  return useQueryClient();
}

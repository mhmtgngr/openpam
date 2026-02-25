/**
 * EmptyState - Component for displaying empty states
 * Used when there's no data to display or when content is loading
 */

import React from 'react';
import { FileText, Inbox } from 'lucide-react';
import { Button } from './Button';

export interface EmptyStateProps {
  title?: string;
  description?: string;
  action?: React.ReactNode;
  icon?: React.ReactNode;
}

export const EmptyState: React.FC<EmptyStateProps> = ({
  title = 'No data found',
  description,
  action,
  icon,
}) => {
  const defaultIcon = icon || <Inbox className="h-12 w-12 text-gray-600" />;

  return (
    <div className="flex flex-col items-center justify-center py-12 text-center">
      <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-full bg-gray-800">
        {defaultIcon}
      </div>
      <h3 className="text-lg font-semibold text-white">{title}</h3>
      {description && (
        <p className="mt-2 text-sm text-gray-400 max-w-sm">{description}</p>
      )}
      {action && <div className="mt-6">{action}</div>}
    </div>
  );
};

export default EmptyState;

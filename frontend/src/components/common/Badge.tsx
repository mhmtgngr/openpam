import React from 'react';
import clsx from 'clsx';

type Variant = 'success' | 'warning' | 'danger' | 'info' | 'neutral';

export interface BadgeProps {
  children: React.ReactNode;
  variant?: Variant;
  size?: 'sm' | 'md';
  dot?: boolean;
  className?: string;
}

export const Badge: React.FC<BadgeProps> = ({
  children,
  variant = 'neutral',
  size = 'sm',
  dot = false,
  className,
}) => {
  const variantStyles = {
    success: 'badge-success',
    warning: 'badge-warning',
    danger: 'badge-danger',
    info: 'badge-info',
    neutral: 'badge-neutral',
  };

  const sizeStyles = {
    sm: 'text-xs px-2 py-0.5',
    md: 'text-sm px-2.5 py-1',
  };

  return (
    <span className={clsx('badge', variantStyles[variant], sizeStyles[size], className)}>
      {dot && (
        <span className="mr-1.5 h-1.5 w-1.5 rounded-full bg-current" />
      )}
      {children}
    </span>
  );
};

// Status-specific badge helpers
export const StatusBadge: React.FC<{ status: string; className?: string }> = ({ status, className }) => {
  const variantMap: Record<string, Variant> = {
    active: 'success',
    online: 'success',
    approved: 'success',
    completed: 'success',
    pending: 'warning',
    warning: 'warning',
    expiring: 'warning',
    offline: 'danger',
    inactive: 'neutral',
    suspended: 'danger',
    denied: 'danger',
    terminated: 'danger',
    failed: 'danger',
    expired: 'neutral',
    checked_in: 'neutral',
  };

  const variant = variantMap[status.toLowerCase()] || 'neutral';

  return (
    <Badge variant={variant} className={className}>
      {status}
    </Badge>
  );
};

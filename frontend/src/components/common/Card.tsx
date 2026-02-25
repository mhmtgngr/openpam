import React from 'react';
import clsx from 'clsx';

export interface CardProps {
  children: React.ReactNode;
  className?: string;
  noPadding?: boolean;
  onClick?: () => void;
}

export const Card: React.FC<CardProps> = ({ children, className, noPadding, onClick }) => {
  return (
    <div className={clsx('card', className)} onClick={onClick}>
      {noPadding ? children : <div className="card-body">{children}</div>}
    </div>
  );
};

export const CardHeader: React.FC<{
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
  icon?: React.ReactNode;
}> = ({ title, subtitle, action, icon }) => {
  return (
    <div className="card-header flex items-center justify-between">
      <div className="flex items-center gap-3">
        {icon && <div className="text-gray-400">{icon}</div>}
        <div>
          <h3 className="text-lg font-semibold text-white">{title}</h3>
          {subtitle && <p className="text-sm text-gray-400">{subtitle}</p>}
        </div>
      </div>
      {action && <div>{action}</div>}
    </div>
  );
};

export const CardFooter: React.FC<{
  children: React.ReactNode;
  className?: string;
}> = ({ children, className }) => {
  return <div className={clsx('card-footer', className)}>{children}</div>;
};

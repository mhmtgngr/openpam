import React from 'react';
import clsx from 'clsx';

export interface LabelProps extends React.LabelHTMLAttributes<HTMLLabelElement> {
  required?: boolean;
}

export const Label: React.FC<LabelProps> = ({
  children,
  required = false,
  className,
  ...props
}) => {
  return (
    <label
      className={clsx('block text-sm font-medium text-gray-300 mb-1', className)}
      {...props}
    >
      {children}
      {required && <span className="text-danger-400 ml-1">*</span>}
    </label>
  );
};

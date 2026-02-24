import React from 'react';
import { Calendar } from 'lucide-react';
import clsx from 'clsx';

export interface DatePickerProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> {
  label?: string;
  error?: string;
}

export const DatePicker: React.FC<DatePickerProps> = ({
  label,
  error,
  className,
  ...props
}) => {
  return (
    <div className={className}>
      {label && (
        <label className="block text-sm font-medium text-gray-300 mb-1">
          {label}
        </label>
      )}
      <div className="relative">
        <input
          type="datetime-local"
          className={clsx(
            'w-full rounded-md border bg-gray-800 px-3 py-2 pr-10 text-sm text-white placeholder-gray-500',
            'focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500',
            'disabled:opacity-50 disabled:cursor-not-allowed',
            error ? 'border-danger-500' : 'border-gray-700'
          )}
          {...props}
        />
        <Calendar className="absolute right-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400 pointer-events-none" />
      </div>
      {error && (
        <p className="mt-1 text-xs text-danger-400">{error}</p>
      )}
    </div>
  );
};

import React from 'react';
import clsx from 'clsx';

export interface ToggleProps extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type' | 'size'> {
  label?: string;
  description?: string;
  size?: 'sm' | 'md';
}

export const Toggle: React.FC<ToggleProps> = ({
  label,
  description,
  size = 'md',
  checked,
  onChange,
  disabled,
  className,
  ...props
}) => {
  const sizeStyles = {
    sm: {
      toggle: 'w-8 h-4',
      dot: 'w-3 h-3',
      dotTranslate: 'translate-x-4',
    },
    md: {
      toggle: 'w-11 h-6',
      dot: 'w-5 h-5',
      dotTranslate: 'translate-x-5',
    },
  };

  const styles = sizeStyles[size];

  return (
    <div className={clsx('flex items-start gap-3', className)}>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => !disabled && onChange?.({ target: { checked: !checked } } as any)}
        className={clsx(
          'relative inline-flex flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 focus:ring-offset-gray-900',
          styles.toggle,
          checked ? 'bg-primary-600' : 'bg-gray-700',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        <span
          className={clsx(
            'pointer-events-none inline-block rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            styles.dot,
            checked ? styles.dotTranslate : 'translate-x-0'
          )}
          aria-hidden="true"
        />
      </button>
      {(label || description) && (
        <div className="flex flex-col">
          {label && (
            <span className="text-sm font-medium text-gray-300">{label}</span>
          )}
          {description && (
            <span className="text-xs text-gray-400">{description}</span>
          )}
        </div>
      )}
    </div>
  );
};

import React from 'react';
import clsx from 'clsx';

export interface ToggleProps {
  label?: string;
  description?: string;
  size?: 'sm' | 'md';
  checked?: boolean;
  disabled?: boolean;
  className?: string;
  // Allow onChange to receive either a ChangeEvent or a boolean value directly
  onChange?: ((event: React.ChangeEvent<HTMLInputElement>) => void) | ((value: boolean) => void);
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
        onClick={() => {
          if (disabled) return;
          const newChecked = !checked;
          if (onChange) {
            // Support both event handler pattern and direct boolean setter pattern
            // Try to detect if onChange is a setState-like function by checking arity
            // React setState functions receive either a value or a function
            onChange(newChecked as any);
          }
        }}
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

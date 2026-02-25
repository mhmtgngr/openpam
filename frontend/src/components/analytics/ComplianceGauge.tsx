/**
 * ComplianceGauge - Visual gauge component for displaying compliance scores
 * Shows overall compliance percentage with color-coded indicators
 */

import React from 'react';
import clsx from 'clsx';

export interface ComplianceGaugeProps {
  score: number; // 0-100
  label?: string;
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showPercentage?: boolean;
  showLabel?: boolean;
  threshold?: {
    warning: number;
    success: number;
  };
  className?: string;
  animated?: boolean;
}

const sizeConfig = {
  sm: { size: 80, strokeWidth: 8, text: 'text-lg' },
  md: { size: 120, strokeWidth: 10, text: 'text-2xl' },
  lg: { size: 160, strokeWidth: 12, text: 'text-3xl' },
  xl: { size: 200, strokeWidth: 14, text: 'text-4xl' },
};

export const ComplianceGauge: React.FC<ComplianceGaugeProps> = ({
  score,
  label,
  size = 'md',
  showPercentage = true,
  showLabel = true,
  threshold = { warning: 60, success: 80 },
  className,
  animated = true,
}) => {
  const config = sizeConfig[size];
  const radius = (config.size - config.strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const strokeDashoffset = circumference - (score / 100) * circumference;

  const getScoreColor = () => {
    if (score >= threshold.success) return {
      color: 'text-success-400',
      track: 'stroke-success-500',
      bg: 'bg-success-500/10',
      bar: 'bg-success-500',
    };
    if (score >= threshold.warning) return {
      color: 'text-warning-400',
      track: 'stroke-warning-500',
      bg: 'bg-warning-500/10',
      bar: 'bg-warning-500',
    };
    return {
      color: 'text-danger-400',
      track: 'stroke-danger-500',
      bg: 'bg-danger-500/10',
      bar: 'bg-danger-500',
    };
  };

  const scoreColor = getScoreColor();

  return (
    <div className={clsx('flex flex-col items-center', className)}>
      {/* SVG Gauge */}
      <div
        className={clsx('relative', scoreColor.bg)}
        style={{ width: config.size, height: config.size }}
      >
        <svg className="h-full w-full" viewBox={`0 0 ${config.size} ${config.size}`}>
          {/* Background Circle */}
          <circle
            cx={config.size / 2}
            cy={config.size / 2}
            r={radius}
            fill="none"
            className="stroke-gray-700"
            strokeWidth={config.strokeWidth}
          />

          {/* Progress Circle */}
          <circle
            cx={config.size / 2}
            cy={config.size / 2}
            r={radius}
            fill="none"
            className={clsx(
              scoreColor.track,
              'transition-all duration-1000 ease-out',
              animated && 'animate-pulse-slow'
            )}
            strokeWidth={config.strokeWidth}
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            strokeLinecap="round"
            transform={`rotate(-90 ${config.size / 2} ${config.size / 2})`}
          />
        </svg>

        {/* Center Text */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className={clsx('font-bold', config.text, scoreColor.color)}>
            {showPercentage ? `${score}%` : score}
          </span>
        </div>
      </div>

      {/* Label */}
      {showLabel && label && (
        <span className="mt-3 text-sm font-medium text-gray-400">{label}</span>
      )}
    </div>
  );
};

export default ComplianceGauge;

/**
 * MultiGauge - Multiple gauges in a row for comparison
 */
export interface MultiGaugeProps {
  data: Array<{
    label: string;
    score: number;
    color?: 'success' | 'warning' | 'danger' | 'neutral';
  }>;
  size?: 'sm' | 'md';
  showLabels?: boolean;
}

export const MultiGauge: React.FC<MultiGaugeProps> = ({
  data,
  size = 'md',
  showLabels = true,
}) => {
  const colorClasses = {
    success: 'text-success-400 bg-success-500/10',
    warning: 'text-warning-400 bg-warning-500/10',
    danger: 'text-danger-400 bg-danger-500/10',
    neutral: 'text-gray-400 bg-gray-500/10',
  };

  return (
    <div className="flex items-center justify-center gap-6 flex-wrap">
      {data.map((item, index) => (
        <div key={index} className="flex flex-col items-center">
          <ComplianceGauge
            score={item.score}
            size={size === 'md' ? 'sm' : 'sm'}
            showPercentage
            showLabel={false}
          />
          {showLabels && (
            <span className={clsx(
              'mt-2 text-xs font-medium uppercase px-2 py-1 rounded',
              colorClasses[item.color || (
                item.score >= 80 ? 'success' : item.score >= 60 ? 'warning' : 'danger'
              )]
            )}>
              {item.label}
            </span>
          )}
          <span className="text-sm font-bold text-white mt-1">{item.score}%</span>
        </div>
      ))}
    </div>
  );
};

/**
 * ComplianceBar - Horizontal progress bar version
 */
export interface ComplianceBarProps {
  score: number;
  label?: string;
  showThresholds?: boolean;
  threshold?: {
    warning: number;
    success: number;
  };
  height?: 'sm' | 'md' | 'lg';
}

export const ComplianceBar: React.FC<ComplianceBarProps> = ({
  score,
  label,
  showThresholds = true,
  threshold = { warning: 60, success: 80 },
  height = 'md',
}) => {
  const heightClass = {
    sm: 'h-2',
    md: 'h-3',
    lg: 'h-4',
  }[height];

  const getScoreColor = () => {
    if (score >= threshold.success) return 'bg-success-500';
    if (score >= threshold.warning) return 'bg-warning-500';
    return 'bg-danger-500';
  };

  const scoreColor = getScoreColor();

  return (
    <div className="space-y-1">
      {(label || showThresholds) && (
        <div className="flex items-center justify-between text-xs">
          {label && <span className="text-gray-400">{label}</span>}
          {showThresholds && (
            <div className="flex items-center gap-3 text-gray-500">
              <span>{threshold.success}%+</span>
              <span>{threshold.warning}%+</span>
            </div>
          )}
        </div>
      )}
      <div className="relative">
        {/* Background bar */}
        <div className={clsx('w-full rounded-full bg-gray-800', heightClass)} />
        {/* Progress bar */}
        <div
          className={clsx(
            'absolute top-0 left-0 rounded-full transition-all duration-500',
            scoreColor,
            heightClass
          )}
          style={{ width: `${Math.min(score, 100)}%` }}
        />
        {/* Threshold markers */}
        {showThresholds && (
          <>
            <div
              className="absolute top-0 bottom-0 w-0.5 bg-gray-600"
              style={{ left: `${threshold.warning}%` }}
            />
            <div
              className="absolute top-0 bottom-0 w-0.5 bg-gray-600"
              style={{ left: `${threshold.success}%` }}
            />
          </>
        )}
      </div>
      <div className="flex justify-between">
        <span className="text-xs text-gray-500">0%</span>
        <span className={clsx('text-sm font-bold', score === 100 ? 'text-success-400' : 'text-white')}>
          {score}%
        </span>
        <span className="text-xs text-gray-500">100%</span>
      </div>
    </div>
  );
};

/**
 * ComplianceBreakdown - Detailed breakdown with multiple categories
 */
export interface ComplianceBreakdownProps {
  categories: Array<{
    name: string;
    score: number;
    count: number;
    total: number;
  }>;
  showBars?: boolean;
}

export const ComplianceBreakdown: React.FC<ComplianceBreakdownProps> = ({
  categories,
  showBars = true,
}) => {
  const getScoreColor = (score: number) => {
    if (score >= 80) return 'bg-success-500';
    if (score >= 60) return 'bg-warning-500';
    return 'bg-danger-500';
  };

  return (
    <div className="space-y-4">
      {categories.map((category, index) => (
        <div key={index} className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-white">{category.name}</span>
            <div className="flex items-center gap-2">
              <span className="text-xs text-gray-500">
                {category.count}/{category.total}
              </span>
              <span className={clsx(
                'text-sm font-bold',
                category.score >= 80 ? 'text-success-400' :
                category.score >= 60 ? 'text-warning-400' :
                'text-danger-400'
              )}>
                {category.score}%
              </span>
            </div>
          </div>
          {showBars && (
            <div className="relative h-2 w-full rounded-full bg-gray-800">
              <div
                className={clsx(
                  'absolute top-0 left-0 h-full rounded-full transition-all duration-500',
                  getScoreColor(category.score)
                )}
                style={{ width: `${category.score}%` }}
              />
            </div>
          )}
        </div>
      ))}
    </div>
  );
};

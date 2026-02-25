/**
<<<<<<< HEAD
 * ComplianceGauge Component
 *
 * Visualizes compliance score as an arc gauge with color coding.
 * Supports different sizes, animations, and thresholds for compliance levels.
 */

import React, { useMemo } from 'react';

interface ComplianceGaugeProps {
  score: number; // 0-100
  size?: 'sm' | 'md' | 'lg' | 'xl';
  showLabel?: boolean;
  showValue?: boolean;
  animated?: boolean;
  thresholds?: {
    warning: number;
    success: number;
  };
  label?: string;
  className?: string;
  formatValue?: (value: number) => string;
}

const SIZE_CONFIG = {
  sm: { width: 80, height: 48, strokeWidth: 8, fontSize: 'text-lg' },
  md: { width: 120, height: 70, strokeWidth: 12, fontSize: 'text-2xl' },
  lg: { width: 160, height: 96, strokeWidth: 16, fontSize: 'text-3xl' },
  xl: { width: 200, height: 120, strokeWidth: 20, fontSize: 'text-4xl' },
};

const DEFAULT_THRESHOLDS = {
  warning: 70,
  success: 90,
=======
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
>>>>>>> team/fix-srcapianalyticstestts-typescript-com-1772017573
};

export const ComplianceGauge: React.FC<ComplianceGaugeProps> = ({
  score,
<<<<<<< HEAD
  size = 'md',
  showLabel = true,
  showValue = true,
  animated = true,
  thresholds = DEFAULT_THRESHOLDS,
  label = 'Compliance',
  className = '',
  formatValue,
}) => {
  const config = SIZE_CONFIG[size];

  // Calculate gauge properties
  const gaugeProps = useMemo(() => {
    const clampedScore = Math.max(0, Math.min(100, score));
    const radius = (config.width - config.strokeWidth) / 2;
    const circumference = radius * Math.PI; // Semi-circle (180 degrees)
    const strokeDasharray = circumference;
    const strokeDashoffset = circumference - (clampedScore / 100) * circumference;

    // Determine color based on thresholds
    let color = '#ef4444'; // red
    if (clampedScore >= thresholds.success) {
      color = '#22c55e'; // green
    } else if (clampedScore >= thresholds.warning) {
      color = '#eab308'; // yellow
    }

    return {
      radius,
      circumference,
      strokeDashoffset,
      color,
      clampedScore,
    };
  }, [score, config, thresholds]);

  // Animation class
  const animationClass = animated ? 'transition-all duration-700 ease-out' : '';

  // Format display value
  const displayValue = formatValue ? formatValue(gaugeProps.clampedScore) : `${Math.round(gaugeProps.clampedScore)}%`;

  return (
    <div className={`compliance-gauge inline-flex flex-col items-center ${className}`}>
      <svg
        width={config.width}
        height={config.height}
        viewBox={`0 0 ${config.width} ${config.height}`}
        className="overflow-visible"
      >
        {/* Background arc */}
        <path
          d={`M ${config.strokeWidth / 2} ${config.height - config.strokeWidth / 2} A ${gaugeProps.radius} ${gaugeProps.radius} 0 0 1 ${config.width - config.strokeWidth / 2} ${config.height - config.strokeWidth / 2}`}
          fill="none"
          stroke="currentColor"
          strokeWidth={config.strokeWidth}
          strokeLinecap="round"
          className="text-gray-200 dark:text-gray-700"
        />

        {/* Foreground arc (colored) */}
        <path
          d={`M ${config.strokeWidth / 2} ${config.height - config.strokeWidth / 2} A ${gaugeProps.radius} ${gaugeProps.radius} 0 0 1 ${config.width - config.strokeWidth / 2} ${config.height - config.strokeWidth / 2}`}
          fill="none"
          stroke={gaugeProps.color}
          strokeWidth={config.strokeWidth}
          strokeLinecap="round"
          strokeDasharray={gaugeProps.circumference}
          strokeDashoffset={gaugeProps.strokeDashoffset}
          className={`${animationClass}`}
          style={{
            transformOrigin: 'center',
          }}
        />

        {/* Value text */}
        {showValue && (
          <text
            x={config.width / 2}
            y={config.height - 10}
            textAnchor="middle"
            className={`font-bold ${config.fontSize} fill-gray-900 dark:fill-gray-100`}
          >
            {displayValue}
          </text>
        )}
      </svg>

      {showLabel && (
        <div className="text-sm font-medium text-gray-600 dark:text-gray-400 mt-2">
          {label}
        </div>
      )}
    </div>
  );
};

// Mini compliance badge version
interface ComplianceBadgeProps {
  score: number;
  showScore?: boolean;
  className?: string;
}

export const ComplianceBadge: React.FC<ComplianceBadgeProps> = ({
  score,
  showScore = true,
  className = '',
}) => {
  const status = useMemo(() => {
    if (score >= 90) return { label: 'Compliant', color: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200' };
    if (score >= 70) return { label: 'Partial', color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200' };
    return { label: 'Non-Compliant', color: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200' };
  }, [score]);

  return (
    <span className={`inline-flex items-center gap-2 px-3 py-1 rounded-full text-sm font-medium ${status.color} ${className}`}>
      {showScore && <span>{score}%</span>}
      <span>{status.label}</span>
    </span>
  );
};

// Detailed compliance score card with breakdown
interface ComplianceScoreCardProps {
  score: number;
  framework: string;
  lastAssessed?: string;
  controls?: {
    compliant: number;
    partial: number;
    nonCompliant: number;
    notApplicable: number;
  };
  breakdown?: Array<{
    category: string;
    score: number;
    count: number;
  }>;
  className?: string;
}

export const ComplianceScoreCard: React.FC<ComplianceScoreCardProps> = ({
  score,
  framework,
  lastAssessed,
  controls,
  breakdown,
  className = '',
}) => {
  const totalControls = controls
    ? controls.compliant + controls.partial + controls.nonCompliant + controls.notApplicable
    : 0;

  return (
    <div className={`compliance-score-card bg-white dark:bg-gray-800 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700 p-6 ${className}`}>
      <div className="flex items-start justify-between">
        <div>
          <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
            {framework.replace('_', ' ').toUpperCase()} Compliance
          </h3>
          {lastAssessed && (
            <p className="text-sm text-gray-500 mt-1">
              Last assessed: {new Date(lastAssessed).toLocaleDateString()}
            </p>
          )}
        </div>
        <ComplianceBadge score={score} />
      </div>

      <div className="flex items-center justify-center py-6">
        <ComplianceGauge score={score} size="lg" />
      </div>

      {controls && (
        <div className="grid grid-cols-4 gap-4 mt-4 pt-4 border-t border-gray-200 dark:border-gray-700">
          <div className="text-center">
            <div className="text-2xl font-bold text-green-600 dark:text-green-400">
              {controls.compliant}
            </div>
            <div className="text-xs text-gray-500">Compliant</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-yellow-600 dark:text-yellow-400">
              {controls.partial}
            </div>
            <div className="text-xs text-gray-500">Partial</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-red-600 dark:text-red-400">
              {controls.nonCompliant}
            </div>
            <div className="text-xs text-gray-500">Non-Compliant</div>
          </div>
          <div className="text-center">
            <div className="text-2xl font-bold text-gray-600 dark:text-gray-400">
              {controls.notApplicable}
            </div>
            <div className="text-xs text-gray-500">N/A</div>
          </div>
        </div>
      )}

      {breakdown && breakdown.length > 0 && (
        <div className="mt-6">
          <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
            Score by Category
          </h4>
          <div className="space-y-3">
            {breakdown.map((item) => (
              <div key={item.category} className="flex items-center gap-3">
                <div className="flex-1">
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-gray-700 dark:text-gray-300">{item.category}</span>
                    <span className="font-medium text-gray-900 dark:text-white">{item.score}%</span>
                  </div>
                  <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
                    <div
                      className={`h-2 rounded-full transition-all duration-500 ${
                        item.score >= 90
                          ? 'bg-green-500'
                          : item.score >= 70
                          ? 'bg-yellow-500'
                          : 'bg-red-500'
                      }`}
                      style={{ width: `${item.score}%` }}
                    />
                  </div>
                  <div className="text-xs text-gray-500 mt-1">{item.count} controls</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

// Compact compliance meter (horizontal bar)
interface ComplianceMeterProps {
  score: number;
  height?: number;
  showMarkers?: boolean;
  showLabels?: boolean;
  className?: string;
}

export const ComplianceMeter: React.FC<ComplianceMeterProps> = ({
  score,
  height = 12,
  showMarkers = true,
  showLabels = true,
  className = '',
}) => {
  const color = score >= 90 ? 'bg-green-500' : score >= 70 ? 'bg-yellow-500' : 'bg-red-500';

  return (
    <div className={`compliance-meter w-full ${className}`}>
      <div className="relative h-full">
        {/* Background track */}
        <div
          className="absolute inset-0 bg-gray-200 dark:bg-gray-700 rounded-full"
          style={{ height: `${height}px` }}
        />

        {/* Progress bar */}
        <div
          className={`absolute left-0 top-0 ${color} rounded-full transition-all duration-500 ease-out`}
          style={{ width: `${score}%`, height: `${height}px` }}
        />

        {/* Markers */}
        {showMarkers && (
          <>
            <div
              className="absolute top-0 bottom-0 w-0.5 bg-gray-400 dark:bg-gray-500"
              style={{ left: '70%' }}
            />
            <div
              className="absolute top-0 bottom-0 w-0.5 bg-gray-400 dark:bg-gray-500"
              style={{ left: '90%' }}
            />
          </>
        )}
      </div>

      {showLabels && (
        <div className="flex justify-between text-xs text-gray-500 mt-1">
          <span>0%</span>
          <span>70%</span>
          <span>90%</span>
          <span>100%</span>
        </div>
=======
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
>>>>>>> team/fix-srcapianalyticstestts-typescript-com-1772017573
      )}
    </div>
  );
};

export default ComplianceGauge;
<<<<<<< HEAD
=======

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
>>>>>>> team/fix-srcapianalyticstestts-typescript-com-1772017573

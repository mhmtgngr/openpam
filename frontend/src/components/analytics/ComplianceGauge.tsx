/**
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
};

export const ComplianceGauge: React.FC<ComplianceGaugeProps> = ({
  score,
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
      )}
    </div>
  );
};

export default ComplianceGauge;

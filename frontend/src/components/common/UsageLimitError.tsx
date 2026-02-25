import React from 'react';
import { Clock, AlertTriangle, RefreshCw } from 'lucide-react';
import clsx from 'clsx';

export interface UsageLimitErrorProps {
  code: string;
  message: string;
  resetAt: string;
  requestId?: string;
  onRetry?: () => void;
  isInline?: boolean;
}

export const UsageLimitError: React.FC<UsageLimitErrorProps> = ({
  code,
  message,
  resetAt,
  requestId,
  onRetry,
  isInline = false,
}) => {
  const [timeRemaining, setTimeRemaining] = React.useState<string>('');
  const [isRetrying, setIsRetrying] = React.useState(false);

  // Calculate time remaining
  React.useEffect(() => {
    const calculateTimeRemaining = () => {
      const resetTime = new Date(resetAt).getTime();
      const now = Date.now();
      const diff = resetTime - now;

      if (diff <= 0) {
        return 'Limit should be reset';
      }

      const hours = Math.floor(diff / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      if (hours > 0) {
        return `${hours}h ${minutes}m ${seconds}s`;
      } else if (minutes > 0) {
        return `${minutes}m ${seconds}s`;
      } else {
        return `${seconds}s`;
      }
    };

    setTimeRemaining(calculateTimeRemaining());

    const interval = setInterval(() => {
      setTimeRemaining(calculateTimeRemaining());
    }, 1000);

    return () => clearInterval(interval);
  }, [resetAt]);

  const handleRetry = () => {
    if (onRetry) {
      setIsRetrying(true);
      onRetry();
      setTimeout(() => setIsRetrying(false), 2000);
    }
  };

  const resetTimeFormatted = new Date(resetAt).toLocaleString();

  if (isInline) {
    return (
      <div className="rounded-lg bg-warning-500/10 border border-warning-500/30 p-4">
        <div className="flex items-start gap-3">
          <AlertTriangle className="h-5 w-5 text-warning-500 flex-shrink-0 mt-0.5" />
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-warning-500">Usage Limit Reached</p>
            <p className="mt-1 text-sm text-gray-400">{message}</p>
            <div className="mt-2 flex items-center gap-2 text-xs text-gray-500">
              <Clock className="h-3 w-3" />
              <span>Resets in {timeRemaining}</span>
            </div>
          </div>
          {onRetry && (
            <button
              onClick={handleRetry}
              disabled={isRetrying}
              className="flex-shrink-0 text-warning-500 hover:text-warning-400 transition-colors disabled:opacity-50"
              title="Retry now"
            >
              <RefreshCw className={clsx('h-4 w-4', isRetrying && 'animate-spin')} />
            </button>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-[400px] items-center justify-center">
      <div className="max-w-md w-full mx-auto">
        <div className="card">
          <div className="flex flex-col items-center text-center py-8">
            {/* Icon */}
            <div className="relative">
              <div className="absolute inset-0 bg-warning-500/20 rounded-full blur-xl" />
              <div className="relative bg-gradient-to-br from-warning-500/20 to-orange-500/20 rounded-full p-6 border border-warning-500/30">
                <Clock className="h-12 w-12 text-warning-500" />
              </div>
            </div>

            {/* Title */}
            <h2 className="mt-6 text-2xl font-bold text-white">
              Usage Limit Reached
            </h2>

            {/* Error Code Badge */}
            <div className="mt-3">
              <span className="inline-flex items-center px-3 py-1 rounded-full text-xs font-medium bg-warning-500/10 text-warning-500 border border-warning-500/20">
                Error Code: {code}
              </span>
            </div>

            {/* Message */}
            <p className="mt-4 text-gray-400">
              {message}
            </p>

            {/* Countdown Timer */}
            <div className="mt-6 w-full">
              <div className="rounded-lg bg-gray-800/50 border border-gray-700 p-4">
                <p className="text-xs text-gray-500 uppercase tracking-wide mb-2">
                  Time Until Reset
                </p>
                <p className="text-3xl font-mono font-bold text-warning-500">
                  {timeRemaining}
                </p>
                <p className="mt-2 text-xs text-gray-500">
                  Resets at: {resetTimeFormatted}
                </p>
              </div>
            </div>

            {/* Request ID */}
            {requestId && (
              <div className="mt-4 text-xs text-gray-600">
                Request ID: {requestId}
              </div>
            )}

            {/* Actions */}
            <div className="mt-8 flex flex-col sm:flex-row gap-3 w-full">
              {onRetry && (
                <button
                  onClick={handleRetry}
                  disabled={isRetrying}
                  className={clsx(
                    'flex-1 btn btn-primary',
                    isRetrying && 'opacity-75 cursor-wait'
                  )}
                >
                  {isRetrying ? (
                    <>
                      <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      Retrying...
                    </>
                  ) : (
                    <>
                      <RefreshCw className="mr-2 h-4 w-4" />
                      Retry Now
                    </>
                  )}
                </button>
              )}
              <button
                onClick={() => window.location.reload()}
                className="flex-1 btn btn-secondary"
              >
                Reload Page
              </button>
            </div>

            {/* Help Text */}
            <p className="mt-6 text-xs text-gray-500">
              Need higher limits? Contact your administrator to upgrade your plan.
            </p>
          </div>
        </div>

        {/* Additional Info Card */}
        <div className="mt-4 card">
          <div className="flex items-start gap-3">
            <AlertTriangle className="h-5 w-5 text-warning-500 flex-shrink-0 mt-0.5" />
            <div className="flex-1">
              <h4 className="text-sm font-medium text-white">About Usage Limits</h4>
              <p className="mt-1 text-xs text-gray-400">
                Usage limits help ensure fair resource allocation across all users.
                Limits reset automatically at the scheduled time shown above.
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default UsageLimitError;

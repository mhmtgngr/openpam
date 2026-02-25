import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import UsageLimitError from '@/components/common/UsageLimitError';

interface LocationState {
  errorCode?: string;
  errorMessage?: string;
  resetAt?: string;
  requestId?: string;
}

const UsageLimitPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const state = location.state as LocationState;

  // Try to get error details from multiple sources
  const errorCode = state?.errorCode || localStorage.getItem('usage_limit_code') || '1308';
  const errorMessage = state?.errorMessage ||
    localStorage.getItem('usage_limit_message') ||
    'Usage limit reached. Please try again later.';
  const resetAt = state?.resetAt ||
    localStorage.getItem('usage_limit_reset_at') ||
    new Date(Date.now() + 5 * 60 * 60 * 1000).toISOString();
  const requestId = state?.requestId || localStorage.getItem('usage_limit_request_id') || '';

  const handleRetry = () => {
    // Clear stored usage limit data
    localStorage.removeItem('usage_limit_code');
    localStorage.removeItem('usage_limit_message');
    localStorage.removeItem('usage_limit_reset_at');
    localStorage.removeItem('usage_limit_request_id');

    // Navigate back or to home
    const from = (location.state as { from?: string })?.from;
    if (from) {
      navigate(from);
    } else {
      navigate('/', { replace: true });
    }
  };

  return (
    <div className="min-h-screen bg-gray-900 py-12 px-4">
      <UsageLimitError
        code={errorCode}
        message={errorMessage}
        resetAt={resetAt}
        requestId={requestId}
        onRetry={handleRetry}
      />
    </div>
  );
};

export default UsageLimitPage;

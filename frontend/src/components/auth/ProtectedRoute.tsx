import React from 'react';
import { Navigate, useLocation, Link } from 'react-router-dom';
import { ShieldOff } from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { LoadingState } from '@/components/common';

interface ProtectedRouteProps {
  children: React.ReactNode;
  requiredRoles?: string[];
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  requiredRoles,
}) => {
  const { user, isLoading, isAuthenticated } = useAuth();
  const location = useLocation();

  if (isLoading) {
    return <LoadingState message="Verifying authentication..." />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  if (requiredRoles && user && !requiredRoles.includes(user.role)) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="mx-auto mb-6 flex h-20 w-20 items-center justify-center rounded-full bg-warning-400/10">
          <ShieldOff className="h-10 w-10 text-warning-400" />
        </div>
        <h2 className="text-2xl font-bold text-white">Access Denied</h2>
        <p className="mt-2 text-gray-400">
          You don't have permission to access this page.
        </p>
        <p className="mt-1 text-sm text-gray-500">
          Required role: {requiredRoles.join(' or ')} — Your role: {user.role.replace('_', ' ')}
        </p>
        <div className="mt-6 flex gap-3">
          <button
            onClick={() => window.history.back()}
            className="rounded-md bg-gray-700 px-4 py-2 text-sm font-medium text-gray-100 hover:bg-gray-600 transition-colors"
          >
            Go Back
          </button>
          <Link
            to="/dashboard"
            className="rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700 transition-colors"
          >
            Go to Dashboard
          </Link>
        </div>
      </div>
    );
  }

  return <>{children}</>;
};

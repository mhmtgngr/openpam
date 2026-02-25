import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { QueryClientProvider, QueryClient } from '@tanstack/react-query';
import { AuthProvider, useAuth } from '@/contexts/AuthContext';
import { ThemeProvider } from '@/contexts/ThemeContext';
import { AppLayout } from '@/components/layout/AppLayout';
import { ProtectedRoute } from '@/components/auth/ProtectedRoute';
import { LoginPage } from '@/pages/LoginPage';
import { DashboardPage } from '@/pages/DashboardPage';
import { UserListPage } from '@/pages/users/UserListPage';
import { UserFormPage } from '@/pages/users/UserFormPage';
import { RoleListPage } from '@/pages/roles/RoleListPage';
import { RoleFormPage } from '@/pages/roles/RoleFormPage';
import { TargetListPage } from '@/pages/targets/TargetListPage';
import { TargetFormPage } from '@/pages/targets/TargetFormPage';
import { CredentialListPage } from '@/pages/credentials/CredentialListPage';
import { CredentialFormPage } from '@/pages/credentials/CredentialFormPage';
import { MyRequestsPage } from '@/pages/requests/MyRequestsPage';
import { CreateRequestPage } from '@/pages/requests/CreateRequestPage';
import { RequestDetailPage } from '@/pages/requests/RequestDetailPage';
import { ApprovalsPage } from '@/pages/approvals/ApprovalsPage';
import { SessionsPage } from '@/pages/sessions/SessionsPage';
import { SessionDetailPage } from '@/pages/sessions/SessionDetailPage';
import { AuditPage } from '@/pages/audit/AuditPage';
import { MFASetup } from '@/components/auth/MFASetup';
import { TenantListPage } from '@/pages/tenants/TenantListPage';
import { TenantFormPage } from '@/pages/tenants/TenantFormPage';
import { PasswordPolicyListPage } from '@/pages/policies/PasswordPolicyListPage';
import { PasswordPolicyFormPage } from '@/pages/policies/PasswordPolicyFormPage';
import { SessionPolicyListPage } from '@/pages/policies/SessionPolicyListPage';
import { SessionPolicyFormPage } from '@/pages/policies/SessionPolicyFormPage';
import { AccessPolicyListPage } from '@/pages/policies/AccessPolicyListPage';
import { AccessPolicyFormPage } from '@/pages/policies/AccessPolicyFormPage';
import { PolicyTestPage } from '@/pages/policies/PolicyTestPage';
import {
  AnalyticsPage,
  AnomalyListPage,
  AnomalyDetailPage,
  ReportsPage,
  ReportDetailPage,
  ReportGeneratorPage,
  ExceptionManagementPage,
} from '@/pages/analytics';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
});

const AppRoutes: React.FC = () => {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-gray-900">
        <div className="text-center">
          <div className="mx-auto h-12 w-12 animate-spin rounded-full border-4 border-primary-500 border-t-transparent" />
          <p className="mt-4 text-gray-400">Loading...</p>
        </div>
      </div>
    );
  }

  return (
    <Routes>
      {/* Public routes */}
      <Route path="/login" element={<LoginPage />} />

      {/* Protected routes */}
      <Route element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
        <Route path="/dashboard" element={<DashboardPage />} />

        {/* User management */}
        <Route
          path="/users"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <UserListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/users/new"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <UserFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/users/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <UserFormPage />
            </ProtectedRoute>
          }
        />

        {/* Role management */}
        <Route
          path="/roles"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <RoleListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/roles/new"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <RoleFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/roles/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <RoleFormPage />
            </ProtectedRoute>
          }
        />

        {/* Targets */}
        <Route path="/targets" element={<TargetListPage />} />
        <Route path="/targets/new" element={<TargetFormPage />} />
        <Route path="/targets/:id" element={<TargetFormPage />} />

        {/* Credentials */}
        <Route path="/credentials" element={<CredentialListPage />} />
        <Route path="/credentials/new" element={<CredentialFormPage />} />
        <Route path="/credentials/:id" element={<CredentialFormPage />} />

        {/* Requests */}
        <Route path="/requests/my" element={<MyRequestsPage />} />
        <Route path="/requests/new" element={<CreateRequestPage />} />
        <Route path="/requests/:id" element={<RequestDetailPage />} />

        {/* Approvals */}
        <Route
          path="/approvals"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'operator']}>
              <ApprovalsPage />
            </ProtectedRoute>
          }
        />

        {/* Sessions */}
        <Route path="/sessions" element={<SessionsPage />} />
        <Route path="/sessions/:id" element={<SessionDetailPage />} />

        {/* Audit */}
        <Route
          path="/audit"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <AuditPage />
            </ProtectedRoute>
          }
        />

        {/* Tenant management (super_admin only) */}
        <Route
          path="/tenants"
          element={
            <ProtectedRoute requiredRoles={['super_admin']}>
              <TenantListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/tenants/new"
          element={
            <ProtectedRoute requiredRoles={['super_admin']}>
              <TenantFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/tenants/:id"
          element={
            <ProtectedRoute requiredRoles={['super_admin']}>
              <TenantFormPage />
            </ProtectedRoute>
          }
        />

        {/* Password policies */}
        <Route
          path="/policies/password"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <PasswordPolicyListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/password/new"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <PasswordPolicyFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/password/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <PasswordPolicyFormPage />
            </ProtectedRoute>
          }
        />

        {/* Session policies */}
        <Route
          path="/policies/session"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <SessionPolicyListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/session/new"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <SessionPolicyFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/session/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <SessionPolicyFormPage />
            </ProtectedRoute>
          }
        />

        {/* MFA Setup */}
        <Route path="/mfa/setup" element={<MFASetup onComplete={() => window.location.reload()} />} />

        {/* Access Policies */}
        <Route
          path="/policies/access"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <AccessPolicyListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/access/new"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <AccessPolicyFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/access/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <AccessPolicyFormPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/policies/access/test"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin']}>
              <PolicyTestPage />
            </ProtectedRoute>
          }
        />

        {/* Analytics */}
        <Route
          path="/analytics"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <AnalyticsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics/anomalies"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <AnomalyListPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics/anomalies/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <AnomalyDetailPage />
            </ProtectedRoute>
          }
        />

        {/* Reports - support both /reports and /analytics/reports routes */}
        <Route
          path="/reports"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics/reports"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportsPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/reports/generate"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportGeneratorPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics/reports/generate"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportGeneratorPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/reports/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportDetailPage />
            </ProtectedRoute>
          }
        />
        <Route
          path="/analytics/reports/:id"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ReportDetailPage />
            </ProtectedRoute>
          }
        />

        {/* Exceptions */}
        <Route
          path="/compliance/exceptions"
          element={
            <ProtectedRoute requiredRoles={['admin', 'super_admin', 'auditor']}>
              <ExceptionManagementPage />
            </ProtectedRoute>
          }
        />
      </Route>

      {/* Default redirect */}
      <Route
        path="/"
        element={
          <Navigate to={isAuthenticated ? '/dashboard' : '/login'} replace />
        }
      />

      {/* 404 */}
      <Route
        path="*"
        element={
          <div className="flex min-h-screen items-center justify-center bg-gray-900">
            <div className="text-center">
              <h1 className="text-4xl font-bold text-white">404</h1>
              <p className="mt-2 text-gray-400">Page not found</p>
            </div>
          </div>
        }
      />
    </Routes>
  );
};

const App: React.FC = () => {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <AuthProvider>
          <AppRoutes />
        </AuthProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
};

export default App;

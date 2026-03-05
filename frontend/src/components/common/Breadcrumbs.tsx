import React from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ChevronRight, Home } from 'lucide-react';

interface BreadcrumbItem {
  label: string;
  path?: string;
}

const routeLabels: Record<string, string> = {
  dashboard: 'Dashboard',
  users: 'Users',
  roles: 'Roles',
  targets: 'Targets',
  credentials: 'Credentials',
  requests: 'Requests',
  approvals: 'Approvals',
  sessions: 'Sessions',
  audit: 'Audit Logs',
  analytics: 'Analytics',
  reports: 'Reports',
  policies: 'Policies',
  tenants: 'Tenants',
  compliance: 'Compliance',
  settings: 'Settings',
  new: 'New',
  my: 'My Requests',
  password: 'Password Policies',
  session: 'Session Policies',
  access: 'Access Policies',
  test: 'Policy Test',
  anomalies: 'Anomalies',
  exceptions: 'Exceptions',
  generate: 'Generate Report',
};

export const Breadcrumbs: React.FC = () => {
  const location = useLocation();
  const pathSegments = location.pathname.split('/').filter(Boolean);

  if (pathSegments.length <= 1) return null;

  const breadcrumbs: BreadcrumbItem[] = [];
  let currentPath = '';

  for (let i = 0; i < pathSegments.length; i++) {
    const segment = pathSegments[i];
    currentPath += `/${segment}`;

    // Skip UUID-like segments, show them as "Details"
    const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(segment);
    const isNumeric = /^\d+$/.test(segment);

    const label = isUuid || isNumeric
      ? 'Details'
      : routeLabels[segment] || segment.charAt(0).toUpperCase() + segment.slice(1).replace(/-/g, ' ');

    const isLast = i === pathSegments.length - 1;

    breadcrumbs.push({
      label,
      path: isLast ? undefined : currentPath,
    });
  }

  return (
    <nav aria-label="Breadcrumb" className="mb-4">
      <ol className="flex items-center gap-1.5 text-sm">
        <li>
          <Link
            to="/dashboard"
            className="text-gray-500 hover:text-gray-300 transition-colors"
            aria-label="Home"
          >
            <Home className="h-4 w-4" />
          </Link>
        </li>
        {breadcrumbs.map((crumb, index) => (
          <li key={index} className="flex items-center gap-1.5">
            <ChevronRight className="h-3.5 w-3.5 text-gray-600" />
            {crumb.path ? (
              <Link
                to={crumb.path}
                className="text-gray-400 hover:text-gray-200 transition-colors"
              >
                {crumb.label}
              </Link>
            ) : (
              <span className="text-gray-200 font-medium">{crumb.label}</span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
};

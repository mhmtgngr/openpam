import type { User, UserRole } from '@/types';

export const hasPermission = (
  user: User | null,
  resource: string,
  action: string
): boolean => {
  if (!user) return false;

  // Super admins have all permissions
  if (user.role === 'super_admin') return true;

  // Define role permissions
  const rolePermissions: Record<string, string[]> = {
    super_admin: ['*'],
    admin: [
      'users.create',
      'users.read',
      'users.update',
      'users.delete',
      'roles.create',
      'roles.read',
      'roles.update',
      'targets.create',
      'targets.read',
      'targets.update',
      'targets.delete',
      'credentials.create',
      'credentials.read',
      'credentials.update',
      'credentials.checkout',
      'requests.create',
      'requests.read',
      'requests.approve',
      'sessions.read',
      'sessions.terminate',
      'audit.read',
    ],
    operator: [
      'targets.read',
      'credentials.read',
      'credentials.checkout',
      'requests.create',
      'requests.read',
      'requests.approve',
      'sessions.read',
      'sessions.terminate',
    ],
    auditor: [
      'users.read',
      'targets.read',
      'credentials.read',
      'requests.read',
      'sessions.read',
      'audit.read',
      'compliance.read',
    ],
    requester: [
      'targets.read',
      'credentials.read',
      'requests.create',
      'requests.read',
      'sessions.read',
    ],
    user: [
      'targets.read',
      'requests.create',
      'requests.read',
    ],
  };

  const permissions = rolePermissions[user.role] || [];

  // Check for wildcard permission
  if (permissions.includes('*')) return true;

  // Check for exact permission match
  const permission = `${resource}.${action}`;
  if (permissions.includes(permission)) return true;

  // Check for resource wildcard
  if (permissions.includes(`${resource}.*`)) return true;

  return false;
};

export const canApproveRequests = (user: User | null): boolean => {
  return hasPermission(user, 'requests', 'approve');
};

export const canCheckoutCredentials = (user: User | null): boolean => {
  return hasPermission(user, 'credentials', 'checkout');
};

export const canTerminateSessions = (user: User | null): boolean => {
  return hasPermission(user, 'sessions', 'terminate');
};

export const canManageUsers = (user: User | null): boolean => {
  return hasPermission(user, 'users', 'create');
};

export const canManageTargets = (user: User | null): boolean => {
  return hasPermission(user, 'targets', 'create');
};

export const canViewAudit = (user: User | null): boolean => {
  return hasPermission(user, 'audit', 'read');
};

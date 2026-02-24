import React from 'react';
import { NavLink } from 'react-router-dom';
import {
  LayoutDashboard,
  Users,
  Shield,
  Server,
  Key,
  FileKey,
  Clock,
  ClipboardList,
  Activity,
  BarChart3,
  Settings,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import clsx from 'clsx';

interface NavItem {
  name: string;
  path: string;
  icon: React.ElementType;
  roles?: string[];
  badge?: number;
}

const navItems: NavItem[] = [
  { name: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
  { name: 'Users', path: '/users', icon: Users, roles: ['admin', 'super_admin'] },
  { name: 'Roles', path: '/roles', icon: Shield, roles: ['admin', 'super_admin'] },
  { name: 'Targets', path: '/targets', icon: Server },
  { name: 'Credentials', path: '/credentials', icon: Key },
  { name: 'My Requests', path: '/requests/my', icon: FileKey },
  { name: 'Approvals', path: '/approvals', icon: Clock, roles: ['admin', 'super_admin', 'operator'] },
  { name: 'Sessions', path: '/sessions', icon: Activity },
  { name: 'Analytics', path: '/analytics', icon: BarChart3, roles: ['admin', 'super_admin', 'auditor'] },
  { name: 'Audit Logs', path: '/audit', icon: ClipboardList, roles: ['admin', 'super_admin', 'auditor'] },
  { name: 'Settings', path: '/settings', icon: Settings, roles: ['admin', 'super_admin'] },
];

interface SidebarProps {
  collapsed: boolean;
  onToggle: () => void;
}

export const Sidebar: React.FC<SidebarProps> = ({ collapsed, onToggle }) => {
  const { user } = useAuth();

  const canAccess = (roles?: string[]) => {
    if (!roles) return true;
    return user?.role && roles.includes(user.role);
  };

  const getBadge = (path: string) => {
    if (path === '/approvals') {
      // Return pending approvals count from notification context
      return 0; // Placeholder
    }
    return undefined;
  };

  return (
    <aside
      className={clsx(
        'flex flex-col bg-gray-800 border-r border-gray-700 transition-all duration-300',
        collapsed ? 'w-16' : 'w-64'
      )}
    >
      {/* Logo */}
      <div className="flex h-16 items-center justify-between border-b border-gray-700 px-4">
        {!collapsed && (
          <div className="flex items-center gap-2">
            <div className="flex h-8 w-8 items-center justify-center rounded bg-primary-600">
              <Shield className="h-5 w-5 text-white" />
            </div>
            <span className="text-lg font-semibold text-white">OpenPAM</span>
          </div>
        )}
        <button
          onClick={onToggle}
          className="rounded p-1 text-gray-400 hover:bg-gray-700 hover:text-white"
        >
          {collapsed ? (
            <ChevronRight className="h-5 w-5" />
          ) : (
            <ChevronLeft className="h-5 w-5" />
          )}
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 overflow-y-auto p-2 scrollbar-thin">
        {navItems.map((item) => {
          if (!canAccess(item.roles)) return null;

          const Icon = item.icon;
          const badge = getBadge(item.path);

          return (
            <NavLink
              key={item.path}
              to={item.path}
              className={({ isActive: isNavActive }) =>
                clsx(
                  'flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                  isNavActive
                    ? 'bg-primary-600/20 text-primary-400'
                    : 'text-gray-300 hover:bg-gray-700 hover:text-white'
                )
              }
            >
              <Icon className="h-5 w-5 shrink-0" />
              {!collapsed && <span>{item.name}</span>}
              {!collapsed && badge !== undefined && badge > 0 && (
                <span className="ml-auto rounded-full bg-danger-500 px-2 py-0.5 text-xs text-white">
                  {badge > 9 ? '9+' : badge}
                </span>
              )}
            </NavLink>
          );
        })}
      </nav>

      {/* User info */}
      {!collapsed && user && (
        <div className="border-t border-gray-700 p-4">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-primary-600 text-white">
              {user.first_name[0]}{user.last_name[0]}
            </div>
            <div className="flex-1 min-w-0">
              <p className="truncate text-sm font-medium text-white">
                {user.first_name} {user.last_name}
              </p>
              <p className="truncate text-xs text-gray-400">{user.email}</p>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
};

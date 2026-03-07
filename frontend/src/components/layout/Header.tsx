import React, { useState, useRef, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Bell,
  Search,
  User,
  LogOut,
  Settings,
  ChevronDown,
  RefreshCw,
  Command,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifications } from '@/contexts/NotificationContext';
import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';

export const Header: React.FC = () => {
  const { user, logout } = useAuth();
  const { notifications, unreadCount, markAsRead, markAllAsRead, dismissNotification } =
    useNotifications();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [isRefreshing, setIsRefreshing] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const notifRef = useRef<HTMLDivElement>(null);
  const userMenuButtonRef = useRef<HTMLButtonElement>(null);
  const notifButtonRef = useRef<HTMLButtonElement>(null);
  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        userMenuRef.current &&
        !userMenuRef.current.contains(event.target as Node) &&
        !userMenuButtonRef.current?.contains(event.target as Node)
      ) {
        setShowUserMenu(false);
      }
      if (
        notifRef.current &&
        !notifRef.current.contains(event.target as Node) &&
        !notifButtonRef.current?.contains(event.target as Node)
      ) {
        setShowNotifications(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Keyboard shortcut: Ctrl+K / Cmd+K to focus search
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    await queryClient.invalidateQueries();
    // Brief delay so user sees the animation
    setTimeout(() => setIsRefreshing(false), 600);
  }, [queryClient]);

  const handleSearchSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.trim()) {
      navigate(`/audit?search=${encodeURIComponent(searchQuery.trim())}`);
      setSearchQuery('');
      searchInputRef.current?.blur();
    }
  };

  const handleLogout = async () => {
    await logout();
  };

  const handleNotificationClick = (id: string, read: boolean) => {
    if (!read) {
      markAsRead(id);
    }
    // Navigate to related resource if applicable
    dismissNotification(id);
    setShowNotifications(false);
  };

  const getUserInitials = () => {
    if (!user) return '';
    return `${user.first_name[0]}${user.last_name[0]}`.toUpperCase();
  };

  return (
    <header className="flex h-16 items-center justify-between border-b border-gray-700 bg-gray-800 px-6">
      {/* Search */}
      <div className="flex-1">
        <form onSubmit={handleSearchSubmit} className="relative w-96">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            ref={searchInputRef}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search targets, credentials, users..."
            className="input pl-10 pr-20"
          />
          <kbd className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 hidden items-center gap-0.5 rounded border border-gray-600 bg-gray-700 px-1.5 py-0.5 text-[10px] font-medium text-gray-400 sm:inline-flex">
            <Command className="h-2.5 w-2.5" />K
          </kbd>
        </form>
      </div>

      {/* Right side */}
      <div className="flex items-center gap-4">
        {/* Quick refresh */}
        <button
          onClick={handleRefresh}
          disabled={isRefreshing}
          className="rounded p-2 text-gray-400 hover:bg-gray-700 hover:text-white disabled:opacity-50 transition-colors"
          title="Refresh all data"
        >
          <RefreshCw className={clsx('h-5 w-5', isRefreshing && 'animate-spin')} />
        </button>

        {/* Notifications */}
        <div className="relative">
          <button
            ref={notifButtonRef}
            onClick={() => setShowNotifications(!showNotifications)}
            className="relative rounded p-2 text-gray-400 hover:bg-gray-700 hover:text-white"
            aria-label={`Notifications${unreadCount > 0 ? ` (${unreadCount} unread)` : ''}`}
          >
            <Bell className="h-5 w-5" />
            {unreadCount > 0 && (
              <span className="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-danger-500 text-xs text-white">
                {unreadCount > 9 ? '9+' : unreadCount}
              </span>
            )}
          </button>

          {showNotifications && (
            <div
              ref={notifRef}
              className="dropdown-menu right-0 w-80 p-2"
              onClick={(e) => e.stopPropagation()}
            >
              <div className="flex items-center justify-between border-b border-gray-700 px-3 py-2">
                <h3 className="text-sm font-medium text-white">Notifications</h3>
                {unreadCount > 0 && (
                  <button
                    onClick={markAllAsRead}
                    className="text-xs text-primary-400 hover:text-primary-300"
                  >
                    Mark all read
                  </button>
                )}
              </div>
              <div className="max-h-80 overflow-y-auto scrollbar-thin">
                {notifications.length === 0 ? (
                  <div className="px-3 py-8 text-center">
                    <Bell className="mx-auto h-8 w-8 text-gray-600" />
                    <p className="mt-2 text-sm text-gray-400">No notifications yet</p>
                    <p className="mt-1 text-xs text-gray-500">
                      You'll be notified about approvals, sessions, and security events
                    </p>
                  </div>
                ) : (
                  notifications.map((notif) => (
                    <button
                      key={notif.id}
                      onClick={() => handleNotificationClick(notif.id, notif.read)}
                      className={clsx(
                        'w-full rounded px-3 py-2 text-left transition-colors hover:bg-gray-700',
                        !notif.read && 'bg-gray-700/50'
                      )}
                    >
                      <div className="flex items-start gap-2">
                        {!notif.read && (
                          <span className="mt-1.5 h-2 w-2 flex-shrink-0 rounded-full bg-primary-400" />
                        )}
                        <div className={clsx(!notif.read ? '' : 'pl-4')}>
                          <p className="text-sm font-medium text-white">{notif.title}</p>
                          <p className="text-xs text-gray-400">{notif.message}</p>
                        </div>
                      </div>
                    </button>
                  ))
                )}
              </div>
            </div>
          )}
        </div>

        {/* User menu */}
        <div className="relative">
          <button
            ref={userMenuButtonRef}
            onClick={() => setShowUserMenu(!showUserMenu)}
            className="flex items-center gap-2 rounded p-1 text-gray-300 hover:bg-gray-700"
            aria-label="User menu"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary-600 text-sm font-medium text-white">
              {getUserInitials()}
            </div>
            <span className="hidden text-sm md:block">{user?.email}</span>
            <ChevronDown className="h-4 w-4" />
          </button>

          {showUserMenu && (
            <div ref={userMenuRef} className="dropdown-menu right-0 w-48" onClick={(e) => e.stopPropagation()}>
              <div className="border-b border-gray-700 px-4 py-3">
                <p className="text-sm font-medium text-white">
                  {user?.first_name} {user?.last_name}
                </p>
                <p className="text-xs text-gray-400">{user?.email}</p>
                <p className="mt-1 text-xs text-gray-500 capitalize">{user?.role?.replace('_', ' ')}</p>
              </div>
              <button
                onClick={() => {
                  setShowUserMenu(false);
                  navigate('/settings');
                }}
                className="dropdown-item w-full flex items-center gap-2"
              >
                <Settings className="h-4 w-4" />
                Settings
              </button>
              <button
                onClick={() => {
                  setShowUserMenu(false);
                  navigate('/profile');
                }}
                className="dropdown-item w-full flex items-center gap-2"
              >
                <User className="h-4 w-4" />
                Profile
              </button>
              <div className="border-t border-gray-700">
                <button
                  onClick={handleLogout}
                  className="dropdown-item w-full flex items-center gap-2 text-danger-400 hover:text-danger-300"
                >
                  <LogOut className="h-4 w-4" />
                  Logout
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
};

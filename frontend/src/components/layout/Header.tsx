import React, { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Bell,
  Search,
  User,
  LogOut,
  Settings,
  ChevronDown,
  RefreshCw,
} from 'lucide-react';
import { useAuth } from '@/contexts/AuthContext';
import { useNotifications } from '@/contexts/NotificationContext';
import clsx from 'clsx';

export const Header: React.FC = () => {
  const { user, logout } = useAuth();
  const { notifications, unreadCount, markAsRead, markAllAsRead, dismissNotification } =
    useNotifications();
  const navigate = useNavigate();
  const [showUserMenu, setShowUserMenu] = useState(false);
  const [showNotifications, setShowNotifications] = useState(false);
  const userMenuRef = useRef<HTMLDivElement>(null);
  const notifRef = useRef<HTMLDivElement>(null);
  const userMenuButtonRef = useRef<HTMLButtonElement>(null);
  const notifButtonRef = useRef<HTMLButtonElement>(null);

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
        <div className="relative w-96">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search targets, credentials, users..."
            className="input pl-10"
          />
        </div>
      </div>

      {/* Right side */}
      <div className="flex items-center gap-4">
        {/* Quick refresh */}
        <button
          className="rounded p-2 text-gray-400 hover:bg-gray-700 hover:text-white"
          title="Refresh data"
        >
          <RefreshCw className="h-5 w-5" />
        </button>

        {/* Notifications */}
        <div className="relative">
          <button
            ref={notifButtonRef}
            onClick={() => setShowNotifications(!showNotifications)}
            className="relative rounded p-2 text-gray-400 hover:bg-gray-700 hover:text-white"
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
                  <p className="px-3 py-4 text-center text-sm text-gray-400">No notifications</p>
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
                      <p className="text-sm font-medium text-white">{notif.title}</p>
                      <p className="text-xs text-gray-400">{notif.message}</p>
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

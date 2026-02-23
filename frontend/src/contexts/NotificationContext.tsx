import React, { createContext, useContext, useState, useCallback, ReactNode, useEffect } from 'react';
import { api } from '@/api/client';
import type { Notification } from '@/types';

interface NotificationContextType {
  notifications: Notification[];
  unreadCount: number;
  isLoading: boolean;
  fetchNotifications: () => Promise<void>;
  markAsRead: (id: string) => Promise<void>;
  markAllAsRead: () => Promise<void>;
  dismissNotification: (id: string) => void;
}

const NotificationContext = createContext<NotificationContextType | undefined>(undefined);

interface NotificationProviderProps {
  children: ReactNode;
  enabled?: boolean;
}

export const NotificationProvider: React.FC<NotificationProviderProps> = ({
  children,
  enabled = true,
}) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [unreadCount, setUnreadCount] = useState(0);

  const fetchNotifications = useCallback(async () => {
    if (!enabled) return;

    setIsLoading(true);
    try {
      const response = await api.get<{ data: Notification[] }>('/notifications');
      setNotifications(response.data.data);
      setUnreadCount(response.data.data.filter((n) => !n.read).length);
    } catch {
      // Handle error silently
    } finally {
      setIsLoading(false);
    }
  }, [enabled]);

  const markAsRead = async (id: string) => {
    try {
      await api.patch(`/notifications/${id}`, { read: true });
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read: true } : n))
      );
      setUnreadCount((prev) => Math.max(0, prev - 1));
    } catch {
      // Handle error silently
    }
  };

  const markAllAsRead = async () => {
    try {
      await api.post('/notifications/mark-all-read', {});
      setNotifications((prev) => prev.map((n) => ({ ...n, read: true })));
      setUnreadCount(0);
    } catch {
      // Handle error silently
    }
  };

  const dismissNotification = useCallback((id: string) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id));
    setUnreadCount((prev) => {
      const notification = notifications.find((n) => n.id === id);
      return notification && !notification.read ? prev - 1 : prev;
    });
  }, [notifications]);

  useEffect(() => {
    if (enabled) {
      fetchNotifications();
      // Poll for new notifications every 30 seconds
      const interval = setInterval(fetchNotifications, 30000);
      return () => clearInterval(interval);
    }
  }, [enabled, fetchNotifications]);

  const value: NotificationContextType = {
    notifications,
    unreadCount,
    isLoading,
    fetchNotifications,
    markAsRead,
    markAllAsRead,
    dismissNotification,
  };

  return (
    <NotificationContext.Provider value={value}>
      {children}
    </NotificationContext.Provider>
  );
};

export const useNotifications = (): NotificationContextType => {
  const context = useContext(NotificationContext);
  if (context === undefined) {
    throw new Error('useNotifications must be used within a NotificationProvider');
  }
  return context;
};

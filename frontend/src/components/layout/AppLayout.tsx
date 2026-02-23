import React, { useState } from 'react';
import { Outlet } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { NotificationProvider } from '@/contexts/NotificationContext';
import clsx from 'clsx';

export const AppLayout: React.FC = () => {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  return (
    <NotificationProvider>
      <div className="flex h-screen overflow-hidden">
        <Sidebar
          collapsed={sidebarCollapsed}
          onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
        />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Header />
          <main
            className={clsx(
              'flex-1 overflow-y-auto bg-gray-900 p-6 scrollbar-thin',
              sidebarCollapsed ? 'ml-0' : ''
            )}
          >
            <Outlet />
          </main>
        </div>
      </div>
    </NotificationProvider>
  );
};

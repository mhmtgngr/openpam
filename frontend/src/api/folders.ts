import { api } from './client';
import type { Folder } from '@/types';

export const foldersApi = {
  // List folders by type
  list: (type: 'targets' | 'credentials' | 'policies') =>
    api.get<Folder[]>('/folders', { type }),

  // Get folder by ID
  get: (id: string) => api.get<Folder>(`/folders/${id}`),

  // Create folder
  create: (data: { name: string; type: string; parent_id?: string; description?: string }) =>
    api.post<Folder>('/folders', data),

  // Update folder
  update: (id: string, data: { name?: string; description?: string }) =>
    api.patch<Folder>(`/folders/${id}`, data),

  // Delete folder
  delete: (id: string) => api.delete<void>(`/folders/${id}`),

  // Move items to folder
  moveItems: (folderId: string, itemType: string, itemIds: string[]) =>
    api.post<void>(`/folders/${folderId}/items`, { item_type: itemType, item_ids: itemIds }),
};

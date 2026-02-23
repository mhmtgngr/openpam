import { format, formatDistanceToNow, parseISO } from 'date-fns';

export const formatDate = (date: string | Date): string => {
  const d = typeof date === 'string' ? parseISO(date) : date;
  return format(d, 'MMM dd, yyyy');
};

export const formatDateTime = (date: string | Date): string => {
  const d = typeof date === 'string' ? parseISO(date) : date;
  return format(d, 'MMM dd, yyyy HH:mm');
};

export const formatRelativeTime = (date: string | Date): string => {
  const d = typeof date === 'string' ? parseISO(date) : date;
  return formatDistanceToNow(d, { addSuffix: true });
};

export const formatDuration = (seconds: number): string => {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  }
  return `${secs}s`;
};

export const isExpired = (date: string | Date): boolean => {
  const d = typeof date === 'string' ? parseISO(date) : date;
  return d < new Date();
};

export const isExpiringSoon = (date: string | Date, daysThreshold = 7): boolean => {
  const d = typeof date === 'string' ? parseISO(date) : date;
  const threshold = new Date();
  threshold.setDate(threshold.getDate() + daysThreshold);
  return d < threshold && d > new Date();
};

import { type ClassValue, clsx } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatBytes(bytes: number, decimals = 2) {
  if (bytes === 0) return '0 Bytes';
  
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
}

export function formatDate(date: string | number) {
  return new Date(date).toLocaleString();
}

export function getStatusColor(status: string) {
  const statusLower = status.toLowerCase();
  if (statusLower.includes('running') || statusLower.includes('up')) return 'text-green-500';
  if (statusLower.includes('exited') || statusLower.includes('stopped')) return 'text-gray-500';
  if (statusLower.includes('restarting')) return 'text-yellow-500';
  if (statusLower.includes('paused')) return 'text-blue-500';
  if (statusLower.includes('dead') || statusLower.includes('error')) return 'text-red-500';
  return 'text-gray-400';
}

export function getStatusBadgeClass(status: string) {
  const statusLower = status.toLowerCase();
  if (statusLower.includes('running') || statusLower.includes('up')) {
    return 'bg-green-500/10 text-green-500 border-green-500/20';
  }
  if (statusLower.includes('exited') || statusLower.includes('stopped')) {
    return 'bg-gray-500/10 text-gray-400 border-gray-500/20';
  }
  if (statusLower.includes('restarting')) {
    return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20';
  }
  if (statusLower.includes('paused')) {
    return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
  }
  if (statusLower.includes('dead') || statusLower.includes('error')) {
    return 'bg-red-500/10 text-red-500 border-red-500/20';
  }
  return 'bg-gray-500/10 text-gray-400 border-gray-500/20';
}

export function truncateId(id: string, length = 12) {
  return id.substring(0, length);
}

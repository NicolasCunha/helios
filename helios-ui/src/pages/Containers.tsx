import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Play, Square, RotateCw, Trash2, Search, Filter, CheckSquare, XSquare } from 'lucide-react';
import { bulkStartContainers, bulkStopContainers, bulkRemoveContainers } from '../services/api';
import api from '../services/api';
import type { Container } from '../types';
import ConfirmDialog from '../components/ConfirmDialog';
import ErrorDialog from '../components/ErrorDialog';

type FilterStatus = 'all' | 'running' | 'stopped';

interface ConfirmState {
  isOpen: boolean;
  action: 'start' | 'stop' | 'restart' | 'remove' | null;
  container: Container | null;
  containerIds?: string[];
}

interface ErrorState {
  isOpen: boolean;
  title: string;
  error: string;
  details?: string;
}

export default function Containers() {
  const [filterStatus, setFilterStatus] = useState<FilterStatus>('all');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [confirmState, setConfirmState] = useState<ConfirmState>({
    isOpen: false,
    action: null,
    container: null,
    containerIds: undefined,
  });
  const [errorState, setErrorState] = useState<ErrorState>({
    isOpen: false,
    title: '',
    error: '',
    details: '',
  });
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ['containers', filterStatus],
    queryFn: async () => {
      const response = await api.get('/containers', {
        params: { all: filterStatus === 'all' || filterStatus === 'stopped' }
      });
      return response.data;
    },
    refetchInterval: 5000,
  });

  const handleError = (action: string, error: any) => {
    const errorMessage = error.response?.data?.error || error.message || 'Unknown error';
    const errorDetail = error.response?.data?.detail || '';
    
    setErrorState({
      isOpen: true,
      title: `Failed to ${action}`,
      error: errorMessage,
      details: errorDetail,
    });
  };

  const startMutation = useMutation({
    mutationFn: (id: string) => api.post(`/containers/${id}/start`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['containers'] }),
    onError: (error) => handleError('start container', error),
  });

  const stopMutation = useMutation({
    mutationFn: (id: string) => api.post(`/containers/${id}/stop`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['containers'] }),
    onError: (error) => handleError('stop container', error),
  });

  const restartMutation = useMutation({
    mutationFn: (id: string) => api.post(`/containers/${id}/restart`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['containers'] }),
    onError: (error) => handleError('restart container', error),
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/containers/${id}`, { params: { force: true } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['containers'] }),
    onError: (error) => handleError('remove container', error),
  });

  const bulkStartMutation = useMutation({
    mutationFn: bulkStartContainers,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['containers'] });
      setSelectedIds([]);
      if (data.failed > 0) {
        handleError('start containers', {
          message: `${data.failed} of ${data.total} containers failed to start`,
          response: { data: { detail: data.results.filter((r: any) => !r.success).map((r: any) => `${r.container_name || r.container_id}: ${r.error}`).join('\n') } }
        });
      }
    },
    onError: (error) => handleError('start containers', error),
  });

  const bulkStopMutation = useMutation({
    mutationFn: bulkStopContainers,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['containers'] });
      setSelectedIds([]);
      if (data.failed > 0) {
        handleError('stop containers', {
          message: `${data.failed} of ${data.total} containers failed to stop`,
          response: { data: { detail: data.results.filter((r: any) => !r.success).map((r: any) => `${r.container_name || r.container_id}: ${r.error}`).join('\n') } }
        });
      }
    },
    onError: (error) => handleError('stop containers', error),
  });

  const bulkRemoveMutation = useMutation({
    mutationFn: (ids: string[]) => bulkRemoveContainers(ids, true),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['containers'] });
      setSelectedIds([]);
      if (data.failed > 0) {
        handleError('remove containers', {
          message: `${data.failed} of ${data.total} containers failed to be removed`,
          response: { data: { detail: data.results.filter((r: any) => !r.success).map((r: any) => `${r.container_name || r.container_id}: ${r.error}`).join('\n') } }
        });
      }
    },
    onError: (error) => handleError('remove containers', error),
  });

  const handleAction = (action: 'start' | 'stop' | 'restart' | 'remove', container: Container) => {
    setConfirmState({
      isOpen: true,
      action,
      container,
      containerIds: undefined,
    });
  };

  const handleBulkAction = (action: 'start' | 'stop' | 'remove') => {
    if (selectedIds.length === 0) return;
    
    setConfirmState({
      isOpen: true,
      action,
      container: null,
      containerIds: selectedIds,
    });
  };

  const handleConfirm = async () => {
    const { action, container, containerIds } = confirmState;

    // Bulk action
    if (containerIds && containerIds.length > 0) {
      switch (action) {
        case 'start':
          bulkStartMutation.mutate(containerIds);
          break;
        case 'stop':
          bulkStopMutation.mutate(containerIds);
          break;
        case 'remove':
          bulkRemoveMutation.mutate(containerIds);
          break;
      }
      return;
    }

    // Single action
    if (!container) return;

    switch (action) {
      case 'start':
        startMutation.mutate(container.id);
        break;
      case 'stop':
        stopMutation.mutate(container.id);
        break;
      case 'restart':
        restartMutation.mutate(container.id);
        break;
      case 'remove':
        removeMutation.mutate(container.id);
        break;
    }
  };

  const toggleSelectAll = () => {
    if (selectedIds.length === filteredContainers.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filteredContainers.map(c => c.id));
    }
  };

  const toggleSelect = (id: string) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
    );
  };

  const getConfirmMessage = () => {
    const { action, container, containerIds } = confirmState;

    // Bulk action
    if (containerIds && containerIds.length > 0) {
      const count = containerIds.length;
      switch (action) {
        case 'start':
          return `Are you sure you want to start ${count} container${count > 1 ? 's' : ''}?`;
        case 'stop':
          return `Are you sure you want to stop ${count} container${count > 1 ? 's' : ''}?`;
        case 'remove':
          return `Are you sure you want to remove ${count} container${count > 1 ? 's' : ''}? This action cannot be undone.`;
        default:
          return '';
      }
    }

    // Single action
    if (!container) return '';

    switch (action) {
      case 'start':
        return `Are you sure you want to start "${container.name}"?`;
      case 'stop':
        return `Are you sure you want to stop "${container.name}"?`;
      case 'restart':
        return `Are you sure you want to restart "${container.name}"?`;
      case 'remove':
        return `Are you sure you want to remove "${container.name}"? This action cannot be undone.`;
      default:
        return '';
    }
  };

  const containers: Container[] = data?.containers || [];
  
  const filteredContainers = containers.filter((container) => {
    const matchesSearch = 
      container.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      container.image.toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesStatus = 
      filterStatus === 'all' ||
      (filterStatus === 'running' && container.state === 'running') ||
      (filterStatus === 'stopped' && container.state !== 'running');

    return matchesSearch && matchesStatus;
  });

  const getStatusColor = (state: string) => {
    switch (state) {
      case 'running':
        return 'bg-green-500/20 text-green-400 border-green-500/50';
      case 'paused':
        return 'bg-yellow-500/20 text-yellow-400 border-yellow-500/50';
      case 'exited':
      case 'stopped':
        return 'bg-gray-500/20 text-gray-400 border-gray-500/50';
      default:
        return 'bg-blue-500/20 text-blue-400 border-blue-500/50';
    }
  };

  const formatPorts = (ports: Array<string | { PrivatePort?: number; PublicPort?: number; Type?: string; private_port?: number; public_port?: number; type?: string }>) => {
    if (!ports || ports.length === 0) return 'None';
    
    const formatted = ports.map(port => {
      if (typeof port === 'string') return port;
      
      const privatePort = port.PrivatePort || port.private_port;
      const publicPort = port.PublicPort || port.public_port;
      const type = (port.Type || port.type || 'tcp').toLowerCase();
      
      if (publicPort) {
        return `${publicPort}:${privatePort}/${type}`;
      }
      return `${privatePort}/${type}`;
    });
    
    if (formatted.length <= 2) {
      return formatted.join(', ');
    }
    return formatted.slice(0, 2).join(', ') + ` +${formatted.length - 2}`;
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  const formatCPU = (cpuPercent?: number) => {
    if (cpuPercent === undefined) return '-';
    return `${cpuPercent.toFixed(1)}%`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4">
        <p className="text-red-400">Failed to load containers: {error.message}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <h1 className="text-3xl font-bold text-white">Containers</h1>
          {selectedIds.length > 0 && (
            <span className="text-sm text-blue-400 bg-blue-500/20 px-3 py-1 rounded-full">
              {selectedIds.length} selected
            </span>
          )}
        </div>
        <div className="text-sm text-gray-400">
          Total: {filteredContainers.length} / {containers.length}
        </div>
      </div>

      {/* Bulk Actions */}
      {selectedIds.length > 0 && (
        <div className="bg-blue-500/10 border border-blue-500/50 rounded-lg p-4 mb-6">
          <div className="flex items-center justify-between">
            <span className="text-blue-400 font-medium">
              {selectedIds.length} container{selectedIds.length > 1 ? 's' : ''} selected
            </span>
            <div className="flex items-center gap-2">
              <button
                onClick={() => handleBulkAction('start')}
                className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <Play className="w-4 h-4" />
                Start All
              </button>
              <button
                onClick={() => handleBulkAction('stop')}
                className="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <Square className="w-4 h-4" />
                Stop All
              </button>
              <button
                onClick={() => handleBulkAction('remove')}
                className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <Trash2 className="w-4 h-4" />
                Remove All
              </button>
              <button
                onClick={() => setSelectedIds([])}
                className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <XSquare className="w-4 h-4" />
                Clear
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Filters */}
      <div className="bg-gray-800 rounded-lg border border-gray-700 p-4 mb-6">
        <div className="flex flex-col sm:flex-row gap-4">
          {/* Search */}
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Search by name or image..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
            />
          </div>

          {/* Status Filter */}
          <div className="flex items-center gap-2">
            <Filter className="w-5 h-5 text-gray-400" />
            <select
              value={filterStatus}
              onChange={(e) => setFilterStatus(e.target.value as FilterStatus)}
              className="px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-blue-500"
            >
              <option value="all">All Containers</option>
              <option value="running">Running Only</option>
              <option value="stopped">Stopped Only</option>
            </select>
          </div>
        </div>
      </div>

      {/* Containers Table */}
      {filteredContainers.length === 0 ? (
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-8 text-center">
          <p className="text-gray-400">No containers found</p>
        </div>
      ) : (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700/50">
                <tr>
                  <th className="px-6 py-3 text-left">
                    <button
                      onClick={toggleSelectAll}
                      className="text-gray-300 hover:text-white transition-colors"
                      title={selectedIds.length === filteredContainers.length ? 'Deselect All' : 'Select All'}
                    >
                      <CheckSquare className="w-5 h-5" />
                    </button>
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Name
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Image
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    CPU
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Memory
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Ports
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {filteredContainers.map((container) => (
                  <tr key={container.id} className="hover:bg-gray-700/30 transition-colors">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(container.id)}
                        onChange={() => toggleSelect(container.id)}
                        className="w-4 h-4 rounded border-gray-600 bg-gray-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-800"
                      />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <Link 
                        to={`/containers/${container.id}`}
                        className="text-blue-400 hover:text-blue-300 font-medium"
                      >
                        {container.name}
                      </Link>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                      {container.image}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <span className={`inline-flex px-2 py-1 text-xs font-semibold rounded-full border ${getStatusColor(container.state)}`}>
                        {container.state}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {container.state === 'running' ? (
                        <span className={`font-mono ${container.stats && container.stats.cpu_percent > 80 ? 'text-red-400' : 'text-gray-400'}`}>
                          {formatCPU(container.stats?.cpu_percent)}
                        </span>
                      ) : (
                        <span className="text-gray-600">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {container.state === 'running' ? (
                        <div className="flex flex-col">
                          <span className={`font-mono text-xs ${container.stats && container.stats.memory_percent > 80 ? 'text-red-400' : 'text-gray-400'}`}>
                            {container.stats ? `${formatBytes(container.stats.memory_usage)} / ${formatBytes(container.stats.memory_limit)}` : '-'}
                          </span>
                          <span className={`font-mono text-xs ${container.stats && container.stats.memory_percent > 80 ? 'text-red-400' : 'text-gray-500'}`}>
                            {container.stats ? `${container.stats.memory_percent.toFixed(1)}%` : ''}
                          </span>
                        </div>
                      ) : (
                        <span className="text-gray-600">-</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                      {formatPorts(container.ports)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex justify-end gap-2">
                        {container.state === 'running' ? (
                          <>
                            <button
                              onClick={() => handleAction('stop', container)}
                              disabled={stopMutation.isPending}
                              className="p-2 text-yellow-400 hover:bg-yellow-500/20 rounded-lg transition-colors disabled:opacity-50"
                              title="Stop"
                            >
                              <Square className="w-4 h-4" />
                            </button>
                            <button
                              onClick={() => handleAction('restart', container)}
                              disabled={restartMutation.isPending}
                              className="p-2 text-blue-400 hover:bg-blue-500/20 rounded-lg transition-colors disabled:opacity-50"
                              title="Restart"
                            >
                              <RotateCw className="w-4 h-4" />
                            </button>
                          </>
                        ) : (
                          <button
                            onClick={() => handleAction('start', container)}
                            disabled={startMutation.isPending}
                            className="p-2 text-green-400 hover:bg-green-500/20 rounded-lg transition-colors disabled:opacity-50"
                            title="Start"
                          >
                            <Play className="w-4 h-4" />
                          </button>
                        )}
                        <button
                          onClick={() => handleAction('remove', container)}
                          disabled={removeMutation.isPending}
                          className="p-2 text-red-400 hover:bg-red-500/20 rounded-lg transition-colors disabled:opacity-50"
                          title="Remove"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Confirm Dialog */}
      <ConfirmDialog
        isOpen={confirmState.isOpen}
        onClose={() => setConfirmState({ isOpen: false, action: null, container: null, containerIds: undefined })}
        onConfirm={handleConfirm}
        title={`${confirmState.action?.charAt(0).toUpperCase()}${confirmState.action?.slice(1)} Container${confirmState.containerIds && confirmState.containerIds.length > 1 ? 's' : ''}`}
        message={getConfirmMessage()}
        confirmText={confirmState.action === 'remove' ? 'Remove' : 'Confirm'}
        variant={confirmState.action === 'remove' ? 'danger' : 'warning'}
      />

      {/* Error Dialog */}
      <ErrorDialog
        isOpen={errorState.isOpen}
        onClose={() => setErrorState({ isOpen: false, title: '', error: '', details: '' })}
        title={errorState.title}
        error={errorState.error}
        details={errorState.details}
      />
    </div>
  );
}

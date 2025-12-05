import { useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { 
  Play, Square, RotateCw, Trash2, ArrowLeft,
  Activity, HardDrive, Network, Clock, Box
} from 'lucide-react';
import api from '../services/api';
import ConfirmDialog from '../components/ConfirmDialog';
import ErrorDialog from '../components/ErrorDialog';

export default function ContainerDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [confirmState, setConfirmState] = useState<{
    isOpen: boolean;
    action: 'start' | 'stop' | 'restart' | 'remove' | null;
  }>({ isOpen: false, action: null });
  const [errorState, setErrorState] = useState({
    isOpen: false,
    title: '',
    error: '',
    details: '',
  });

  const { data: container, isLoading, error } = useQuery({
    queryKey: ['container', id],
    queryFn: async () => {
      const response = await api.get(`/containers/${id}`);
      return response.data;
    },
    refetchInterval: 5000,
    enabled: !!id,
  });

  const handleError = (action: string, error: any) => {
    setErrorState({
      isOpen: true,
      title: `Failed to ${action}`,
      error: error.response?.data?.error || error.message,
      details: error.response?.data?.detail || '',
    });
  };

  const startMutation = useMutation({
    mutationFn: () => api.post(`/containers/${id}/start`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['container', id] }),
    onError: (error) => handleError('start container', error),
  });

  const stopMutation = useMutation({
    mutationFn: () => api.post(`/containers/${id}/stop`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['container', id] }),
    onError: (error) => handleError('stop container', error),
  });

  const restartMutation = useMutation({
    mutationFn: () => api.post(`/containers/${id}/restart`),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['container', id] }),
    onError: (error) => handleError('restart container', error),
  });

  const removeMutation = useMutation({
    mutationFn: () => api.delete(`/containers/${id}`, { params: { force: true } }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['containers'] });
      navigate('/containers');
    },
    onError: (error) => handleError('remove container', error),
  });

  const handleAction = (action: 'start' | 'stop' | 'restart' | 'remove') => {
    setConfirmState({ isOpen: true, action });
  };

  const handleConfirm = () => {
    switch (confirmState.action) {
      case 'start':
        startMutation.mutate();
        break;
      case 'stop':
        stopMutation.mutate();
        break;
      case 'restart':
        restartMutation.mutate();
        break;
      case 'remove':
        removeMutation.mutate();
        break;
    }
  };

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

  const formatUptime = (created: number) => {
    const now = Math.floor(Date.now() / 1000);
    const diff = now - created;
    const days = Math.floor(diff / 86400);
    const hours = Math.floor((diff % 86400) / 3600);
    const minutes = Math.floor((diff % 3600) / 60);
    
    if (days > 0) return `${days}d ${hours}h`;
    if (hours > 0) return `${hours}h ${minutes}m`;
    return `${minutes}m`;
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatPort = (port: any) => {
    if (typeof port === 'string') return port;
    
    const privatePort = port.PrivatePort || port.private_port;
    const publicPort = port.PublicPort || port.public_port;
    const ip = port.IP || port.ip || '0.0.0.0';
    const type = (port.Type || port.type || 'tcp').toLowerCase();
    
    if (publicPort) {
      return `${ip}:${publicPort} → ${privatePort}/${type}`;
    }
    return `${privatePort}/${type}`;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
    );
  }

  if (error || !container) {
    return (
      <div>
        <Link to="/containers" className="inline-flex items-center gap-2 text-blue-400 hover:text-blue-300 mb-6">
          <ArrowLeft className="w-4 h-4" />
          Back to Containers
        </Link>
        <div className="bg-red-500/10 border border-red-500/50 rounded-lg p-4">
          <p className="text-red-400">Container not found</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-4">
          <Link to="/containers" className="text-gray-400 hover:text-white">
            <ArrowLeft className="w-6 h-6" />
          </Link>
          <div>
            <h1 className="text-3xl font-bold text-white">{container.name}</h1>
            <p className="text-sm text-gray-400 mt-1">{container.id.substring(0, 12)}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {container.state === 'running' ? (
            <>
              <button
                onClick={() => handleAction('stop')}
                className="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <Square className="w-4 h-4" />
                Stop
              </button>
              <button
                onClick={() => handleAction('restart')}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
              >
                <RotateCw className="w-4 h-4" />
                Restart
              </button>
            </>
          ) : (
            <button
              onClick={() => handleAction('start')}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors flex items-center gap-2"
            >
              <Play className="w-4 h-4" />
              Start
            </button>
          )}
          <button
            onClick={() => handleAction('remove')}
            className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors flex items-center gap-2"
          >
            <Trash2 className="w-4 h-4" />
            Remove
          </button>
        </div>
      </div>

      {/* Status and Info Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-6">
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <div className="flex items-center gap-3 mb-2">
            <Activity className="w-5 h-5 text-blue-400" />
            <span className="text-sm text-gray-400">Status</span>
          </div>
          <span className={`inline-flex px-3 py-1 text-sm font-semibold rounded-full border ${getStatusColor(container.state)}`}>
            {container.state}
          </span>
        </div>

        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <div className="flex items-center gap-3 mb-2">
            <Clock className="w-5 h-5 text-green-400" />
            <span className="text-sm text-gray-400">Uptime</span>
          </div>
          <p className="text-xl font-bold text-white">{formatUptime(container.created)}</p>
        </div>

        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <div className="flex items-center gap-3 mb-2">
            <HardDrive className="w-5 h-5 text-purple-400" />
            <span className="text-sm text-gray-400">Memory</span>
          </div>
          <p className="text-xl font-bold text-white">
            {container.stats?.memory_percent?.toFixed(1) || 0}%
          </p>
          <p className="text-xs text-gray-400 mt-1">
            {formatBytes(container.stats?.memory_usage || 0)} / {formatBytes(container.stats?.memory_limit || 0)}
          </p>
        </div>

        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <div className="flex items-center gap-3 mb-2">
            <Activity className="w-5 h-5 text-orange-400" />
            <span className="text-sm text-gray-400">CPU</span>
          </div>
          <p className="text-xl font-bold text-white">
            {container.stats?.cpu_percent?.toFixed(1) || 0}%
          </p>
        </div>
      </div>

      {/* Details Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Overview */}
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
            <Box className="w-5 h-5" />
            Overview
          </h2>
          <div className="space-y-3">
            <div>
              <span className="text-sm text-gray-400">Image</span>
              <p className="text-white font-mono text-sm mt-1">{container.image}</p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Command</span>
              <p className="text-white font-mono text-sm mt-1">{container.command || 'N/A'}</p>
            </div>
            <div>
              <span className="text-sm text-gray-400">Created</span>
              <p className="text-white text-sm mt-1">
                {new Date(container.created * 1000).toLocaleString()}
              </p>
            </div>
          </div>
        </div>

        {/* Network */}
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
            <Network className="w-5 h-5" />
            Network
          </h2>
          {container.ports && container.ports.length > 0 ? (
            <div className="space-y-2">
              {container.ports.map((port: any, idx: number) => (
                <div key={idx} className="flex items-center justify-between py-2 border-b border-gray-700 last:border-0">
                  <span className="text-gray-300 text-sm font-mono">{formatPort(port)}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-gray-400 text-sm">No ports exposed</p>
          )}
        </div>
      </div>

      {/* Environment Variables */}
      {container.env && container.env.length > 0 && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6 mb-6">
          <h2 className="text-xl font-semibold text-white mb-4">Environment Variables</h2>
          <div className="space-y-2 max-h-64 overflow-y-auto">
            {container.env.map((env: string, idx: number) => {
              const [key, ...valueParts] = env.split('=');
              const value = valueParts.join('=');
              return (
                <div key={idx} className="flex items-start gap-4 py-2 border-b border-gray-700 last:border-0">
                  <span className="text-blue-400 font-mono text-sm min-w-0 flex-shrink-0">{key}</span>
                  <span className="text-gray-300 font-mono text-sm break-all">{value}</span>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Mounts */}
      {container.mounts && container.mounts.length > 0 && (
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-6">
          <h2 className="text-xl font-semibold text-white mb-4 flex items-center gap-2">
            <HardDrive className="w-5 h-5" />
            Volumes & Mounts
          </h2>
          <div className="space-y-3">
            {container.mounts.map((mount: any, idx: number) => (
              <div key={idx} className="p-3 bg-gray-700/50 rounded border border-gray-600">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-semibold text-blue-400">{mount.Type || 'volume'}</span>
                  <span className="text-xs text-gray-400">{mount.Mode || 'rw'}</span>
                </div>
                <div className="space-y-1">
                  <div>
                    <span className="text-xs text-gray-400">Source:</span>
                    <p className="text-sm text-white font-mono break-all">{mount.Source || mount.Name}</p>
                  </div>
                  <div>
                    <span className="text-xs text-gray-400">Destination:</span>
                    <p className="text-sm text-white font-mono">{mount.Destination}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Confirm Dialog */}
      <ConfirmDialog
        isOpen={confirmState.isOpen}
        onClose={() => setConfirmState({ isOpen: false, action: null })}
        onConfirm={handleConfirm}
        title={`${confirmState.action?.charAt(0).toUpperCase()}${confirmState.action?.slice(1)} Container`}
        message={
          confirmState.action === 'remove'
            ? `Are you sure you want to remove "${container.name}"? This action cannot be undone.`
            : `Are you sure you want to ${confirmState.action} "${container.name}"?`
        }
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

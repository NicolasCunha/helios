import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Trash2, Plus, Search, CheckSquare, XSquare, HardDrive } from 'lucide-react';
import api from '../services/api';
import type { Volume } from '../types';
import ConfirmDialog from '../components/ConfirmDialog';
import ErrorDialog from '../components/ErrorDialog';
import SuccessDialog from '../components/SuccessDialog';
import Modal from '../components/Modal';

export default function Volumes() {
  const [searchTerm, setSearchTerm] = useState('');
  const [usageFilter, setUsageFilter] = useState<'all' | 'in-use' | 'unused'>('all');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [volumeName, setVolumeName] = useState('');
  const [volumeDriver, setVolumeDriver] = useState('local');
  const [confirmState, setConfirmState] = useState<{
    isOpen: boolean;
    action: 'remove' | 'prune' | 'bulk-remove' | null;
    volume: Volume | null;
    volumeIds?: string[];
  }>({ isOpen: false, action: null, volume: null, volumeIds: [] });
  const [errorState, setErrorState] = useState({
    isOpen: false,
    title: '',
    error: '',
    details: '',
  });
  const [successState, setSuccessState] = useState({
    isOpen: false,
    title: '',
    message: '',
    details: '',
  });
  const queryClient = useQueryClient();

  const { data, isLoading, error } = useQuery({
    queryKey: ['volumes'],
    queryFn: async () => {
      const response = await api.get('/volumes');
      return response.data;
    },
    refetchInterval: 10000,
  });

  const { data: containersData } = useQuery({
    queryKey: ['containers'],
    queryFn: async () => {
      const response = await api.get('/containers');
      return response.data;
    },
    refetchInterval: 10000,
  });

  const removeMutation = useMutation({
    mutationFn: async (name: string) => {
      await api.delete(`/volumes/${name}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes'] });
    },
  });

  const bulkRemoveMutation = useMutation({
    mutationFn: async (volumeNames: string[]) => {
      await api.post('/volumes/bulk/remove', { volume_names: volumeNames });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes'] });
      setSelectedIds([]);
    },
  });

  const pruneMutation = useMutation({
    mutationFn: async () => {
      const response = await api.post('/volumes/prune');
      return response.data;
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['volumes'] });
      const reclaimed = data.space_reclaimed || 0;
      const deletedVolumes = data.volumes_deleted || [];
      const count = deletedVolumes.length;
      
      // Format the list of deleted volumes
      const volumeList = deletedVolumes.length > 0 
        ? deletedVolumes.join('\n') 
        : 'No volumes removed';
      
      setSuccessState({
        isOpen: true,
        title: 'Volumes Pruned Successfully',
        message: `Removed ${count} unused volume(s)`,
        details: `Space reclaimed: ${formatBytes(reclaimed)}\n\nVolumes removed:\n${volumeList}`,
      });
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: { name: string; driver: string }) => {
      await api.post('/volumes', data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['volumes'] });
      setShowCreateModal(false);
      setVolumeName('');
      setVolumeDriver('local');
    },
  });

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const getContainersUsingVolume = (volumeName: string) => {
    if (!containersData?.containers) return [];
    return containersData.containers.filter((container: any) => {
      if (!container.mounts) return false;
      return container.mounts.some((mount: any) => mount.name === volumeName);
    });
  };

  const handleError = (action: string, error: unknown) => {
    const message = error instanceof Error ? error.message : 'Unknown error occurred';
    setErrorState({
      isOpen: true,
      title: `Failed to ${action}`,
      error: message,
      details: '',
    });
  };

  const handleAction = (action: 'remove' | 'prune' | 'bulk-remove', volume: Volume | null, volumeIds?: string[]) => {
    setConfirmState({
      isOpen: true,
      action,
      volume,
      volumeIds,
    });
  };

  const handleConfirm = async () => {
    const { action, volume, volumeIds } = confirmState;
    setConfirmState({ isOpen: false, action: null, volume: null, volumeIds: [] });

    try {
      if (action === 'remove' && volume) {
        await removeMutation.mutateAsync(volume.name);
      } else if (action === 'prune') {
        await pruneMutation.mutateAsync();
      } else if (action === 'bulk-remove' && volumeIds) {
        await bulkRemoveMutation.mutateAsync(volumeIds);
      }
    } catch (error: unknown) {
      handleError(action || 'operation', error);
    }
  };

  const handleCreateVolume = async () => {
    const name = volumeName?.trim();
    if (!name) {
      setErrorState({
        isOpen: true,
        title: 'Validation Error',
        error: 'Volume name is required',
        details: '',
      });
      return;
    }
    try {
      await createMutation.mutateAsync({ name, driver: volumeDriver });
    } catch (error: unknown) {
      handleError('create volume', error);
    }
  };

  const handleSelectAll = () => {
    if (selectedIds.length === filteredVolumes?.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filteredVolumes?.map((v: Volume) => v.name) || []);
    }
  };

  const handleSelectVolume = (name: string) => {
    setSelectedIds(prev =>
      prev.includes(name) ? prev.filter(id => id !== name) : [...prev, name]
    );
  };

  if (isLoading) {
    return <div className="flex items-center justify-center h-64">Loading volumes...</div>;
  }

  if (error) {
    return <div className="text-red-400">Error loading volumes: {(error as Error).message}</div>;
  }

  const volumes: Volume[] = data?.volumes || [];

  const filteredVolumes = volumes.filter((volume) => {
    const matchesSearch = volume.name.toLowerCase().includes(searchTerm.toLowerCase());
    
    const containersUsing = getContainersUsingVolume(volume.name);
    const isInUse = containersUsing.length > 0;
    
    if (usageFilter === 'in-use' && !isInUse) return false;
    if (usageFilter === 'unused' && isInUse) return false;
    
    return matchesSearch;
  });

  const getConfirmMessage = () => {
    const { action, volume, volumeIds } = confirmState;
    if (action === 'remove' && volume) {
      const containers = getContainersUsingVolume(volume.name);
      if (containers.length > 0) {
        return `Volume "${volume.name}" is currently used by ${containers.length} container(s). Removing it may cause issues.`;
      }
      return `Are you sure you want to remove volume "${volume.name}"?`;
    }
    if (action === 'bulk-remove' && volumeIds) {
      return `Are you sure you want to remove ${volumeIds.length} volume(s)?`;
    }
    if (action === 'prune') {
      return 'This will remove all unused volumes. Are you sure?';
    }
    return '';
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold text-white">Volumes</h1>
        <div className="flex gap-3">
          <button
            onClick={() => handleAction('prune', null)}
            className="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded flex items-center gap-2 transition-colors"
          >
            <Trash2 className="w-4 h-4" />
            Prune Unused
          </button>
          <button
            onClick={() => setShowCreateModal(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded flex items-center gap-2 transition-colors"
          >
            <Plus className="w-4 h-4" />
            Create Volume
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-4 items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
          <input
            type="text"
            placeholder="Search volumes..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
          />
        </div>
        <select
          value={usageFilter}
          onChange={(e) => setUsageFilter(e.target.value as 'all' | 'in-use' | 'unused')}
          className="px-4 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:border-blue-500"
        >
          <option value="all">All Volumes</option>
          <option value="in-use">In Use</option>
          <option value="unused">Not Used</option>
        </select>
      </div>

      {/* Bulk Actions Bar */}
      {selectedIds.length > 0 && (
        <div className="bg-blue-900 border border-blue-700 rounded p-3 flex items-center justify-between">
          <span className="text-white">{selectedIds.length} volume(s) selected</span>
          <div className="flex gap-2">
            <button
              onClick={() => handleAction('bulk-remove', null, selectedIds)}
              className="px-3 py-1 bg-red-600 hover:bg-red-700 text-white rounded text-sm transition-colors"
            >
              Remove All
            </button>
            <button
              onClick={() => setSelectedIds([])}
              className="px-3 py-1 bg-gray-600 hover:bg-gray-700 text-white rounded text-sm transition-colors"
            >
              Clear Selection
            </button>
          </div>
        </div>
      )}

      {/* Table */}
      <div className="bg-gray-800 rounded-lg overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-700">
            <tr>
              <th className="px-4 py-3 text-left">
                <button
                  onClick={handleSelectAll}
                  className="text-gray-400 hover:text-white transition-colors"
                >
                  {selectedIds.length === filteredVolumes.length ? (
                    <CheckSquare className="w-5 h-5" />
                  ) : (
                    <XSquare className="w-5 h-5" />
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Name</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Driver</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Mountpoint</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Used By</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Created</th>
              <th className="px-4 py-3 text-right text-gray-300 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            {filteredVolumes.map((volume) => {
              const containers = getContainersUsingVolume(volume.name);
              return (
                <tr key={volume.name} className="hover:bg-gray-750">
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selectedIds.includes(volume.name)}
                      onChange={() => handleSelectVolume(volume.name)}
                      className="w-4 h-4 rounded bg-gray-700 border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-800"
                    />
                  </td>
                  <td className="px-4 py-3 font-mono text-sm text-white">{volume.name}</td>
                  <td className="px-4 py-3 text-gray-300">{volume.driver}</td>
                  <td className="px-4 py-3 text-gray-400 text-xs font-mono truncate max-w-xs">
                    {volume.mountpoint}
                  </td>
                  <td className="px-4 py-3">
                    {containers.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {containers.slice(0, 2).map((container: any) => (
                          <span
                            key={container.id}
                            className="text-xs px-2 py-0.5 bg-blue-900 text-blue-200 rounded"
                          >
                            {container.name}
                          </span>
                        ))}
                        {containers.length > 2 && (
                          <span className="text-xs px-2 py-0.5 bg-gray-700 text-gray-300 rounded">
                            +{containers.length - 2} more
                          </span>
                        )}
                      </div>
                    ) : (
                      <span className="text-gray-500 text-sm">Not used</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-sm">
                    {new Date(volume.created_at).toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => handleAction('remove', volume)}
                      className="p-2 text-red-400 hover:text-red-300 hover:bg-red-900/20 rounded transition-colors"
                      title="Remove volume"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredVolumes.length === 0 && (
          <div className="text-center py-12 text-gray-400">
            <HardDrive className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No volumes found</p>
          </div>
        )}
      </div>

      {/* Create Modal */}
      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Create Volume">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Volume Name</label>
            <input
              type="text"
              value={volumeName}
              onChange={(e) => setVolumeName(e.target.value)}
              placeholder="my-volume"
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Driver</label>
            <select
              value={volumeDriver}
              onChange={(e) => setVolumeDriver(e.target.value)}
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:border-blue-500"
            >
              <option value="local">local</option>
            </select>
          </div>
          <div className="flex justify-end gap-3 pt-4">
            <button
              onClick={() => setShowCreateModal(false)}
              className="px-4 py-2 bg-gray-600 hover:bg-gray-700 text-white rounded transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleCreateVolume}
              disabled={!volumeName.trim() || createMutation.isPending}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </button>
          </div>
        </div>
      </Modal>

      {/* Confirm Dialog */}
      <ConfirmDialog
        isOpen={confirmState.isOpen}
        onClose={() => setConfirmState({ isOpen: false, action: null, volume: null, volumeIds: [] })}
        onConfirm={handleConfirm}
        title={`Confirm ${confirmState.action}`}
        message={getConfirmMessage()}
      />

      {/* Error Dialog */}
      <ErrorDialog
        isOpen={errorState.isOpen}
        onClose={() => setErrorState({ ...errorState, isOpen: false })}
        title={errorState.title}
        error={errorState.error}
        details={errorState.details}
      />

      {/* Success Dialog */}
      <SuccessDialog
        isOpen={successState.isOpen}
        onClose={() => setSuccessState({ ...successState, isOpen: false })}
        title={successState.title}
        message={successState.message}
        details={successState.details}
      />
    </div>
  );
}

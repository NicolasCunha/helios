import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Trash2, Plus, Search, CheckSquare, XSquare, Network } from 'lucide-react';
import api from '../services/api';
import type { Network as NetworkType } from '../types';
import ConfirmDialog from '../components/ConfirmDialog';
import ErrorDialog from '../components/ErrorDialog';
import Modal from '../components/Modal';

export default function Networks() {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [networkName, setNetworkName] = useState('');
  const [networkDriver, setNetworkDriver] = useState('bridge');
  const [confirmState, setConfirmState] = useState<{
    isOpen: boolean;
    action: 'remove' | 'prune' | 'bulk-remove' | null;
    network: NetworkType | null;
    networkIds?: string[];
  }>({ isOpen: false, action: null, network: null, networkIds: [] });
  const [errorState, setErrorState] = useState({
    isOpen: false,
    title: '',
    error: '',
    details: '',
  });
  const queryClient = useQueryClient();

  const { data: networksResponse, isLoading, error } = useQuery({
    queryKey: ['networks'],
    queryFn: async () => {
      const response = await api.get('/networks');
      return response.data;
    },
    refetchInterval: 5000,
  });

  const { data: _containersData } = useQuery({
    queryKey: ['containers'],
    queryFn: async () => {
      const response = await api.get('/containers');
      return response.data;
    },
    refetchInterval: 10000,
  });

  const removeMutation = useMutation({
    mutationFn: async (id: string) => {
      await api.delete(`/networks/${id}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks'] });
    },
  });

  const bulkRemoveMutation = useMutation({
    mutationFn: async (networkIds: string[]) => {
      await api.post('/networks/bulk/remove', { network_ids: networkIds });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks'] });
      setSelectedIds([]);
    },
  });

  const pruneMutation = useMutation({
    mutationFn: async () => {
      const response = await api.post('/networks/prune');
      return response.data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks'] });
    },
  });

  const createMutation = useMutation({
    mutationFn: async (data: { name: string; driver: string }) => {
      await api.post('/networks', data);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['networks'] });
      setShowCreateModal(false);
      setNetworkName('');
      setNetworkDriver('bridge');
    },
  });

  const getContainersOnNetwork = (network: NetworkType) => {
    if (!network.containers) return [];
    // The containers field is a map with container IDs as keys
    // Each value has Name, EndpointID, etc.
    return Object.entries(network.containers).map(([id, info]: [string, any]) => ({
      id,
      name: info.Name || id.substring(0, 12),
    }));
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

  const handleAction = (action: 'remove' | 'prune' | 'bulk-remove', network: NetworkType | null, networkIds?: string[]) => {
    setConfirmState({
      isOpen: true,
      action,
      network,
      networkIds,
    });
  };

  const handleConfirm = async () => {
    const { action, network, networkIds } = confirmState;
    setConfirmState({ isOpen: false, action: null, network: null, networkIds: [] });

    try {
      if (action === 'remove' && network) {
        await removeMutation.mutateAsync(network.id);
      } else if (action === 'prune') {
        await pruneMutation.mutateAsync();
      } else if (action === 'bulk-remove' && networkIds) {
        await bulkRemoveMutation.mutateAsync(networkIds);
      }
    } catch (error: unknown) {
      handleError(action || 'operation', error);
    }
  };

  const handleCreateNetwork = async () => {
    const name = networkName?.trim();
    if (!name) {
      setErrorState({
        isOpen: true,
        title: 'Validation Error',
        error: 'Network name is required',
        details: '',
      });
      return;
    }
    try {
      await createMutation.mutateAsync({ name, driver: networkDriver });
    } catch (error: unknown) {
      handleError('create network', error);
    }
  };

  const handleSelectAll = () => {
    const selectableNetworks = filteredNetworks.filter((n: NetworkType) => !['bridge', 'host', 'none'].includes(n.name));
    if (selectedIds.length === selectableNetworks.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(selectableNetworks.map((n: NetworkType) => n.id));
    }
  };

  const handleSelectNetwork = (id: string) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(nid => nid !== id) : [...prev, id]
    );
  };

  if (isLoading) {
    return <div className="flex items-center justify-center h-64">Loading networks...</div>;
  }

  if (error) {
    return <div className="text-red-400">Error loading networks: {(error as Error).message}</div>;
  }

  const networks: NetworkType[] = networksResponse?.networks || [];

  const filteredNetworks = networks.filter((network) =>
    network.name.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getConfirmMessage = () => {
    const { action, network, networkIds } = confirmState;
    if (action === 'remove' && network) {
      const containers = getContainersOnNetwork(network);
      if (containers.length > 0) {
        return `Network "${network.name}" is currently used by ${containers.length} container(s). Removing it may cause issues.`;
      }
      return `Are you sure you want to remove network "${network.name}"?`;
    }
    if (action === 'bulk-remove' && networkIds) {
      return `Are you sure you want to remove ${networkIds.length} network(s)?`;
    }
    if (action === 'prune') {
      return 'This will remove all unused networks. Are you sure?';
    }
    return '';
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <h1 className="text-3xl font-bold text-white">Networks</h1>
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
            Create Network
          </button>
        </div>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400 w-5 h-5" />
        <input
          type="text"
          placeholder="Search networks..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
        />
      </div>

      {/* Bulk Actions Bar */}
      {selectedIds.length > 0 && (
        <div className="bg-blue-900 border border-blue-700 rounded p-3 flex items-center justify-between">
          <span className="text-white">{selectedIds.length} network(s) selected</span>
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
                  {selectedIds.length > 0 ? (
                    <CheckSquare className="w-5 h-5" />
                  ) : (
                    <XSquare className="w-5 h-5" />
                  )}
                </button>
              </th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Name</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">ID</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Driver</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Scope</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Connected Containers</th>
              <th className="px-4 py-3 text-left text-gray-300 font-medium">Created</th>
              <th className="px-4 py-3 text-right text-gray-300 font-medium">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-700">
            {filteredNetworks.map((network) => {
              const containers = getContainersOnNetwork(network);
              const isSystemNetwork = ['bridge', 'host', 'none'].includes(network.name);
              return (
                <tr key={network.id} className="hover:bg-gray-750">
                  <td className="px-4 py-3">
                    {!isSystemNetwork && (
                      <input
                        type="checkbox"
                        checked={selectedIds.includes(network.id)}
                        onChange={() => handleSelectNetwork(network.id)}
                        className="w-4 h-4 rounded bg-gray-700 border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-800"
                      />
                    )}
                  </td>
                  <td className="px-4 py-3 font-medium text-white">{network.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-gray-400">
                    {network.id.substring(0, 12)}
                  </td>
                  <td className="px-4 py-3 text-gray-300">{network.driver}</td>
                  <td className="px-4 py-3 text-gray-300">{network.scope}</td>
                  <td className="px-4 py-3">
                    {containers.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {containers.slice(0, 2).map((container: any) => (
                          <span
                            key={container.id}
                            className="text-xs px-2 py-0.5 bg-green-900 text-green-200 rounded"
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
                      <span className="text-gray-500 text-sm">No containers</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-gray-400 text-sm">
                    {network.created ? new Date(network.created).toLocaleString() : 'N/A'}
                  </td>
                  <td className="px-4 py-3 text-right">
                    {!isSystemNetwork && (
                      <button
                        onClick={() => handleAction('remove', network)}
                        className="p-2 text-red-400 hover:text-red-300 hover:bg-red-900/20 rounded transition-colors"
                        title="Remove network"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>

        {filteredNetworks.length === 0 && (
          <div className="text-center py-12 text-gray-400">
            <Network className="w-12 h-12 mx-auto mb-3 opacity-50" />
            <p>No networks found</p>
          </div>
        )}
      </div>

      {/* Create Modal */}
      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Create Network">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Network Name</label>
            <input
              type="text"
              value={networkName}
              onChange={(e) => setNetworkName(e.target.value)}
              placeholder="my-network"
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">Driver</label>
            <select
              value={networkDriver}
              onChange={(e) => setNetworkDriver(e.target.value)}
              className="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white focus:outline-none focus:border-blue-500"
            >
              <option value="bridge">bridge</option>
              <option value="overlay">overlay</option>
              <option value="macvlan">macvlan</option>
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
              onClick={handleCreateNetwork}
              disabled={!networkName.trim() || createMutation.isPending}
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
        onClose={() => setConfirmState({ isOpen: false, action: null, network: null, networkIds: [] })}
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
    </div>
  );
}

import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Download, Trash2, Search, CheckSquare, XSquare } from 'lucide-react';
import api, { bulkRemoveImages } from '../services/api';
import type { Image } from '../types';
import ConfirmDialog from '../components/ConfirmDialog';
import ErrorDialog from '../components/ErrorDialog';
import SuccessDialog from '../components/SuccessDialog';
import Modal from '../components/Modal';

export default function Images() {
  const [searchTerm, setSearchTerm] = useState('');
  const [usageFilter, setUsageFilter] = useState<'all' | 'in-use' | 'unused'>('all');
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [showPullModal, setShowPullModal] = useState(false);
  const [pullImageName, setPullImageName] = useState('');
  const [pullProgress, setPullProgress] = useState<string[]>([]);
  const [isPulling, setIsPulling] = useState(false);
  const [availableTags, setAvailableTags] = useState<string[]>([]);
  const [isLoadingTags, setIsLoadingTags] = useState(false);
  const [selectedTag, setSelectedTag] = useState('');
  const [pullPercentage, setPullPercentage] = useState(0);
  const [confirmState, setConfirmState] = useState<{
    isOpen: boolean;
    action: 'remove' | 'prune' | 'bulk-remove' | null;
    image: Image | null;
    imageIds?: string[];
    affectedContainers?: string[];
  }>({ isOpen: false, action: null, image: null, imageIds: [], affectedContainers: [] });
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
    queryKey: ['images'],
    queryFn: async () => {
      const response = await api.get('/images');
      return response.data;
    },
    refetchInterval: 10000,
  });

  const { data: containersData } = useQuery({
    queryKey: ['containers'],
    queryFn: async () => {
      const response = await api.get('/containers', { params: { all: true } });
      return response.data;
    },
    refetchInterval: 10000,
  });

  const handleError = (action: string, error: unknown) => {
    const err = error as { response?: { data?: { error?: string; detail?: string } }; message?: string };
    setErrorState({
      isOpen: true,
      title: `Failed to ${action}`,
      error: err.response?.data?.error || err.message || 'Unknown error',
      details: err.response?.data?.detail || '',
    });
  };

  const removeMutation = useMutation({
    mutationFn: (id: string) => api.delete(`/images/${id}`, { params: { force: true } }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['images'] }),
    onError: (error) => handleError('remove image', error),
  });

  const bulkRemoveMutation = useMutation({
    mutationFn: (imageIds: string[]) => bulkRemoveImages(imageIds, true),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['images'] });
      setSelectedIds([]);
      
      const failed = response.failed || 0;
      const total = response.total || 0;
      
      if (failed > 0) {
        const failedResults = response.results?.filter((r: { Success: boolean }) => !r.Success) || [];
        const errorDetails = failedResults.map((r: { ContainerName: string; Error: string }) => 
          `${r.ContainerName}: ${r.Error}`
        ).join('\n');
        
        handleError('remove images', { 
          message: `${failed} of ${total} images failed to remove`,
          response: { data: { detail: errorDetails } }
        });
      }
    },
    onError: (error) => handleError('remove images', error),
  });

  const pruneMutation = useMutation({
    mutationFn: () => api.post('/images/prune', {}, { params: { all: true } }),
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['images'] });
      const spaceReclaimed = (response.data as { space_reclaimed?: number })?.space_reclaimed || 0;
      setSuccessState({
        isOpen: true,
        title: 'Images Pruned Successfully',
        message: 'Unused images have been removed.',
        details: `Space reclaimed: ${formatBytes(spaceReclaimed)}`,
      });
    },
    onError: (error) => handleError('prune images', error),
  });

  const fetchAvailableTags = async (imageName: string) => {
    setIsLoadingTags(true);
    setAvailableTags([]);
    
    try {
      const response = await api.get('/images/tags', {
        params: {
          image: imageName.trim(),
          limit: 20
        }
      });
      
      const tags = response.data.tags || [];
      console.log('Fetched tags:', tags);
      setAvailableTags(tags);
    } catch (error) {
      console.error('Failed to fetch tags:', error);
      // Don't set fallback, let user type the tag manually
      setAvailableTags([]);
    } finally {
      setIsLoadingTags(false);
    }
  };

  const handleImageNameChange = (value: string) => {
    setPullImageName(value);
    setSelectedTag('');
    setAvailableTags([]);
    
    // Check if tag is not specified and has at least 5 characters
    if (value && !value.includes(':') && value.length >= 5) {
      // Extract image name without tag
      const imageName = value.split(':')[0];
      fetchAvailableTags(imageName);
    }
  };

  const handlePullImage = async () => {
    if (!pullImageName.trim()) return;
    
    // Determine final image name with tag
    let finalImageName = pullImageName;
    if (!pullImageName.includes(':')) {
      const tag = selectedTag || 'latest';
      finalImageName = `${pullImageName}:${tag}`;
    }

    setIsPulling(true);
    setPullProgress([]);
    setPullPercentage(0);

    try {
      // Prepare request body
      const requestBody = {
        image: finalImageName,
      };

      const response = await fetch(`${import.meta.env.VITE_API_URL || 'http://localhost:8080/helios'}/images/pull`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
      });

      const reader = response.body?.getReader();
      const decoder = new TextDecoder();

      if (!reader) throw new Error('Failed to read response');

      const layerProgress: Record<string, number> = {};
      let totalLayers = 0;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const chunk = decoder.decode(value);
        const lines = chunk.split('\n').filter(line => line.trim());

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.substring(6);
            setPullProgress(prev => [...prev, data]);
            
            // Try to parse progress from Docker pull messages
            try {
              const progressData = JSON.parse(data);
              
              // Check for authentication or other errors
              if (progressData.error || progressData.errorDetail) {
                const errorMsg = progressData.error || 'Pull failed';
                const errorDetail = progressData.errorDetail || '';
                const fullError = `${errorMsg}${errorDetail ? ': ' + errorDetail : ''}`;
                setPullProgress(prev => [...prev, `Error: ${fullError}`]);
                reader.cancel();
                throw new Error(fullError);
              }
              
              if (progressData.status && progressData.id) {
                const layerId = progressData.id;
                
                if (progressData.status === 'Downloading' || progressData.status === 'Extracting') {
                  if (!layerProgress[layerId]) {
                    totalLayers++;
                  }
                  
                  if (progressData.progressDetail?.current && progressData.progressDetail?.total) {
                    const percent = (progressData.progressDetail.current / progressData.progressDetail.total) * 100;
                    layerProgress[layerId] = percent;
                  }
                } else if (progressData.status === 'Download complete' || progressData.status === 'Pull complete') {
                  layerProgress[layerId] = 100;
                }
                
                // Calculate overall progress
                if (totalLayers > 0) {
                  const totalProgress = Object.values(layerProgress).reduce((sum, val) => sum + val, 0);
                  const avgProgress = totalProgress / totalLayers;
                  setPullPercentage(Math.min(Math.round(avgProgress), 100));
                }
              }
            } catch (err) {
              // Check if it's an actual error (not just non-JSON)
              if (err instanceof Error && err.message && !err.message.includes('JSON')) {
                throw err;
              }
              // Otherwise not JSON or doesn't have expected format, ignore
            }
          }
        }
      }

      queryClient.invalidateQueries({ queryKey: ['images'] });
      setPullPercentage(100);
      setTimeout(() => {
        setShowPullModal(false);
        setPullImageName('');
        setPullProgress([]);
        setAvailableTags([]);
        setSelectedTag('');
        setPullPercentage(0);
      }, 2000);
    } catch (error: unknown) {
      handleError('pull image', error);
      setPullPercentage(0);
    } finally {
      setIsPulling(false);
    }
  };

  const handleAction = (action: 'remove' | 'prune' | 'bulk-remove', image: Image | null, imageIds?: string[]) => {
    let affectedContainers: string[] = [];
    
    if (action === 'remove' && image) {
      const usingContainers = getContainersUsingImage(image.id, image.repo_tags || []);
      affectedContainers = usingContainers.map((c: { name: string }) => c.name);
    } else if (action === 'bulk-remove' && imageIds) {
      // Collect all affected containers from all selected images
      imageIds.forEach(id => {
        const image = images.find(img => img.id === id);
        if (image) {
          const usingContainers = getContainersUsingImage(image.id, image.repo_tags || []);
          affectedContainers.push(...usingContainers.map((c: { name: string }) => c.name));
        }
      });
      // Remove duplicates
      affectedContainers = Array.from(new Set(affectedContainers));
    }
    
    setConfirmState({ isOpen: true, action, image, imageIds, affectedContainers });
  };

  const handleConfirm = () => {
    if (confirmState.action === 'remove' && confirmState.image) {
      removeMutation.mutate(confirmState.image.id);
    } else if (confirmState.action === 'bulk-remove' && confirmState.imageIds) {
      bulkRemoveMutation.mutate(confirmState.imageIds);
    } else if (confirmState.action === 'prune') {
      pruneMutation.mutate();
    }
  };

  const images: Image[] = data?.images || [];
  const containers = containersData?.containers || [];
  
  const getContainersUsingImage = (imageId: string, repoTags: string[]) => {
    return containers.filter((container: { image: string }) => {
      const containerImage = container.image.toLowerCase();
      const shortId = imageId.substring(7, 19); // Remove 'sha256:' prefix and get short ID
      
      // Match by short image ID (12 chars)
      if (containerImage.includes(shortId)) {
        return true;
      }
      
      // Match by full image ID
      if (containerImage.includes(imageId.toLowerCase())) {
        return true;
      }
      
      // Match by repo:tag (exact match)
      if (repoTags.some(tag => containerImage === tag.toLowerCase())) {
        return true;
      }
      
      // Match by repo:tag without registry prefix (e.g., container uses 'nginx:latest' but image is 'docker.io/library/nginx:latest')
      for (const tag of repoTags) {
        const tagLower = tag.toLowerCase();
        // Remove common registry prefixes
        const withoutRegistry = tagLower.replace(/^(docker\.io\/|registry\.hub\.docker\.com\/)/, '');
        const withoutLibrary = withoutRegistry.replace(/^library\//, '');
        
        if (containerImage === withoutRegistry || containerImage === withoutLibrary) {
          return true;
        }
        
        // Check if container image is missing :latest tag
        // If image is "nginx:latest" and container uses "nginx", they match
        if (tagLower.endsWith(':latest')) {
          const repoWithoutTag = tagLower.slice(0, -7); // Remove ':latest'
          if (containerImage === repoWithoutTag) {
            return true;
          }
          
          // Also check without registry prefix
          const repoWithoutRegistryNoTag = withoutRegistry.endsWith(':latest') 
            ? withoutRegistry.slice(0, -7) 
            : withoutRegistry;
          const repoWithoutLibraryNoTag = withoutLibrary.endsWith(':latest')
            ? withoutLibrary.slice(0, -7)
            : withoutLibrary;
            
          if (containerImage === repoWithoutRegistryNoTag || containerImage === repoWithoutLibraryNoTag) {
            return true;
          }
        }
        
        // Also check if container image ends with the tag (handles different registry prefixes)
        const tagParts = tagLower.split('/');
        const lastPart = tagParts[tagParts.length - 1];
        if (containerImage.endsWith(lastPart)) {
          return true;
        }
        
        // Check if container image without tag matches repo without tag
        const containerWithoutTag = containerImage.split(':')[0];
        const tagWithoutVersion = tagLower.split(':')[0];
        if (containerWithoutTag === tagWithoutVersion) {
          return true;
        }
      }
      
      return false;
    });
  };
  
  const filteredImages = images.filter(image => {
    // Search filter
    const matchesSearch = image.repo_tags?.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase())) ||
      image.id.toLowerCase().includes(searchTerm.toLowerCase());
    
    if (!matchesSearch) return false;
    
    // Usage filter
    if (usageFilter === 'all') return true;
    
    const usingContainers = getContainersUsingImage(image.id, image.repo_tags || []);
    if (usageFilter === 'in-use') return usingContainers.length > 0;
    if (usageFilter === 'unused') return usingContainers.length === 0;
    
    return true;
  });

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatDate = (timestamp: number) => {
    return new Date(timestamp * 1000).toLocaleDateString();
  };

  const toggleSelectAll = () => {
    if (selectedIds.length === filteredImages.length) {
      setSelectedIds([]);
    } else {
      setSelectedIds(filteredImages.map(img => img.id));
    }
  };

  const toggleSelect = (id: string) => {
    setSelectedIds(prev =>
      prev.includes(id) ? prev.filter(i => i !== id) : [...prev, id]
    );
  };

  const handleBulkRemove = () => {
    handleAction('bulk-remove', null, selectedIds);
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
        <p className="text-red-400">Failed to load images: {error.message}</p>
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-3xl font-bold text-white">Images</h1>
        <div className="flex items-center gap-3">
          <button
            onClick={() => handleAction('prune', null)}
            className="px-4 py-2 bg-yellow-600 hover:bg-yellow-700 text-white rounded-lg transition-colors"
          >
            Prune Unused
          </button>
          <button
            onClick={() => setShowPullModal(true)}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors flex items-center gap-2"
          >
            <Download className="w-4 h-4" />
            Pull Image
          </button>
        </div>
      </div>

      {/* Search and Filters */}
      <div className="bg-gray-800 rounded-lg border border-gray-700 p-4 mb-6">
        <div className="flex gap-4">
          <div className="flex-1 relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              placeholder="Search by tag or ID..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-10 pr-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-blue-500"
            />
          </div>
          <div className="w-48">
            <select
              value={usageFilter}
              onChange={(e) => setUsageFilter(e.target.value as 'all' | 'in-use' | 'unused')}
              className="w-full px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-blue-500"
            >
              <option value="all">All Images</option>
              <option value="in-use">In Use</option>
              <option value="unused">Not Used</option>
            </select>
          </div>
        </div>
      </div>

      {/* Bulk Actions Bar */}
      {selectedIds.length > 0 && (
        <div className="bg-blue-600 rounded-lg p-4 mb-6 flex items-center justify-between">
          <div className="flex items-center gap-4">
            <span className="text-white font-medium">
              {selectedIds.length} image{selectedIds.length !== 1 ? 's' : ''} selected
            </span>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleBulkRemove}
              disabled={bulkRemoveMutation.isPending}
              className="px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors flex items-center gap-2 disabled:opacity-50"
            >
              <Trash2 className="w-4 h-4" />
              Remove All
            </button>
            <button
              onClick={() => setSelectedIds([])}
              className="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white rounded-lg transition-colors"
            >
              Clear Selection
            </button>
          </div>
        </div>
      )}

      {/* Images Table */}
      {filteredImages.length === 0 ? (
        <div className="bg-gray-800 rounded-lg border border-gray-700 p-8 text-center">
          <p className="text-gray-400">No images found</p>
        </div>
      ) : (
        <div className="bg-gray-800 rounded-lg border border-gray-700 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-700/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider w-12">
                    <button
                      onClick={toggleSelectAll}
                      className="p-1 hover:bg-gray-600 rounded transition-colors"
                      title={selectedIds.length === filteredImages.length ? 'Deselect All' : 'Select All'}
                    >
                      {selectedIds.length === filteredImages.length ? (
                        <CheckSquare className="w-5 h-5 text-blue-400" />
                      ) : (
                        <XSquare className="w-5 h-5 text-gray-400" />
                      )}
                    </button>
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Repository
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Tag
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Image ID
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Size
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Used By
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-300 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-700">
                {filteredImages.map((image) => {
                  const repoTag = image.repo_tags?.[0] || '<none>:<none>';
                  const [repo, tag] = repoTag.split(':');
                  const usingContainers = getContainersUsingImage(image.id, image.repo_tags || []);
                  
                  return (
                    <tr key={image.id} className="hover:bg-gray-700/30 transition-colors">
                      <td className="px-6 py-4 whitespace-nowrap">
                        <input
                          type="checkbox"
                          checked={selectedIds.includes(image.id)}
                          onChange={() => toggleSelect(image.id)}
                          className="w-4 h-4 rounded border-gray-600 text-blue-600 focus:ring-blue-500 focus:ring-offset-gray-800 bg-gray-700 cursor-pointer"
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-300 font-mono">
                        {repo || '<none>'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="inline-flex px-2 py-1 text-xs font-semibold rounded bg-blue-500/20 text-blue-400 border border-blue-500/50">
                          {tag || '<none>'}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400 font-mono">
                        {image.id.substring(0, 12)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                        {formatDate(image.created)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-400">
                        {formatBytes(image.size)}
                      </td>
                      <td className="px-6 py-4 text-sm text-gray-400">
                        {usingContainers.length > 0 ? (
                          <div className="flex flex-col gap-1">
                            {usingContainers.slice(0, 3).map((container: { name: string }) => (
                              <Link
                                key={container.name}
                                to={`/containers`}
                                className="text-blue-400 hover:text-blue-300 hover:underline"
                              >
                                {container.name}
                              </Link>
                            ))}
                            {usingContainers.length > 3 && (
                              <span className="text-xs text-gray-500">
                                +{usingContainers.length - 3} more
                              </span>
                            )}
                          </div>
                        ) : (
                          <span className="text-gray-500 italic">None</span>
                        )}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                        <button
                          onClick={() => handleAction('remove', image)}
                          disabled={removeMutation.isPending}
                          className="p-2 text-red-400 hover:bg-red-500/20 rounded-lg transition-colors disabled:opacity-50"
                          title="Remove"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Pull Image Modal */}
      <Modal
        isOpen={showPullModal}
        onClose={() => !isPulling && setShowPullModal(false)}
        title="Pull Docker Image"
        footer={
          <>
            <button
              onClick={() => setShowPullModal(false)}
              disabled={isPulling}
              className="px-4 py-2 text-gray-300 hover:text-white transition-colors disabled:opacity-50"
            >
              Cancel
            </button>
            <button
              onClick={handlePullImage}
              disabled={isPulling || !pullImageName.trim()}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors disabled:opacity-50"
            >
              {isPulling ? 'Pulling...' : 'Pull'}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-300 mb-2">
              Image Name
            </label>
            <input
              type="text"
              value={pullImageName}
              onChange={(e) => handleImageNameChange(e.target.value)}
              placeholder="e.g., nginx, ubuntu, redis"
              disabled={isPulling}
              className="w-full px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white placeholder-gray-400 focus:outline-none focus:border-blue-500 disabled:opacity-50"
            />
            <p className="text-xs text-gray-400 mt-1">
              Enter image name (minimum 5 characters to fetch tags). For private registries, use full path: registry.example.com/image
            </p>
          </div>

          {isLoadingTags && (
            <div className="text-sm text-gray-400 flex items-center gap-2">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-500"></div>
              Loading available tags...
            </div>
          )}

          {availableTags.length > 0 && !pullImageName.includes(':') && (
            <div>
              <label className="block text-sm font-medium text-gray-300 mb-2">
                Select Tag
              </label>
              <select
                value={selectedTag}
                onChange={(e) => setSelectedTag(e.target.value)}
                disabled={isPulling}
                className="w-full px-4 py-2 bg-gray-700 border border-gray-600 rounded-lg text-white focus:outline-none focus:border-blue-500 disabled:opacity-50"
              >
                <option value="">latest (default)</option>
                {availableTags.map((tag) => (
                  <option key={tag} value={tag}>
                    {tag}
                  </option>
                ))}
              </select>
            </div>
          )}

          {isPulling && pullPercentage > 0 && (
            <div>
              <div className="flex justify-between text-sm text-gray-400 mb-2">
                <span>Pulling image...</span>
                <span>{pullPercentage}%</span>
              </div>
              <div className="w-full bg-gray-700 rounded-full h-2.5">
                <div
                  className="bg-blue-600 h-2.5 rounded-full transition-all duration-300"
                  style={{ width: `${pullPercentage}%` }}
                ></div>
              </div>
            </div>
          )}

          {pullProgress.length > 0 && (
            <div className="bg-gray-900/50 rounded-lg border border-gray-700 p-4 max-h-64 overflow-y-auto">
              <div className="space-y-1">
                {pullProgress.map((msg, idx) => (
                  <div key={idx} className="text-xs font-mono text-gray-400">
                    {msg}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </Modal>

      {/* Confirm Dialog */}
      <ConfirmDialog
        isOpen={confirmState.isOpen}
        onClose={() => setConfirmState({ isOpen: false, action: null, image: null, imageIds: [], affectedContainers: [] })}
        onConfirm={handleConfirm}
        title={
          confirmState.action === 'prune' 
            ? 'Prune Unused Images' 
            : confirmState.action === 'bulk-remove'
            ? 'Remove Selected Images'
            : 'Remove Image'
        }
        message={
          confirmState.action === 'prune'
            ? 'Are you sure you want to remove all unused images? This will free up disk space.'
            : confirmState.action === 'bulk-remove'
            ? confirmState.affectedContainers && confirmState.affectedContainers.length > 0
              ? `You are about to remove ${confirmState.imageIds?.length} image(s). These images are being used by ${confirmState.affectedContainers.length} container(s). Removing these images will force remove the following containers:\n\n${confirmState.affectedContainers.join(', ')}\n\nThis action cannot be undone.`
              : `Are you sure you want to remove ${confirmState.imageIds?.length} selected image(s)? This action cannot be undone.`
            : confirmState.affectedContainers && confirmState.affectedContainers.length > 0
            ? `This image is being used by ${confirmState.affectedContainers.length} container(s). Removing this image will force remove the following containers:\n\n${confirmState.affectedContainers.join(', ')}\n\nThis action cannot be undone.`
            : `Are you sure you want to remove this image? This action cannot be undone.`
        }
        confirmText={confirmState.action === 'prune' ? 'Prune' : 'Remove'}
        variant="danger"
      />

      {/* Error Dialog */}
      <ErrorDialog
        isOpen={errorState.isOpen}
        onClose={() => setErrorState({ isOpen: false, title: '', error: '', details: '' })}
        title={errorState.title}
        error={errorState.error}
        details={errorState.details}
      />

      {/* Success Dialog */}
      <SuccessDialog
        isOpen={successState.isOpen}
        onClose={() => setSuccessState({ isOpen: false, title: '', message: '', details: '' })}
        title={successState.title}
        message={successState.message}
        details={successState.details}
      />
    </div>
  );
}

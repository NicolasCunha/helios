import axios from 'axios';
import type {
  Container,
  Image,
  ImageDetail,
  Volume,
  Network,
} from '../types';

// Use relative path in production (served by nginx), absolute for dev
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 
  (import.meta.env.DEV ? 'http://localhost:5000/helios' : '/helios');

const api = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response) {
      // Server responded with error
      const message = error.response.data?.detail || error.response.data?.error || 'An error occurred';
      throw new Error(message);
    } else if (error.request) {
      // Request made but no response
      throw new Error('No response from server. Is the backend running?');
    } else {
      // Error setting up request
      throw new Error(error.message);
    }
  }
);

// Health
export const getHealth = async () => {
  const { data } = await api.get('/health');
  return data;
};

// Containers
export const listContainers = async (params?: { all?: boolean; limit?: number; filter?: string }) => {
  const { data } = await api.get<{ containers: Container[]; count: number }>('/containers', { params });
  return data;
};

export const getContainer = async (id: string) => {
  const { data } = await api.get<Container>(`/containers/${id}`);
  return data;
};

export const startContainer = async (id: string) => {
  const { data } = await api.post(`/containers/${id}/start`);
  return data;
};

export const stopContainer = async (id: string) => {
  const { data } = await api.post(`/containers/${id}/stop`);
  return data;
};

export const restartContainer = async (id: string) => {
  const { data } = await api.post(`/containers/${id}/restart`);
  return data;
};

export const removeContainer = async (id: string, force?: boolean) => {
  const { data } = await api.delete(`/containers/${id}`, { params: { force } });
  return data;
};

// Bulk container operations
export const bulkStartContainers = async (containerIds: string[]) => {
  const { data } = await api.post('/containers/bulk/start', { container_ids: containerIds });
  return data;
};

export const bulkStopContainers = async (containerIds: string[]) => {
  const { data } = await api.post('/containers/bulk/stop', { container_ids: containerIds });
  return data;
};

export const bulkRemoveContainers = async (containerIds: string[], force: boolean = true) => {
  const { data } = await api.post('/containers/bulk/remove', { container_ids: containerIds, force });
  return data;
};

// Container logs
export const downloadContainerLogs = (id: string) => {
  window.open(`${API_BASE_URL}/containers/${id}/logs/download`, '_blank');
};

// Images
export const listImages = async (all?: boolean) => {
  const { data } = await api.get<{ images: Image[]; count: number }>('/images', { params: { all } });
  return data;
};

export const getImage = async (id: string) => {
  const { data } = await api.get<ImageDetail>(`/images/${id}`);
  return data;
};

export const pullImage = async (image: string, auth?: { username: string; password: string; serveraddress: string }) => {
  // Returns EventSource for SSE streaming
  return new EventSource(`${API_BASE_URL}/images/pull?image=${encodeURIComponent(image)}${auth ? `&auth=${JSON.stringify(auth)}` : ''}`);
};

export const removeImage = async (id: string, force?: boolean) => {
  const { data } = await api.delete(`/images/${id}`, { params: { force } });
  return data;
};

export const bulkRemoveImages = async (imageIds: string[], force: boolean = true) => {
  const { data } = await api.post('/images/bulk/remove', { image_ids: imageIds, force });
  return data;
};

export const pruneImages = async (all?: boolean) => {
  const { data } = await api.post('/images/prune', {}, { params: { all } });
  return data;
};

export const searchImages = async (term: string, limit?: number) => {
  const { data } = await api.get('/images/search', { params: { term, limit } });
  return data;
};

// Volumes
export const listVolumes = async () => {
  const { data } = await api.get<{ volumes: Volume[]; count: number }>('/volumes');
  return data;
};

export const getVolume = async (name: string) => {
  const { data } = await api.get<Volume>(`/volumes/${name}`);
  return data;
};

export const createVolume = async (volume: { name: string; driver?: string; driver_opts?: Record<string, string>; labels?: Record<string, string> }) => {
  const { data } = await api.post<Volume>('/volumes', volume);
  return data;
};

export const removeVolume = async (name: string, force?: boolean) => {
  const { data } = await api.delete(`/volumes/${name}`, { params: { force } });
  return data;
};

export const pruneVolumes = async (filters?: Record<string, string[]>) => {
  const { data } = await api.post('/volumes/prune', { filters });
  return data;
};

// Networks
export const listNetworks = async () => {
  const { data } = await api.get<{ networks: Network[]; count: number }>('/networks');
  return data;
};

export const getNetwork = async (id: string) => {
  const { data } = await api.get<Network>(`/networks/${id}`);
  return data;
};

export const createNetwork = async (network: {
  name: string;
  driver?: string;
  internal?: boolean;
  attachable?: boolean;
  ipam?: {
    driver?: string;
    config?: Array<{
      subnet?: string;
      gateway?: string;
    }>;
  };
  options?: Record<string, string>;
  labels?: Record<string, string>;
}) => {
  const { data } = await api.post<Network>('/networks', network);
  return data;
};

export const removeNetwork = async (id: string) => {
  const { data } = await api.delete(`/networks/${id}`);
  return data;
};

export const pruneNetworks = async (filters?: Record<string, string[]>) => {
  const { data} = await api.post('/networks/prune', { filters });
  return data;
};

export default api;

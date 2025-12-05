import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import api from '../services/api';

interface DashboardStats {
  containers: {
    total: number;
    running: number;
    stopped: number;
  };
  images: {
    total: number;
  };
  volumes: {
    total: number;
  };
  networks: {
    total: number;
  };
}

interface ResourceSummary {
  total_cpu_percent: number;
  total_memory_usage: number;
  total_memory_limit: number;
  total_memory_percent: number;
  total_network_rx: number;
  total_network_tx: number;
  container_count: number;
}

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats>({
    containers: { total: 0, running: 0, stopped: 0 },
    images: { total: 0 },
    volumes: { total: 0 },
    networks: { total: 0 },
  });
  const [resourceSummary, setResourceSummary] = useState<ResourceSummary | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadStats();
    // Refresh every 5 seconds
    const interval = setInterval(loadStats, 5000);
    return () => clearInterval(interval);
  }, []);

  const loadStats = async () => {
    try {
      const [containers, images, volumes, networks, summary] = await Promise.all([
        api.get('/containers?all=true&stats=false'), // Disable stats for faster loading
        api.get('/images'),
        api.get('/volumes'),
        api.get('/networks'),
        api.get('/dashboard/summary'),
      ]);

      const runningCount = containers.data.containers.filter(
        (c: any) => c.state === 'running'
      ).length;

      setStats({
        containers: {
          total: containers.data.count,
          running: runningCount,
          stopped: containers.data.count - runningCount,
        },
        images: {
          total: images.data.count,
        },
        volumes: {
          total: volumes.data.count,
        },
        networks: {
          total: networks.data.count,
        },
      });
      
      setResourceSummary(summary.data);
    } catch (error) {
      console.error('Failed to load stats:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    );
  }
  
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i];
  };

  return (
    <div>
      <h1 className="text-3xl font-bold text-white mb-8">Dashboard</h1>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
        {/* Containers Card */}
        <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-200">Containers</h3>
            <div className="p-3 bg-blue-500/20 rounded-lg">
              <svg className="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            </div>
          </div>
          <div className="text-3xl font-bold text-white mb-2">{stats.containers.total}</div>
          <div className="flex items-center text-sm">
            <span className="text-green-400 font-medium">{stats.containers.running} running</span>
            <span className="mx-2 text-gray-600">·</span>
            <span className="text-gray-400">{stats.containers.stopped} stopped</span>
          </div>
        </div>

        {/* Images Card */}
        <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-200">Images</h3>
            <div className="p-3 bg-purple-500/20 rounded-lg">
              <svg className="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
          </div>
          <div className="text-3xl font-bold text-white mb-2">{stats.images.total}</div>
          <div className="text-sm text-gray-400">Docker images</div>
        </div>

        {/* Volumes Card */}
        <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-200">Volumes</h3>
            <div className="p-3 bg-green-500/20 rounded-lg">
              <svg className="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z" />
              </svg>
            </div>
          </div>
          <div className="text-3xl font-bold text-white mb-2">{stats.volumes.total}</div>
          <div className="text-sm text-gray-400">Storage volumes</div>
        </div>

        {/* Networks Card */}
        <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg font-semibold text-gray-200">Networks</h3>
            <div className="p-3 bg-orange-500/20 rounded-lg">
              <svg className="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
              </svg>
            </div>
          </div>
          <div className="text-3xl font-bold text-white mb-2">{stats.networks.total}</div>
          <div className="text-sm text-gray-400">Docker networks</div>
        </div>
      </div>

      {/* Resource Usage Summary */}
      {resourceSummary && resourceSummary.container_count > 0 && (
        <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6 mb-8">
          <h2 className="text-xl font-semibold text-white mb-4">Resource Usage (Running Containers)</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {/* CPU Usage */}
            <div className="flex flex-col">
              <span className="text-sm text-gray-400 mb-2">Total CPU</span>
              <div className="text-2xl font-bold text-white mb-1">
                {resourceSummary.total_cpu_percent.toFixed(1)}%
              </div>
              <span className="text-xs text-gray-500">
                Across {resourceSummary.container_count} container{resourceSummary.container_count !== 1 ? 's' : ''}
              </span>
            </div>

            {/* Memory Usage */}
            <div className="flex flex-col">
              <span className="text-sm text-gray-400 mb-2">Memory Usage</span>
              <div className="text-2xl font-bold text-white mb-1">
                {formatBytes(resourceSummary.total_memory_usage)}
              </div>
              <span className="text-xs text-gray-500">
                of {formatBytes(resourceSummary.total_memory_limit)} ({resourceSummary.total_memory_percent.toFixed(1)}%)
              </span>
            </div>

            {/* Network RX */}
            <div className="flex flex-col">
              <span className="text-sm text-gray-400 mb-2">Network Received</span>
              <div className="text-2xl font-bold text-white mb-1">
                {formatBytes(resourceSummary.total_network_rx)}
              </div>
              <span className="text-xs text-gray-500">Total received</span>
            </div>

            {/* Network TX */}
            <div className="flex flex-col">
              <span className="text-sm text-gray-400 mb-2">Network Transmitted</span>
              <div className="text-2xl font-bold text-white mb-1">
                {formatBytes(resourceSummary.total_network_tx)}
              </div>
              <span className="text-xs text-gray-500">Total transmitted</span>
            </div>
          </div>
        </div>
      )}

      {/* Quick Links */}
      <div className="bg-gray-800 rounded-lg shadow-lg border border-gray-700 p-6">
        <h2 className="text-xl font-semibold text-white mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <Link
            to="/containers"
            className="flex items-center p-4 border border-gray-600 rounded-lg hover:border-blue-500 hover:bg-gray-700/50 transition-all"
          >
            <div className="p-2 bg-blue-500/20 rounded">
              <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
              </svg>
            </div>
            <span className="ml-3 font-medium text-gray-200">View Containers</span>
          </Link>

          <Link
            to="/images"
            className="flex items-center p-4 border border-gray-600 rounded-lg hover:border-purple-500 hover:bg-gray-700/50 transition-all"
          >
            <div className="p-2 bg-purple-500/20 rounded">
              <svg className="w-5 h-5 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </div>
            <span className="ml-3 font-medium text-gray-200">Manage Images</span>
          </Link>

          <Link
            to="/volumes"
            className="flex items-center p-4 border border-gray-600 rounded-lg hover:border-green-500 hover:bg-gray-700/50 transition-all"
          >
            <div className="p-2 bg-green-500/20 rounded">
              <svg className="w-5 h-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 19a2 2 0 01-2-2V7a2 2 0 012-2h4l2 2h4a2 2 0 012 2v1M5 19h14a2 2 0 002-2v-5a2 2 0 00-2-2H9a2 2 0 00-2 2v5a2 2 0 01-2 2z" />
              </svg>
            </div>
            <span className="ml-3 font-medium text-gray-200">View Volumes</span>
          </Link>

          <Link
            to="/networks"
            className="flex items-center p-4 border border-gray-600 rounded-lg hover:border-orange-500 hover:bg-gray-700/50 transition-all"
          >
            <div className="p-2 bg-orange-500/20 rounded">
              <svg className="w-5 h-5 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 919-9" />
              </svg>
            </div>
            <span className="ml-3 font-medium text-gray-200">View Networks</span>
          </Link>
        </div>
      </div>
    </div>
  );
}


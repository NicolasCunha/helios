export interface Container {
  id: string;
  name: string;
  image: string;
  state: string;
  status: string;
  created: string;
  ports: Port[];
  mounts: Mount[];
  labels: Record<string, string>;
  stats?: ContainerStats;
}

export interface ContainerStats {
  cpu_percent: number;
  memory_usage: number;
  memory_limit: number;
  memory_percent: number;
  network_rx: number;
  network_tx: number;
  block_read: number;
  block_write: number;
}

export interface Port {
  IP?: string;
  PrivatePort: number;
  PublicPort?: number;
  Type: string;
}

export interface Mount {
  Type: string;
  Name?: string;
  Source: string;
  Destination: string;
  Driver?: string;
  Mode: string;
  RW: boolean;
  Propagation: string;
}

export interface Image {
  id: string;
  repo_tags: string[];
  repo_digests: string[];
  created: number;
  size: number;
  virtual_size: number;
  shared_size: number;
  labels: Record<string, string>;
  containers: number;
}

export interface ImageDetail extends Image {
  parent: string;
  comment: string;
  docker_version: string;
  author: string;
  architecture: string;
  os: string;
  exposed_ports: string[];
  env: string[];
  cmd: string[];
  entrypoint: string[];
  volumes: string[];
  working_dir: string;
  user: string;
  rootfs?: {
    type: string;
    layers: string[];
  };
}

export interface Volume {
  name: string;
  driver: string;
  mountpoint: string;
  created_at: string;
  labels: Record<string, string>;
  scope: string;
  options: Record<string, string>;
  usage_data?: {
    size: number;
    ref_count: number;
  };
}

export interface Network {
  id: string;
  name: string;
  driver: string;
  scope: string;
  internal: boolean;
  attachable: boolean;
  ingress: boolean;
  ipam: {
    driver: string;
    config: Array<{
      subnet?: string;
      gateway?: string;
    }>;
  };
  containers: Record<string, any>;
  options: Record<string, string>;
  labels: Record<string, string>;
  created?: string;
}

export interface HealthCheckLog {
  id: number;
  container_id: string;
  container_name: string;
  status: string;
  resource_cpu: number;
  resource_memory: number;
  resource_memory_limit: number;
  resource_network_rx: number;
  resource_network_tx: number;
  error_message?: string;
  checked_at: string;
}

export interface ActionLog {
  id: number;
  action_type: string;
  resource_type: string;
  resource_id: string;
  resource_name: string;
  success: boolean;
  error_message?: string;
  executed_at: string;
}

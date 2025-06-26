import { pocketbase } from '$lib/stores/pocketbase';
import { get } from 'svelte/store';

export interface PortScanRequest {
  client_id: string;
  target_ips?: string[];              // Manual IP list
  include_domains: boolean;           // Include IPs from discovered domains
  include_netblocks: boolean;         // Include IPs from netblocks
  ports?: string;                     // Custom ports (e.g., "80,443,8080-8090")
  top_ports?: string;                 // Top ports preset (100, 1000, full)
  exclude_ports?: string;             // Ports to exclude
  scan_type: string;                  // "SYN" or "CONNECT"
  rate?: number;                      // Packets per second
  threads?: number;                   // Worker threads
  timeout?: number;                   // Timeout in milliseconds
  retries?: number;                   // Number of retries
  host_discovery?: boolean;           // Enable host discovery
  exclude_cdn?: boolean;              // Skip full scans for CDN/WAF
  verify?: boolean;                   // Verify ports with TCP
  execution_mode: string;             // "local" or "cloud"
  cloud_provider?: string;            // Cloud provider for remote execution
  nmap_integration?: boolean;         // Run nmap for service detection
  nmap_command?: string;              // Custom nmap command
}

export interface PortScanResult {
  ip: string;
  port: number;
  protocol?: string;
  service?: string;
  state: string;
  host?: string;
  source: string; // "domains", "netblocks", "manual"
}

export interface PortScanJobResult {
  client_id: string;
  scan_id: string;
  start_time: string;
  end_time: string;
  duration: string;
  total_targets: number;
  total_ports: number;
  open_ports: number;
  target_ips: string[];
  results: PortScanResult[];
  stats: Record<string, number>;
  execution_mode: string;
  cloud_provider?: string;
  naabu_version?: string;
  error?: string;
}

export interface Port {
  id: string;
  scan_id: string;
  ip: string;
  port: number;
  protocol?: string;
  service?: string;
  state: string;
  host?: string;
  source: string;
  discovered_at: string;
  created: string;
  updated: string;
}

export interface PortScan {
  id: string;
  scan_id: string;
  start_time: string;
  end_time?: string;
  duration?: string;
  total_targets: number;
  total_ports: number;
  open_ports: number;
  execution_mode: string;
  cloud_provider?: string;
  naabu_version?: string;
  error?: string;
  stats?: Record<string, number>;
  target_ips?: string[];
  created: string;
  updated: string;
}

export interface PortStats {
  total_open_ports: number;
  unique_hosts: number;
  total_scans: number;
  top_ports: Array<{ port: number; count: number }>;
  service_breakdown: Record<string, number>;
  source_breakdown: Record<string, number>;
  latest_scan?: {
    scan_id: string;
    start_time: string;
    end_time?: string;
    duration?: string;
    total_targets: number;
    open_ports: number;
    execution_mode: string;
    stats?: Record<string, number>;
  };
  }

export class PortScanService {
  /**
   * Get authentication headers
   */
  private getAuthHeaders(): Record<string, string> {
    const pb = get(pocketbase);
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    
    if (pb.authStore.token) {
      headers['Authorization'] = `Bearer ${pb.authStore.token}`;
    }
    
    return headers;
  }

  /**
   * Start a new port scan
   */
  async startPortScan(request: PortScanRequest): Promise<{ success: boolean; result?: PortScanJobResult; message?: string }> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ports/scan`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Port scan failed:', error);
      throw error;
    }
  }

  /**
   * Get all ports for a client
   */
  async getPorts(clientId: string): Promise<{ success: boolean; ports: Port[]; total: number }> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ports?client_id=${clientId}`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get ports:', error);
      throw error;
    }
  }

  /**
   * Get port scan history for a client
   */
  async getPortScans(clientId: string): Promise<{ success: boolean; scans: PortScan[]; total: number }> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ports/scans?client_id=${clientId}`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get port scans:', error);
      throw error;
    }
  }

  /**
   * Get port statistics for a client
   */
  async getPortStats(clientId: string): Promise<{ success: boolean; stats: PortStats }> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ports/stats?client_id=${clientId}`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get port stats:', error);
      throw error;
    }
  }

  /**
   * Get the progress of a running port scan
   */
  async getScanProgress(scanId: string): Promise<{ success: boolean; progress?: any; message?: string }> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ports/scan/${scanId}/progress`, {
        method: 'GET',
        headers: this.getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Failed to get scan progress:', error);
      throw error;
    }
  }

  /**
   * Export ports to CSV format
   */
  exportPortsToCSV(ports: Port[]): string {
    const headers = ['IP', 'Port', 'Protocol', 'Service', 'State', 'Host', 'Source', 'Discovered At'];
    const rows = ports.map(port => [
      port.ip,
      port.port.toString(),
      port.protocol || '',
      port.service || '',
      port.state,
      port.host || '',
      port.source,
      new Date(port.discovered_at).toLocaleString()
    ]);

    return [headers, ...rows]
      .map(row => row.map(cell => `"${cell}"`).join(','))
      .join('\n');
  }

  /**
   * Get common port presets
   */
  getPortPresets(): Array<{ value: string; label: string; description: string }> {
    return [
      { value: '100', label: 'Top 100', description: 'Most common 100 ports' },
      { value: '1000', label: 'Top 1000', description: 'Most common 1000 ports' },
      { value: 'full', label: 'Full Range', description: 'All 65535 ports (slower)' },
      { value: 'custom', label: 'Custom', description: 'Specify custom ports' }
    ];
  }

  /**
   * Validate port specification
   */
  validatePorts(ports: string): { valid: boolean; error?: string } {
    if (!ports || ports.trim() === '') {
      return { valid: false, error: 'Ports cannot be empty' };
    }

    const portRegex = /^(\d+(-\d+)?)(,\s*\d+(-\d+)?)*$/;
    if (!portRegex.test(ports.trim())) {
      return { valid: false, error: 'Invalid port format. Use: 80,443,8080-8090' };
    }

    // Check individual ports
    const parts = ports.split(',');
    for (const part of parts) {
      const trimmed = part.trim();
      if (trimmed.includes('-')) {
        const [start, end] = trimmed.split('-').map(p => parseInt(p.trim()));
        if (start < 1 || start > 65535 || end < 1 || end > 65535 || start >= end) {
          return { valid: false, error: `Invalid port range: ${trimmed}` };
        }
      } else {
        const port = parseInt(trimmed);
        if (port < 1 || port > 65535) {
          return { valid: false, error: `Invalid port: ${port}` };
        }
      }
    }

    return { valid: true };
  }

  /**
   * Get common port services
   */
  getCommonPorts(): Array<{ port: number; service: string; description: string }> {
    return [
      { port: 21, service: 'FTP', description: 'File Transfer Protocol' },
      { port: 22, service: 'SSH', description: 'Secure Shell' },
      { port: 23, service: 'Telnet', description: 'Telnet protocol' },
      { port: 25, service: 'SMTP', description: 'Simple Mail Transfer Protocol' },
      { port: 53, service: 'DNS', description: 'Domain Name System' },
      { port: 80, service: 'HTTP', description: 'HyperText Transfer Protocol' },
      { port: 110, service: 'POP3', description: 'Post Office Protocol v3' },
      { port: 143, service: 'IMAP', description: 'Internet Message Access Protocol' },
      { port: 443, service: 'HTTPS', description: 'HTTP Secure' },
      { port: 993, service: 'IMAPS', description: 'IMAP over SSL' },
      { port: 995, service: 'POP3S', description: 'POP3 over SSL' },
      { port: 3389, service: 'RDP', description: 'Remote Desktop Protocol' },
      { port: 5432, service: 'PostgreSQL', description: 'PostgreSQL Database' },
      { port: 3306, service: 'MySQL', description: 'MySQL Database' },
      { port: 27017, service: 'MongoDB', description: 'MongoDB Database' },
      { port: 6379, service: 'Redis', description: 'Redis Database' },
      { port: 8080, service: 'HTTP-Alt', description: 'Alternative HTTP port' },
      { port: 8443, service: 'HTTPS-Alt', description: 'Alternative HTTPS port' },
      { port: 9200, service: 'Elasticsearch', description: 'Elasticsearch REST API' },
      { port: 5601, service: 'Kibana', description: 'Kibana Dashboard' }
    ];
  }

  /**
   * Estimate scan duration based on parameters
   */
  estimateScanDuration(targetCount: number, portCount: number, scanType: string, rate: number = 1000): string {
    const baseTimePerPort = scanType === 'SYN' ? 0.001 : 0.01; // seconds
    const totalChecks = targetCount * portCount;
    const adjustedRate = Math.min(rate, 1000); // Cap at reasonable rate
    const estimatedSeconds = (totalChecks * baseTimePerPort) * (1000 / adjustedRate);
    
    if (estimatedSeconds < 60) {
      return `~${Math.ceil(estimatedSeconds)} seconds`;
    } else if (estimatedSeconds < 3600) {
      return `~${Math.ceil(estimatedSeconds / 60)} minutes`;
    } else {
      return `~${Math.ceil(estimatedSeconds / 3600)} hours`;
    }
  }

  /**
   * Format port scan results for display
   */
  formatPortResult(port: Port): string {
    const service = port.service ? ` (${port.service})` : '';
    return `${port.ip}:${port.port}${service}`;
  }

  /**
   * Group ports by IP address
   */
  groupPortsByIP(ports: Port[]): Record<string, Port[]> {
    return ports.reduce((groups, port) => {
      if (!groups[port.ip]) {
        groups[port.ip] = [];
      }
      groups[port.ip].push(port);
      return groups;
    }, {} as Record<string, Port[]>);
  }

  /**
   * Get scan type options
   */
  getScanTypes(): Array<{ value: string; label: string; description: string; requiresRoot?: boolean }> {
    return [
      { 
        value: 'CONNECT', 
        label: 'TCP Connect', 
        description: 'Full TCP connection (slower, more reliable, no root required)' 
      },
      { 
        value: 'SYN', 
        label: 'SYN Scan', 
        description: 'Half-open SYN scan (faster, stealthier, requires root)', 
        requiresRoot: true 
      }
    ];
  }

  /**
   * Get execution mode options
   */
  getExecutionModes(): Array<{ value: string; label: string; description: string; recommended?: boolean }> {
    return [
      { 
        value: 'local', 
        label: 'Local Execution', 
        description: 'Run scan on this machine (requires naabu installed)' 
      },
      { 
        value: 'cloud', 
        label: 'Cloud Execution', 
        description: 'Run scan on a cloud instance (recommended for large scans)', 
        recommended: true 
      }
    ];
  }

  /**
   * Get cloud provider options
   */
  getCloudProviders(): Array<{ value: string; label: string; description: string }> {
    return [
      { value: 'aws', label: 'Amazon Web Services', description: 'AWS EC2 instances' },
      { value: 'gcp', label: 'Google Cloud Platform', description: 'GCP Compute Engine' },
      { value: 'azure', label: 'Microsoft Azure', description: 'Azure Virtual Machines' },
      { value: 'digitalocean', label: 'DigitalOcean', description: 'DigitalOcean Droplets' }
    ];
  }
} 
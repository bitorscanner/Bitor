import { pocketbase } from '$lib/stores/pocketbase';
import { get } from 'svelte/store';

export interface URLScanRequest {
  client_id: string;
  target_urls?: string[];           // Manual URL list
  include_ports: boolean;           // Include URLs from port scan results
  include_domains: boolean;         // Include URLs from discovered domains
  include_subdomains: boolean;      // Include URLs from subdomains
  schemes?: string[];               // URL schemes (http, https)
  ports?: number[];                 // Specific ports to scan
  only_web_ports?: boolean;         // Only scan common web ports
  threads?: number;                 // Worker threads
  timeout?: number;                 // Request timeout in seconds
  retries?: number;                 // Number of retries
  follow_redirects?: boolean;       // Follow HTTP redirects
  tech_detection?: boolean;         // Enable technology detection
  status_code?: boolean;            // Include status codes
  content_length?: boolean;         // Include content length
  response_time?: boolean;          // Include response time
  match_regex?: string;             // Custom regex to match
  filter_regex?: string;            // Custom regex to filter
  output_all?: boolean;             // Output all URLs (even failed)
  silent?: boolean;                 // Silent mode
  execution_mode: string;           // "local" or "cloud"
  cloud_provider?: string;          // Cloud provider for remote execution
}

export interface URLScanResult {
  id: string;
  url: string;
  scheme: string;
  host: string;
  port: number;
  path: string;
  status_code: number;
  content_length: number;
  response_time: string;
  title?: string;
  server?: string;
  content_type?: string;
  final_url?: string;
  source: string; // "ports", "domains", "subdomains", "manual"
  ip?: string;
  cdn?: string;
  webserver?: string;
  technologies?: string[];
  hash?: Record<string, string>;
  cnames?: string[];
  chain?: string[];
  scan_id: string;
  discovered_at: string;
  created: string;
  updated: string;
}

export interface URLScanProgress {
  scan_id: string;
  client_id: string;
  status: string;           // "running", "completed", "failed"
  progress: number;         // 0-100
  message: string;
  start_time: string;
  end_time?: string;
  total_targets: number;
  live_urls: number;
  error?: string;
}

export interface URLScanSummary {
  id: string;
  scan_id: string;
  start_time: string;
  end_time: string;
  duration: string;
  total_targets: number;
  live_urls: number;
  execution_mode: string;
  cloud_provider?: string;
  httpx_version?: string;
  stats?: Record<string, number>;
  created: string;
  updated: string;
}

export interface URLScanStats {
  total_urls: number;
  unique_hosts: number;
  total_scans: number;
  status_breakdown: Record<string, number>;
  scheme_breakdown: Record<string, number>;
  top_technologies: Array<{ technology: string; count: number }>;
  source_breakdown: Record<string, number>;
  latest_scan?: URLScanSummary;
}

export interface URLScanResponse {
  success: boolean;
  scan_id?: string;
  message?: string;
  error?: string;
}

export interface URLScanProgressResponse {
  success: boolean;
  progress?: URLScanProgress;
  message?: string;
}

export interface URLScanResultsResponse {
  success: boolean;
  urls: URLScanResult[];
  total: number;
  message?: string;
}

export interface URLScanSummaryResponse {
  success: boolean;
  scans: URLScanSummary[];
  total: number;
  message?: string;
}

export interface URLScanStatsResponse {
  success: boolean;
  stats: URLScanStats;
  message?: string;
}

class URLScanService {
  private baseUrl: string;

  constructor() {
    this.baseUrl = import.meta.env.VITE_API_BASE_URL || '';
  }

  private getAuthHeaders() {
    const pb = get(pocketbase);
    return {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${pb.authStore.token}`
    };
  }

  async startScan(request: URLScanRequest): Promise<URLScanResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/attack-surface/urls/scan`, {
        method: 'POST',
        headers: this.getAuthHeaders(),
        body: JSON.stringify(request)
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error starting URL scan:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  async getScanProgress(scanId: string): Promise<URLScanProgressResponse> {
    try {
      const response = await fetch(`${this.baseUrl}/api/attack-surface/urls/scan/${scanId}/progress`, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error getting scan progress:', error);
      return {
        success: false,
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  async getURLs(clientId: string, host?: string): Promise<URLScanResultsResponse> {
    try {
      const params = new URLSearchParams({ client_id: clientId });
      if (host) {
        params.append('host', host);
      }

      const response = await fetch(`${this.baseUrl}/api/attack-surface/urls?${params}`, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error getting URLs:', error);
      return {
        success: false,
        urls: [],
        total: 0,
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  async getScans(clientId: string): Promise<URLScanSummaryResponse> {
    try {
      const params = new URLSearchParams({ client_id: clientId });

      const response = await fetch(`${this.baseUrl}/api/attack-surface/urls/scans?${params}`, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error getting URL scans:', error);
      return {
        success: false,
        scans: [],
        total: 0,
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  async getStats(clientId: string): Promise<URLScanStatsResponse> {
    try {
      const params = new URLSearchParams({ client_id: clientId });

      const response = await fetch(`${this.baseUrl}/api/attack-surface/urls/stats?${params}`, {
        method: 'GET',
        headers: this.getAuthHeaders()
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      console.error('Error getting URL stats:', error);
      return {
        success: false,
        stats: {
          total_urls: 0,
          unique_hosts: 0,
          total_scans: 0,
          status_breakdown: {},
          scheme_breakdown: {},
          top_technologies: [],
          source_breakdown: {}
        },
        message: error instanceof Error ? error.message : 'Unknown error occurred'
      };
    }
  }

  // Utility method to get common web ports
  getCommonWebPorts(): number[] {
    return [80, 443, 8080, 8443, 8000, 8888, 9000, 9001, 3000, 5000];
  }

  // Utility method to get default schemes
  getDefaultSchemes(): string[] {
    return ['http', 'https'];
  }

  // Utility method to get execution modes
  getExecutionModes(): Array<{ value: string; label: string }> {
    return [
      { value: 'local', label: 'Local' },
      { value: 'cloud', label: 'Cloud' }
    ];
  }

  // Utility method to get cloud providers
  getCloudProviders(): Array<{ value: string; label: string }> {
    return [
      { value: 'aws', label: 'Amazon Web Services (AWS)' },
      { value: 'gcp', label: 'Google Cloud Platform (GCP)' },
      { value: 'azure', label: 'Microsoft Azure' },
      { value: 'digitalocean', label: 'DigitalOcean' }
    ];
  }

  // Utility method to validate URL
  isValidURL(url: string): boolean {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  }

  // Utility method to get status code color class
  getStatusCodeColor(statusCode: number): string {
    if (statusCode >= 200 && statusCode < 300) {
      return 'text-green-600';
    } else if (statusCode >= 300 && statusCode < 400) {
      return 'text-yellow-600';
    } else if (statusCode >= 400 && statusCode < 500) {
      return 'text-red-600';
    } else if (statusCode >= 500) {
      return 'text-red-800';
    }
    return 'text-gray-600';
  }

  // Utility method to format response time
  formatResponseTime(responseTime: string): string {
    if (!responseTime) return 'N/A';
    
    // Parse duration string (e.g., "1.234567ms", "2.5s")
    const match = responseTime.match(/^(\d+(?:\.\d+)?)(ms|s|μs|ns)$/);
    if (!match) return responseTime;
    
    const value = parseFloat(match[1]);
    const unit = match[2];
    
    switch (unit) {
      case 's':
        return `${(value * 1000).toFixed(0)}ms`;
      case 'μs':
        return `${(value / 1000).toFixed(2)}ms`;
      case 'ns':
        return `${(value / 1000000).toFixed(3)}ms`;
      case 'ms':
      default:
        return `${value.toFixed(0)}ms`;
    }
  }

  // Utility method to format content length
  formatContentLength(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B';
    
    const units = ['B', 'KB', 'MB', 'GB'];
    let size = bytes;
    let unitIndex = 0;
    
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    
    return `${size.toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`;
  }
}

export const urlScanService = new URLScanService(); 
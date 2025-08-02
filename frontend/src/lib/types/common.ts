// Common type definitions for the application

// Error types
export interface AppError extends Error {
  message: string;
  status?: number;
  code?: string;
}

// API Response types
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

// Progress types
export interface ScanProgress {
  percentage: number;
  status: 'running' | 'completed' | 'failed' | 'pending';
  currentTarget?: string;
  totalTargets?: number;
  completedTargets?: number;
  startTime?: string;
  estimatedTimeRemaining?: string;
}

// Message types
export interface UserMessage {
  id: string;
  userId: string;
  title: string;
  content: string;
  type: 'info' | 'warning' | 'error' | 'success';
  read: boolean;
  created: string;
  updated: string;
  collectionId?: string;
  collectionName?: string;
}

// Settings types
export interface AppSettings {
  theme?: 'light' | 'dark' | 'auto';
  notifications?: boolean;
  autoRefresh?: boolean;
  refreshInterval?: number;
  language?: string;
  timezone?: string;
}

// PocketBase types
export interface PocketBaseOptions {
  fetch?: typeof fetch;
  headers?: Record<string, string>;
  body?: unknown;
  query?: Record<string, unknown>;
}
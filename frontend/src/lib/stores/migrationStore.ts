import { writable } from 'svelte/store';

export interface MigrationState {
  isProcessing: boolean;
  totalCount: number;
  processedCount: number;
  progress: number;
  error: string;
  currentStatus: string;
}

const initialState: MigrationState = {
  isProcessing: false,
  totalCount: 0,
  processedCount: 0,
  progress: 0,
  error: '',
  currentStatus: ''
};

export const migrationStore = writable<MigrationState>(initialState);

export const resetMigrationStore = () => {
  migrationStore.set(initialState);
}; 
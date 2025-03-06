<!-- Migration Progress Component -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { Button, Progressbar } from 'flowbite-svelte';
  import { migrationStore } from '@lib/stores/migrationStore';
  import type { MigrationState } from '@lib/stores/migrationStore';

  let pollInterval: number;

  // Subscribe to the store
  $: ({ isProcessing, totalCount, processedCount, progress, error, currentStatus } = $migrationStore);

  async function checkMigrationStatus() {
    try {
      // Check if we have a valid admin token
      if (!$pocketbase.authStore.isValid || !$pocketbase.authStore.isAdmin) {
        migrationStore.update((s: MigrationState) => ({
          ...s,
          error: 'Admin access required.',
          isProcessing: false
        }));
        return;
      }

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/findings/migration-status`, {
        method: 'GET',
        headers: {
          'Authorization': `Bearer ${$pocketbase.authStore.token}`
        }
      });

      if (!response.ok) {
        throw new Error('Failed to fetch migration status');
      }

      const data = await response.json();
      
      // Update store with all fields from response
      migrationStore.update((s: MigrationState) => ({
        isProcessing: data.isProcessing,
        totalCount: data.totalCount,
        processedCount: data.processedCount,
        progress: data.progress,
        error: data.error || '',
        currentStatus: data.currentStatus || (data.totalCount === 0 && data.isProcessing ? 'Calculating total findings to process...' : '')
      }));

      // If we're still processing but don't have a total count yet, check more frequently
      if (data.isProcessing && data.totalCount === 0) {
        await new Promise(resolve => setTimeout(resolve, 500)); // Check every 500ms during counting
        await checkMigrationStatus();
      }
    } catch (err: any) {
      console.error('Error checking migration status:', err);
      migrationStore.update((s: MigrationState) => ({
        ...s,
        error: err?.message || 'Failed to check migration status',
        isProcessing: false
      }));
    }
  }

  // Process a single batch
  async function processBatch(offset: number, limit: number): Promise<number> {
    try {
      const result = await $pocketbase.send('/api/findings/migrate-batch', {
        method: 'POST',
        params: {
          offset: offset.toString(),
          limit: limit.toString(),
          migrationOnly: 'true'
        }
      });
      return result.processedCount;
    } catch (err: any) {
      console.error('Batch processing error:', err);
      migrationStore.update((s: MigrationState) => ({
        ...s,
        error: err.message || 'Unknown error occurred',
        currentStatus: `Error processing batch ${offset}-${offset + limit}, continuing...`
      }));
      return 0;
    }
  }

  async function startMigration() {
    try {
      // Check if we have a valid admin token
      if (!$pocketbase.authStore.isValid || !$pocketbase.authStore.isAdmin) {
        migrationStore.update((s: MigrationState) => ({
          ...s,
          error: 'Admin access required.',
          isProcessing: false
        }));
        return;
      }

      // Reset the store state
      migrationStore.set({
        isProcessing: true,
        totalCount: 0,
        processedCount: 0,
        progress: 0,
        error: '',
        currentStatus: 'Calculating total findings to process...'
      });

      // Start the migration with initial batch
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/findings/migrate-batch?offset=0&limit=25&migrationOnly=true`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${$pocketbase.authStore.token}`
        }
      });

      if (!response.ok) {
        throw new Error('Failed to start migration');
      }

      // Let the polling handle the rest
      await checkMigrationStatus();

    } catch (err: any) {
      console.error('Migration error:', err);
      migrationStore.update((s: MigrationState) => ({
        ...s,
        error: err.message || 'Failed to start migration',
        isProcessing: false
      }));
    }
  }

  onMount(async () => {
    // Check initial status
    await checkMigrationStatus();
    
    // Start polling
    pollInterval = window.setInterval(checkMigrationStatus, 2000);
  });

  onDestroy(() => {
    if (pollInterval) {
      clearInterval(pollInterval);
    }
  });
</script>

<div class="space-y-4">
  {#if error}
    <div class="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded" role="alert">
      <p class="font-bold">Error</p>
      <p>{error}</p>
    </div>
  {/if}

  <div class="bg-white dark:bg-gray-800 rounded-lg p-4 shadow">
    <h3 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">Migration Status</h3>
    
    {#if isProcessing}
      <div class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">{currentStatus || 'Processing...'}</p>
        <div class="flex items-center justify-between mb-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
            Progress: {processedCount} of {totalCount} findings
          </span>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-200">
            {progress}%
          </span>
        </div>
        <Progressbar 
          progress={progress}
          size="h-4"
          color={error ? "red" : "blue"}
        />
      </div>
    {:else if processedCount > 0}
      <div class="text-green-600 dark:text-green-400">
        <p>{currentStatus || 'Migration completed'}</p>
        <p class="mt-1">Processed {processedCount} findings.</p>
      </div>
    {:else}
      <div class="space-y-3">
        <p class="text-sm text-gray-600 dark:text-gray-300">
          Click the button below to start the findings migration process.
          This will generate hashes and create history records for all findings.
          The process will continue even if you navigate away from this page.
        </p>
        <Button color="blue" on:click={startMigration}>
          Start Migration
        </Button>
      </div>
    {/if}
  </div>
</div>

<style>
  /* Add any custom styles here */
</style> 
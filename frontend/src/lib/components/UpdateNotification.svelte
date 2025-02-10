<script lang="ts">
  import { onMount } from 'svelte';
  import { Alert, Button } from 'flowbite-svelte';
  import { InfoCircleSolid } from 'flowbite-svelte-icons';
  import { pocketbase } from '$lib/stores/pocketbase';

  let currentVersion: string = '';
  let updateAvailable: boolean = false;
  let isDocker: boolean = false;
  let latestVersion: string = '';
  let releaseNotes: string = '';
  let loading = false;
  let error = '';
  let showNotification: boolean = false;

  interface VersionResponse {
    message: string;
    current_version: string;
    update_available: boolean;
    is_docker: boolean;
    latest_version: string;
    release_notes?: string;
  }

  async function checkForUpdates(): Promise<void> {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/version/check`);
      const data: VersionResponse = await response.json();
      
      if (!response.ok) {
        throw new Error(data.message);
      }

      currentVersion = data.current_version;
      
      // Don't show update notification for development version
      if (currentVersion === "development") {
        updateAvailable = false;
        return;
      }

      updateAvailable = data.update_available;
      isDocker = data.is_docker;
      latestVersion = data.latest_version;
      if (data.release_notes) {
        releaseNotes = data.release_notes;
      }
      showNotification = true; // Show notification when update is available
    } catch (error) {
      console.error('Error checking for updates:', error);
    }
  }

  async function performUpdate() {
    if (isDocker) {
      return; // Docker updates are handled differently
    }

    loading = true;
    error = '';

    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/version/update`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${$pocketbase.authStore.token}`,
        },
      });

      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.message || 'Failed to start update');
      }

      // Show message about update in progress
      alert('Update process started. The application will restart automatically.');
    } catch (err: any) {
      error = err.message || 'Failed to perform update';
      console.error('Update error:', err);
    } finally {
      loading = false;
    }
  }

  function closeNotification() {
    showNotification = false;
  }

  onMount(() => {
    checkForUpdates();
    // Check for updates every hour
    const interval = setInterval(checkForUpdates, 3600000);
    return () => clearInterval(interval);
  });
</script>

{#if updateAvailable && showNotification && currentVersion !== "development"}
  <Alert color="blue" class="mb-4">
    <div class="flex items-center justify-between w-full">
      <div class="flex items-center gap-2">
        <InfoCircleSolid class="w-4 h-4" />
        <div>
          <span class="font-medium">Update Available!</span>
          <p class="text-sm">
            Version {latestVersion} is available (current: {currentVersion})
          </p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        {#if isDocker}
          <div class="text-sm">
            <span class="block">Pull latest Docker image:</span>
            <code class="bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded text-xs">docker pull orbitscanner/orbit:latest</code>
          </div>
        {:else}
          <Button
            size="xs"
            color="blue"
            disabled={loading}
            on:click={performUpdate}
          >
            {loading ? 'Updating...' : 'Update Now'}
          </Button>
        {/if}
        <button
          type="button"
          class="ml-2 text-gray-500 hover:text-gray-900 dark:hover:text-white"
          on:click={closeNotification}
        >
          <span class="sr-only">Close</span>
          <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
            <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"></path>
          </svg>
        </button>
      </div>
    </div>
    {#if error}
      <p class="text-red-500 mt-2 text-sm">{error}</p>
    {/if}
  </Alert>
{/if} 
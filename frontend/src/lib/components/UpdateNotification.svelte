<script lang="ts">
  import { onMount } from 'svelte';
  import { Alert } from 'flowbite-svelte';
  import { InfoCircleSolid } from 'flowbite-svelte-icons';
  import { pocketbase } from '@lib/stores/pocketbase';

  let hasUpdate = false;
  let currentVersion = '';
  let latestVersion = '';

  onMount(async () => {
    try {
      const response = await fetch('/api/version/check', {
        headers: {
          'Authorization': `Bearer ${$pocketbase.authStore.token}`
        }
      });
      const data = await response.json();
      hasUpdate = data.hasUpdate;
      currentVersion = data.currentVersion;
      latestVersion = data.latestVersion;
    } catch (error) {
      console.error('Error checking for updates:', error);
    }
  });
</script>

{#if hasUpdate}
  <Alert color="blue" class="mb-4">
    <svelte:fragment slot="icon">
      <InfoCircleSolid class="w-5 h-5" />
    </svelte:fragment>
    <span class="font-medium">Update Available!</span>
    <p>
      A new version of Bitor ({latestVersion}) is available. You are currently running version {currentVersion}.
      Please check the documentation for update instructions.
    </p>
  </Alert>
{/if} 
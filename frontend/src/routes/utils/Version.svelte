<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';

  let version = 'unknown';

  onMount(async () => {
    try {
      const token = $pocketbase.authStore.token;

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/version`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (response.ok) {
        const data = await response.json();
        version = data.version;
      } else {
        console.error('Failed to fetch version');
      }
    } catch (error) {
      console.error('Error fetching version:', error);
    }
  });
</script>

<footer class="text-center text-gray-500 text-sm mt-4">
  <p>Version: {version}</p>
</footer>

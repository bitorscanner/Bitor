<!-- frontend/src/routes/utils/dashboard/+page.svelte -->
<script lang="ts">
  import DraggableDashboard from './DraggableDashboard.svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';

  onMount(async () => {
    // Check if user is authenticated
    if (!$pocketbase.authStore.isValid) {
      goto('/login');
    }
  });
</script>

<div class="container mx-auto">
  <DraggableDashboard />
</div>

<style>
  .container {
    max-width: 100%;
    padding: 1rem;
  }
</style> 
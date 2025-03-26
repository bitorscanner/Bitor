<script lang="ts">
  import { Checkbox } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { createEventDispatcher } from 'svelte';

  // Whether to show only the current user's findings
  export let checked = false;

  // Get the current user ID
  let currentUserId = $pocketbase.authStore.model?.id ?? '';

  // Create a dispatcher to notify parent component of changes
  const dispatch = createEventDispatcher<{
    change: { checked: boolean; userId: string };
  }>();

  // Handle changes to the checkbox
  function handleChange() {
    dispatch('change', { checked, userId: currentUserId });
  }
</script>

<div class="flex flex-col justify-end mb-1">
  <Checkbox bind:checked on:change={handleChange} id="myFindingsFilter">
    <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
      Show My Findings Only
    </span>
  </Checkbox>
</div> 
<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { Select, Toast } from 'flowbite-svelte';
  import { slide } from 'svelte/transition';
  import { CheckCircleSolid, ExclamationCircleSolid } from 'flowbite-svelte-icons';

  export let userId: string;
  export let currentGroupId: string = '';
  export let disabled: boolean = false;
  export let onGroupChange: (groupId: string) => void = () => {};

  let groups = [];
  let loading = true;
  let showSuccessToast = false;
  let showErrorToast = false;
  let errorMessage = '';

  onMount(async () => {
    await loadGroups();
    if (typeof currentGroupId === 'object' && currentGroupId?.id) {
      currentGroupId = currentGroupId.id;
    }
  });

  async function loadGroups() {
    try {
      loading = true;
      const records = await $pocketbase.collection('groups').getFullList();
      groups = records;
    } catch (error) {
      errorMessage = 'Error loading groups';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      console.error('Error loading groups:', error);
    } finally {
      loading = false;
    }
  }

  async function handleGroupChange(event) {
    const newGroupId = event.target.value;
    try {
      await $pocketbase.collection('users').update(userId, {
        group: newGroupId
      });
      currentGroupId = newGroupId;
      onGroupChange(newGroupId);
      showSuccessToast = true;
      setTimeout(() => showSuccessToast = false, 3000);
    } catch (error) {
      errorMessage = 'Error updating group';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      console.error('Error updating group:', error);
    }
  }
</script>

{#if showSuccessToast}
  <Toast transition={slide} color="green" class="mb-4">
    <CheckCircleSolid slot="icon" class="w-5 h-5" />
    Group updated successfully
  </Toast>
{/if}

{#if showErrorToast}
  <Toast transition={slide} color="red" class="mb-4">
    <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
    {errorMessage}
  </Toast>
{/if}

{#if loading}
  <div class="text-sm text-gray-500">Loading groups...</div>
{:else}
  <Select 
    class="w-full"
    value={currentGroupId}
    on:change={handleGroupChange}
    {disabled}
  >
    <option value="" disabled>Select a group</option>
    {#each groups as group}
      <option value={group.id}>{group.name}</option>
    {/each}
  </Select>
{/if} 
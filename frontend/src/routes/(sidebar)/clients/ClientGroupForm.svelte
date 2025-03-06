<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { Modal, Input, Button, Label, Checkbox } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';

  export let open = false;
  export let size = 'lg';

  let groupName = '';
  let selectedClients = new Set<string>();
  let clients = [];
  let clientGroups = [];

  const dispatch = createEventDispatcher();

  onMount(async () => {
    await fetchClients();
    await fetchClientGroups();
  });

  async function fetchClients() {
    try {
      const result = await $pocketbase.collection('clients').getList(1, 50);
      clients = result.items.map((client) => ({
        id: client.id,
        name: client.name,
      }));
    } catch (error) {
      console.error('Error fetching clients:', error);
    }
  }

  async function fetchClientGroups() {
    try {
      const result = await $pocketbase.collection('client_groups').getList();
      clientGroups = result.items;
    } catch (error) {
      console.error('Error fetching client groups:', error);
    }
  }

  async function deleteClientGroup(groupId: string) {
    try {
      await $pocketbase.collection('client_groups').delete(groupId);
      await fetchClientGroups(); // Refresh the list after deletion
    } catch (error) {
      console.error('Error deleting client group:', error);
    }
  }

  function toggleClientSelection(clientId: string) {
    if (selectedClients.has(clientId)) {
      selectedClients.delete(clientId);
    } else {
      selectedClients.add(clientId);
    }
    // Reassign to trigger reactivity
    selectedClients = new Set(selectedClients);
  }

  function handleSave() {
    const groupData = {
      name: groupName,
    };

    // Dispatch both group data and selected client IDs
    dispatch('save', { groupData, selectedClients: Array.from(selectedClients) });
    open = false;
  }
</script>

<Modal bind:open size={size} title="Add Client Group">
  <form on:submit|preventDefault={handleSave} class="space-y-6">
    <Label>
      <span>Group Name</span>
      <Input bind:value={groupName} placeholder="Enter group name" required />
    </Label>

    <div>
      <h3 class="text-lg font-medium">Select Clients</h3>
      {#each clients as client}
        <div class="flex items-center mt-2">
          <Checkbox
            checked={selectedClients.has(client.id)}
            on:change={() => toggleClientSelection(client.id)}
          />
          <span class="ml-2">{client.name}</span>
        </div>
      {/each}
    </div>

    <Button type="submit" class="w-full">Save Group</Button>
  </form>

  <div class="mt-6">
    <h3 class="text-lg font-medium">Existing Client Groups</h3>
    <ul class="space-y-2">
      {#each clientGroups as group}
        <li class="flex justify-between items-center">
          <span>{group.name}</span>
          <Button color="red" size="sm" on:click={() => deleteClientGroup(group.id)}>Delete</Button>
        </li>
      {/each}
    </ul>
  </div>
</Modal> 
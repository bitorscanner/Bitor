<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Button,
    Input,
    Label,
    Modal,
    Table,
    TableBody,
    TableBodyCell,
    TableBodyRow,
    TableHead,
    TableHeadCell,
    Heading
  } from 'flowbite-svelte';
  import Card from '@utils/Card.svelte';
  import { pocketbase } from '@lib/stores/pocketbase';

  let servers = [];
  let newServer = { name: '', url: '', token: '' };
  let showModal = false;

  async function fetchServers() {
    try {
      const result = await $pocketbase.collection('nuclei_interact').getList();
      servers = result.items;
    } catch (error) {
      console.error('Error fetching servers:', error);
    }
  }

  onMount(() => {
    fetchServers();
  });

  async function addServer() {
    try {
      await $pocketbase.collection('nuclei_interact').create(newServer);
      fetchServers();
      newServer = { name: '', url: '', token: '' };
      showModal = false;
    } catch (error) {
      console.error('Error adding server:', error);
    }
  }

  async function removeServer(id: string) {
    try {
      await $pocketbase.collection('nuclei_interact').delete(id);
      fetchServers();
    } catch (error) {
      console.error('Error removing server:', error);
    }
  }
</script>

<main class="p-4">
  <Card class="w-full">
    <div class="space-y-8">
      <!-- Heading for Interact Servers -->
      <Heading tag="h2" class="text-xl font-semibold mb-4">Interact Servers</Heading>

      <!-- Table of Interact Servers -->
      <div>
        <Table class="w-full border border-gray-200 dark:border-gray-700">
          <TableHead class="bg-gray-100 dark:bg-gray-700">
            {#each ['Name', 'URL', 'Created', 'Actions'] as title}
              <TableHeadCell class="ps-4 font-normal">{title}</TableHeadCell>
            {/each}
          </TableHead>
          <TableBody>
            {#each servers as server}
              <TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
                <TableBodyCell class="p-4">{server.name}</TableBodyCell>
                <TableBodyCell class="p-4">{server.url}</TableBodyCell>
                <TableBodyCell class="p-4">{new Date(server.created).toLocaleDateString()}</TableBodyCell>
                <TableBodyCell class="space-x-2">
                  <Button size="sm" color="failure" on:click={() => removeServer(server.id)}>Remove</Button>
                </TableBodyCell>
              </TableBodyRow>
            {/each}
          </TableBody>
        </Table>
      </div>

      <!-- Add Server Button -->
      <Button class="mt-4" on:click={() => showModal = true}>Add Server</Button>

      <!-- Modal for Adding a New Server -->
      {#if showModal}
        <Modal bind:open={showModal}>
          <div slot="header" class="text-lg font-semibold">
            Add New Server
          </div>
          <div class="p-4 space-y-4">
            <Label for="name">Name</Label>
            <Input id="name" bind:value={newServer.name} placeholder="Server Name" required />

            <Label for="url">URL</Label>
            <Input id="url" bind:value={newServer.url} placeholder="Server URL" required />

            <Label for="token">Token</Label>
            <Input
              id="token"
              bind:value={newServer.token}
              type="password"
              placeholder="Token"
            />

            <div class="flex justify-end space-x-4 mt-4">
              <Button color="success" on:click={addServer}>Add Server</Button>
              <Button color="gray" on:click={() => (showModal = false)}>Cancel</Button>
            </div>
          </div>
        </Modal>
      {/if}
    </div>
  </Card>
</main>

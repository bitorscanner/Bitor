<script lang="ts">
    import {
      Breadcrumb,
      BreadcrumbItem,
      Heading,
      Table,
      TableBody,
      TableBodyCell,
      TableBodyRow,
      TableHead,
      TableHeadCell,
      Checkbox,
      Button,
      Select,
    } from 'flowbite-svelte';
    import { onMount } from 'svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import ClientForm from './ClientForm.svelte';
    import Delete from './Delete.svelte';
    import MetaTag from '@utils/MetaTag.svelte';
    import ClientGroupForm from './ClientGroupForm.svelte';
  
    interface Client {
      id: string;
      name: string;
      logo?: string;
      hidden_name: string;
      group?: ClientGroup | null;
      homepage?: string;
      favicon?: string;
    }
  
    interface ClientData {
      name: string;
      hidden_name: string;
      logo?: File;
      group?: string;
      homepage?: string;
    }
  
    interface ClientGroupEvent {
      detail: {
        groupData: any;
        selectedClients: Set<string>;
      }
    }
  
    interface ClientSaveEvent {
      detail: ClientData;
    }
  
    interface ClientGroup {
      id: string;
      name: string;
    }
  
    const path: string = '/clients';
    const description: string = 'Nuclei clients management';
    const title: string = 'Nuclei Clients';
    const subtitle: string = 'Manage your nuclei clients';
  
    let currentClientData: Client | null = null;
    let clients: Client[] = [];
    let showClientModal = false;
    let showDeleteClientModal = false;
    let currentClientId = '';
    let modalMode: 'add' | 'edit' = 'add';
    let showClientGroupModal = false;
    let selectedClients = new Set<string>();
    let selectedGroupId = '';
    let clientGroups: any[] = [];
  
    // Reactive statement to check if at least one client is selected
    $: multipleClientsSelected = selectedClients.size >= 1;
  
    onMount(() => {
      fetchClients();
      fetchClientGroups();
    });
  
    async function fetchClients() {
      try {
        const result = await $pocketbase.collection('clients').getList(1, 50, { expand: 'group' });
        console.log('Fetched clients:', result.items);
        clients = result.items.map((item) => ({
          id: item.id,
          name: item.name,
          logo: item.logo ? $pocketbase.getFileUrl(item, item.logo) : '',
          hidden_name: item.hidden_name || '',
          group: item.expand?.group ? { id: item.expand.group.id, name: item.expand.group.name } : null,
          homepage: item.homepage || '',
          favicon: item.favicon ? $pocketbase.getFileUrl(item, item.favicon) : '',
        }));
      } catch (error) {
        console.error('Error fetching clients:', error);
      }
    }
  
    async function fetchClientGroups() {
      try {
        const result = await $pocketbase.collection('client_groups').getList();
        clientGroups = result.items.map((group) => ({
          id: group.id,
          name: group.name,
        }));
      } catch (error) {
        console.error('Error fetching client groups:', error);
      }
    }
  
    async function saveClient(event: ClientSaveEvent) {
      // The client is already saved in the ClientForm component
      // We just need to refresh the client list
      await fetchClients();
      showClientModal = false;
    }
  
    async function deleteClient() {
      try {
        await $pocketbase.collection('clients').delete(currentClientId);
        await fetchClients();
        showDeleteClientModal = false;
      } catch (error) {
        console.error('Error deleting client:', error);
      }
    }
  
    function openAddClientModal() {
      modalMode = 'add';
      currentClientData = null;
      showClientModal = true;
    }
  
    function openEditModal(client: Client) {
      console.log('Editing client:', client);
      modalMode = 'edit';
      currentClientData = { ...client };
      showClientModal = true;
    }
  
    function openDeleteModal(id: string) {
      currentClientId = id;
      showDeleteClientModal = true;
    }
  
    function openAddClientGroupModal() {
      showClientGroupModal = true;
    }
  
    async function saveClientGroup(event: ClientGroupEvent) {
      const { groupData, selectedClients } = event.detail;
      try {
        // Create the new client group
        const newGroup = await $pocketbase.collection('client_groups').create(groupData);

        // Assign selected clients to the new group
        for (const clientId of selectedClients) {
          await $pocketbase.collection('clients').update(clientId, { group: newGroup.id });
        }

        showClientGroupModal = false;
        await fetchClients(); // Refresh clients to get updated group assignments
        await fetchClientGroups(); // Refresh client groups after adding a new one
      } catch (error) {
        console.error('Error saving client group:', error);
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
  
    async function assignClientsToGroup(groupId: string) {
      if (!groupId) {
        alert('Please select a group to assign the selected clients.');
        return;
      }
      try {
        for (let clientId of selectedClients) {
          await $pocketbase.collection('clients').update(clientId, { group: groupId });
        }
        await fetchClients();
        selectedClients.clear();
        selectedGroupId = ''; // Reset selected group
      } catch (error) {
        console.error('Error assigning clients to group:', error);
      }
    }
  </script>
  
  <MetaTag {path} {description} {title} {subtitle} />
  <main class="p-4">
    <Breadcrumb class="mb-6">
        <BreadcrumbItem home>Home</BreadcrumbItem>
        <BreadcrumbItem href="/clients">Clients</BreadcrumbItem>
    </Breadcrumb>

    <Heading tag="h1" class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
        Clients
    </Heading>
  
    <Button class="mt-4" on:click={openAddClientModal}>Add Client</Button>
    <Button class="mt-4" on:click={openAddClientGroupModal}>Add Client Group</Button>
  
    {#if multipleClientsSelected}
      <Select placeholder="Select a group" bind:value={selectedGroupId} class="mt-4">
        {#each clientGroups as group}
          <option value={group.id}>{group.name}</option>
        {/each}
      </Select>
      <Button class="mt-4" on:click={() => assignClientsToGroup(selectedGroupId)}>Assign to Group</Button>
    {/if}
  
    <Table class="mt-4 border border-gray-200 dark:border-gray-700">
      <TableHead class="bg-gray-100 dark:bg-gray-700">
        {#each ['Select', 'Name', 'Logo', 'Homepage', 'Group', 'Actions'] as title}
          <TableHeadCell class="ps-4 font-normal">{title}</TableHeadCell>
        {/each}
      </TableHead>
      <TableBody>
        {#each clients as client}
          <TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
            <TableBodyCell class="w-4 p-4">
              <Checkbox
                checked={selectedClients.has(client.id)}
                on:change={() => toggleClientSelection(client.id)}
              />
            </TableBodyCell>
            <TableBodyCell class="p-4">{client.name}</TableBodyCell>
            <TableBodyCell class="px-4 font-normal">
              {#if client.logo}
                <img src={client.logo} alt="{client.name} Logo" class="h-10 w-auto" />
              {:else if client.favicon}
                <img src={client.favicon} alt="{client.name} Favicon" class="h-8 w-8" />
              {/if}
            </TableBodyCell>
            <TableBodyCell class="p-4">
              {#if client.homepage}
                <a href={client.homepage} target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-200">
                  {new URL(client.homepage).hostname}
                </a>
              {/if}
            </TableBodyCell>
            <TableBodyCell class="p-4">{client.group?.name || 'No Group'}</TableBodyCell>
            <TableBodyCell class="space-x-2">
              <Button
                size="sm"
                class="gap-2 px-3"
                on:click={() => openEditModal(client)}
              >
                Edit
              </Button>
              <Button
                color="red"
                size="sm"
                class="gap-2 px-3"
                on:click={() => openDeleteModal(client.id)}
              >
                Delete
              </Button>
            </TableBodyCell>
          </TableBodyRow>
        {/each}
      </TableBody>
    </Table>
  
    <ClientForm
      bind:open={showClientModal}
      client={currentClientData}
      on:save={saveClient}
      mode={modalMode}
      size="medium"
    />
    <Delete bind:open={showDeleteClientModal} onDelete={deleteClient} />
    <ClientGroupForm bind:open={showClientGroupModal} on:save={saveClientGroup} />
  </main>

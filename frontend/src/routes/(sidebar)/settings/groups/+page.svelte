<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { 
    Button, 
    Table, 
    TableBody,
    TableBodyCell,
    TableBodyRow,
    TableHead,
    TableHeadCell,
    Modal, 
    Label, 
    Input, 
    Checkbox, 
    Toast,
    Card,
    Badge,
    Spinner
  } from 'flowbite-svelte';
  import { slide } from 'svelte/transition';
  import { 
    CheckCircleSolid, 
    ExclamationCircleSolid,
    UsersGroupSolid,
    PenSolid,
    TrashBinSolid
  } from 'flowbite-svelte-icons';

  interface GroupPermissions {
    read: string[];
    write: string[];
    delete: string[];
    manage_users: boolean;
    manage_system: boolean;
    manage_groups: boolean;
    manage_providers: boolean;
    manage_notifications: boolean;
    manage_api_keys: boolean;
    settings: boolean;
  }

  interface Group {
    id: string;
    name: string;
    description: string;
    permissions: GroupPermissions;
  }

  let groups: Group[] = [];
  let loading = true;
  let editModalOpen = false;
  let deleteModalOpen = false;
  let currentGroup: Group | null = null;
  let showSuccessToast = false;
  let showErrorToast = false;
  let toastMessage = '';

  // Form data for new/edit group
  let formData = {
    name: '',
    description: '',
    permissions: {
      read: [] as string[],
      write: [] as string[],
      delete: [] as string[],
      manage_users: false,
      manage_system: false,
      manage_groups: false,
      manage_providers: false,
      manage_notifications: false,
      manage_api_keys: false,
      settings: false
    }
  };

  // Available permissions for resources
  const resources = ['clients', 'findings', 'scan_profiles', 'users'];

  onMount(async () => {
    await loadGroups();
  });

  async function loadGroups() {
    try {
      loading = true;
      const records = await $pocketbase.collection('groups').getFullList();
      groups = records;
    } catch (error) {
      toastMessage = 'Error loading groups';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      console.error('Error loading groups:', error);
    } finally {
      loading = false;
    }
  }

  function openEditModal(group = null) {
    currentGroup = group;
    if (group) {
      formData = {
        name: group.name,
        description: group.description,
        permissions: {
          ...group.permissions
        }
      };
    } else {
      formData = {
        name: '',
        description: '',
        permissions: {
          read: [],
          write: [],
          delete: [],
          manage_users: false,
          manage_system: false,
          manage_groups: false,
          manage_providers: false,
          manage_notifications: false,
          manage_api_keys: false,
          settings: false
        }
      };
    }
    editModalOpen = true;
  }

  function openDeleteModal(group) {
    currentGroup = group;
    deleteModalOpen = true;
  }

  async function saveGroup() {
    try {
      if (currentGroup) {
        await $pocketbase.collection('groups').update(currentGroup.id, formData);
        toastMessage = 'Group updated successfully';
      } else {
        await $pocketbase.collection('groups').create(formData);
        toastMessage = 'Group created successfully';
      }
      showSuccessToast = true;
      setTimeout(() => showSuccessToast = false, 3000);
      editModalOpen = false;
      await loadGroups();
    } catch (error) {
      toastMessage = 'Error saving group';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      console.error('Error saving group:', error);
    }
  }

  async function deleteGroup() {
    try {
      await $pocketbase.collection('groups').delete(currentGroup.id);
      toastMessage = 'Group deleted successfully';
      showSuccessToast = true;
      setTimeout(() => showSuccessToast = false, 3000);
      deleteModalOpen = false;
      await loadGroups();
    } catch (error) {
      toastMessage = 'Error deleting group';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      console.error('Error deleting group:', error);
    }
  }

  function toggleResource(type: 'read' | 'write' | 'delete', resource: string) {
    const index = formData.permissions[type].indexOf(resource);
    if (index === -1) {
      formData.permissions[type] = [...formData.permissions[type], resource];
    } else {
      formData.permissions[type] = formData.permissions[type].filter(r => r !== resource);
    }
    formData = { ...formData }; // Trigger reactivity
  }

  function getBadgeColor(type: string): "blue" | "green" | "red" {
    switch (type) {
      case 'read': return 'blue';
      case 'write': return 'green';
      case 'delete': return 'red';
      default: return 'blue';
    }
  }
</script>

{#if showSuccessToast}
  <Toast transition={slide} color="green" class="fixed bottom-4 right-4">
    <CheckCircleSolid slot="icon" class="w-5 h-5" />
    {toastMessage}
  </Toast>
{/if}

{#if showErrorToast}
  <Toast transition={slide} color="red" class="fixed bottom-4 right-4">
    <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
    {toastMessage}
  </Toast>
{/if}

<div class="container mx-auto p-4">
  <!-- Breadcrumb -->
  <div class="mb-6">
    <nav class="flex" aria-label="Breadcrumb">
      <ol class="inline-flex items-center space-x-1 md:space-x-3">
        <li class="inline-flex items-center">
          <a href="/" class="inline-flex items-center text-sm font-medium text-gray-700 hover:text-blue-600 dark:text-gray-400 dark:hover:text-white">
            Home
          </a>
        </li>
        <li>
          <div class="flex items-center">
            <svg class="w-3 h-3 text-gray-400 mx-1" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 6 10">
              <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 9 4-4-4-4"/>
            </svg>
            <a href="/settings" class="ml-1 text-sm font-medium text-gray-700 hover:text-blue-600 md:ml-2 dark:text-gray-400 dark:hover:text-white">
              Settings
            </a>
          </div>
        </li>
        <li aria-current="page">
          <div class="flex items-center">
            <svg class="w-3 h-3 text-gray-400 mx-1" aria-hidden="true" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 6 10">
              <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m1 9 4-4-4-4"/>
            </svg>
            <span class="ml-1 text-sm font-medium text-gray-500 md:ml-2 dark:text-gray-400">Groups</span>
          </div>
        </li>
      </ol>
    </nav>
  </div>

  <Card padding="xl" class="min-w-full">
    <div class="flex justify-between items-center mb-8">
      <div class="flex items-center gap-3">
        <UsersGroupSolid class="w-8 h-8 text-primary-600 dark:text-primary-500" />
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Group Management</h1>
          <p class="text-sm text-gray-600 dark:text-gray-400">Manage user groups and their permissions</p>
        </div>
      </div>
      <Button color="primary" size="lg" class="px-4" on:click={() => openEditModal()}>
        <UsersGroupSolid class="w-4 h-4 mr-2" />
        Create New Group
      </Button>
    </div>

    {#if loading}
      <div class="flex justify-center items-center p-8">
        <Spinner size="8" color="primary" />
      </div>
    {:else}
      <Table striped={true} hoverable={true} shadow>
        <TableHead>
          <TableHeadCell class="!p-4 bg-gray-50 dark:bg-gray-700">Name</TableHeadCell>
          <TableHeadCell class="!p-4 bg-gray-50 dark:bg-gray-700">Description</TableHeadCell>
          <TableHeadCell class="!p-4 w-1/3 bg-gray-50 dark:bg-gray-700">Permissions</TableHeadCell>
          <TableHeadCell class="!p-4 w-24 bg-gray-50 dark:bg-gray-700">Actions</TableHeadCell>
        </TableHead>
        <TableBody class="divide-y divide-gray-200 dark:divide-gray-700">
          {#each groups as group}
            <TableBodyRow class="bg-white dark:bg-gray-800">
              <TableBodyCell class="!p-4">
                <div class="font-medium text-gray-900 dark:text-white">{group.name}</div>
              </TableBodyCell>
              <TableBodyCell class="!p-4 text-gray-600 dark:text-gray-400">
                {group.description}
              </TableBodyCell>
              <TableBodyCell class="!p-4">
                <div class="space-y-2">
                  {#each ['read', 'write', 'delete'] as type}
                    {#if group.permissions[type]?.length > 0}
                      <div class="flex flex-wrap gap-1.5">
                        <Badge color={getBadgeColor(type)} class="capitalize font-semibold">{type}:</Badge>
                        {#each group.permissions[type] as resource}
                          <Badge color={getBadgeColor(type)} class="opacity-75 capitalize">{resource}</Badge>
                        {/each}
                      </div>
                    {/if}
                  {/each}
                  <div class="flex flex-wrap gap-1.5 mt-1">
                    {#if group.permissions.manage_users}
                      <Badge color="primary">Manage Users</Badge>
                    {/if}
                    {#if group.permissions.manage_groups}
                      <Badge color="primary">Manage Groups</Badge>
                    {/if}
                    {#if group.permissions.manage_providers}
                      <Badge color="primary">Manage Providers</Badge>
                    {/if}
                    {#if group.permissions.manage_notifications}
                      <Badge color="primary">Manage Notifications</Badge>
                    {/if}
                    {#if group.permissions.manage_system}
                      <Badge color="primary">Manage System</Badge>
                    {/if}
                    {#if group.permissions.manage_api_keys}
                      <Badge color="primary">Manage API Keys</Badge>
                    {/if}
                    {#if group.permissions.settings}
                      <Badge color="primary">Settings</Badge>
                    {/if}
                  </div>
                </div>
              </TableBodyCell>
              <TableBodyCell class="!p-4">
                <div class="flex items-center gap-2">
                  <Button size="xs" color="primary" class="px-2" on:click={() => openEditModal(group)}>
                    <PenSolid class="w-3.5 h-3.5" />
                  </Button>
                  <Button size="xs" color="red" class="px-2" on:click={() => openDeleteModal(group)}>
                    <TrashBinSolid class="w-3.5 h-3.5" />
                  </Button>
                </div>
              </TableBodyCell>
            </TableBodyRow>
          {/each}
        </TableBody>
      </Table>
    {/if}
  </Card>
</div>

<!-- Edit Modal -->
<Modal bind:open={editModalOpen} size="lg" autoclose={false}>
  <div class="flex items-start justify-between p-5 border-b rounded-t dark:border-gray-600">
    <h3 class="text-xl font-semibold text-gray-900 dark:text-white">
      {currentGroup ? 'Edit Group' : 'Create New Group'}
    </h3>
  </div>
  <div class="p-6 space-y-6">
    <div class="space-y-4">
      <!-- Name -->
      <div>
        <Label for="name" class="mb-2">Group Name</Label>
        <Input id="name" type="text" bind:value={formData.name} required />
      </div>

      <!-- Description -->
      <div>
        <Label for="description" class="mb-2">Description</Label>
        <Input id="description" type="text" bind:value={formData.description} />
      </div>

      <!-- Permissions -->
      <div class="space-y-4">
        <h4 class="text-lg font-semibold text-gray-900 dark:text-white">Permissions</h4>
        
        <!-- Resource Permissions -->
        <div class="grid gap-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
          <h5 class="font-medium text-gray-900 dark:text-white">Resource Access</h5>
          {#each resources as resource}
            <div class="pl-4 space-y-2">
              <div class="font-medium capitalize text-gray-700 dark:text-gray-300">{resource}</div>
              <div class="flex flex-wrap gap-4">
                {#each ['read', 'write', 'delete'] as type}
                  <div class="flex items-center gap-2">
                    <Checkbox
                      checked={formData.permissions[type].includes(resource)}
                      on:change={() => toggleResource(type, resource)}
                    />
                    <span class="capitalize text-sm text-gray-600 dark:text-gray-400">{type}</span>
                  </div>
                {/each}
              </div>
            </div>
          {/each}
        </div>

        <!-- Special Permissions -->
        <div class="grid gap-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
          <h5 class="font-medium text-gray-900 dark:text-white">Administrative Access</h5>
          <div class="space-y-3">
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_users} />
              Manage Users
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_groups} />
              Manage Groups
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_providers} />
              Manage Providers
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_notifications} />
              Manage Notifications
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_system} />
              Manage System
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.manage_api_keys} />
              Manage API Keys
            </Label>
            <Label class="flex items-center gap-2">
              <Checkbox bind:checked={formData.permissions.settings} />
              Settings Access
            </Label>
          </div>
        </div>
      </div>
    </div>
  </div>
  <div class="flex items-center justify-end p-6 space-x-2 border-t border-gray-200 rounded-b dark:border-gray-600">
    <Button color="alternative" on:click={() => editModalOpen = false}>Cancel</Button>
    <Button color="primary" on:click={saveGroup}>Save</Button>
  </div>
</Modal>

<!-- Delete Modal -->
<Modal bind:open={deleteModalOpen} size="md" autoclose={false}>
  <div class="text-center p-6">
    <ExclamationCircleSolid class="mx-auto mb-4 text-red-600 w-12 h-12" />
    <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
      Are you sure you want to delete the group "{currentGroup?.name}"?
    </h3>
    <div class="flex justify-center gap-4">
      <Button color="red" class="px-4" on:click={deleteGroup}>Yes, delete</Button>
      <Button color="alternative" class="px-4" on:click={() => deleteModalOpen = false}>No, cancel</Button>
    </div>
  </div>
</Modal> 
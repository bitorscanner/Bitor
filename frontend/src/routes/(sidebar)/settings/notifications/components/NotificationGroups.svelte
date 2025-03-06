<script lang="ts">
    import { Button, Modal, Label, Select, Input, Card, Toggle } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { onMount } from 'svelte';
    import type { RecordModel } from 'pocketbase';

    interface User {
        id: string;
        email: string;
        name?: string;
    }

    interface Group extends RecordModel {
        name: string;
        description: string;
        members: string[];
        enabled: boolean;
        expand?: {
            members: User[];
        };
    }

    let groups: Group[] = [];
    let users: User[] = [];
    let selectedGroup: Group | null = null;
    let showModal = false;
    let name = '';
    let description = '';
    let selectedMembers: string[] = [];

    onMount(async () => {
        await loadGroups();
        await loadUsers();
    });

    async function loadGroups() {
        try {
            const records = await $pocketbase.collection('notification_groups').getFullList({
                expand: 'members'
            });
            groups = records as unknown as Group[];
        } catch (error) {
            console.error('Error loading notification groups:', error);
        }
    }

    async function loadUsers() {
        try {
            const records = await $pocketbase.collection('users').getFullList();
            users = records.map(record => ({
                id: record.id,
                email: record.email,
                name: record.name
            }));
        } catch (error) {
            console.error('Error loading users:', error);
        }
    }

    async function toggleGroup(group: Group) {
        try {
            await $pocketbase.collection('notification_groups').update(group.id, {
                enabled: !group.enabled
            });
            await loadGroups();
        } catch (error) {
            console.error('Error toggling group:', error);
        }
    }

    function openCreateModal() {
        selectedGroup = null;
        name = '';
        description = '';
        selectedMembers = [];
        showModal = true;
    }

    function openEditModal(group: Group) {
        selectedGroup = group;
        name = group.name;
        description = group.description;
        selectedMembers = group.members;
        showModal = true;
    }

    async function saveGroup() {
        try {
            const data = {
                name,
                description,
                members: selectedMembers,
                enabled: selectedGroup ? selectedGroup.enabled : true
            };

            if (selectedGroup) {
                await $pocketbase.collection('notification_groups').update(selectedGroup.id, data);
            } else {
                await $pocketbase.collection('notification_groups').create(data);
            }

            showModal = false;
            await loadGroups();
        } catch (error) {
            console.error('Error saving notification group:', error);
        }
    }

    async function deleteGroup(id: string) {
        if (confirm('Are you sure you want to delete this group?')) {
            try {
                await $pocketbase.collection('notification_groups').delete(id);
                await loadGroups();
            } catch (error) {
                console.error('Error deleting notification group:', error);
            }
        }
    }
</script>

<div class="p-4">
    <div class="flex justify-between items-center mb-4">
        <h2 class="text-xl font-bold">Notification Groups</h2>
        <Button color="blue" on:click={openCreateModal}>Create Group</Button>
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each groups as group}
            <Card class="relative">
                <div class="flex flex-col h-full">
                    <div class="flex justify-between items-start mb-2">
                        <div class="flex items-center gap-3">
                            <Toggle checked={group.enabled} on:change={() => toggleGroup(group)} />
                            <h3 class="text-lg font-semibold">{group.name}</h3>
                        </div>
                        <div class="flex gap-2">
                            <Button color="blue" size="xs" on:click={() => openEditModal(group)}>Edit</Button>
                            <Button color="red" size="xs" on:click={() => deleteGroup(group.id)}>Delete</Button>
                        </div>
                    </div>
                    <p class="text-gray-600 dark:text-gray-400 mb-4">{group.description}</p>
                    <div class="mt-auto">
                        <h4 class="font-medium mb-2">Members:</h4>
                        <div class="text-sm text-gray-600 dark:text-gray-400">
                            {#if group.expand?.members}
                                {#each group.expand.members as member, i}
                                    {member.email}{#if i < group.expand.members.length - 1}, {/if}
                                {/each}
                            {:else}
                                No members
                            {/if}
                        </div>
                    </div>
                </div>
            </Card>
        {:else}
            <div class="col-span-full text-center py-8 text-gray-500 dark:text-gray-400">
                No notification groups created yet. Click "Create Group" to add one.
            </div>
        {/each}
    </div>
</div>

<Modal bind:open={showModal} size="lg" autoclose>
    <div class="p-4">
        <h3 class="text-xl font-bold mb-4">{selectedGroup ? 'Edit' : 'Create'} Notification Group</h3>
        <form class="space-y-4" on:submit|preventDefault={saveGroup}>
            <div>
                <Label for="name">Name</Label>
                <Input id="name" bind:value={name} required />
            </div>
            <div>
                <Label for="description">Description</Label>
                <Input id="description" bind:value={description} />
            </div>
            <div>
                <Label for="members">Members</Label>
                <Select multiple bind:value={selectedMembers}>
                    {#each users as user}
                        <option value={user.id}>{user.email}</option>
                    {/each}
                </Select>
            </div>
            <div class="flex justify-end gap-2">
                <Button color="alternative" on:click={() => (showModal = false)}>Cancel</Button>
                <Button type="submit" color="blue">{selectedGroup ? 'Update' : 'Create'}</Button>
            </div>
        </form>
    </div>
</Modal> 
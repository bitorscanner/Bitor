<script lang="ts">
    import { Breadcrumb, BreadcrumbItem, Heading, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Checkbox, Button } from 'flowbite-svelte';
    import { onMount } from 'svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import TargetForm from './TargetForm.svelte';
    import AttackSurfaceTargetForm from './AttackSurfaceTargetForm.svelte';
    import Delete from './Delete.svelte';
	import MetaTag from '@utils/MetaTag.svelte';
    interface Target {
        id: string;
        name: string;
        targets: string[];
        count: number;
        clientName: string;
        client: string;
        favicon?: string;
    }

    const path: string = '/targets';
    const description: string = 'Nuclei targets management';
    const title: string = 'Nuclei Targets';
    const subtitle: string = 'Manage your nuclei targets';

    let currentTargetData: Target | null = null;
    let targets: Target[] = [];
    let showAddTargetModal = false;
    let showDeleteTargetModal = false;
    let currentTargetId = '';
    let showEditTargetModal = false; // Declare the variable
    let modalMode: 'add' | 'edit' = 'add'; // Declare and initialize modalMode
    let showTargetModal = false; // Add this line to declare the variable
    let showAttackSurfaceModal = false;

    // Add PocketBase base URL
    const pbUrl = $pocketbase.baseUrl;

    onMount(() => {
        fetchTargets();
    });

    async function fetchTargets() {
        try {
            console.log('Fetching targets');
            const result = await $pocketbase.collection('nuclei_targets').getList(1, 50, {
                expand: 'client'
            });
            console.log('Got targets result:', result);
            targets = result.items.map(item => {
                const clientRecord = item.expand?.client;
                console.log('Client record for target:', item.id, clientRecord);
                // Use PocketBase's getFileUrl method for the favicon
                const faviconUrl = clientRecord?.favicon ? 
                    $pocketbase.getFileUrl(clientRecord, clientRecord.favicon) : 
                    null;
                console.log('Generated favicon URL:', faviconUrl);
                
                return {
                    id: item.id,
                    name: item.name,
                    targets: item.targets || [],
                    count: item.count || 0,
                    clientName: clientRecord?.name || 'N/A',
                    client: item.client,
                    favicon: faviconUrl || undefined
                };
            });
            console.log('Processed targets:', targets);
        } catch (error) {
            console.error('Error fetching targets:', error);
        }
    }

    function openDeleteModal(id: string) {
        currentTargetId = id;
        showDeleteTargetModal = true;
    }

    async function saveTarget(targetData: Target) {
        try {
            const data = {
                ...targetData,
            };
            
            if (modalMode === 'edit' && currentTargetData && currentTargetData.id) {
                await $pocketbase.collection('nuclei_targets').update(currentTargetData.id, data);
            } else {
                await $pocketbase.collection('nuclei_targets').create(data);
            }
            fetchTargets();
            showTargetModal = false;
        } catch (error) {
            console.error('Error saving target:', error);
        }
    }

    async function deleteTarget() {
        try {
            await $pocketbase.collection('nuclei_targets').delete(currentTargetId);
            fetchTargets(); // Refresh the list
            showDeleteTargetModal = false;
        } catch (error) {
            console.error('Error deleting target:', error);
        }
    }

    function openAddTargetModal() {
        modalMode = 'add';
        currentTargetData = null;
        showTargetModal = true;
    }
    
    function openAttackSurfaceModal() {
        showAttackSurfaceModal = true;
    }
    
    function onAttackSurfaceSuccess() {
        fetchTargets();
    }
    
    function openEditModal(target: Target) {
        console.log('Editing target:', target); // Add this line
        modalMode = 'edit';
        currentTargetData = { ...target };
        showTargetModal = true;
    }
</script>

<MetaTag {path} {description} {title} {subtitle} />
<main class="p-4">
    <Breadcrumb class="mb-6">
        <BreadcrumbItem home>Home</BreadcrumbItem>
        <BreadcrumbItem href="/targets">Targets</BreadcrumbItem>
    </Breadcrumb>

    <Heading tag="h1" class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
        Targets
    </Heading>

    <div class="mt-4 flex space-x-2">
        <Button on:click={() => openAddTargetModal()}>Add Target</Button>
        <Button color="purple" on:click={() => openAttackSurfaceModal()}>Create from Attack Surface</Button>
    </div>

    <Table class="mt-4 border border-gray-200 dark:border-gray-700">
        <TableHead class="bg-gray-100 dark:bg-gray-700">
            <TableHeadCell class="w-4 p-4"><Checkbox /></TableHeadCell>
            {#each ['Name', 'Total Targets', 'Client', 'Actions'] as title}
                <TableHeadCell class="ps-4 font-normal">{title}</TableHeadCell>
            {/each}
        </TableHead>
        <TableBody>
            {#each targets as target}
                <TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
                    <TableBodyCell class="w-4 p-4"><Checkbox /></TableBodyCell>
                    <TableBodyCell class="p-4">{target.name}</TableBodyCell>
                    <TableBodyCell class="p-4">{target.count}</TableBodyCell>
                    <TableBodyCell class="p-4 flex items-center">
                        {#if target.favicon}
                            <img src={target.favicon} alt="Client Favicon" class="w-4 h-4 mr-2" />
                        {/if}
                        {target.clientName}
                    </TableBodyCell>
                    <TableBodyCell class="space-x-2">
                        <Button size="sm" class="gap-2 px-3" on:click={() => openEditModal(target)}>Edit</Button>
                        <Button color="red" size="sm" class="gap-2 px-3" on:click={() => openDeleteModal(target.id)}>Delete</Button>
                    </TableBodyCell>
                </TableBodyRow>
            {/each}
        </TableBody>
    </Table>
    
    <TargetForm
        bind:open={showTargetModal}
        target={currentTargetData}
        onSave={saveTarget}
        mode={modalMode}
    />
    <AttackSurfaceTargetForm
        bind:open={showAttackSurfaceModal}
        onSuccess={onAttackSurfaceSuccess}
    />
    <Delete bind:open={showDeleteTargetModal} onDelete={deleteTarget} />
</main>

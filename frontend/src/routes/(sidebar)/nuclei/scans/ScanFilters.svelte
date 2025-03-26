<script lang="ts">
    import { MultiSelect, Toggle, Button, Checkbox } from 'flowbite-svelte';
    import { createEventDispatcher } from 'svelte';
    import { slide } from 'svelte/transition';
    import type { Client, Provider } from './types';
    import { SiDigitalocean, SiAmazon, SiAmazons3 } from '@icons-pack/svelte-simple-icons';

    export let clients: Client[] = [];
    export let providers: Provider[] = [];
    export let showManualScans = false;
    export let showDestroyedScans = true;
    export let showArchivedScans = false;
    export let showMyScansOnly = true;

    let showFilters = false; // Start collapsed by default
    const dispatch = createEventDispatcher();

    let selectedStatuses: string[] = [];
    let selectedClients: string[] = [];
    let selectedProviders: string[] = [];

    const statusOptions = [
        { value: 'Created', name: 'Created' },
        { value: 'Started', name: 'Started' },
        { value: 'Generating', name: 'Generating' },
        { value: 'Deploying', name: 'Deploying' },
        { value: 'Running', name: 'Running' },
        { value: 'Finished', name: 'Finished' },
        { value: 'Failed', name: 'Failed' },
        { value: 'Stopped', name: 'Stopped' },
        { value: 'Manual', name: 'Manual' }
    ];

    $: clientOptions = clients.map(client => ({
        value: client.id,
        name: client.name,
        html: client.favicon ? 
            `<div class="flex items-center gap-2">
                <img src="${client.favicon}" alt="${client.name}" class="w-6 h-6" />
                <span>${client.name}</span>
            </div>` :
            client.name
    }));

    $: providerOptions = providers.map(provider => ({
        value: provider.id,
        name: provider.name,
        html: `<div class="flex items-center gap-2">
            ${provider.provider_type === 'digitalocean' ? '<SiDigitalocean class="w-6 h-6" />' :
              provider.provider_type === 'aws' ? '<SiAmazon class="w-6 h-6" />' :
              provider.provider_type === 's3' ? '<SiAmazons3 class="w-6 h-6" />' : ''}
            <span>${provider.name}</span>
        </div>`
    }));

    function handleChange() {
        dispatch('filterChange', {
            statuses: selectedStatuses,
            clients: selectedClients,
            providers: selectedProviders,
            showManual: showManualScans,
            showDestroyed: showDestroyedScans,
            showArchived: showArchivedScans,
            showMyScansOnly: showMyScansOnly
        });
    }

    function clearFilters() {
        selectedStatuses = [];
        selectedClients = [];
        selectedProviders = [];
        showManualScans = false;
        showDestroyedScans = true;
        showArchivedScans = false;
        showMyScansOnly = true;
        handleChange();
    }

    $: {
        // Trigger handleChange whenever any filter changes
        selectedStatuses;
        selectedClients;
        selectedProviders;
        showManualScans;
        showDestroyedScans;
        showArchivedScans;
        showMyScansOnly;
        handleChange();
    }
</script>

<div class="flex flex-col gap-4 mb-6 bg-white dark:bg-gray-800 p-4 rounded-lg shadow">
    <div class="flex items-center justify-between mb-2">
        <div class="flex items-center gap-2">
            <button 
                class="text-lg font-medium text-gray-900 dark:text-white flex items-center gap-2"
                on:click={() => showFilters = !showFilters}
            >
                <span>Filters</span>
                <svg 
                    class="w-4 h-4 transition-transform {showFilters ? 'rotate-180' : ''}" 
                    fill="none" 
                    stroke="currentColor" 
                    viewBox="0 0 24 24"
                >
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
                </svg>
            </button>
            {#if selectedStatuses.length > 0 || selectedClients.length > 0 || selectedProviders.length > 0 || showManualScans || showDestroyedScans || showArchivedScans}
                <span class="text-sm text-gray-500 dark:text-gray-400">
                    ({selectedStatuses.length + selectedClients.length + selectedProviders.length + (showManualScans ? 1 : 0) + (showDestroyedScans ? 1 : 0) + (showArchivedScans ? 1 : 0)} active)
                </span>
            {/if}
        </div>
        <Button size="xs" color="light" on:click={clearFilters}>Clear Filters</Button>
    </div>

    {#if showFilters}
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4" transition:slide>
            <!-- Status Filter -->
            <div>
                <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">
                    Status
                </label>
                <MultiSelect
                    class="w-full"
                    items={statusOptions}
                    bind:value={selectedStatuses}
                    placeholder="Select statuses..."
                />
            </div>

            <!-- Client Filter -->
            <div>
                <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">
                    Client
                </label>
                <MultiSelect
                    class="w-full"
                    items={clientOptions}
                    bind:value={selectedClients}
                    placeholder="Select clients..."
                />
            </div>

            <!-- Provider Filter -->
            <div>
                <label class="block mb-2 text-sm font-medium text-gray-900 dark:text-white">
                    Provider
                </label>
                <MultiSelect
                    class="w-full"
                    items={providerOptions}
                    bind:value={selectedProviders}
                    placeholder="Select providers..."
                />
            </div>

            <div class="flex items-center gap-4 mt-2 md:col-span-3">
                <Checkbox bind:checked={showMyScansOnly} on:change={handleChange}>
                    <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
                        Show My Scans Only
                    </span>
                </Checkbox>
                <Checkbox bind:checked={showArchivedScans} on:change={handleChange}>
                    <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
                        Show Archived Scans
                    </span>
                </Checkbox>
                <Checkbox bind:checked={showManualScans} on:change={handleChange}>
                    <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
                        Show Manual Scans
                    </span>
                </Checkbox>
                <Checkbox bind:checked={showDestroyedScans} on:change={handleChange}>
                    <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
                        Show Destroyed Scans
                    </span>
                </Checkbox>
            </div>
        </div>
    {/if}
</div> 
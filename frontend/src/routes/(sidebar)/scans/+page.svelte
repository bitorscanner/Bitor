<script lang="ts">
    import { Breadcrumb, BreadcrumbItem, Heading, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Checkbox, Button, Pagination } from 'flowbite-svelte';
    import ScanForm from './ScanForm.svelte';
    import Delete from './Delete.svelte';
    import { onMount, onDestroy } from 'svelte';
    import MetaTag from '../../utils/MetaTag.svelte';
    import { pocketbase } from '$lib/stores/pocketbase';
    import StartScan from './StartScan.svelte';
    import StopScan from './StopScan.svelte';
    import LogModal from './LogModal.svelte';
    import { BookOpenSolid, TerminalSolid } from 'flowbite-svelte-icons';
    import StatusBadge from '../../utils/dashboard/StatusBadge.svelte';
    import ResultsModal from './ResultsModal.svelte';
    import ManualScanModal from './ManualScanModal.svelte';
    import ScanFilters from './ScanFilters.svelte';
    import { goto } from '$app/navigation';
    import type { ScanData, ScanFormData, Client, Provider } from './types';
    import TerminalModal from './TerminalModal.svelte';
    import { SiDigitalocean, SiAmazon, SiAmazons3 } from '@icons-pack/svelte-simple-icons';
    import DestroyScan from './DestroyScan.svelte';
    import Archive from './Archive.svelte';

    let scans: ScanData[] = [];
    let filteredScans: ScanData[] = [];
    let showAddScanModal = false;
    let showDeleteScanModal = false;
    let currentScanId = '';
    let selectedScanId = '';
    let currentScanData = {};
    let showStartScanModal = false;
    let showStopScanModal = false;
    let currentScan: ScanData | null = null;
    let showLogModal = false;
    let userToken = $pocketbase.authStore.token;
    let showResultsModal = false;
    let showManualScanModal = false;
    let intervalId: NodeJS.Timeout;
    let showTerminalModal = false;
    let showDestroyModal = false;
    let showArchiveModal = false;

    // Filter-related state
    let providers: Provider[] = [];
    let clients: Client[] = [];
    let showManualScans = false;
    let showDestroyedScans = true;
    let showArchivedScans = false;
    let filterStatuses: string[] = [];
    let filterClients: string[] = [];
    let filterProviders: string[] = [];

    // Pagination state
    let currentPage = 1;
    let totalPages = 1;
    let itemsPerPage = 10;
    let paginatedScans: ScanData[] = [];

    let modalMode: 'add' | 'edit' = 'add';

    // Add after other state variables
    let selectedScans: string[] = [];
    let selectAll = false;

    const path: string = '/scans';
    const description: string = 'Nuclei scan management';
    const title: string = 'Nuclei Scans';
    const subtitle: string = 'Manage your nuclei scans';

    interface Client {
        id: string | null;
        name: string;
        api_key: string | null;
        favicon: string;
    }

    interface ScanData {
        id: string;
        name: string;
        status: string;
        destroyed: boolean;
        start_time?: string;
        end_time?: string;
        progress?: number;
        vm_provider?: string;
        vm_provider_name?: string;
        nuclei_profile?: string;
        nuclei_profile_name?: string;
        nuclei_targets?: string;
        nuclei_interact?: string;
        nuclei_interact_name?: string;
        client?: {
            id: string | null;
            name: string;
            api_key: string | null;
            favicon: string;
        };
        ansible_logs?: any[];
        state_bucket?: string;
        scan_bucket?: string;
        cron?: string;
        api_key?: string;
        ip_address?: string;
        cost?: number;

        cost_per_hour?: number;
        vm_size?: string;
        start_time_display?: string;
        end_time_display?: string;
        startImmediately?: boolean;
        archived?: boolean;
        vm_start_time?: string;
        vm_stop_time?: string;
    }

    interface ScanFormData {
        startImmediately?: boolean;
        [key: string]: any;
    }

    async function fetchProviders() {
        try {
            const result = await $pocketbase.collection('providers').getFullList();
            console.log('All providers with details:', result.map(p => ({
                id: p.id,
                name: p.name,
                provider_type: p.provider_type,
                use: p.use
            })));
            
            // Filter for providers that have 'compute' in their use array
            providers = result
                .filter(item => Array.isArray(item.use) && item.use.includes('compute'))
                .map(item => ({
                    id: item.id,
                    name: item.name,
                    provider_type: item.provider_type
                }));
            console.log('Filtered compute providers:', providers);
        } catch (error) {
            console.error('Error fetching providers:', error);
        }
    }

    async function fetchClients() {
        try {
            const result = await $pocketbase.collection('clients').getFullList();
            clients = result.map(item => ({
                id: item.id,
                name: item.name,
                api_key: item.api_key || null,
                favicon: item.favicon ? $pocketbase.getFileUrl(item, item.favicon) : ''
            }));
        } catch (error) {
            console.error('Error fetching clients:', error);
        }
    }

    async function fetchScans() {
        try {
            const filter = buildFilter();
            
            const result = await $pocketbase.collection('nuclei_scans').getList(currentPage, itemsPerPage, {
                sort: '-start_time,status', // Primary sort by start time desc, secondary by status
                filter: filter,
                expand: 'nuclei_profile,vm_provider,nuclei_interact,client'
            });
            
            totalPages = Math.ceil(result.totalItems / itemsPerPage);
            scans = result.items.map(scan => ({
                id: scan.id,
                name: scan.name,
                status: scan.status,
                destroyed: scan.destroyed || false,
                start_time: scan.start_time,
                end_time: scan.end_time,
                start_time_display: scan.start_time ? new Date(scan.start_time).toLocaleString() : 'N/A',
                end_time_display: scan.end_time ? new Date(scan.end_time).toLocaleString() : 'N/A',
                progress: scan.progress,
                vm_provider: scan.vm_provider,
                vm_provider_name: scan.expand?.vm_provider?.name || 'N/A',
                nuclei_profile: scan.nuclei_profile,
                nuclei_profile_name: scan.expand?.nuclei_profile?.name || 'N/A',
                nuclei_targets: scan.nuclei_targets,
                nuclei_interact: scan.nuclei_interact,
                nuclei_interact_name: scan.expand?.nuclei_interact?.name || 'N/A',
                client: scan.expand?.client ? {
                    id: scan.expand.client.id,
                    name: scan.expand.client.name,
                    api_key: scan.expand.client.api_key || null,
                    favicon: scan.expand.client.favicon ? $pocketbase.getFileUrl(scan.expand.client, scan.expand.client.favicon) : ''
                } : { id: null, name: 'N/A', api_key: null, favicon: '' },
                ansible_logs: scan.ansible_logs,
                state_bucket: scan.state_bucket,
                scan_bucket: scan.scan_bucket,
                ip_address: scan.ip_address,
                cost: scan.cost,
                vm_size: scan.vm_size,
                archived: scan.archived || false,
                vm_start_time: scan.vm_start_time || '',
                vm_stop_time: scan.vm_stop_time || '',
                cost_per_hour: scan.cost_per_hour
            }));
            
            paginatedScans = scans;
            updateRunningCosts();
        } catch (error) {
            console.error('Error fetching scans:', error);
        }
    }

    function buildFilter(): string {
        const filters: string[] = [];

        // Filter by archived status
        if (!showArchivedScans) {
            filters.push('archived = false');
        }

        // Filter by manual scans
        if (!showManualScans) {
            filters.push('status != "Manual"');
        }

        // Filter by destroyed scans
        if (!showDestroyedScans) {
            filters.push('destroyed = false');
        }

        // Filter by status
        if (filterStatuses.length > 0) {
            filters.push(`status ~ "${filterStatuses.join('||')}"`)
        }

        // Filter by client
        if (filterClients.length > 0) {
            filters.push(`client ~ "${filterClients.join('||')}"`)
        }

        // Filter by provider
        if (filterProviders.length > 0) {
            filters.push(`vm_provider ~ "${filterProviders.join('||')}"`)
        }

        return filters.join(' && ');
    }

    function applyFilters() {
        filteredScans = scans.filter(scan => {
            // Filter by archived status
            if (!showArchivedScans && scan.archived) return false;

            // Filter by manual scans
            if (!showManualScans && scan.status === 'Manual') return false;

            // Filter by destroyed scans
            if (!showDestroyedScans && scan.destroyed) return false;

            // Filter by status
            if (filterStatuses.length > 0 && !filterStatuses.includes(scan.status)) return false;

            // Filter by client
            if (filterClients.length > 0 && (!scan.client || !filterClients.includes(scan.client.id))) return false;

            // Filter by provider
            if (filterProviders.length > 0 && (!scan.vm_provider || !filterProviders.includes(scan.vm_provider))) return false;

            return true;
        });

        // Update pagination
        totalPages = Math.ceil(filteredScans.length / itemsPerPage);
        currentPage = 1;
        updatePaginatedScans();
    }

    function updatePaginatedScans() {
        const startIndex = (currentPage - 1) * itemsPerPage;
        const endIndex = startIndex + itemsPerPage;
        paginatedScans = filteredScans.slice(startIndex, endIndex);
    }

    function handleFilterChange(event: CustomEvent<{
        statuses: string[];
        clients: string[];
        providers: string[];
        showManual: boolean;
        showDestroyed: boolean;
        showArchived: boolean;
    }>) {
        const { statuses, clients, providers, showManual, showDestroyed, showArchived } = event.detail;
        
        filterStatuses = statuses || [];
        filterClients = clients || [];
        filterProviders = providers || [];
        showManualScans = showManual;
        showDestroyedScans = showDestroyed;
        showArchivedScans = showArchived;

        currentPage = 1;
        fetchScans();
    }

    async function handlePageChange(event: CustomEvent<number>) {
        currentPage = event.detail;
        await fetchScans(); // Await the fetch to ensure data is updated
    }

    async function openDeleteModal(id: string) {
        currentScanId = id;
        showDeleteScanModal = true;
    }

    function openEditModal(scan: ScanData) {
        currentScanData = { ...scan };
        modalMode = 'edit';
        showAddScanModal = true;
    }

    async function handleScanSave(scanData: ScanFormData) {
        try {
            console.log('handleScanSave called with data:', scanData);
            // Store startImmediately flag and remove it from createData
            const shouldStart = scanData.startImmediately;
            const createData = { ...scanData };
            delete createData.startImmediately;

            // Add initial status
            createData.status = 'Created';

            console.log('Creating scan with data:', createData);

            const response = await $pocketbase.collection('nuclei_scans').create(createData);
            console.log('Scan created:', response);

            // Close the modal immediately after creating the scan
            showAddScanModal = false;

            // If startImmediately is true, start the scan right away
            if (shouldStart && response.id) {
                console.log('Starting scan immediately...');
                const startResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/start`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${userToken}`
                    },
                    body: JSON.stringify({ scan_id: response.id })
                });

                if (!startResponse.ok) {
                    const errorText = await startResponse.text();
                    throw new Error(`Failed to start scan: ${errorText}`);
                }
                console.log('Scan started successfully');
            }

            // Refresh the scans list
            await fetchScans();
        } catch (error) {
            console.error('Error saving scan:', error);
            // Ensure modal is closed even if an error occurs
            showAddScanModal = false;
            throw error;
        }
    }

    async function deleteScan() {
        try {
            await $pocketbase.collection('nuclei_scans').delete(currentScanId);
            fetchScans(); // Refresh the list of scans
            showDeleteScanModal = false; // Close the modal
        } catch (error) {
            console.error('Error deleting scan:', error);
        }
    }

    function openStartModal(scan: ScanData) {
        currentScan = scan;
        showStartScanModal = true;
    }

    function openStopModal(scan: ScanData) {
        currentScan = scan;
        showStopScanModal = true;
    }

    function openResultsModal(scan: ScanData) {
        selectedScanId = scan.id;
        showResultsModal = true;
    }

    function handleManualImport() {
        fetchScans(); // Refresh the scans list after importing
    }

    async function fetchProviderPrice(scan: ScanData): Promise<number> {
        if (!scan.vm_provider || !scan.vm_size) return 0;
        
        try {
            const provider = await $pocketbase.collection('providers').getOne(scan.vm_provider);
            const region = provider.settings?.region;
            if (!region) return 0;

            const response = await fetch(
                `${import.meta.env.VITE_API_BASE_URL}/api/providers/digitalocean/price?providerId=${scan.vm_provider}&region=${region}&size=${scan.vm_size}`,
                {
                    headers: {
                        'Authorization': `Bearer ${userToken}`
                    }
                }
            );

            if (!response.ok) return 0;
            const data = await response.json();
            return data.price_hourly || 0;
        } catch (error) {
            console.error('Error fetching provider price:', error);
            return 0;
        }
    }

    async function calculateCost(scan: ScanData): Promise<number> {
        if (!scan.vm_start_time) return 0;
        
        // For finished scans without a cost, calculate it
        if (scan.status === 'Finished' && !scan.cost) {
            // Validate both timestamps exist
            if (!scan.vm_stop_time || !scan.vm_start_time) return 0;
            
            const endTime = new Date(scan.vm_stop_time);
            const startTime = new Date(scan.vm_start_time);
            
            // Validate dates are valid
            if (isNaN(endTime.getTime()) || isNaN(startTime.getTime())) return 0;
            
            const durationHours = (endTime.getTime() - startTime.getTime()) / (1000 * 60 * 60);
            
            // Ensure duration is positive
            if (durationHours < 0) return 0;
            
            // Get hourly cost
            const hourlyRate = scan.cost_per_hour || await fetchProviderPrice(scan);
            return durationHours * hourlyRate;
        }
        
        // If scan has a final cost and is not running, use it
        if (scan.cost && !['Started', 'Generating', 'Deploying', 'Running'].includes(scan.status)) {
            return scan.cost;
        }
        
        // Calculate duration for running scans
        const endTime = scan.vm_stop_time ? new Date(scan.vm_stop_time) : new Date();
        const startTime = new Date(scan.vm_start_time);
        
        // Validate dates are valid
        if (isNaN(endTime.getTime()) || isNaN(startTime.getTime())) return 0;
        
        const durationHours = (endTime.getTime() - startTime.getTime()) / (1000 * 60 * 60);
        
        // Ensure duration is positive
        if (durationHours < 0) return 0;
        
        // Get hourly cost
        const hourlyRate = scan.cost_per_hour || await fetchProviderPrice(scan);
        return durationHours * hourlyRate;
    }

    async function updateRunningCosts() {
        if (!scans) return;
        
        const updatedScans = await Promise.all(
            scans.map(async (scan) => ({
                ...scan,
                calculatedCost: await calculateCost(scan)
            }))
        );
        
        scans = updatedScans;
    }

    function openTerminalModal(scan: ScanData) {
        if (!scan.id) return;
        currentScanId = scan.id;
        showTerminalModal = true;
    }

    async function destroyScan() {
        try {
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/destroy`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${userToken}`,
                },
                body: JSON.stringify({ scan_id: currentScan.id }),
            });

            if (!response.ok) {
                throw new Error('Failed to destroy scan');
            }

            fetchScans();
        } catch (error) {
            console.error('Error destroying scan:', error);
        }
    }

    function openDestroyModal(scan: ScanData) {
        currentScan = scan;
        showDestroyModal = true;
    }

    function openArchiveModal(scan: ScanData) {
        currentScan = scan;
        showArchiveModal = true;
    }

    async function archiveScan() {
        try {
            if (!currentScan?.id) return;
            
            await $pocketbase.collection('nuclei_scans').update(currentScan.id, {
                archived: true,
                archived_date: new Date().toISOString()
            });

            fetchScans();
            showArchiveModal = false;
        } catch (error) {
            console.error('Error archiving scan:', error);
        }
    }

    function handleSelectAll(event: Event) {
        const target = event.target as HTMLInputElement;
        selectAll = target.checked;
        if (selectAll) {
            selectedScans = paginatedScans.map(scan => scan.id).filter(id => id !== undefined) as string[];
        } else {
            selectedScans = [];
        }
    }

    function handleScanSelect(scanId: string, event: Event) {
        const target = event.target as HTMLInputElement;
        if (target.checked) {
            selectedScans = [...selectedScans, scanId];
        } else {
            selectedScans = selectedScans.filter(id => id !== scanId);
            selectAll = false;
        }
    }

    async function bulkArchive() {
        try {
            await Promise.all(
                selectedScans.map(scanId => 
                    $pocketbase.collection('nuclei_scans').update(scanId, {
                        archived: true,
                        archived_date: new Date().toISOString()
                    })
                )
            );
            fetchScans();
            selectedScans = [];
            selectAll = false;
        } catch (error) {
            console.error('Error archiving scans:', error);
        }
    }

    async function bulkStop() {
        try {
            await Promise.all(
                selectedScans.map(scanId =>
                    fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/stop`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': `Bearer ${userToken}`,
                        },
                        body: JSON.stringify({ scan_id: scanId }),
                    })
                )
            );
            fetchScans();
            selectedScans = [];
            selectAll = false;
        } catch (error) {
            console.error('Error stopping scans:', error);
        }
    }

    $: canArchive = selectedScans.length > 0 && selectedScans.every(id => {
        const scan = scans.find(s => s.id === id);
        return scan && !scan.archived && ['Created', 'Finished', 'Stopped', 'Failed'].includes(scan.status);
    });

    $: canStop = selectedScans.length > 0 && selectedScans.every(id => {
        const scan = scans.find(s => s.id === id);
        return scan && ['Started', 'Generating', 'Deploying', 'Running'].includes(scan.status);
    });

    function openLogModal(scan: ScanData) {
        currentScan = scan;
        showLogModal = true;
    }

    async function startScan() {
        try {
            if (!currentScan?.id) return;
            
            const startResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/start`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${userToken}`,
                },
                body: JSON.stringify({ scan_id: currentScan.id }),
            });

            if (!startResponse.ok) {
                const responseText = await startResponse.text();
                throw new Error(`Failed to start scan: ${responseText}`);
            }

            fetchScans();
            showStartScanModal = false;
        } catch (error) {
            console.error('Error starting scan:', error);
        }
    }

    async function stopScan() {
        try {
            if (!currentScan?.id) return;
            
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/stop`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${userToken}`,
                },
                body: JSON.stringify({ scan_id: currentScan.id }),
            });

            if (!response.ok) {
                throw new Error('Failed to stop scan');
            }

            fetchScans();
            showStopScanModal = false;
        } catch (error) {
            console.error('Error stopping scan:', error);
        }
    }

    onMount(() => {
        fetchScans();
        fetchProviders();
        fetchClients();
        
        // Update costs and refresh scans every minute
        const updateInterval = setInterval(() => {
            updateRunningCosts();
            fetchScans();
        }, 60000);
        
        // Initial cost update
        updateRunningCosts();
        
        return () => {
            clearInterval(updateInterval);
        };
    });
</script>

<MetaTag {path} {description} {title} {subtitle} />

<main class="p-4">
    <Breadcrumb class="mb-6">
        <BreadcrumbItem home>Home</BreadcrumbItem>
        <BreadcrumbItem href="/scans">Scans</BreadcrumbItem>
    </Breadcrumb>

    <Heading tag="h1" class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl mb-4">
        Scans
    </Heading>

    <div class="flex gap-2 mb-4">
        <Button on:click={() => {
            modalMode = 'add';
            currentScanData = null;
            showAddScanModal = true;
        }}>Add Scan</Button>

        <Button on:click={() => showManualScanModal = true}>
            Manually Add Scan JSON
        </Button>
    </div>

    <div class="mb-4">
        <ScanFilters
            bind:showDestroyedScans
            bind:showManualScans
            bind:showArchivedScans
            {providers}
            {clients}
            on:filterChange={handleFilterChange}
        />
    </div>

    <!-- Add bulk actions menu after the filters -->
    <div class="mb-4 flex gap-2">
        {#if selectedScans.length > 0}
            {#if canArchive}
                <Button color="red" size="sm" on:click={bulkArchive}>
                    Archive Selected ({selectedScans.length})
                </Button>
            {/if}
            {#if canStop}
                <Button color="red" size="sm" on:click={bulkStop}>
                    Stop Selected ({selectedScans.length})
                </Button>
            {/if}
        {/if}
    </div>

    <Table class="border border-gray-200 dark:border-gray-700">
        <TableHead class="bg-gray-100 dark:bg-gray-700">
            <TableHeadCell class="w-4 p-4">
                <Checkbox on:change={handleSelectAll} checked={selectAll} />
            </TableHeadCell>
            {#each ['Name', 'Status', 'Start Time', 'End Time', 'Profile', 'Provider', 'Client', 'IP Address', 'Cost', 'Actions'] as title}
                <TableHeadCell class="ps-4 font-normal">{title}</TableHeadCell>
            {/each}
        </TableHead>
        <TableBody>
            {#each paginatedScans as scan (scan.id)}
                <TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
                    <TableBodyCell class="w-4 p-4">
                        <Checkbox 
                            checked={selectedScans.includes(scan.id)} 
                            on:change={(e) => handleScanSelect(scan.id, e)} 
                        />
                    </TableBodyCell>
                    <TableBodyCell class="p-4">{scan.name}</TableBodyCell>
                    <TableBodyCell class="p-4">
                        {#if scan.status}
                            <StatusBadge 
                                state={scan.status} 
                                destroyed={scan.destroyed} 
                                end_time={scan.end_time}
                            />
                        {:else}
                            <span>Unknown Status</span>
                        {/if}
                    </TableBodyCell>
                    <TableBodyCell class="p-4">
                        <div class="flex flex-col">
                            <span>{scan.start_time_display?.split(', ')[0]}</span>
                            <span>{scan.start_time_display?.split(', ')[1]}</span>
                        </div>
                    </TableBodyCell>
                    <TableBodyCell class="p-4">
                        <div class="flex flex-col">
                            <span>{scan.end_time_display?.split(', ')[0]}</span>
                            <span>{scan.end_time_display?.split(', ')[1]}</span>
                        </div>
                    </TableBodyCell>
                    <TableBodyCell class="p-4">{scan.nuclei_profile_name || 'N/A'}</TableBodyCell>
                    <TableBodyCell class="p-4">
                        {#if scan.vm_provider}
                            <div class="flex items-center gap-2">
                                {#if providers.find(p => p.id === scan.vm_provider)?.provider_type === 'digitalocean'}
                                    <SiDigitalocean class="w-6 h-6" />
                                {:else if providers.find(p => p.id === scan.vm_provider)?.provider_type === 'aws'}
                                    <SiAmazon class="w-6 h-6" />
                                {:else if providers.find(p => p.id === scan.vm_provider)?.provider_type === 's3'}
                                    <SiAmazons3 class="w-6 h-6" />
                                {/if}
                                <span>{scan.vm_provider_name || 'N/A'}</span>
                            </div>
                        {:else}
                            <span>N/A</span>
                        {/if}
                    </TableBodyCell>
                    <TableBodyCell class="p-4">
                        <div class="flex items-center">
                            {#if scan.client?.favicon}
                                <img src={scan.client.favicon} alt={scan.client.name} class="w-6 h-6 mr-2" />
                            {/if}
                            <span>{scan.client?.name || 'N/A'}</span>
                        </div>
                    </TableBodyCell>
                    <TableBodyCell class="p-4">{scan.ip_address || 'N/A'}</TableBodyCell>
                    <TableBodyCell class="p-4">
                        {#if ['Started', 'Generating', 'Deploying', 'Running', 'Failed', 'Finished'].includes(scan.status) && !scan.cost}
                            {#await calculateCost(scan)}
                                <span class="text-gray-500">Calculating...</span>
                            {:then cost}
                                <span class="text-green-600">${cost.toFixed(2)}</span>
                            {:catch error}
                                <span class="text-red-500" title={error.message}>Error calculating cost</span>
                            {/await}
                        {:else if scan.cost}
                            <span class="text-green-600">${scan.cost.toFixed(2)}</span>
                        {:else}
                            <span class="text-gray-500">N/A</span>
                        {/if}
                    </TableBodyCell>
                    <TableBodyCell class="space-x-1 p-2 flex flex-wrap gap-1">
                        {#if scan.status === 'Created'}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" class="gap-1 px-2" on:click={() => openStartModal(scan)}>Start</Button>
                            <Button size="xs" class="gap-1 px-2" on:click={() => openEditModal(scan)}>Edit</Button>
                            <Button color="red" size="xs" class="gap-1 px-2" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                        {:else if ['Started', 'Generating', 'Deploying', 'Running'].includes(scan.status)}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" class="gap-1 px-2" on:click={() => openTerminalModal(scan)}>
                                <TerminalSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" class="gap-1 px-2" on:click={() => openStopModal(scan)}>Stop</Button>
                        {:else if scan.status === 'Manual'}
                            <Button color="red" size="xs" class="gap-1 px-2" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                        {:else if ['Finished', 'Stopped'].includes(scan.status)}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            {#if scan.status === 'Finished'}
                                <Button size="xs" class="gap-1 px-2" on:click={() => openResultsModal(scan)}>
                                    Results
                                </Button>
                            {/if}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openEditModal(scan)}>
                                Copy
                            </Button>
                            <Button color="red" size="xs" class="gap-1 px-2" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                        {:else if scan.status === 'Failed' && !scan.destroyed}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" class="gap-1 px-2" on:click={() => openTerminalModal(scan)}>
                                <TerminalSolid class="w-4 h-4" />
                            </Button>
                            <Button color="red" size="xs" class="gap-1 px-2" on:click={() => openDestroyModal(scan)}>
                                Destroy
                            </Button>
                        {:else if scan.status === 'Failed' && scan.destroyed}
                            <Button size="xs" class="gap-1 px-2" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button color="red" size="xs" class="gap-1 px-2" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                        {/if}
                    </TableBodyCell>
                </TableBodyRow>
            {/each}
        </TableBody>
    </Table>

    <!-- Pagination -->
    {#if totalPages > 1}
        <div class="flex justify-center mt-4">
            <Pagination
                {totalPages}
                bind:currentPage
                on:pageChange={handlePageChange}
                showFirstLast={true}
            />
        </div>
    {/if}

    <Delete bind:open={showDeleteScanModal} onDelete={deleteScan} />
    <Archive bind:open={showArchiveModal} onArchive={archiveScan} />
    <ScanForm 
        bind:open={showAddScanModal} 
        onSave={handleScanSave} 
        scan={currentScan} 
    />
    <StartScan bind:open={showStartScanModal} onStart={startScan} />
    <StopScan bind:open={showStopScanModal} onStop={stopScan} />
    <LogModal bind:open={showLogModal} scanId={currentScan?.id || ''} />
    <ResultsModal bind:open={showResultsModal} scanId={currentScan?.id || ''} />
    <TerminalModal bind:open={showTerminalModal} scanId={currentScan?.id || ''} />
    <DestroyScan bind:open={showDestroyModal} onDestroy={destroyScan} />

    <!-- Include the ManualScanModal component -->
    <ManualScanModal bind:open={showManualScanModal} on:import={handleManualImport} />

</main>


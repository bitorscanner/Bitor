<script lang="ts">
    import { Breadcrumb, BreadcrumbItem, Heading, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Checkbox, Button, Pagination } from 'flowbite-svelte';
    import ScanForm from './ScanForm.svelte';
    import Delete from './Delete.svelte';
    import { onMount, onDestroy } from 'svelte';
    import MetaTag from '@utils/MetaTag.svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import StartScan from './StartScan.svelte';
    import StopScan from './StopScan.svelte';
    import LogModal from './LogModal.svelte';
    import { BookOpenSolid, TerminalSolid } from 'flowbite-svelte-icons';
    import StatusBadge from '@utils/dashboard/StatusBadge.svelte';
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
    let currentScanData: Record<string, unknown> = {};
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
    let showMyScansOnly = true;
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

    // Add current user ID with proper type check
    let currentUserId = $pocketbase.authStore.model?.id ?? '';

    let userCache: Record<string, { name: string; username: string }> = {};

    const path: string = '/scans';
    const description: string = 'Nuclei scan management';
    const title: string = 'Nuclei Scans';
    const subtitle: string = 'Manage your nuclei scans';

    interface Client {
        id: string;
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
        ip_address?: string;
        cost?: number;
        vm_size?: string;
        archived?: boolean;
        vm_start_time?: string;
        vm_stop_time?: string;
        cost_per_hour?: number;
        created_by?: string;
        created_by_name?: string;
        start_time_display?: string;
        end_time_display?: string;
        scan_profile?: string;
        frequency?: string;
        cron?: string;
        startImmediately?: boolean;
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

    async function resolveUserName(userId: string): Promise<string> {
        try {
            // Check cache first
            if (userCache[userId]) {
                return userCache[userId].name || userCache[userId].username;
            }

            // Fetch user data if not in cache
            const user = await $pocketbase.collection('users').getOne(userId);
            userCache[userId] = {
                name: user.name,
                username: user.username
            };
            return user.name || user.username;
        } catch (error) {
            console.error('Error resolving user name:', error);
            return 'Unknown';
        }
    }

    async function fetchScans() {
        try {
            const filter = buildFilter();
            console.log('Fetching scans with filter:', filter, 'page:', currentPage);
            
            const result = await $pocketbase.collection('nuclei_scans').getList(currentPage, itemsPerPage, {
                sort: '-start_time,status',
                filter: filter,
                expand: 'nuclei_profile,vm_provider,nuclei_interact,client'
            });
            
            totalPages = Math.ceil(result.totalItems / itemsPerPage);
            console.log('Total pages:', totalPages, 'Current page:', currentPage);
            
            // Create a set of unique user IDs
            const userIds = new Set(result.items.map(scan => scan.created_by));
            
            // Fetch all user names in parallel
            const userPromises = Array.from(userIds).map(async userId => {
                const name = await resolveUserName(userId);
                return [userId, name];
            });
            
            // Wait for all user names to be resolved
            const userNames = Object.fromEntries(await Promise.all(userPromises));
            
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
                cost_per_hour: scan.cost_per_hour,
                created_by: scan.created_by || 'Unknown',
                created_by_name: userNames[scan.created_by] || 'Unknown'
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

        // Filter by user's scans
        if (showMyScansOnly && currentUserId) {
            filters.push(`created_by = "${currentUserId}"`);
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

    function handleFilterChange(event: CustomEvent<{
        statuses: string[];
        clients: string[];
        providers: string[];
        showManual: boolean;
        showDestroyed: boolean;
        showArchived: boolean;
        showMyScansOnly: boolean;
    }>) {
        filterStatuses = event.detail.statuses;
        filterClients = event.detail.clients;
        filterProviders = event.detail.providers;
        showManualScans = event.detail.showManual;
        showDestroyedScans = event.detail.showDestroyed;
        showArchivedScans = event.detail.showArchived;
        showMyScansOnly = event.detail.showMyScansOnly;
        currentPage = 1; // Reset to first page when filters change
        fetchScans();
    }

    async function handlePageChange(event: CustomEvent<number>) {
        console.log('Page change event received:', event);
        console.log('Current page before:', currentPage);
        console.log('New page:', event.detail);
        currentPage = event.detail;
        console.log('Current page after:', currentPage);
        await fetchScans();
    }

    async function openDeleteModal(id: string) {
        if (id) {
            currentScanId = id;
            showDeleteScanModal = true;
        }
    }

    function openEditModal(scan: ScanData | null) {
        if (scan) {
            currentScanData = { ...scan };
            modalMode = 'edit';
            showAddScanModal = true;
        }
    }

    async function handleScanSave(scanData: ScanFormData) {
        try {
            console.log('handleScanSave called with data:', scanData);
            // Store startImmediately flag and remove it from createData
            const shouldStart = scanData.startImmediately;
            const createData = { ...scanData } as Record<string, any>;
            delete createData.startImmediately;

            // Add initial status and creator
            createData.status = 'Created';
            createData.created_by = $pocketbase.authStore.model?.id || 'Unknown';

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

    function openStartModal(scan: ScanData | null) {
        if (scan) {
            currentScan = scan;
            showStartScanModal = true;
        }
    }

    function openStopModal(scan: ScanData) {
        currentScan = scan;
        showStopScanModal = true;
    }

    function openResultsModal(scan: ScanData) {
        console.log('Opening results modal for scan:', scan);
        currentScan = scan;
        console.log('currentScan set to:', currentScan);
        showResultsModal = true;
    }

    function handleManualImport() {
        fetchScans(); // Refresh the scans list after importing
    }

    async function fetchProviderPrice(scan: ScanData): Promise<number> {
        if (!scan.vm_provider || !scan.vm_size) {
            console.log('Missing vm_provider or vm_size:', { vm_provider: scan.vm_provider, vm_size: scan.vm_size });
            return 0;
        }
        
        try {
            console.log('Fetching provider details for:', scan.vm_provider);
            const provider = await $pocketbase.collection('providers').getOne(scan.vm_provider);
            if (!provider) {
                console.error('Provider not found:', scan.vm_provider);
                return 0;
            }

            console.log('Provider details:', provider);
            const region = provider.settings?.region;
            if (!region) {
                console.error('Region not found in provider settings:', provider.settings);
                return 0;
            }

            const url = `${import.meta.env.VITE_API_BASE_URL}/api/providers/${provider.provider_type}/price?providerId=${scan.vm_provider}&region=${region}&size=${scan.vm_size}`;
            console.log('Fetching price from URL:', url);
            
            const response = await fetch(url, {
                headers: {
                    'Authorization': `Bearer ${userToken}`
                }
            });

            if (!response.ok) {
                const errorText = await response.text();
                console.error('Error fetching price:', errorText);
                return 0;
            }
            
            const data = await response.json();
            console.log('Price data received:', data);
            return data.price_hourly || 0;
        } catch (error) {
            console.error('Error fetching provider price:', error);
            return 0;
        }
    }

    async function calculateCost(scan: ScanData): Promise<number> {
        console.log('Calculating cost for scan:', {
            id: scan.id,
            status: scan.status,
            vm_start_time: scan.vm_start_time,
            vm_stop_time: scan.vm_stop_time,
            cost: scan.cost,
            cost_per_hour: scan.cost_per_hour
        });

        // If scan has a final cost and is completed, use the stored cost
        if (scan.cost && ['Finished', 'Failed', 'Stopped'].includes(scan.status)) {
            console.log('Using stored cost for completed scan:', scan.cost);
            return scan.cost;
        }

        // Only proceed with calculation if we have a start time
        if (!scan.vm_start_time) {
            console.log('No vm_start_time, returning 0');
            return 0;
        }
        
        // Get hourly cost if not already set
        const hourlyRate = scan.cost_per_hour || await fetchProviderPrice(scan);
        console.log('Hourly rate:', hourlyRate);
        if (!hourlyRate) {
            console.log('No hourly rate available');
            return 0;
        }

        // For finished scans without a cost, calculate and store it
        if ((scan.status === 'Finished' || scan.status === 'Failed' || scan.status === 'Stopped') && !scan.cost && scan.vm_stop_time) {
            const endTime = new Date(scan.vm_stop_time);
            const startTime = new Date(scan.vm_start_time);
            
            // Validate dates are valid
            if (isNaN(endTime.getTime()) || isNaN(startTime.getTime())) {
                console.log('Invalid dates:', { startTime, endTime });
                return 0;
            }
            
            // Calculate duration in milliseconds first for better precision
            const durationMs = endTime.getTime() - startTime.getTime();
            const durationHours = durationMs / (1000 * 60 * 60);
            console.log('Duration in hours:', durationHours);
            
            // Ensure duration is positive
            if (durationHours < 0) {
                console.log('Negative duration, returning 0');
                return 0;
            }
            
            // Use precise calculation
            const finalCost = Number((durationHours * hourlyRate).toFixed(4));
            console.log('Calculated final cost:', finalCost);

            // Update the cost in the database
            try {
                console.log('Updating cost in database:', {
                    scanId: scan.id,
                    finalCost,
                    hourlyRate
                });
                await $pocketbase.collection('nuclei_scans').update(scan.id, {
                    cost: finalCost,
                    cost_per_hour: hourlyRate
                });
                console.log('Successfully updated cost in database');
            } catch (error) {
                console.error('Error updating scan cost:', error);
            }

            return finalCost;
        }
        
        // Only calculate current cost for running scans
        if (['Started', 'Generating', 'Deploying', 'Running'].includes(scan.status)) {
            const endTime = new Date();
            const startTime = new Date(scan.vm_start_time);
            
            // Validate dates are valid
            if (isNaN(endTime.getTime()) || isNaN(startTime.getTime())) {
                console.log('Invalid dates for running scan:', { startTime, endTime });
                return 0;
            }
            
            // Calculate duration in milliseconds first for better precision
            const durationMs = endTime.getTime() - startTime.getTime();
            const durationHours = durationMs / (1000 * 60 * 60);
            console.log('Running scan duration in hours:', durationHours);
            
            // Ensure duration is positive
            if (durationHours < 0) {
                console.log('Negative duration for running scan, returning 0');
                return 0;
            }
            
            // Use precise calculation
            const currentCost = Number((durationHours * hourlyRate).toFixed(4));
            console.log('Calculated current cost for running scan:', currentCost);
            return currentCost;
        }

        return scan.cost || 0;
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
            if (!currentScan?.id) return;
            
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

    function openArchiveModal(scan: ScanData | null) {
        if (scan) {
            currentScan = scan;
            showArchiveModal = true;
        }
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

    async function bulkDelete() {
        try {
            await Promise.all(
                selectedScans.map(scanId => 
                    $pocketbase.collection('nuclei_scans').delete(scanId)
                )
            );
            fetchScans();
            selectedScans = [];
            selectAll = false;
        } catch (error) {
            console.error('Error deleting scans:', error);
        }
    }

    async function bulkDestroy() {
        try {
            await Promise.all(
                selectedScans.map(scanId =>
                    fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/destroy`, {
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
            console.error('Error destroying scans:', error);
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

    $: canDelete = $pocketbase.authStore.isAdmin && selectedScans.length > 0;

    $: canDestroy = selectedScans.length > 0 && selectedScans.every(id => {
        const scan = scans.find(s => s.id === id);
        return scan && scan.status === 'Failed' && !scan.destroyed && scan.ip_address;
    });

    function openLogModal(scan: ScanData | null) {
        if (scan) {
            currentScan = scan;
            showLogModal = true;
        }
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
            currentScanData = {};
            showAddScanModal = true;
        }}>Add Scan</Button>

        <Button on:click={() => showManualScanModal = true}>
            Manually Add Scan JSON
        </Button>
    </div>

    <div class="mb-4">
        <ScanFilters
            {clients}
            {providers}
            bind:showManualScans
            bind:showDestroyedScans
            bind:showArchivedScans
            bind:showMyScansOnly
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
            {#if canDestroy}
                <Button color="red" size="sm" on:click={bulkDestroy}>
                    Destroy Selected ({selectedScans.length})
                </Button>
            {/if}
            {#if canDelete}
                <Button color="red" size="sm" on:click={bulkDelete}>
                    Delete Selected ({selectedScans.length})
                </Button>
            {/if}
        {/if}
    </div>

    <Table class="border border-gray-200 dark:border-gray-700">
        <TableHead class="bg-gray-100 dark:bg-gray-700">
            <TableHeadCell class="w-4 p-4">
                <Checkbox on:change={handleSelectAll} checked={selectAll} />
            </TableHeadCell>
            {#each ['Name', 'Status', 'Start Time', 'End Time', 'Profile', 'Provider', 'Client', 'IP Address', 'Cost', 'Created By', 'Actions'] as title}
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
                                    <SiDigitalocean size={24} />
                                {:else if providers.find(p => p.id === scan.vm_provider)?.provider_type === 'aws'}
                                    <SiAmazon size={24} />
                                {:else if providers.find(p => p.id === scan.vm_provider)?.provider_type === 's3'}
                                    <SiAmazons3 size={24} />
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
                                <span class="text-green-600">${cost.toFixed(4)}</span>
                            {:catch error}
                                <span class="text-red-500" title={error.message}>Error calculating cost</span>
                            {/await}
                        {:else if scan.cost}
                            <span class="text-green-600">${scan.cost.toFixed(4)}</span>
                        {:else}
                            <span class="text-gray-500">N/A</span>
                        {/if}
                    </TableBodyCell>
                    <TableBodyCell class="p-4">{scan.created_by_name}</TableBodyCell>
                    <TableBodyCell class="space-x-1 p-2 flex flex-wrap gap-1">
                        {#if scan.status === 'Created'}
                            <Button size="xs" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" on:click={() => openStartModal(scan)}>Start</Button>
                            <Button size="xs" on:click={() => openEditModal(scan)}>Edit</Button>
                            <Button color="red" size="xs" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                            {#if $pocketbase.authStore.isAdmin}
                                <Button color="red" size="xs" on:click={() => openDeleteModal(scan.id)}>
                                    Delete
                                </Button>
                            {/if}
                        {:else if ['Started', 'Generating', 'Deploying', 'Running'].includes(scan.status)}
                            <Button size="xs" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" on:click={() => openTerminalModal(scan)}>
                                <TerminalSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" on:click={() => openStopModal(scan)}>Stop</Button>
                            {#if $pocketbase.authStore.isAdmin}
                                <Button color="red" size="xs" on:click={() => openDeleteModal(scan.id)}>
                                    Delete
                                </Button>
                            {/if}
                        {:else if scan.status === 'Manual'}
                            <Button color="red" size="xs" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                            <Button size="xs" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" on:click={() => openResultsModal(scan)}>
                                Results
                            </Button>
                            <Button size="xs" on:click={() => openEditModal(scan)}>
                                Copy
                            </Button>
                            <Button color="red" size="xs" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                            {#if $pocketbase.authStore.isAdmin}
                                <Button color="red" size="xs" on:click={() => openDeleteModal(scan.id)}>
                                    Delete
                                </Button>
                            {/if}
                        {:else if ['Finished', 'Stopped'].includes(scan.status)}
                            <Button size="xs" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            {#if scan.status === 'Finished'}
                                <Button size="xs" on:click={() => openResultsModal(scan)}>
                                    Results
                                </Button>
                            {/if}
                            <Button size="xs" on:click={() => openEditModal(scan)}>
                                Copy
                            </Button>
                            <Button color="red" size="xs" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                            {#if $pocketbase.authStore.isAdmin}
                                <Button color="red" size="xs" on:click={() => openDeleteModal(scan.id)}>
                                    Delete
                                </Button>
                            {/if}
                        {:else if scan.status === 'Failed' && !scan.destroyed}
                            <Button size="xs" on:click={() => openLogModal(scan)}>
                                <BookOpenSolid class="w-4 h-4" />
                            </Button>
                            <Button size="xs" on:click={() => openTerminalModal(scan)}>
                                <TerminalSolid class="w-4 h-4" />
                            </Button>
                            {#if scan.ip_address}
                                <Button color="red" size="xs" on:click={() => openDestroyModal(scan)}>
                                    Destroy
                                </Button>
                            {/if}
                            <Button color="red" size="xs" on:click={() => openArchiveModal(scan)}>
                                Archive
                            </Button>
                            {#if $pocketbase.authStore.isAdmin}
                                <Button color="red" size="xs" on:click={() => openDeleteModal(scan.id)}>
                                    Delete
                                </Button>
                            {/if}
                        {/if}
                    </TableBodyCell>
                </TableBodyRow>
            {/each}
        </TableBody>
    </Table>

    <!-- Pagination -->
    {#if totalPages > 1}
        <div class="flex flex-col items-center gap-2 mt-4">
            <div class="text-sm text-gray-700 dark:text-gray-400">
                Showing page {currentPage} of {totalPages}
            </div>
            <div class="flex items-center gap-2">
                <Button 
                    size="sm"
                    disabled={currentPage === 1}
                    on:click={() => handlePageChange(new CustomEvent('pageChange', { detail: currentPage - 1 }))}
                >
                    Previous
                </Button>
                <span class="mx-2">
                    {#each Array(totalPages) as _, i}
                        <Button 
                            size="sm"
                            color={currentPage === i + 1 ? 'blue' : 'light'}
                            class="mx-1"
                            on:click={() => handlePageChange(new CustomEvent('pageChange', { detail: i + 1 }))}
                        >
                            {i + 1}
                        </Button>
                    {/each}
                </span>
                <Button 
                    size="sm"
                    disabled={currentPage === totalPages}
                    on:click={() => handlePageChange(new CustomEvent('pageChange', { detail: currentPage + 1 }))}
                >
                    Next
                </Button>
            </div>
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
    <ManualScanModal bind:open={showManualScanModal} on:import={handleManualImport} />
</main>


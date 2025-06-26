<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { Button, Input, Label, Modal, Select, Toggle, Heading, Alert } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';

    interface Client {
        id: string;
        name: string;
    }

    export let open = false;
    export let onSuccess: () => void = () => {};

    let targetName = '';
    let selectedClient = '';
    let clients: Client[] = [];
    
    // Attack surface options
    let includeDomains = true;
    let includeSubdomains = true;
    let includePorts = true;
    let includeNetblocks = false;
    let includeURLs = true;
    let onlyWebPorts = true;
    let schemes = ['http', 'https'];
    let customPorts = '';
    let manualTargets = '';
    
    // Derived scheme checkboxes
    let httpEnabled = true;
    let httpsEnabled = true;

    // State
    let loading = false;
    let error = '';
    let success = '';
    let previewData: any = null;
    let showPreview = false;

    const dispatch = createEventDispatcher();

    onMount(async () => {
        try {
            const result = await $pocketbase.collection('clients').getList();
            clients = result.items.map(client => ({
                id: client.id,
                name: client.name
            }));
        } catch (error) {
            console.error('Error fetching clients:', error);
        }
    });

    function resetForm() {
        targetName = '';
        selectedClient = '';
        includeDomains = true;
        includeSubdomains = true;
        includePorts = true;
        includeNetblocks = false;
        includeURLs = true;
        onlyWebPorts = true;
        schemes = ['http', 'https'];
        httpEnabled = true;
        httpsEnabled = true;
        customPorts = '';
        manualTargets = '';
        error = '';
        success = '';
        previewData = null;
        showPreview = false;
    }
    
    // Update schemes when checkboxes change
    function updateSchemes() {
        schemes = [];
        if (httpEnabled) schemes.push('http');
        if (httpsEnabled) schemes.push('https');
    }

    $: if (open && !showPreview) {
        resetForm();
    }

    async function handlePreview() {
        if (!selectedClient) {
            error = 'Please select a client first';
            return;
        }

        loading = true;
        error = '';
        
        try {
            const ports = customPorts ? customPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p)) : [];
            const manualTargetList = manualTargets ? manualTargets.split('\n').filter(t => t.trim() !== '') : [];

            const response = await fetch(`${$pocketbase.baseUrl}api/attack-surface/nuclei/collect-targets`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`,
                },
                body: JSON.stringify({
                    client_id: selectedClient,
                    include_domains: includeDomains,
                    include_subdomains: includeSubdomains,
                    include_ports: includePorts,
                    include_netblocks: includeNetblocks,
                    include_urls: includeURLs,
                    schemes: schemes,
                    ports: ports,
                    only_web_ports: onlyWebPorts,
                    manual_targets: manualTargetList
                })
            });

            const result = await response.json();
            
            if (!result.success) {
                throw new Error(result.message || 'Failed to collect targets');
            }

            previewData = result;
            showPreview = true;
            error = '';
        } catch (err: any) {
            error = err.message || 'Failed to collect targets';
            console.error('Error collecting targets:', err);
        } finally {
            loading = false;
        }
    }

    async function handleCreate() {
        if (!targetName.trim()) {
            error = 'Please enter a target name';
            return;
        }

        loading = true;
        error = '';
        
        try {
            const ports = customPorts ? customPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p)) : [];
            const manualTargetList = manualTargets ? manualTargets.split('\n').filter(t => t.trim() !== '') : [];

            const response = await fetch(`${$pocketbase.baseUrl}api/attack-surface/nuclei/create-target`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`,
                },
                body: JSON.stringify({
                    name: targetName,
                    client_id: selectedClient,
                    include_domains: includeDomains,
                    include_subdomains: includeSubdomains,
                    include_ports: includePorts,
                    include_netblocks: includeNetblocks,
                    include_urls: includeURLs,
                    schemes: schemes,
                    ports: ports,
                    only_web_ports: onlyWebPorts,
                    manual_targets: manualTargetList
                })
            });

            const result = await response.json();
            
            if (!result.success) {
                throw new Error(result.message || 'Failed to create target');
            }

            success = result.message || 'Target created successfully';
            setTimeout(() => {
                open = false;
                onSuccess();
            }, 1500);
        } catch (err: any) {
            error = err.message || 'Failed to create target';
            console.error('Error creating target:', err);
        } finally {
            loading = false;
        }
    }

    function goBack() {
        showPreview = false;
        previewData = null;
    }
</script>

<Modal bind:open size="xl" title="Create Target from Attack Surface">
    {#if !showPreview}
        <!-- Configuration Form -->
        <form on:submit|preventDefault={handlePreview} class="space-y-6">
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div class="space-y-4">
                    <Label class="space-y-2">
                        <span class="text-gray-700 dark:text-gray-300">Target Name</span>
                        <Input bind:value={targetName} placeholder="Enter target name" required />
                    </Label>
                    
                    <Label class="space-y-2">
                        <span class="text-gray-700 dark:text-gray-300">Client</span>
                        <Select bind:value={selectedClient} placeholder="Select a client" required>
                            {#each clients as client}
                                <option value={client.id}>{client.name}</option>
                            {/each}
                        </Select>
                    </Label>

                    <div class="space-y-3">
                        <Heading tag="h4" class="text-lg">Attack Surface Sources</Heading>
                        
                        <div class="space-y-2">
                            <Label class="flex items-center space-x-2">
                                <Toggle bind:checked={includeDomains} />
                                <span>Include Domains</span>
                            </Label>
                            
                            <Label class="flex items-center space-x-2">
                                <Toggle bind:checked={includeSubdomains} />
                                <span>Include Subdomains</span>
                            </Label>
                            
                            <Label class="flex items-center space-x-2">
                                <Toggle bind:checked={includePorts} />
                                <span>Include Open Ports</span>
                            </Label>
                            
                            <Label class="flex items-center space-x-2">
                                <Toggle bind:checked={includeNetblocks} />
                                <span>Include Netblocks</span>
                            </Label>
                            
                            <Label class="flex items-center space-x-2">
                                <Toggle bind:checked={includeURLs} />
                                <span>Include Discovered URLs</span>
                            </Label>
                        </div>
                    </div>
                </div>

                <div class="space-y-4">
                    <div class="space-y-3">
                        <Heading tag="h4" class="text-lg">Options</Heading>
                        
                        <Label class="flex items-center space-x-2">
                            <Toggle bind:checked={onlyWebPorts} />
                            <span>Only Web Ports (80, 443, 8080, etc.)</span>
                        </Label>
                        <p class="text-sm text-gray-500 dark:text-gray-400 ml-6">
                            When disabled, includes all discovered ports (SSH, FTP, databases, etc.) as IP:port targets
                        </p>

                        <Label class="space-y-2">
                            <span class="text-gray-700 dark:text-gray-300">Custom Ports (comma-separated)</span>
                            <Input bind:value={customPorts} placeholder="e.g., 8080,8443,9000" />
                        </Label>

                        <Label class="space-y-2">
                            <span class="text-gray-700 dark:text-gray-300">URL Schemes</span>
                            <div class="flex space-x-4">
                                <Label class="flex items-center space-x-2">
                                    <input type="checkbox" bind:checked={httpEnabled} on:change={updateSchemes} />
                                    <span>HTTP</span>
                                </Label>
                                <Label class="flex items-center space-x-2">
                                    <input type="checkbox" bind:checked={httpsEnabled} on:change={updateSchemes} />
                                    <span>HTTPS</span>
                                </Label>
                            </div>
                        </Label>

                        <Label class="space-y-2">
                            <span class="text-gray-700 dark:text-gray-300">Manual Targets (one per line)</span>
                            <textarea bind:value={manualTargets} rows="4" class="mt-1 block w-full border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white p-2" placeholder="https://example.com&#10;192.168.1.1:8080"></textarea>
                        </Label>
                    </div>
                </div>
            </div>

            {#if error}
                <Alert color="red" class="mt-4">
                    {error}
                </Alert>
            {/if}

            <div class="flex justify-end space-x-2">
                <Button color="light" on:click={() => open = false}>Cancel</Button>
                <Button type="submit" disabled={loading || !selectedClient}>
                    {loading ? 'Loading...' : 'Preview Targets'}
                </Button>
            </div>
        </form>
    {:else}
        <!-- Preview and Create -->
        <div class="space-y-6">
            <div class="flex items-center justify-between">
                <Heading tag="h3" class="text-xl">Target Preview</Heading>
                <Button color="light" on:click={goBack}>← Back to Configuration</Button>
            </div>

            {#if previewData}
                <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                    <div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
                        <h4 class="font-semibold text-blue-800 dark:text-blue-200">Total Targets</h4>
                        <p class="text-2xl font-bold text-blue-600 dark:text-blue-400">{previewData.total_targets}</p>
                    </div>
                    
                    <div class="bg-green-50 dark:bg-green-900/20 p-4 rounded-lg">
                        <h4 class="font-semibold text-green-800 dark:text-green-200">Sources Used</h4>
                        <p class="text-2xl font-bold text-green-600 dark:text-green-400">{Object.keys(previewData.sources || {}).length}</p>
                    </div>
                    
                    <div class="bg-purple-50 dark:bg-purple-900/20 p-4 rounded-lg">
                        <h4 class="font-semibold text-purple-800 dark:text-purple-200">Client</h4>
                        <p class="text-sm text-purple-600 dark:text-purple-400">{clients.find(c => c.id === selectedClient)?.name || 'Unknown'}</p>
                    </div>
                </div>

                <!-- Source Breakdown -->
                <div class="space-y-4">
                    <Heading tag="h4" class="text-lg">Source Breakdown</Heading>
                    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
                        {#if previewData.stats}
                            {#each Object.entries(previewData.stats) as [key, value]}
                                {#if key !== 'total_targets' && Number(value) > 0}
                                    <div class="bg-gray-50 dark:bg-gray-800 p-3 rounded">
                                        <p class="text-sm text-gray-600 dark:text-gray-400 capitalize">{key.replace('_', ' ')}</p>
                                        <p class="text-lg font-semibold">{value}</p>
                                    </div>
                                {/if}
                            {/each}
                        {/if}
                    </div>
                </div>

                <!-- Sample Targets -->
                <div class="space-y-4">
                    <Heading tag="h4" class="text-lg">Sample Targets (first 10)</Heading>
                    <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg max-h-64 overflow-y-auto">
                        <pre class="text-sm whitespace-pre-wrap">{previewData.targets ? previewData.targets.slice(0, 10).join('\n') : 'No targets found'}</pre>
                        {#if previewData.targets && previewData.targets.length > 10}
                            <p class="text-sm text-gray-500 mt-2">... and {previewData.targets.length - 10} more targets</p>
                        {/if}
                    </div>
                </div>
            {/if}

            {#if error}
                <Alert color="red">
                    {error}
                </Alert>
            {/if}

            {#if success}
                <Alert color="green">
                    {success}
                </Alert>
            {/if}

            <div class="flex justify-end space-x-2">
                <Button color="light" on:click={goBack}>Back</Button>
                <Button on:click={handleCreate} disabled={loading || !targetName.trim()}>
                    {loading ? 'Creating...' : 'Create Target'}
                </Button>
            </div>
        </div>
    {/if}
</Modal>

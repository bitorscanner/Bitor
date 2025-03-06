<script lang="ts">
    import { Label, Input, Select, Alert, Button, Badge, Spinner } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider, DigitalOceanSettings } from '../types';
    import { fetchDigitalOceanData } from '../utils/digitalocean';
    import { onMount } from 'svelte';
    import DigitalOceanAPIKeyModal from './DigitalOceanAPIKeyModal.svelte';

    export let provider: Provider;
    export let onSave: (provider: Provider) => void;

    let error = '';
    let success = '';
    let loading = false;
    let isInitialLoading = false;
    let regions: { id: string; name: string }[] = [];
    let projects: { id: string; name: string }[] = [];
    let sizes: { slug: string; memory: number; vcpus: number; disk: number; transfer: number; price_monthly: number }[] = [];
    let domains: { name: string; ttl: number }[] = [];
    let hasApiKey = false;
    let showApiKeyModal = false;
    let newTag = '';

    // Ensure settings object exists and initialize with default values
    if (!provider.settings || provider.provider_type !== 'digitalocean') {
        provider.settings = {
            region: '',
            do_project: '',
            size: '',
            tags: [],
            dns_domain: ''
        } as DigitalOceanSettings;
    }

    const settings = provider.settings as DigitalOceanSettings;

    // Ensure tags is an array
    if (!Array.isArray(settings.tags)) {
        settings.tags = typeof settings.tags === 'string' 
            ? (settings.tags as string).split(',').map((t: string) => t.trim()).filter(Boolean)
            : [];
    }

    function addTag(event: KeyboardEvent) {
        if (event.key === 'Enter' || event.key === ',') {
            event.preventDefault();
            const tag = newTag.trim();
            if (tag && !settings.tags.includes(tag)) {
                settings.tags = [...settings.tags, tag];
                saveSettings();
            }
            newTag = '';
        }
    }

    function removeTag(tagToRemove: string) {
        settings.tags = settings.tags.filter(tag => tag !== tagToRemove);
        saveSettings();
    }

    async function checkApiKey() {
        try {
            console.log('Checking for API key for provider:', provider.id);
            const apiKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "api_key"`,
                sort: '-created'
            });
            console.log('API keys found:', apiKeys.totalItems, apiKeys.items);
            hasApiKey = apiKeys.totalItems > 0;
            console.log('hasApiKey set to:', hasApiKey);
        } catch (e) {
            console.error('Error checking API key:', e);
            hasApiKey = false;
        }
    }

    async function loadDomains() {
        if (!provider?.id) return;
        
        try {
            const baseUrl = `${import.meta.env.VITE_API_BASE_URL}/api/providers/digitalocean`;
            const response = await fetch(`${baseUrl}/domains?providerId=${provider.id}`, {
                headers: {
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText);
            }

            domains = await response.json();
        } catch (e) {
            console.error('Error loading domains:', e);
            error = e.message || 'Failed to load domains';
        }
    }

    async function loadData() {
        isInitialLoading = true;
        loading = true;
        error = '';
        try {
            // Check if provider has an API key
            const apiKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "api_key"`
            });
            hasApiKey = apiKeys.totalItems > 0;

            if (!hasApiKey) {
                loading = false;
                isInitialLoading = false;
                return;
            }

            console.log('Fetching DigitalOcean data...');
            const data = await fetchDigitalOceanData(provider);
            console.log('Fetched data:', data);
            regions = data.regions || [];
            projects = data.projects || [];
            
            // Load domains if dns use is enabled
            if (provider.uses?.includes('dns')) {
                await loadDomains();
            }

            // Set default values if none are set
            if (!settings.region && regions.length > 0) {
                settings.region = regions[0].id;
                await saveSettings();
            }
            if (!settings.do_project && projects.length > 0) {
                settings.do_project = projects[0].name;
                await saveSettings();
            }

            // Only fetch sizes if we have a region
            if (settings.region) {
                await loadSizes();
            }
        } catch (e: any) {
            console.error('Error loading DigitalOcean data:', e);
            error = e.message || 'Failed to load DigitalOcean data';
        } finally {
            loading = false;
            isInitialLoading = false;
            console.log('Final state:', { hasApiKey, loading, error, regions, projects, sizes, domains });
        }
    }

    async function loadSizes() {
        if (!provider?.id || !settings.region) return;
        
        try {
            const baseUrl = `${import.meta.env.VITE_API_BASE_URL}/api/providers/digitalocean`;
            const response = await fetch(`${baseUrl}/sizes?providerId=${provider.id}&region=${settings.region}`, {
                headers: {
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`,
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText);
            }

            sizes = await response.json();
            
            // If we have sizes but no size is selected, set the first one as default
            if (!settings.size && sizes.length > 0) {
                settings.size = sizes[0].slug;
                await saveSettings();
            }
        } catch (e: any) {
            console.error('Error loading sizes:', e);
            if (e.message.includes('max cost')) {
                const settings = await $pocketbase.collection('system_settings').getFirstListItem('');
                error = `Some droplet sizes are hidden due to the maximum monthly cost limit ($${settings?.max_cost_per_month}). Adjust this in System Settings if needed.`;
            } else {
                error = e.message || 'Failed to load droplet sizes';
            }
        }
    }

    async function handleRegionChange() {
        // Clear the current size selection since it might not be available in the new region
        settings.size = '';
        // Save the region change
        await saveSettings();
        // Load the sizes for the new region
        await loadSizes();
    }

    async function saveSettings() {
        if (!provider?.id) return;
        
        // Validate required fields
        if (!settings.region) {
            error = 'Region is required';
            return;
        }
        if (!settings.do_project) {
            error = 'Project is required';
            return;
        }
        if (provider.uses?.includes('compute') && !settings.size) {
            error = 'Droplet size is required for compute usage';
            return;
        }

        error = '';
        try {
            const updated = await $pocketbase.collection('providers').update(provider.id, {
                settings: settings,
                updated: new Date().toISOString()
            });
            
            onSave(updated as unknown as Provider);
            success = 'Settings saved successfully';
        } catch (err: unknown) {
            console.error('Error saving settings:', err);
            if (err instanceof Error) {
                error = err.message;
            } else {
                error = 'Failed to save settings';
            }
        }
    }

    async function handleApiKeyAdded() {
        await checkApiKey();
        showApiKeyModal = false;
        success = 'API key added successfully';
        await loadData();
    }

    $: if (provider.uses?.includes('dns') && hasApiKey && !loading) {
        loadDomains();
    }

    onMount(loadData);
</script>

<div class="space-y-4">
    {#if error}
        <Alert color="red" class="mb-4">
            <span class="font-medium">Error!</span> {error}
        </Alert>
    {/if}

    {#if success}
        <Alert color="green" class="mb-4">
            <span class="font-medium">Success!</span> {success}
        </Alert>
    {/if}

    {#if isInitialLoading}
        <div class="flex flex-col items-center justify-center py-8">
            <Spinner size="8" />
            <p class="mt-4 text-gray-600 dark:text-gray-400">Loading DigitalOcean Configuration...</p>
        </div>
    {:else}
        <div class="flex justify-between items-center">
            <h3 class="text-lg font-medium text-gray-900 dark:text-white">DigitalOcean Configuration</h3>
            <Button color="blue" on:click={() => (showApiKeyModal = true)}>
                {hasApiKey ? 'Update API Key' : 'Add API Key'}
            </Button>
        </div>

        {#if hasApiKey}
            <div class="space-y-4">
                <div>
                    <Label for="do_region" class="mb-2">
                        Region <span class="text-red-500">*</span>
                    </Label>
                    <Select
                        id="do_region"
                        bind:value={settings.region}
                        on:change={handleRegionChange}
                        required
                    >
                        <option value="">Select a region</option>
                        {#each regions as region}
                            <option value={region.id}>{region.name}</option>
                        {/each}
                    </Select>
                </div>

                <div>
                    <Label for="do_project" class="mb-2">
                        Project <span class="text-red-500">*</span>
                    </Label>
                    <Select
                        id="do_project"
                        bind:value={settings.do_project}
                        on:change={saveSettings}
                        required
                    >
                        <option value="">Select a project</option>
                        {#each projects as project}
                            <option value={project.name}>{project.name}</option>
                        {/each}
                    </Select>
                </div>

                <div>
                    <Label for="size" class="mb-2">
                        Droplet Size <span class="text-red-500">*</span>
                    </Label>
                    <div class="text-sm text-gray-600 dark:text-gray-400 mb-2">
                        <span class="font-medium">Note:</span> Available droplet sizes are limited to a maximum monthly cost of $
                        {#await $pocketbase.collection('system_settings').getFirstListItem('') then settings}
                            {settings?.max_cost_per_month || 'loading...'}
                        {/await}
                    </div>
                    <Select
                        id="size"
                        bind:value={settings.size}
                        on:change={saveSettings}
                        required
                    >
                        <option value="">Select a size</option>
                        {#each sizes as size}
                            <option value={size.slug}>
                                {size.slug} ({size.vcpus} vCPUs, {size.memory/1024}GB RAM, {size.disk}GB SSD) - ${size.price_monthly}/mo
                            </option>
                        {/each}
                    </Select>
                </div>

                <div>
                    <Label for="tags" class="mb-2">
                        Tags <span class="text-gray-500 text-sm">(optional)</span>
                    </Label>
                    <div class="flex flex-wrap gap-2 mb-2">
                        {#each settings.tags as tag}
                            <Badge
                                color="blue"
                                class="flex items-center gap-1"
                            >
                                {tag}
                                <button
                                    type="button"
                                    class="ml-1 hover:text-red-500"
                                    on:click={() => removeTag(tag)}
                                >
                                    ×
                                </button>
                            </Badge>
                        {/each}
                    </div>
                    <Input
                        id="tags"
                        type="text"
                        placeholder="Type a tag and press Enter (optional)"
                        bind:value={newTag}
                        on:keydown={addTag}
                    />
                </div>

                {#if provider.uses?.includes('dns')}
                    <div>
                        <Label for="dns_domain" class="mb-2">
                            Domain <span class="text-red-500">*</span>
                        </Label>
                        <Select
                            id="dns_domain"
                            bind:value={settings.dns_domain}
                            on:change={saveSettings}
                            required
                        >
                            <option value="">Select a domain</option>
                            {#each domains as domain}
                                <option value={domain.name}>{domain.name}</option>
                            {/each}
                        </Select>
                    </div>
                {/if}
            </div>
        {/if}
    {/if}

    <DigitalOceanAPIKeyModal
        bind:show={showApiKeyModal}
        {provider}
        onSave={handleApiKeyAdded}
        onClose={() => showApiKeyModal = false}
    />
</div> 
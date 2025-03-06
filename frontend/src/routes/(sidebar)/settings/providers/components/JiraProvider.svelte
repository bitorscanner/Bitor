<script lang="ts">
    import { Label, Input, Select, Alert, Button, Badge, Spinner } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider, JiraSettings } from '../types';
    import { onMount } from 'svelte';
    import JiraCredentialsModal from './JiraCredentialsModal.svelte';
    import JiraClientMappings from './JiraClientMappings.svelte';

    export let provider: Provider;
    export let onSave: (provider: Provider) => void;

    let error = '';
    let success = '';
    let loading = false;
    let isInitialLoading = false;
    let projects: { key: string; name: string }[] = [];
    let issueTypes: { id: string; name: string }[] = [];
    let hasCredentials = false;
    let showCredentialsModal = false;
    let organizations: { id: string; name: string }[] = [];
    let clients: { id: string; name: string }[] = [];

    // Ensure settings object exists and initialize with default values
    if (!provider.settings || provider.provider_type !== 'jira') {
        provider.settings = {
            jira_url: '',
            project_key: '',
            issue_type: '',
            client_mappings: []
        } as JiraSettings;
    }

    const settings = provider.settings as JiraSettings;

    async function checkCredentials() {
        try {
            const [usernameKeys, apiKeys] = await Promise.all([
                $pocketbase.collection('api_keys').getList(1, 1, {
                    filter: `provider = "${provider.id}" && key_type = "username"`,
                    sort: '-created'
                }),
                $pocketbase.collection('api_keys').getList(1, 1, {
                    filter: `provider = "${provider.id}" && key_type = "api_key"`,
                    sort: '-created'
                })
            ]);
            hasCredentials = usernameKeys.totalItems > 0 && apiKeys.totalItems > 0;
        } catch (e) {
            console.error('Error checking credentials:', e);
            hasCredentials = false;
        }
    }

    async function loadData() {
        isInitialLoading = true;
        loading = true;
        error = '';
        try {
            await checkCredentials();
            await loadClients();

            if (!hasCredentials) {
                loading = false;
                isInitialLoading = false;
                return;
            }

            if (settings.jira_url && settings.project_key) {
                await Promise.all([
                    fetchProjects(),
                    fetchIssueTypes(),
                    fetchOrganizations()
                ]);
            } else if (settings.jira_url) {
                await fetchProjects();
            }
        } catch (e: any) {
            console.error('Error loading Jira data:', e);
            error = e.message || 'Failed to load Jira data';
        } finally {
            loading = false;
            isInitialLoading = false;
        }
    }

    async function loadClients() {
        try {
            const records = await $pocketbase.collection('clients').getFullList();
            clients = records.map(record => ({
                id: record.id,
                name: record.name
            }));
        } catch (e) {
            console.error('Error loading clients:', e);
            error = 'Failed to load clients';
        }
    }

    async function fetchProjects() {
        try {
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/projects`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    url: settings.jira_url,
                    provider_id: provider.id
                })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to fetch projects');
            }

            const data = await response.json();
            projects = data.projects;
        } catch (e: any) {
            console.error('Error fetching projects:', e);
            error = e.message || 'Failed to fetch projects';
            projects = [];
        }
    }

    async function fetchIssueTypes() {
        try {
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/issuetypes`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    url: settings.jira_url,
                    project_key: settings.project_key,
                    provider_id: provider.id
                })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to fetch issue types');
            }

            const data = await response.json();
            // Initialize as empty array if data.issueTypes is undefined
            issueTypes = data.issueTypes || [];
            console.log('Received issue types:', issueTypes); // Debug log
        } catch (e: any) {
            console.error('Error fetching issue types:', e);
            error = e.message || 'Failed to fetch issue types';
            issueTypes = [];
        }
    }

    async function fetchOrganizations() {
        try {
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/organizations`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    url: settings.jira_url,
                    project_key: settings.project_key,
                    provider_id: provider.id
                })
            });

            if (!response.ok) {
                const data = await response.json();
                throw new Error(data.error || 'Failed to fetch organizations');
            }

            const data = await response.json();
            organizations = data.organizations || [];
            console.log('Received organizations:', organizations); // Debug log
        } catch (e: any) {
            console.error('Error fetching organizations:', e);
            error = e.message || 'Failed to fetch organizations';
            organizations = [];
        }
    }

    async function saveSettings() {
        if (!provider?.id) return;
        
        error = '';
        try {
            const updated = await $pocketbase.collection('providers').update(provider.id, {
                settings: settings
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

    async function handleCredentialsAdded() {
        await checkCredentials();
        showCredentialsModal = false;
        success = 'Credentials saved successfully';
        await loadData();
    }

    function handleMappingsChange(event: CustomEvent<Array<{ client_id: string; organization_id: string }>>) {
        settings.client_mappings = event.detail;
        saveSettings();
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
            <p class="mt-4 text-gray-600 dark:text-gray-400">Loading Jira Configuration...</p>
        </div>
    {:else}
        <div class="flex justify-between items-center">
            <h3 class="text-lg font-medium text-gray-900 dark:text-white">Jira Configuration</h3>
            <Button color="blue" on:click={() => (showCredentialsModal = true)}>
                {hasCredentials ? 'Update Credentials' : 'Add Credentials'}
            </Button>
        </div>

        {#if hasCredentials}
            <div class="space-y-4">
                <div>
                    <Label for="jira_url">Jira URL</Label>
                    <Input
                        id="jira_url"
                        type="url"
                        placeholder="https://your-domain.atlassian.net"
                        bind:value={settings.jira_url}
                        on:blur={saveSettings}
                    />
                    <p class="text-sm text-gray-500 mt-1">Your Jira instance URL (e.g., https://your-domain.atlassian.net)</p>
                </div>

                {#if settings.jira_url}
                    <div>
                        <Label for="project_key">Project</Label>
                        <Select
                            id="project_key"
                            bind:value={settings.project_key}
                            on:change={() => {
                                saveSettings();
                                fetchIssueTypes();
                            }}
                        >
                            <option value="">Select a project</option>
                            {#each projects as project}
                                <option value={project.key}>{project.name}</option>
                            {/each}
                        </Select>
                    </div>

                    {#if settings.project_key}
                        <div>
                            <Label for="issue_type">Issue Type</Label>
                            <Select
                                id="issue_type"
                                bind:value={settings.issue_type}
                                on:change={saveSettings}
                            >
                                <option value="">Select an issue type</option>
                                {#each issueTypes || [] as type}
                                    <option value={type.name}>{type.name}</option>
                                {/each}
                            </Select>
                        </div>

                        <div class="mt-6">
                            <JiraClientMappings
                                {clients}
                                organizations={organizations}
                                mappings={settings.client_mappings || []}
                                on:change={handleMappingsChange}
                            />
                        </div>
                    {/if}
                {/if}
            </div>
        {:else}
            <div class="text-center py-4">
                <p class="text-gray-600 dark:text-gray-400">Please add your Jira credentials to continue.</p>
            </div>
        {/if}
    {/if}

    <JiraCredentialsModal
        bind:show={showCredentialsModal}
        {provider}
        onSave={handleCredentialsAdded}
        onClose={() => showCredentialsModal = false}
    />
</div> 
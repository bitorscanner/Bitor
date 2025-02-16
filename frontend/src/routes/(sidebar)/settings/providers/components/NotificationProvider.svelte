<script lang="ts">
    import { Label, Input, Select, Button } from 'flowbite-svelte';
    import { pocketbase } from '$lib/stores/pocketbase';
    import { onMount } from 'svelte';
    import type { Provider, EmailSettings, WebhookSettings, TelegramSettings, JiraSettings, JiraClientMapping } from '../types';
    import JiraClientMappings from './JiraClientMappings.svelte';

    interface JiraProject {
        key: string;
        name: string;
    }

    interface JiraIssueType {
        id: string;
        name: string;
    }

    export let provider: Provider;
    export let onSave: (provider: Provider) => void;

    let error = '';
    let projects: JiraProject[] = [];
    let issueTypes: JiraIssueType[] = [];
    let organizations: Array<{ id: string; name: string }> = [];
    let settings: EmailSettings | WebhookSettings | TelegramSettings | JiraSettings;
    let loading = true;
    let clients: Array<{ id: string; name: string }> = [];

    // Type guard functions
    function isJiraSettings(settings: any): settings is JiraSettings {
        return settings && 
               typeof settings === 'object' && 
               'jira_url' in settings && 
               'username' in settings && 
               'api_key' in settings &&
               (!settings.client_mappings || Array.isArray(settings.client_mappings));
    }

    function isEmailSettings(settings: any): settings is EmailSettings {
        return provider.provider_type === 'email';
    }

    function isWebhookSettings(settings: any): settings is WebhookSettings {
        return ['slack', 'teams', 'discord'].includes(provider.provider_type);
    }

    function isTelegramSettings(settings: any): settings is TelegramSettings {
        return provider.provider_type === 'telegram';
    }

    $: {
        if (isEmailSettings(provider.settings)) {
            settings = getEmailSettings();
        } else if (isWebhookSettings(provider.settings)) {
            settings = getWebhookSettings();
        } else if (isTelegramSettings(provider.settings)) {
            settings = getTelegramSettings();
        } else if (isJiraSettings(provider.settings)) {
            console.log('Reactive statement: Loading Jira settings from provider:', provider.settings);
            settings = getJiraSettings();
            console.log('Reactive statement: Loaded Jira settings:', settings);
        }
    }

    function getJiraSettings(): JiraSettings {
        if (!isJiraSettings(provider.settings)) {
            throw new Error('Provider is not a Jira provider');
        }
        console.log('getJiraSettings: Raw provider settings:', provider.settings);
        const settings = provider.settings;
        const jiraSettings = {
            jira_url: settings.jira_url || '',
            username: settings.username || '',
            api_key: settings.api_key || '',
            project_key: settings.project_key || settings.jira_project || '',
            issue_type: settings.issue_type || 'Task',
            client_mappings: settings.client_mappings || []
        };
        console.log('getJiraSettings: Processed Jira settings:', jiraSettings);
        return jiraSettings;
    }

    function getEmailSettings(): EmailSettings {
        if (!isEmailSettings(provider.settings)) {
            throw new Error('Provider is not an Email provider');
        }
        const settings = provider.settings;
        // Ensure all required fields exist
        return {
            smtp_host: settings.smtp_host || '',
            smtp_port: settings.smtp_port || 587,
            from_address: settings.from_address || '',
            encryption: settings.encryption || 'tls'
        };
    }

    function getWebhookSettings(): WebhookSettings {
        if (!isWebhookSettings(provider.settings)) {
            throw new Error('Provider is not a webhook-based provider');
        }
        const settings = provider.settings;
        // Ensure webhook_url exists
        return {
            webhook_url: settings.webhook_url || ''
        };
    }

    function getTelegramSettings(): TelegramSettings {
        if (!isTelegramSettings(provider.settings)) {
            throw new Error('Provider is not a Telegram provider');
        }
        const settings = provider.settings;
        // Ensure all required fields exist
        return {
            bot_token: settings.bot_token || '',
            chat_id: settings.chat_id || ''
        };
    }

    async function fetchJiraData() {
        const jiraSettings = getJiraSettings();
        if (!jiraSettings.jira_url || !jiraSettings.username || !jiraSettings.api_key) {
            return;
        }

        try {
            console.log('Fetching Jira projects with settings:', {
                url: jiraSettings.jira_url,
                username: jiraSettings.username,
                apiKeyLength: jiraSettings.api_key?.length || 0
            });

            // Make API call to backend to fetch Jira projects
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/projects`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    url: jiraSettings.jira_url,
                    username: jiraSettings.username,
                    api_key: jiraSettings.api_key
                })
            });

            console.log('Jira projects response status:', response.status);
            const data = await response.json();
            console.log('Jira projects response data:', data);

            if (!response.ok) {
                throw new Error(data.error || 'Failed to fetch Jira projects');
            }

            projects = data.projects;
            console.log('Fetched projects:', projects);
            
            // If a project is selected, fetch its issue types
            if (jiraSettings.project_key) {
                await fetchIssueTypes();
            }
        } catch (e: any) {
            console.error('Error fetching Jira data:', e);
            error = e.message || 'Failed to fetch Jira data';
        }
    }

    async function fetchIssueTypes() {
        const jiraSettings = getJiraSettings();
        if (!jiraSettings.project_key) {
            return;
        }

        try {
            console.log('Fetching Jira issue types with settings:', {
                url: jiraSettings.jira_url,
                username: jiraSettings.username,
                apiKeyLength: jiraSettings.api_key?.length || 0,
                projectKey: jiraSettings.project_key
            });

            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/issuetypes`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    url: jiraSettings.jira_url,
                    username: jiraSettings.username,
                    api_key: jiraSettings.api_key,
                    project_key: jiraSettings.project_key
                })
            });

            console.log('Jira issue types response status:', response.status);
            const data = await response.json();
            console.log('Jira issue types response data:', data);

            if (!response.ok) {
                throw new Error(data.error || 'Failed to fetch issue types');
            }

            issueTypes = data.issueTypes;
            console.log('Fetched issue types:', issueTypes);
        } catch (e: any) {
            console.error('Error fetching issue types:', e);
            error = e.message || 'Failed to fetch issue types';
        }
    }

    async function fetchJiraOrganizations() {
        const jiraSettings = getJiraSettings();
        if (!jiraSettings.project_key || !jiraSettings.issue_type) {
            console.log('Skipping organization fetch - missing required fields:', {
                project_key: jiraSettings.project_key,
                issue_type: jiraSettings.issue_type
            });
            return;
        }

        try {
            const requestBody = {
                url: jiraSettings.jira_url,
                username: jiraSettings.username,
                api_key: jiraSettings.api_key,
                project_key: jiraSettings.project_key,
                issue_type: jiraSettings.issue_type
            };
            
            console.log('Fetching Jira organizations with request:', {
                ...requestBody,
                api_key: '[REDACTED]'
            });

            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/jira/organizations`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify(requestBody)
            });

            console.log('Organizations response status:', response.status);
            const data = await response.json();
            console.log('Organizations response data:', data);

            if (!response.ok) {
                throw new Error(data.error || 'Failed to fetch organizations');
            }

            organizations = data.organizations;
            console.log('Fetched organizations:', organizations);
        } catch (e: any) {
            console.error('Error fetching organizations:', e);
            error = e.message || 'Failed to fetch organizations';
        }
    }

    async function saveSettings() {
        try {
            if (!provider.id) {
                console.error('No provider ID available');
                error = 'Provider ID is missing';
                return;
            }

            console.log('saveSettings: Starting save with current settings:', settings);
            console.log('saveSettings: Current provider state:', provider);

            // Ensure we're using the correct field names
            if (provider.provider_type === 'jira') {
                const jiraSettings = settings as JiraSettings;
                const newSettings = {
                    ...jiraSettings,
                    project_key: jiraSettings.project_key || jiraSettings.jira_project || '',
                    issue_type: jiraSettings.issue_type,
                    client_mappings: jiraSettings.client_mappings || []
                };
                console.log('saveSettings: Prepared Jira settings for save:', newSettings);
                settings = newSettings;
            }

            // Update the provider's settings
            provider.settings = settings;
            console.log('saveSettings: Updated provider settings before save:', provider.settings);

            const updatedProvider = await $pocketbase.collection('providers').update(provider.id, {
                settings: provider.settings
            });

            console.log('saveSettings: Raw response from PocketBase:', updatedProvider);
            
            // Update both the provider and settings objects with the response data
            provider.settings = updatedProvider.settings;
            console.log('saveSettings: Updated provider settings after save:', provider.settings);

            if (isJiraSettings(provider.settings)) {
                settings = {
                    jira_url: provider.settings.jira_url || '',
                    username: provider.settings.username || '',
                    api_key: provider.settings.api_key || '',
                    project_key: provider.settings.project_key || provider.settings.jira_project || '',
                    issue_type: provider.settings.issue_type,
                    client_mappings: provider.settings.client_mappings || []
                };
                console.log('saveSettings: Updated local settings after save:', settings);
            }
            onSave(provider);
            console.log('saveSettings: Save completed successfully');
        } catch (e: any) {
            console.error('Error saving settings:', e);
            error = e.message || 'Failed to save settings';
        }
    }

    // Update the project selection handler to also fetch organizations
    async function handleProjectChange(event: Event) {
        if (isJiraSettings(settings)) {
            const select = event.target as HTMLSelectElement;
            console.log('handleProjectChange: Project selection changed to:', select.value);
            
            settings.project_key = select.value;
            
            // First save the project selection
            await saveSettings();
            
            // Then fetch both issue types and organizations
            await Promise.all([
                fetchIssueTypes(),
                fetchJiraOrganizations()
            ]);
        }
    }

    // Update the issue type selection handler to also fetch organizations
    async function handleIssueTypeChange(event: Event) {
        if (isJiraSettings(settings)) {
            const select = event.target as HTMLSelectElement;
            console.log('handleIssueTypeChange: Issue type changed to:', select.value);
            console.log('handleIssueTypeChange: Current settings:', settings);
            
            // Only update and save if a valid issue type is selected
            if (select.value) {
                settings.issue_type = select.value;
                provider.settings = { ...provider.settings, issue_type: select.value };
                
                // First save the issue type
                await saveSettings();
                console.log('handleIssueTypeChange: Settings saved, now fetching organizations with:', {
                    project_key: settings.project_key,
                    issue_type: settings.issue_type
                });
                
                // Then fetch organizations with the new issue type
                await fetchJiraOrganizations();
            }
        }
    }

    onMount(async () => {
        loading = true;
        try {
            // Fetch clients
            const clientsResponse = await $pocketbase.collection('clients').getFullList();
            clients = clientsResponse.map(client => ({
                id: client.id,
                name: client.name
            }));

            // Fetch Jira data
            if (provider.provider_type === 'jira') {
                await fetchJiraData();
                
                // If we have both project_key and issue_type, fetch organizations
                if (isJiraSettings(settings) && settings.project_key && settings.issue_type) {
                    console.log('Initial load: Fetching organizations with existing project and issue type');
                    await fetchJiraOrganizations();
                }
            }
        } catch (error) {
            console.error('Error loading data:', error);
        } finally {
            loading = false;
        }
    });

    function handleClientMappingsChange(event: CustomEvent<JiraClientMapping[]>) {
        if (isJiraSettings(settings)) {
            console.log('handleClientMappingsChange: Received new mappings:', event.detail);
            settings.client_mappings = event.detail;
            provider.settings = { ...provider.settings, client_mappings: event.detail };
            console.log('handleClientMappingsChange: Updated settings:', settings);
            saveSettings();
        }
    }
</script>

<div class="space-y-4">
    {#if isEmailSettings(settings)}
        <div>
            <Label for="smtp_host">SMTP Host</Label>
            <Input
                id="smtp_host"
                bind:value={settings.smtp_host}
                on:blur={saveSettings}
                placeholder="Enter SMTP host"
            />
        </div>

        <div>
            <Label for="smtp_port">SMTP Port</Label>
            <Input
                id="smtp_port"
                type="number"
                bind:value={settings.smtp_port}
                on:blur={saveSettings}
                placeholder="Enter SMTP port"
            />
        </div>

        <div>
            <Label for="from_address">From Address</Label>
            <Input
                id="from_address"
                type="email"
                bind:value={settings.from_address}
                on:blur={saveSettings}
                placeholder="Enter from address"
            />
        </div>

        <div>
            <Label for="encryption">Encryption</Label>
            <Select
                id="encryption"
                bind:value={settings.encryption}
                on:change={saveSettings}
            >
                <option value="none">None</option>
                <option value="tls">TLS</option>
                <option value="starttls">STARTTLS</option>
            </Select>
        </div>
    {:else if isWebhookSettings(settings)}
        <div>
            <Label for="webhook_url">Webhook URL</Label>
            <Input
                id="webhook_url"
                bind:value={settings.webhook_url}
                on:blur={saveSettings}
                placeholder="Enter webhook URL"
            />
        </div>
    {:else if isTelegramSettings(settings)}
        <div>
            <Label for="bot_token">Bot Token</Label>
            <Input
                id="bot_token"
                bind:value={settings.bot_token}
                on:blur={saveSettings}
                placeholder="Enter bot token"
            />
        </div>

        <div>
            <Label for="chat_id">Chat ID</Label>
            <Input
                id="chat_id"
                bind:value={settings.chat_id}
                on:blur={saveSettings}
                placeholder="Enter chat ID"
            />
        </div>
    {:else if isJiraSettings(settings)}
        <div>
            {#if !settings.jira_url || !settings.username || !settings.api_key}
                <div>
                    <Label for="jira_url">Jira URL</Label>
                    <Input
                        id="jira_url"
                        bind:value={settings.jira_url}
                        on:blur={saveSettings}
                        placeholder="https://your-domain.atlassian.net"
                    />
                    <p class="text-sm text-gray-500 mt-1">Your Jira instance URL (e.g., https://your-domain.atlassian.net)</p>
                </div>

                <div>
                    <Label for="username">Username (Email)</Label>
                    <Input
                        id="username"
                        bind:value={settings.username}
                        on:blur={saveSettings}
                        placeholder="user@example.com"
                    />
                    <p class="text-sm text-gray-500 mt-1">Your Jira account email address</p>
                </div>

                <div>
                    <Label for="api_key">API Token</Label>
                    <Input
                        id="api_key"
                        type="password"
                        bind:value={settings.api_key}
                        on:blur={saveSettings}
                        placeholder="Enter API token"
                    />
                    <p class="text-sm text-gray-500 mt-1">
                        Generate an API token from 
                        <a 
                            href="https://id.atlassian.com/manage-profile/security/api-tokens" 
                            target="_blank" 
                            rel="noopener noreferrer"
                            class="text-blue-600 hover:underline"
                        >
                            Atlassian Account Settings
                        </a>
                    </p>
                </div>
            {:else}
                <div class="mb-4">
                    <div class="flex justify-between items-center">
                        <div>
                            <h3 class="text-lg font-medium">Jira Connection</h3>
                            <p class="text-sm text-gray-500">Connected to {settings.jira_url}</p>
                            <p class="text-sm text-gray-500">User: {settings.username}</p>
                        </div>
                        <Button size="xs" on:click={() => {
                            if (isJiraSettings(settings)) {
                                settings.jira_url = '';
                                settings.username = '';
                                settings.api_key = '';
                                saveSettings();
                            }
                        }}>Reset Connection</Button>
                    </div>
                </div>

                <div>
                    <Label for="api_key">Update API Token</Label>
                    <Input
                        id="api_key"
                        type="password"
                        bind:value={settings.api_key}
                        on:blur={saveSettings}
                        placeholder="Enter new API token"
                    />
                </div>

                {#if loading}
                    <div class="text-sm text-gray-500">Loading...</div>
                {:else}
                    <div>
                        <Label for="project_key">Project</Label>
                        <Select
                            id="project_key"
                            bind:value={settings.project_key}
                            on:change={handleProjectChange}
                        >
                            <option value="">Select a project</option>
                            {#each projects || [] as project}
                                <option value={project.key} selected={project.key === settings.project_key}>
                                    {project.name} ({project.key})
                                </option>
                            {/each}
                        </Select>
                    </div>

                    <div>
                        <Label for="issue_type">Issue Type</Label>
                        <Select
                            id="issue_type"
                            value={settings.issue_type}
                            on:change={handleIssueTypeChange}
                            disabled={!settings.project_key}
                        >
                            <option value="">Select an issue type</option>
                            {#each issueTypes || [] as issueType}
                                <option value={issueType.name}>
                                    {issueType.name}
                                </option>
                            {/each}
                        </Select>
                        {#if settings.issue_type}
                            <p class="text-sm text-gray-500 mt-1">Current Issue Type: {settings.issue_type}</p>
                        {/if}
                    </div>
                {/if}
            {/if}
        </div>
    {/if}

    {#if provider.provider_type === 'jira' && !loading}
        <div class="mt-6">
            {#if isJiraSettings(settings)}
                <JiraClientMappings
                    {clients}
                    organizations={organizations}
                    mappings={settings.client_mappings || []}
                    on:change={handleClientMappingsChange}
                />
            {/if}
        </div>
    {/if}

    {#if error}
        <p class="text-red-500 text-sm">{error}</p>
    {/if}
</div> 
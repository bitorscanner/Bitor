<script lang="ts">
    import { onMount } from 'svelte';
    import { get } from 'svelte/store';
    import { pocketbase } from '$lib/stores/pocketbase';
    import { Card } from 'flowbite-svelte';
    import NotificationRules from './components/NotificationRules.svelte';

    type Provider = {
        id: string;
        name: string;
        type: string;
    };

    type Client = {
        id: string;
        name: string;
    };

    type Settings = {
        rules: Array<{
            id: string;
            name: string;
            type: string;
            severity: string[];
            channels: string[];
            enabled: boolean;
            message: string;
            clients: string[];
            allClients: boolean;
        }>;
    };

    let settings: Settings = {
        rules: []
    };

    let loading = true;
    let error = '';
    let providers: Provider[] = [];
    let clients: Client[] = [];

    onMount(async () => {
        try {
            const pb = get(pocketbase);
            console.log('Fetching data from PocketBase...');

            // First get all providers without filter to debug
            const allProviders = await pb.collection('providers').getFullList();
            console.log('All providers (no filter):', allProviders);

            const [settingsRecords, providerRecords, clientRecords] = await Promise.all([
                pb.collection('notification_settings').getFullList(),
                pb.collection('providers').getFullList({
                    filter: 'enabled = true && use ?~ "notification" && (provider_type = "email" || provider_type = "slack" || provider_type = "discord" || provider_type = "telegram" || provider_type = "jira")'
                }),
                pb.collection('clients').getFullList()
            ]);

            console.log('Raw provider records (with filter):', providerRecords);
            console.log('Filter used:', 'enabled = true && use ?~ "notification" && (provider_type = "email" || provider_type = "slack" || provider_type = "discord" || provider_type = "telegram" || provider_type = "jira")');

            if (settingsRecords.length > 0) {
                const record = settingsRecords[0];
                console.log('Raw settings record:', record);
                
                try {
                    let rulesData;
                    if (record.data) {
                        // Handle both string and object data
                        const parsedData = typeof record.data === 'string' 
                            ? JSON.parse(record.data) 
                            : record.data;
                        rulesData = parsedData.rules;
                        console.log('Parsed rules from data:', rulesData);
                    }
                    
                    settings = {
                        rules: Array.isArray(rulesData) ? rulesData : []
                    };
                    console.log('Final settings object:', settings);
                } catch (e) {
                    console.error('Error parsing settings data:', e);
                    settings = { rules: [] };
                }
            }

            providers = providerRecords.map(record => ({
                id: record.id,
                name: record.name,
                type: record.provider_type
            }));

            clients = clientRecords.map(record => ({
                id: record.id,
                name: record.name
            }));

            console.log('Processed providers:', providers);
            console.log('Provider types available:', providers.map(p => p.type));
            console.log('Number of providers:', providers.length);
            console.log('Loaded clients:', clients);
        } catch (e) {
            console.error('Error loading data:', e);
            error = 'Failed to load settings';
        } finally {
            loading = false;
        }
    });

    async function saveSettings(updatedSettings: Settings) {
        try {
            const pb = get(pocketbase);
            const records = await pb.collection('notification_settings').getFullList();
            
            console.log('Saving rules:', updatedSettings.rules); // Debug log
            console.log('Rule being tested:', updatedSettings.rules[0]); // Log first rule

            // Create the data object with the correct structure
            const data = {
                data: JSON.stringify({
                    rules: updatedSettings.rules
                })
            };

            console.log('Saving data structure:', data); // Debug log
            console.log('Raw data being saved:', JSON.stringify(data, null, 2)); // Pretty print the data

            if (records.length > 0) {
                await pb.collection('notification_settings').update(records[0].id, data);
                console.log('Settings saved successfully with ID:', records[0].id); // Debug log
            } else {
                const newRecord = await pb.collection('notification_settings').create(data);
                console.log('New settings created successfully with ID:', newRecord.id); // Debug log
            }
            error = '';
        } catch (e) {
            console.error('Error saving notification settings:', e);
            error = 'Failed to save notification settings';
        }
    }

    function handleSettingsChange(event: CustomEvent) {
        console.log('Settings change event received:', event.detail);
        settings.rules = event.detail;
        saveSettings(settings);
    }
</script>

<div class="container mx-auto p-4">
    <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Notification Rules</h1>
        <p class="mt-2 text-gray-600 dark:text-gray-400">Configure when and how notifications are sent based on scan events and findings.</p>
    </div>

    {#if loading}
        <div class="text-center text-gray-900 dark:text-white">Loading...</div>
    {:else if error}
        <div class="mt-4 p-4 bg-red-100 text-red-700 rounded">
            {error}
        </div>
    {:else}
        <NotificationRules 
            rules={settings.rules}
            {providers}
            {clients}
            on:change={(e) => handleSettingsChange(e)} 
        />
    {/if}
</div>
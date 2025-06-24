<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { 
        Table, 
        TableBody, 
        TableBodyCell, 
        TableBodyRow, 
        TableHead, 
        TableHeadCell,
        Button,
        Toggle,
        Textarea,
        Dropdown,
        DropdownItem,
        Card,
        Alert
    } from 'flowbite-svelte';
    import {
        SiGmail,
        SiSlack,
        SiDiscord,
        SiJira,
        SiTelegram
    } from '@icons-pack/svelte-simple-icons';
    import { generateUUID } from '$lib/utils/uuid';
    import { pocketbase } from '@lib/stores/pocketbase';

    export let rules: Array<{
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

    export let providers: Array<{
        id: string;
        name: string;
        type: string;
    }>;

    export let clients: Array<{
        id: string;
        name: string;
    }> = [];

    const dispatch = createEventDispatcher();

    const notificationTypes = [
        { 
            value: 'scan_started', 
            label: 'Scan Started',
            description: 'Send notifications when a scan begins'
        },
        { 
            value: 'scan_finished', 
            label: 'Scan Finished',
            description: 'Send notifications when a scan completes successfully'
        },
        { 
            value: 'scan_failed', 
            label: 'Scan Failed',
            description: 'Send notifications when a scan encounters an error'
        },
        { 
            value: 'scan_stopped', 
            label: 'Scan Stopped',
            description: 'Send notifications when a scan is manually stopped'
        },
        { 
            value: 'finding', 
            label: 'Finding Alert',
            description: 'Send notifications for findings based on selected severity levels'
        }
    ];

    const severityLevels = ['info', 'low', 'medium', 'high', 'critical'];
    const channels = ['email', 'jira', 'slack', 'discord', 'telegram'];

    let expandedRule: string | null = null;
    let editingName: string | null = null;
    let tempName = '';
    let testError = '';
    let testSuccess = '';
    let templates: Record<string, string> = {};

    onMount(async () => {
        console.log('NotificationRules mounted');
        console.log('Available providers:', providers);
        console.log('Current rules:', rules);
        
        try {
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/notification-templates`, {
                headers: {
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                }
            });
            
            if (!response.ok) {
                throw new Error('Failed to fetch notification templates');
            }
            
            templates = await response.json();
            console.log('Loaded templates:', templates);
        } catch (error) {
            console.error('Error loading templates:', error);
        }
    });

    function addRule(type: string) {
        const typeConfig = notificationTypes.find(t => t.value === type);
        if (!typeConfig) return;

        const newRule = {
            id: generateUUID(),
            name: `${typeConfig.label} Rule`,
            type: typeConfig.value,
            severity: [],
            channels: [],
            enabled: true,
            message: templates[type] || '',
            clients: [],
            allClients: true
        };
        rules = [...rules, newRule];
        handleChange();
    }

    function deleteRule(id: string) {
        rules = rules.filter(rule => rule.id !== id);
        handleChange();
    }

    function toggleSeverity(rule: any, severity: string) {
        if (rule.severity.includes(severity)) {
            rule.severity = rule.severity.filter((s: string) => s !== severity);
        } else {
            rule.severity = [...rule.severity, severity];
        }
        handleChange();
    }

    function toggleChannel(rule: any, providerId: string) {
        if (rule.channels.includes(providerId)) {
            rule.channels = rule.channels.filter((c: string) => c !== providerId);
        } else {
            rule.channels = [...rule.channels, providerId];
        }
        handleChange();
    }

    function toggleClient(rule: any, clientId: string) {
        if (rule.clients.includes(clientId)) {
            rule.clients = rule.clients.filter((c: string) => c !== clientId);
        } else {
            rule.clients = [...rule.clients, clientId];
        }
        handleChange();
    }

    function toggleAllClients(rule: any) {
        rule.allClients = !rule.allClients;
        if (rule.allClients) {
            rule.clients = [];
        }
        handleChange();
    }

    function handleChange() {
        console.log('Rules changed, dispatching update:', rules);
        dispatch('change', rules);
    }

    function resetTemplate(rule: any) {
        rule.message = templates[rule.type] || '';
        handleChange();
    }

    function toggleExpand(ruleId: string) {
        expandedRule = expandedRule === ruleId ? null : ruleId;
    }

    function getTypeLabel(type: string): string {
        return notificationTypes.find(t => t.value === type)?.label || type;
    }

    function getProviderName(providerId: string): string {
        console.log('Getting provider name for:', providerId);
        console.log('Available providers:', providers);
        const provider = providers.find(p => p.id === providerId);
        console.log('Found provider:', provider);
        return provider ? provider.name : providerId;
    }

    function getProviderType(providerId: string): string {
        const provider = providers.find(p => p.id === providerId);
        return provider ? provider.type : '';
    }

    function startEditingName(rule: any) {
        editingName = rule.id;
        tempName = rule.name;
    }

    function saveRuleName(rule: any) {
        rule.name = tempName;
        editingName = null;
        handleChange();
    }

    function handleKeyDown(event: KeyboardEvent, rule: any) {
        if (event.key === 'Enter') {
            saveRuleName(rule);
        } else if (event.key === 'Escape') {
            editingName = null;
        }
    }

    function handleMessageChange(rule: any) {
        handleChange();
    }

    function handleEnabledChange(rule: any) {
        handleChange();
    }

    async function testNotification(rule: any) {
        testError = '';
        testSuccess = '';
        
        try {
            console.log('Testing notification for rule:', rule);
            console.log('Rule ID being sent:', rule.id);
            const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/test-notification`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${$pocketbase.authStore.token}`
                },
                body: JSON.stringify({
                    rule_id: rule.id
                })
            });

            console.log('Test notification response status:', response.status);
            const data = await response.json();
            console.log('Test notification response data:', data);
            
            if (!response.ok) {
                throw new Error(data.message || 'Failed to send test notification');
            }

            testSuccess = 'Test notification sent successfully';
            setTimeout(() => {
                testSuccess = '';
            }, 3000);
        } catch (e: any) {
            console.error('Error testing notification:', e);
            testError = e.message || 'Failed to send test notification';
            setTimeout(() => {
                testError = '';
            }, 3000);
        }
    }
</script>

<div class="w-full">
    {#if testError}
        <Alert color="red" class="mb-4">
            <span class="font-medium">Error!</span> {testError}
        </Alert>
    {/if}

    {#if testSuccess}
        <Alert color="green" class="mb-4">
            <span class="font-medium">Success!</span> {testSuccess}
        </Alert>
    {/if}

    <div class="relative mb-6 inline-block">
        <Button>Add Rule</Button>
        <Dropdown placement="bottom" class="z-50 !fixed bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 shadow-lg rounded-lg">
            {#each notificationTypes as type}
                <DropdownItem class="hover:bg-gray-100 dark:hover:bg-gray-700" on:click={() => addRule(type.value)}>{type.label}</DropdownItem>
            {/each}
        </Dropdown>
    </div>

    <Table hoverable={true} class="w-full">
        <TableHead>
            <TableHeadCell>Type</TableHeadCell>
            <TableHeadCell>Channels</TableHeadCell>
            <TableHeadCell>Status</TableHeadCell>
            <TableHeadCell>Actions</TableHeadCell>
        </TableHead>
        <TableBody>
            {#if rules.length === 0}
                <TableBodyRow>
                    <TableBodyCell colspan={4} class="text-center py-4 text-gray-500 dark:text-gray-400">
                        No notification rules configured. Click "Add Rule" to create one.
                    </TableBodyCell>
                </TableBodyRow>
            {:else}
                {#each rules as rule}
                    <TableBodyRow>
                        <TableBodyCell>
                            <div class="font-medium text-gray-900 dark:text-white">
                                {#if editingName === rule.id}
                                    <input
                                        type="text"
                                        class="w-full px-2 py-1 bg-white dark:bg-gray-700 border border-blue-500 dark:border-blue-500 rounded focus:outline-none focus:ring-2 focus:ring-blue-500"
                                        bind:value={tempName}
                                        on:blur={() => saveRuleName(rule)}
                                        on:keydown={(e) => handleKeyDown(e, rule)}
                                        autofocus
                                    />
                                {:else}
                                    <div 
                                        class="cursor-pointer hover:text-blue-600 dark:hover:text-blue-400"
                                        on:click={() => startEditingName(rule)}
                                    >
                                        {rule.name}
                                    </div>
                                {/if}
                </div>
                            <div class="text-sm text-gray-500 dark:text-gray-400">
                                {getTypeLabel(rule.type)}
            </div>
                            <div class="text-xs text-gray-500 dark:text-gray-400">
                                {notificationTypes.find(t => t.value === rule.type)?.description}
                </div>
                            {#if !rule.allClients && rule.clients?.length > 0 && clients?.length > 0}
                                <div class="mt-1 flex flex-wrap gap-1">
                                    {#each rule.clients as clientId}
                                        <span class="px-1.5 py-0.5 text-xs font-medium rounded bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300">
                                            {(clients.find(c => c.id === clientId)?.name) || clientId}
                                        </span>
                                    {/each}
                </div>
                            {/if}
                        </TableBodyCell>
                        <TableBodyCell>
                            <div class="flex flex-wrap gap-2">
                                {#each rule.channels as channel}
                                    <span class="px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300">
                                        {getProviderName(channel)}
                                    </span>
                                {/each}
                            </div>
                        </TableBodyCell>
                        <TableBodyCell>
                            <Toggle 
                                bind:checked={rule.enabled}
                                on:change={() => handleEnabledChange(rule)}
                            />
                        </TableBodyCell>
                        <TableBodyCell>
                            <div class="flex items-center space-x-2">
                                <Button size="xs" on:click={() => toggleExpand(rule.id)}>Configure</Button>
                                <Button size="xs" color="green" on:click={() => testNotification(rule)}>Test</Button>
                                <button
                                    class="text-red-500 hover:text-red-700"
                                    on:click={() => deleteRule(rule.id)}
                                >
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                    </svg>
                                </button>
                            </div>
                        </TableBodyCell>
                    </TableBodyRow>
                    {#if expandedRule === rule.id}
                        <TableBodyRow>
                            <TableBodyCell colspan={4} class="!p-0">
                                <div class="min-h-screen bg-gray-50 dark:bg-gray-800">
                                    <div class="max-w-[1920px] mx-auto p-6">
                                        <div class="grid grid-cols-12 gap-8">
                                            <div class="col-span-3">
                                                <div class="space-y-6">
                                                    <div>
                                                        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Notification Channels</h3>
                                                        <div class="flex flex-wrap gap-2">
                                                            {#each providers as provider}
                                                                <button
                                                                    class="px-3 py-2 rounded-lg text-sm font-medium transition-colors duration-200 flex items-center gap-2"
                                                                    class:bg-blue-100={rule.channels.includes(provider.id)}
                                                                    class:text-blue-800={rule.channels.includes(provider.id)}
                                                                    class:hover:bg-blue-200={rule.channels.includes(provider.id)}
                                                                    class:bg-gray-100={!rule.channels.includes(provider.id)}
                                                                    class:text-gray-800={!rule.channels.includes(provider.id)}
                                                                    class:hover:bg-gray-200={!rule.channels.includes(provider.id)}
                                                                    class:dark:bg-gray-700={!rule.channels.includes(provider.id)}
                                                                    class:dark:text-gray-300={!rule.channels.includes(provider.id)}
                                                                    on:click={() => toggleChannel(rule, provider.id)}
                                                                >
                                                                    {#if provider.type === 'email'}
                                                                        <SiGmail />
                                                                    {:else if provider.type === 'slack'}
                                                                        <SiSlack />
                                                                    {:else if provider.type === 'discord'}
                                                                        <SiDiscord />
                                                                    {:else if provider.type === 'jira'}
                                                                        <SiJira />
                                                                    {:else if provider.type === 'telegram'}
                                                                        <SiTelegram />
                                                                    {/if}
                                                                    {provider.name}
                                                                </button>
                                                            {/each}
                                                        </div>
                                                    </div>

                                                    {#if rule.type.includes('finding')}
                                                        <div>
                                                            <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Severity Levels</h3>
                                                            <div class="flex flex-wrap gap-2">
                                                                {#each severityLevels as severity}
                                                                    <button
                                                                        class="px-3 py-2 rounded-lg text-sm font-medium transition-colors duration-200"
                                                                        class:bg-blue-100={rule.severity.includes(severity)}
                                                                        class:text-blue-800={rule.severity.includes(severity)}
                                                                        class:hover:bg-blue-200={rule.severity.includes(severity)}
                                                                        class:bg-gray-100={!rule.severity.includes(severity)}
                                                                        class:text-gray-800={!rule.severity.includes(severity)}
                                                                        class:hover:bg-gray-200={!rule.severity.includes(severity)}
                                                                        class:dark:bg-gray-700={!rule.severity.includes(severity)}
                                                                        class:dark:text-gray-300={!rule.severity.includes(severity)}
                                                                        on:click={() => toggleSeverity(rule, severity)}
                                                                    >
                                                                        {severity}
                                                                    </button>
                                                                {/each}
                                                            </div>
                                                        </div>
                                                    {/if}

                                                    <div>
                                                        <h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Client Filter</h3>
                                                        <div class="mb-4">
                                                            <label class="flex items-center space-x-2 text-sm font-medium text-gray-900 dark:text-white cursor-pointer">
                                                                <input
                                                                    type="checkbox"
                                                                    class="form-checkbox h-4 w-4 text-blue-600"
                                                                    checked={rule.allClients}
                                                                    on:change={() => toggleAllClients(rule)}
                                                                />
                                                                <span>Apply to all clients</span>
                                </label>
                                                        </div>
                                                        {#if !rule.allClients && clients?.length > 0}
                                                            <div class="flex flex-wrap gap-2">
                                                                {#each clients as client}
                                                                    <button
                                                                        class="px-3 py-2 rounded-lg text-sm font-medium transition-colors duration-200"
                                                                        class:bg-blue-100={rule.clients.includes(client.id)}
                                                                        class:text-blue-800={rule.clients.includes(client.id)}
                                                                        class:hover:bg-blue-200={rule.clients.includes(client.id)}
                                                                        class:bg-gray-100={!rule.clients.includes(client.id)}
                                                                        class:text-gray-800={!rule.clients.includes(client.id)}
                                                                        class:hover:bg-gray-200={!rule.clients.includes(client.id)}
                                                                        class:dark:bg-gray-700={!rule.clients.includes(client.id)}
                                                                        class:dark:text-gray-300={!rule.clients.includes(client.id)}
                                                                        on:click={() => toggleClient(rule, client.id)}
                                                                    >
                                                                        {client.name}
                                                                    </button>
                                                                {/each}
                                                            </div>
                                                        {:else if !rule.allClients}
                                                            <p class="text-sm text-gray-500 dark:text-gray-400">No clients available to select.</p>
                                                        {/if}
                                                    </div>
                                                </div>
                                            </div>

                                            <div class="col-span-9">
                                                <div class="space-y-8">
                                                    <Card padding="xl" class="w-full min-h-[800px] max-w-none">
                                                        <div class="flex justify-between items-center mb-8">
                                                            <h3 class="text-xl font-medium text-gray-900 dark:text-white">Message Template</h3>
                                                            <Button size="sm" color="light" on:click={() => resetTemplate(rule)}>
                                                                Reset to Default
                                                            </Button>
                                                        </div>
                                                        <Textarea
                                                            rows={25}
                                                            bind:value={rule.message}
                                                            on:change={() => handleMessageChange(rule)}
                                                            class="w-full mb-8 text-base"
                                                        />
                                                    </Card>

                                                    <Card padding="xl" class="w-full max-w-none bg-gray-50 dark:bg-gray-800">
                                                        <h4 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Available Placeholders:</h4>
                                                        <div class="grid grid-cols-1 md:grid-cols-2 gap-10 text-sm text-gray-600 dark:text-gray-400 p-4">
                                                            <div class="space-y-4">
                                                                <h5 class="text-base font-medium text-gray-700 dark:text-gray-300 mb-2">Common:</h5>
                                                                <div class="space-y-3">
                                                                    <div class="flex items-center gap-3">
                                                                        <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{scan_name}}'}</code>
                                                                        <span class="text-sm">Name of the scan</span>
                                                                    </div>
                                                                    <div class="flex items-center gap-3">
                                                                        <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{scan_id}}'}</code>
                                                                        <span class="text-sm">ID of the scan</span>
                                                                    </div>
                                                                    <div class="flex items-center gap-3">
                                                                        <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{time}}'}</code>
                                                                        <span class="text-sm">Time of the event</span>
                                                                    </div>
                                                                    <div class="flex items-center gap-3">
                                                                        <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{client_name}}'}</code>
                                                                        <span class="text-sm">Name of the client</span>
                                                                    </div>
                                                                    <div class="flex items-center gap-3">
                                                                        <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{jira_link}}'}</code>
                                                                        <span class="text-sm">Link to Jira issue</span>
                                                                    </div>
                                                                </div>
                                                            </div>
                                                            {#if rule.type.includes('finding')}
                                                                <div class="space-y-4">
                                                                    <h5 class="text-base font-medium text-gray-700 dark:text-gray-300 mb-2">Finding Specific:</h5>
                                                                    <div class="space-y-3">
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{title}}'}</code>
                                                                            <span class="text-sm">Finding title</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{severity}}'}</code>
                                                                            <span class="text-sm">Finding severity</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{target}}'}</code>
                                                                            <span class="text-sm">Finding target</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{description}}'}</code>
                                                                            <span class="text-sm">Finding description</span>
                                                                        </div>
                                                                    </div>
                                                                </div>
                                                            {:else if rule.type.includes('scan')}
                                                                <div class="space-y-4">
                                                                    <h5 class="text-base font-medium text-gray-700 dark:text-gray-300 mb-2">Scan Specific:</h5>
                                                                    <div class="space-y-3">
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{tool}}'}</code>
                                                                            <span class="text-sm">Scan tool name</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{tool_version}}'}</code>
                                                                            <span class="text-sm">Tool version</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{total_targets}}'}</code>
                                                                            <span class="text-sm">Total targets</span>
                                                                        </div>
                                                                        <div class="flex items-center gap-3">
                                                                            <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{scan_time}}'}</code>
                                                                            <span class="text-sm">Total scan time</span>
                                                                        </div>
                                                                        {#if rule.type === 'scan_failed'}
                                                                            <div class="flex items-center gap-3">
                                                                                <code class="bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded text-sm">{'{{error}}'}</code>
                                                                                <span class="text-sm">Error message</span>
                            </div>
                        {/if}
                                                                    </div>
                    </div>
                {/if}
            </div>
        </Card>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </TableBodyCell>
                        </TableBodyRow>
                    {/if}
    {/each}
            {/if}
        </TableBody>
    </Table>
</div> 
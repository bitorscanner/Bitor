<script lang="ts">
    import { Modal, Button, Label, Input, Alert } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider } from '../types';

    export let show = false;
    export let provider: Provider;
    export let onSave: () => void;
    export let onClose: () => void;

    let username = '';
    let apiKey = '';
    let error = '';
    let loading = false;
    let existingUsernameId = '';
    let existingApiKeyId = '';

    async function handleSubmit() {
        if (!username || !apiKey) {
            error = 'Both username and API key are required';
            return;
        }

        loading = true;
        error = '';
        try {
            // Handle username
            if (existingUsernameId) {
                await $pocketbase.collection('api_keys').update(existingUsernameId, {
                    key: username
                });
            } else {
                await $pocketbase.collection('api_keys').create({
                    provider: provider.id,
                    key: username,
                    key_type: 'username',
                    name: 'Jira Username'
                });
            }

            // Handle API key
            if (existingApiKeyId) {
                await $pocketbase.collection('api_keys').update(existingApiKeyId, {
                    key: apiKey
                });
            } else {
                await $pocketbase.collection('api_keys').create({
                    provider: provider.id,
                    key: apiKey,
                    key_type: 'api_key',
                    name: 'Jira API Key'
                });
            }
            
            username = '';
            apiKey = '';
            existingUsernameId = '';
            existingApiKeyId = '';
            if (onSave) onSave();
        } catch (e: any) {
            console.error('Error saving credentials:', e);
            error = e.message || 'Failed to save credentials';
        } finally {
            loading = false;
        }
    }

    function handleClose() {
        username = '';
        apiKey = '';
        error = '';
        existingUsernameId = '';
        existingApiKeyId = '';
        if (onClose) onClose();
    }

    async function checkExistingCredentials() {
        try {
            // Check for existing username
            const usernameKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "username"`,
                sort: '-created'
            });
            if (usernameKeys.totalItems > 0) {
                existingUsernameId = usernameKeys.items[0].id;
            }

            // Check for existing API key
            const apiKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "api_key"`,
                sort: '-created'
            });
            if (apiKeys.totalItems > 0) {
                existingApiKeyId = apiKeys.items[0].id;
            }
        } catch (e) {
            console.error('Error checking existing credentials:', e);
        }
    }

    $: if (show) {
        checkExistingCredentials();
    }
</script>

<Modal bind:open={show} size="md" autoclose={false} onClose={handleClose}>
    <form class="space-y-6" on:submit|preventDefault={handleSubmit}>
        <h3 class="text-xl font-medium text-gray-900 dark:text-white mb-4">
            {existingUsernameId || existingApiKeyId ? 'Update' : 'Add'} {provider.name} Credentials
        </h3>

        {#if error}
            <Alert color="red" class="mb-4">
                <span class="font-medium">Error!</span> {error}
            </Alert>
        {/if}

        <div>
            <Label for="username" class="mb-2">Username (Email)</Label>
            <Input
                id="username"
                type="email"
                required
                placeholder="Enter your Jira account email"
                bind:value={username}
            />
            <p class="mt-1 text-sm text-gray-500">
                Your Jira account email address
            </p>
        </div>

        <div>
            <Label for="apiKey" class="mb-2">API Token</Label>
            <Input
                id="apiKey"
                type="password"
                required
                placeholder="Enter your Jira API token"
                bind:value={apiKey}
            />
            <p class="mt-1 text-sm text-gray-500">
                Generate an API token from <a href="https://id.atlassian.com/manage-profile/security/api-tokens" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">Atlassian Account Settings</a>
            </p>
        </div>

        <div class="flex justify-end space-x-2">
            <Button color="alternative" on:click={handleClose}>
                Cancel
            </Button>
            <Button type="submit" disabled={loading}>
                {loading ? 'Saving...' : existingUsernameId || existingApiKeyId ? 'Update Credentials' : 'Save Credentials'}
            </Button>
        </div>
    </form>
</Modal> 
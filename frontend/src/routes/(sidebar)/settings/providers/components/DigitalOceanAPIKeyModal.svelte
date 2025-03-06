<script lang="ts">
    import { Modal, Button, Label, Input, Alert } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider } from '../types';

    export let show = false;
    export let provider: Provider;
    export let onSave: () => void;
    export let onClose: () => void;

    let apiKey = '';
    let error = '';
    let loading = false;
    let existingKeyId = '';

    async function handleSubmit() {
        if (!apiKey) {
            error = 'API key is required';
            return;
        }

        loading = true;
        error = '';
        try {
            if (existingKeyId) {
                // Update existing key
                await $pocketbase.collection('api_keys').update(existingKeyId, {
                    key: apiKey
                });
            } else {
                // Create new key
                await $pocketbase.collection('api_keys').create({
                    provider: provider.id,
                    key: apiKey,
                    key_type: 'api_key',
                    name: 'DigitalOcean API Key'
                });
            }
            
            apiKey = '';
            existingKeyId = '';
            if (onSave) onSave();
        } catch (e: any) {
            console.error('Error saving API key:', e);
            error = e.message || 'Failed to save API key';
        } finally {
            loading = false;
        }
    }

    function handleClose() {
        apiKey = '';
        error = '';
        existingKeyId = '';
        if (onClose) onClose();
    }

    async function checkExistingKey() {
        try {
            const apiKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "api_key"`,
                sort: '-created'
            });
            if (apiKeys.totalItems > 0) {
                existingKeyId = apiKeys.items[0].id;
            }
        } catch (e) {
            console.error('Error checking existing key:', e);
        }
    }

    $: if (show) {
        checkExistingKey();
    }
</script>

<Modal bind:open={show} size="md" autoclose={false} onClose={handleClose}>
    <form class="space-y-6" on:submit|preventDefault={handleSubmit}>
        <h3 class="text-xl font-medium text-gray-900 dark:text-white mb-4">
            {existingKeyId ? 'Update' : 'Add'} {provider.name} API Key
        </h3>

        {#if error}
            <Alert color="red" class="mb-4">
                <span class="font-medium">Error!</span> {error}
            </Alert>
        {/if}

        <div>
            <Label for="apiKey" class="mb-2">API Key</Label>
            <Input
                id="apiKey"
                type="password"
                required
                placeholder="Enter your DigitalOcean API key"
                bind:value={apiKey}
            />
            <p class="mt-1 text-sm text-gray-500">
                You can generate an API key from the <a href="https://cloud.digitalocean.com/account/api/tokens" target="_blank" rel="noopener noreferrer" class="text-blue-600 hover:underline">DigitalOcean Control Panel</a>
            </p>
        </div>

        <div class="flex justify-end space-x-2">
            <Button color="alternative" on:click={handleClose}>
                Cancel
            </Button>
            <Button type="submit" disabled={loading}>
                {loading ? 'Saving...' : existingKeyId ? 'Update API Key' : 'Save API Key'}
            </Button>
        </div>
    </form>
</Modal> 
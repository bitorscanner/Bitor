<script lang="ts">
    import { Modal, Button, Label, Input, Alert } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider } from '../types';

    export let show = false;
    export let provider: Provider;
    export let onSave: () => void;
    export let onClose: () => void;

    let accessKey = '';
    let secretKey = '';
    let error = '';
    let loading = false;
    let existingAccessKeyId = '';
    let existingSecretKeyId = '';

    async function checkExistingKeys() {
        try {
            const accessKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "access_key"`,
                sort: '-created'
            });
            if (accessKeys.totalItems > 0) {
                existingAccessKeyId = accessKeys.items[0].id;
            }

            const secretKeys = await $pocketbase.collection('api_keys').getList(1, 1, {
                filter: `provider = "${provider.id}" && key_type = "secret_key"`,
                sort: '-created'
            });
            if (secretKeys.totalItems > 0) {
                existingSecretKeyId = secretKeys.items[0].id;
            }
        } catch (e) {
            console.error('Error checking existing keys:', e);
        }
    }

    async function handleSubmit() {
        if (!accessKey || !secretKey) {
            error = 'Both Access Key and Secret Key are required';
            return;
        }

        loading = true;
        error = '';
        try {
            if (existingAccessKeyId) {
                // Update existing access key
                await $pocketbase.collection('api_keys').update(existingAccessKeyId, {
                    key: accessKey
                });
            } else {
                // Create new access key
                await $pocketbase.collection('api_keys').create({
                    provider: provider.id,
                    key: accessKey,
                    key_type: 'access_key'
                });
            }

            if (existingSecretKeyId) {
                // Update existing secret key
                await $pocketbase.collection('api_keys').update(existingSecretKeyId, {
                    key: secretKey
                });
            } else {
                // Create new secret key
                await $pocketbase.collection('api_keys').create({
                    provider: provider.id,
                    key: secretKey,
                    key_type: 'secret_key'
                });
            }
            
            accessKey = '';
            secretKey = '';
            existingAccessKeyId = '';
            existingSecretKeyId = '';
            if (onSave) onSave();
        } catch (e: any) {
            console.error('Error saving API keys:', e);
            error = e.message || 'Failed to save API keys';
        } finally {
            loading = false;
        }
    }

    function handleClose() {
        accessKey = '';
        secretKey = '';
        error = '';
        existingAccessKeyId = '';
        existingSecretKeyId = '';
        if (onClose) onClose();
    }

    $: if (show) {
        checkExistingKeys();
    }
</script>

<Modal bind:open={show} size="md" autoclose={false} onClose={handleClose}>
    <form class="space-y-6" on:submit|preventDefault={handleSubmit}>
        <h3 class="text-xl font-medium text-gray-900 dark:text-white mb-4">
            {existingAccessKeyId || existingSecretKeyId ? 'Update' : 'Add'} {provider.name} API Keys
        </h3>

        {#if error}
            <Alert color="red" class="mb-4">
                <span class="font-medium">Error!</span> {error}
            </Alert>
        {/if}

        <div>
            <Label for="accessKey" class="mb-2">Access Key</Label>
            <Input
                id="accessKey"
                type="password"
                required
                placeholder="Enter your S3 Access Key"
                bind:value={accessKey}
            />
        </div>

        <div>
            <Label for="secretKey" class="mb-2">Secret Key</Label>
            <Input
                id="secretKey"
                type="password"
                required
                placeholder="Enter your S3 Secret Key"
                bind:value={secretKey}
            />
            <p class="mt-1 text-sm text-gray-500">
                You can generate these credentials from your S3-compatible storage provider's control panel
            </p>
        </div>

        <div class="flex justify-end space-x-2">
            <Button color="alternative" on:click={handleClose}>
                Cancel
            </Button>
            <Button type="submit" disabled={loading}>
                {loading ? 'Saving...' : (existingAccessKeyId || existingSecretKeyId) ? 'Update API Keys' : 'Save API Keys'}
            </Button>
        </div>
    </form>
</Modal> 
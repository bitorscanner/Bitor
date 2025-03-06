<script lang="ts">
    import { Label, Input, Toggle, Button, Alert } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { Provider, S3Settings } from '../types';
    import S3APIKeyModal from './S3APIKeyModal.svelte';

    export let provider: Provider;
    export let onSave: (provider: Provider) => void;

    let error = '';
    let success = '';
    let showApiKeyModal = false;
    let hasApiKeys = false;

    // Ensure settings has the correct type
    if (!provider.settings || provider.provider_type !== 's3') {
        provider.settings = {
            endpoint: '',
            region: '',
            bucket: '',
            use_path_style: false,
            statefile_path: '',
            scans_path: ''
        } as S3Settings;
    }

    const settings = provider.settings as S3Settings;

    async function saveSettings() {
        // Validate required fields
        if (!settings.endpoint) {
            error = 'Endpoint is required';
            return;
        }
        if (!settings.region) {
            error = 'Region is required';
            return;
        }
        if (!settings.bucket) {
            error = 'Bucket is required';
            return;
        }
        
        // Validate path fields based on uses
        if (provider.uses?.includes('terraform_storage') && !settings.statefile_path) {
            error = 'State File Path is required for Terraform state storage';
            return;
        }
        if (provider.uses?.includes('scan_storage') && !settings.scans_path) {
            error = 'Scans Path is required for scan results storage';
            return;
        }

        try {
            await $pocketbase.collection('providers').update(provider.id, {
                settings: settings,
                updated: new Date().toISOString()
            });
            onSave(provider);
            success = 'Settings saved successfully';
            error = '';
        } catch (e: any) {
            console.error('Error saving settings:', e);
            error = e.message || 'Failed to save settings';
            success = '';
        }
    }

    async function checkApiKeys() {
        try {
            const [accessKeys, secretKeys] = await Promise.all([
                $pocketbase.collection('api_keys').getList(1, 1, {
                    filter: `provider = "${provider.id}" && key_type = "access_key"`,
                }),
                $pocketbase.collection('api_keys').getList(1, 1, {
                    filter: `provider = "${provider.id}" && key_type = "secret_key"`,
                })
            ]);
            hasApiKeys = accessKeys.totalItems > 0 && secretKeys.totalItems > 0;
        } catch (e) {
            console.error('Error checking API keys:', e);
        }
    }

    function handleApiKeySave() {
        showApiKeyModal = false;
        checkApiKeys();
        success = 'API keys saved successfully';
        error = '';
    }

    // Check for API keys on mount and when the modal is closed
    $: if (!showApiKeyModal) {
        checkApiKeys();
    }

    // Debug log to check uses
    $: console.log('Current uses:', provider.uses);
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

    {#if !hasApiKeys}
        <div class="flex justify-center mb-4">
            <Button color="blue" on:click={() => showApiKeyModal = true}>
                Add API Keys
            </Button>
        </div>
    {:else}
        <div class="flex justify-between items-center mb-4">
            <div class="flex items-center gap-2">
                <div class="w-2 h-2 bg-green-500 rounded-full"></div>
                <span class="text-sm text-gray-600">API Keys Configured</span>
            </div>
            <Button size="xs" color="blue" on:click={() => showApiKeyModal = true}>
                Update API Keys
            </Button>
        </div>

        <div>
            <Label for="endpoint">
                Endpoint <span class="text-red-500">*</span>
            </Label>
            <Input
                id="endpoint"
                bind:value={settings.endpoint}
                on:blur={saveSettings}
                placeholder="Enter S3 endpoint"
                required
            />
        </div>

        <div>
            <Label for="region">
                Region <span class="text-red-500">*</span>
            </Label>
            <Input
                id="region"
                bind:value={settings.region}
                on:blur={saveSettings}
                placeholder="Enter S3 region"
                required
            />
        </div>

        <div>
            <Label for="bucket">
                Bucket <span class="text-red-500">*</span>
            </Label>
            <Input
                id="bucket"
                bind:value={settings.bucket}
                on:blur={saveSettings}
                placeholder="Enter bucket name"
                required
            />
        </div>

        <div>
            <Label for="use_path_style">
                Use Path Style <span class="text-gray-500 text-sm">(optional)</span>
            </Label>
            <Toggle
                id="use_path_style"
                bind:checked={settings.use_path_style}
                on:change={saveSettings}
            />
        </div>

        {#if provider.uses?.includes('terraform_storage')}
            <div>
                <Label for="statefile_path">
                    State File Path <span class="text-red-500">*</span>
                </Label>
                <Input
                    id="statefile_path"
                    bind:value={settings.statefile_path}
                    on:blur={saveSettings}
                    placeholder="Enter state file path"
                    required
                />
            </div>
        {/if}

        {#if provider.uses?.includes('scan_storage')}
            <div>
                <Label for="scans_path">
                    Scans Path <span class="text-red-500">*</span>
                </Label>
                <Input
                    id="scans_path"
                    bind:value={settings.scans_path}
                    on:blur={saveSettings}
                    placeholder="Enter scans path"
                    required
                />
            </div>
        {/if}
    {/if}
</div>

<S3APIKeyModal
    bind:show={showApiKeyModal}
    {provider}
    onSave={handleApiKeySave}
    onClose={() => showApiKeyModal = false}
/> 
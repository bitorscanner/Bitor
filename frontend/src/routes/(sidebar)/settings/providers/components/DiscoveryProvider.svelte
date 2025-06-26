<!-- Full file path: frontend/src/routes/(sidebar)/settings/providers/components/DiscoveryProvider.svelte -->
<script lang="ts">
  import { 
    Button,
    Input,
    Alert
  } from 'flowbite-svelte';
  import { ArrowRightAltSolid } from 'flowbite-svelte-icons';
  import { pocketbase } from '@lib/stores/pocketbase';
  import type { Provider } from '../types';

  export let provider: Provider;
  export let onSave: () => void;

  let apiKey = '';
  let loading = false;
  let error = '';
  let success = '';
  let existingApiKey: string | null = null;

  async function loadApiKey() {
    try {
      const result = await $pocketbase.collection('api_keys').getList(1, 1, {
        filter: `provider = "${provider.id}" && key_type = "api_key"`
      });

      if (result.items.length > 0) {
        existingApiKey = result.items[0].id;
        // Don't show the actual key, just indicate it exists
        apiKey = '••••••••';
      }
    } catch (e: any) {
      console.error('Error loading API key:', e);
      error = e.message || 'Failed to load API key';
    }
  }

  async function saveApiKey() {
    if (!apiKey) {
      error = 'API key is required';
      return;
    }

    try {
      loading = true;
      error = '';
      success = '';

      if (existingApiKey) {
        // Update existing API key
        await $pocketbase.collection('api_keys').update(existingApiKey, {
          key: apiKey,
          key_type: 'api_key',
          provider: provider.id
        });
      } else {
        // Create new API key
        const result = await $pocketbase.collection('api_keys').create({
          name: `${provider.provider_type} Discovery API Key`,
          key: apiKey,
          key_type: 'api_key',
          provider: provider.id
        });
        existingApiKey = result.id;
      }

      success = 'API key saved successfully';
      onSave();
      // Reset form but keep existingApiKey status
      apiKey = '••••••••';
    } catch (e: any) {
      console.error('Error saving API key:', e);
      error = e.message || 'Failed to save API key';
    } finally {
      loading = false;
    }
  }

  async function testApiKey() {
    if (!existingApiKey) {
      error = 'Please save the API key first';
      return;
    }

    try {
      loading = true;
      error = '';
      success = '';

      const result = await $pocketbase.send('/api/discovery/test', {
        method: 'POST',
        body: JSON.stringify({
          service_type: provider.provider_type,
          provider_id: provider.id
        })
      });

      if (result.success) {
        success = 'API key tested successfully';
      } else {
        error = result.message || 'API key test failed';
      }
    } catch (e: any) {
      console.error('Error testing API key:', e);
      error = e.message || 'Failed to test API key';
    } finally {
      loading = false;
    }
  }

  $: {
    if (provider) {
      loadApiKey();
    }
  }
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

  <div class="flex gap-4">
    <div class="flex-1">
      <Input
        type="password"
        placeholder="Enter API key..."
        bind:value={apiKey}
        on:focus={() => {
          if (apiKey === '••••••••') {
            apiKey = '';
          }
        }}
      />
    </div>
    <Button 
      disabled={loading} 
      on:click={saveApiKey}
    >
      {loading ? 'Saving...' : 'Save Key'}
    </Button>
    <Button 
      color="light"
      disabled={loading || !existingApiKey} 
      on:click={testApiKey}
    >
      <ArrowRightAltSolid class="w-4 h-4 mr-1" />
      Test Key
    </Button>
  </div>
</div> 
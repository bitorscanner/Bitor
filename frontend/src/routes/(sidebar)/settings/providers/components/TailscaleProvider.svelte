<!-- Full file path: frontend/src/routes/(sidebar)/settings/providers/components/TailscaleProvider.svelte -->
<script lang="ts">
  import { 
    Button,
    Input,
    Alert,
    Label,
    Textarea
  } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import type { Provider } from '../types';

  export let provider: Provider;
  export let onSave: () => void;

  let apiKey = '';
  let tailnet = '';
  let tags = '';
  let subnetRoutes = '';
  let loading = false;
  let error = '';
  let success = '';
  let existingApiKey: string | null = null;

  async function loadApiKey() {
    try {
      const result = await $pocketbase.collection('api_keys').getList(1, 1, {
        filter: `provider = "${provider.id}" && key_type = "tailscale"`
      });

      if (result.items.length > 0) {
        existingApiKey = result.items[0].id;
        apiKey = '••••••••';
      }

      // Load settings
      const settings = provider.settings as TailscaleSettings;
      tailnet = settings?.tailnet || '';
      tags = (settings?.tags || []).join('\n');
      subnetRoutes = (settings?.subnet_routes || []).join('\n');
    } catch (e: any) {
      console.error('Error loading API key:', e);
      error = e.message || 'Failed to load API key';
    }
  }

  async function saveSettings() {
    if (!apiKey && !existingApiKey) {
      error = 'API key is required';
      return;
    }

    try {
      loading = true;
      error = '';
      success = '';

      // Save API key
      if (existingApiKey && apiKey !== '••••••••') {
        await $pocketbase.collection('api_keys').update(existingApiKey, {
          key: apiKey,
          key_type: 'tailscale',
          provider: provider.id
        });
      } else if (!existingApiKey) {
        const result = await $pocketbase.collection('api_keys').create({
          key: apiKey,
          key_type: 'tailscale',
          provider: provider.id
        });
        existingApiKey = result.id;
      }

      // Save provider settings
      if (provider.id) {
        const settings = {
          ...provider.settings,
          tailnet,
          tags: tags.split('\n').filter(t => t.trim()),
          subnet_routes: subnetRoutes.split('\n').filter(r => r.trim())
        };

        await $pocketbase.collection('providers').update(provider.id, {
          settings
        });

        provider.settings = settings;
      }

      success = 'Settings saved successfully';
      onSave();
      // Reset form but keep existingApiKey status
      if (apiKey !== '••••••••') {
        apiKey = '••••••••';
      }
    } catch (e: any) {
      console.error('Error saving settings:', e);
      error = e.message || 'Failed to save settings';
    } finally {
      loading = false;
    }
  }

  async function testConnection() {
    if (!existingApiKey) {
      error = 'Please save the API key first';
      return;
    }

    try {
      loading = true;
      error = '';
      success = '';

      const result = await $pocketbase.send('/api/tailscale/test', {
        method: 'POST',
        body: JSON.stringify({
          provider_id: provider.id
        })
      });

      if (result.success) {
        success = 'Connection tested successfully';
      } else {
        error = result.message || 'Connection test failed';
      }
    } catch (e: any) {
      console.error('Error testing connection:', e);
      error = e.message || 'Failed to test connection';
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

  <div class="grid grid-cols-1 gap-4">
    <div>
      <Label for="api-key">API Key</Label>
      <Input
        id="api-key"
        type="password"
        placeholder="Enter Tailscale API key..."
        bind:value={apiKey}
        on:focus={() => {
          if (apiKey === '••••••••') {
            apiKey = '';
          }
        }}
      />
    </div>

    <div>
      <Label for="tailnet">Tailnet</Label>
      <Input
        id="tailnet"
        type="text"
        placeholder="example.com"
        bind:value={tailnet}
      />
    </div>

    <div>
      <Label for="tags">Tags (one per line)</Label>
      <Textarea
        id="tags"
        placeholder="tag:prod
tag:web
tag:db"
        rows={4}
        bind:value={tags}
      />
    </div>

    <div>
      <Label for="subnet-routes">Subnet Routes (one per line)</Label>
      <Textarea
        id="subnet-routes"
        placeholder="10.0.0.0/24
172.16.0.0/16"
        rows={4}
        bind:value={subnetRoutes}
      />
    </div>

    <div class="flex gap-4">
      <Button 
        disabled={loading} 
        on:click={saveSettings}
      >
        {loading ? 'Saving...' : 'Save Settings'}
      </Button>
      <Button 
        color="light"
        disabled={loading || !existingApiKey} 
        on:click={testConnection}
      >
        Test Connection
      </Button>
    </div>
  </div>
</div> 
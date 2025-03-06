<!-- Full file path: frontend/src/routes/(sidebar)/settings/providers/components/AIProvider.svelte -->
<script lang="ts">
  import { 
    Button,
    Input,
    Alert,
    Select
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
  let modelType = '';

  // Model options for each provider
  const modelOptions = {
    openai: [
      { value: 'gpt-4', name: 'GPT-4' },
      { value: 'gpt-4-turbo', name: 'GPT-4 Turbo' },
      { value: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo' }
    ],
    anthropic: [
      { value: 'claude-3-opus', name: 'Claude 3 Opus' },
      { value: 'claude-3-sonnet', name: 'Claude 3 Sonnet' },
      { value: 'claude-2.1', name: 'Claude 2.1' }
    ],
    google: [
      { value: 'gemini-pro', name: 'Gemini Pro' },
      { value: 'gemini-ultra', name: 'Gemini Ultra' }
    ],
    mistral: [
      { value: 'mistral-large', name: 'Mistral Large' },
      { value: 'mistral-medium', name: 'Mistral Medium' },
      { value: 'mistral-small', name: 'Mistral Small' }
    ],
    ollama: [
      { value: 'llama2', name: 'Llama 2' },
      { value: 'mistral', name: 'Mistral' },
      { value: 'codellama', name: 'Code Llama' }
    ],
    cohere: [
      { value: 'command', name: 'Command' },
      { value: 'command-light', name: 'Command Light' }
    ]
  };

  async function loadApiKey() {
    try {
      const result = await $pocketbase.collection('api_keys').getList(1, 1, {
        filter: `provider = "${provider.id}" && key_type = "ai"`
      });

      if (result.items.length > 0) {
        existingApiKey = result.items[0].id;
        // Don't show the actual key, just indicate it exists
        apiKey = '••••••••';
        // Load the model type from settings
        modelType = provider.settings?.model_type || '';
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

    if (!modelType) {
      error = 'Please select a model type';
      return;
    }

    try {
      loading = true;
      error = '';
      success = '';

      // Update provider settings with model type
      if (provider.id) {
        await $pocketbase.collection('providers').update(provider.id, {
          settings: {
            ...provider.settings,
            model_type: modelType
          }
        });
      }

      if (existingApiKey) {
        // Update existing API key
        await $pocketbase.collection('api_keys').update(existingApiKey, {
          key: apiKey === '••••••••' ? undefined : apiKey,
          key_type: 'ai',
          provider: provider.id
        });
      } else {
        // Create new API key
        const result = await $pocketbase.collection('api_keys').create({
          key: apiKey,
          key_type: 'ai',
          provider: provider.id
        });
        existingApiKey = result.id;
      }

      success = 'Settings saved successfully';
      onSave();
      // Reset form but keep existingApiKey status
      apiKey = '••••••••';
    } catch (e: any) {
      console.error('Error saving settings:', e);
      error = e.message || 'Failed to save settings';
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

      const result = await $pocketbase.send('/api/ai/test', {
        method: 'POST',
        body: JSON.stringify({
          provider_type: provider.provider_type,
          provider_id: provider.id,
          model_type: modelType
        })
      });

      if (result.success) {
        success = 'API key and model tested successfully';
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

  $: availableModels = modelOptions[provider.provider_type as keyof typeof modelOptions] || [];

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

  <div class="space-y-4">
    <div>
      <Select
        class="w-full"
        items={availableModels}
        bind:value={modelType}
        placeholder="Select a model..."
      />
    </div>

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
        {loading ? 'Saving...' : 'Save Settings'}
      </Button>
      <Button 
        color="light"
        disabled={loading || !existingApiKey} 
        on:click={testApiKey}
      >
        <ArrowRightAltSolid class="w-4 h-4 mr-1" />
        Test Connection
      </Button>
    </div>
  </div>
</div> 
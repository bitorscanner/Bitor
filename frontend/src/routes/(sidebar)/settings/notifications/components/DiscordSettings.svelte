<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input, Label, Toggle } from 'flowbite-svelte';

    export let settings: {
        enabled: boolean;
        webhook_id: string;
        token: string;
    };

    const dispatch = createEventDispatcher();

    function handleChange() {
        dispatch('change', settings);
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <Label>Enable Discord Integration</Label>
        <Toggle 
            bind:checked={settings.enabled} 
            on:change={handleChange}
        />
    </div>

    {#if settings.enabled}
        <div>
            <Label for="webhook_id">Webhook ID</Label>
            <Input
                id="webhook_id"
                type="text"
                placeholder="Discord Webhook ID"
                bind:value={settings.webhook_id}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">The ID part of your Discord webhook URL</p>
        </div>

        <div>
            <Label for="token">Webhook Token</Label>
            <Input
                id="token"
                type="password"
                placeholder="Discord Webhook Token"
                bind:value={settings.token}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">
                Create a webhook in your Discord server's channel settings and extract the ID and token from the webhook URL:
                https://discord.com/api/webhooks/[WEBHOOK_ID]/[TOKEN]
            </p>
        </div>
    {/if}
</div> 
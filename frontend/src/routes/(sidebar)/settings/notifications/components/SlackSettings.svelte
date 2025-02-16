<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input, Label, Toggle } from 'flowbite-svelte';

    export let settings: {
        enabled: boolean;
        token: string;
        channel: string;
    };

    const dispatch = createEventDispatcher();

    function handleChange() {
        dispatch('change', settings);
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <Label>Enable Slack Integration</Label>
        <Toggle 
            bind:checked={settings.enabled} 
            on:change={handleChange}
        />
    </div>

    {#if settings.enabled}
        <div>
            <Label for="token">Bot Token</Label>
            <Input
                id="token"
                type="password"
                placeholder="xoxb-your-token"
                bind:value={settings.token}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">
                Create a Slack app and get the Bot User OAuth Token from 
                <a 
                    href="https://api.slack.com/apps" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    class="text-blue-600 hover:underline"
                >
                    Slack API Dashboard
                </a>
            </p>
        </div>

        <div>
            <Label for="channel">Channel</Label>
            <Input
                id="channel"
                type="text"
                placeholder="#notifications"
                bind:value={settings.channel}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">The channel where notifications will be sent (e.g., #notifications)</p>
        </div>
    {/if}
</div> 
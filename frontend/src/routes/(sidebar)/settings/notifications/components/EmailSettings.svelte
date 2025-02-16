<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input, Label, Toggle, Select } from 'flowbite-svelte';

    export let settings: {
        enabled: boolean;
        host: string;
        port: number;
        username: string;
        password: string;
        from: string;
        to: string[];
    };

    const dispatch = createEventDispatcher();

    function handleChange() {
        dispatch('change', settings);
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <Label>Enable Email Notifications</Label>
        <Toggle 
            bind:checked={settings.enabled} 
            on:change={handleChange}
        />
    </div>

    {#if settings.enabled}
        <div>
            <Label for="host">SMTP Host</Label>
            <Input
                id="host"
                type="text"
                placeholder="smtp.example.com"
                bind:value={settings.host}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">Your SMTP server hostname</p>
        </div>

        <div>
            <Label for="port">SMTP Port</Label>
            <Input
                id="port"
                type="number"
                placeholder="587"
                bind:value={settings.port}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">Common ports: 25 (SMTP), 465 (SMTPS), 587 (Submission)</p>
        </div>

        <div>
            <Label for="username">Username</Label>
            <Input
                id="username"
                type="text"
                placeholder="username@example.com"
                bind:value={settings.username}
                on:change={handleChange}
            />
        </div>

        <div>
            <Label for="password">Password</Label>
            <Input
                id="password"
                type="password"
                placeholder="SMTP password"
                bind:value={settings.password}
                on:change={handleChange}
            />
        </div>

        <div>
            <Label for="from">From Address</Label>
            <Input
                id="from"
                type="email"
                placeholder="notifications@example.com"
                bind:value={settings.from}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">The email address notifications will be sent from</p>
        </div>
    {/if}
</div> 
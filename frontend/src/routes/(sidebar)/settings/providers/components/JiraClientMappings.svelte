<script lang="ts">
    import { Label, Select, Button } from 'flowbite-svelte';
    import type { JiraClientMapping } from '../types';
    import { createEventDispatcher } from 'svelte';

    export let clients: Array<{ id: string; name: string }> = [];
    export let organizations: Array<{ id: string; name: string }> = [];
    export let mappings: JiraClientMapping[] = [];

    const dispatch = createEventDispatcher<{
        change: JiraClientMapping[];
    }>();

    function addMapping() {
        mappings = [...mappings, { client_id: '', organization_id: '' }];
        dispatch('change', mappings);
    }

    function removeMapping(index: number) {
        mappings = mappings.filter((_, i) => i !== index);
        dispatch('change', mappings);
    }

    function updateMapping(index: number, field: keyof JiraClientMapping, value: string) {
        mappings[index] = { ...mappings[index], [field]: value };
        mappings = [...mappings]; // Trigger reactivity
        dispatch('change', mappings);
    }
</script>

<div class="space-y-4">
    <div class="flex justify-between items-center">
        <h3 class="text-lg font-medium">Client Organization Mappings</h3>
        <Button size="xs" on:click={addMapping}>Add Mapping</Button>
    </div>

    {#if mappings.length === 0}
        <p class="text-sm text-gray-500">No mappings configured. Click "Add Mapping" to create one.</p>
    {:else}
        {#each mappings as mapping, index}
            <div class="flex gap-4 items-end">
                <div class="flex-1">
                    <Label>Client</Label>
                    <Select
                        value={mapping.client_id}
                        on:change={(e) => updateMapping(index, 'client_id', e.target.value)}
                    >
                        <option value="">Select a client</option>
                        {#each clients as client}
                            <option value={client.id}>{client.name}</option>
                        {/each}
                    </Select>
                </div>

                <div class="flex-1">
                    <Label>Organization</Label>
                    <Select
                        value={mapping.organization_id}
                        on:change={(e) => updateMapping(index, 'organization_id', e.target.value)}
                    >
                        <option value="">Select an organization</option>
                        {#each organizations as org}
                            <option value={org.id}>{org.name}</option>
                        {/each}
                    </Select>
                </div>

                <Button color="red" size="xs" on:click={() => removeMapping(index)}>Remove</Button>
            </div>
        {/each}
    {/if}
</div> 
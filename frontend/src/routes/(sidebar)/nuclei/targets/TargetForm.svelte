<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { Button, Input, Label, Modal, Select } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';

    interface Client {
        id: string;
        name: string;
    }

    interface TargetData {
        name: string;
        targets: string[];
        count: number;
        client: string;
        clientName?: string;
        id?: string;
    }

    export let open = false;
    export let target: TargetData | null = null;
    export let onSave: (targetData: TargetData) => void;
    export let mode: 'add' | 'edit' = 'add';

    let targetName = '';
    let addresses = '';
    let selectedClient = '';
    let clients: Client[] = [];

    const dispatch = createEventDispatcher<{
        save: TargetData;
    }>();

    onMount(async () => {
        try {
            const result = await $pocketbase.collection('clients').getList();
            clients = result.items.map(client => ({
                id: client.id,
                name: client.name
            }));
        } catch (error) {
            console.error('Error fetching clients:', error);
        }
    });

    let prevOpen = false;
    let prevTarget: TargetData | null = null;
    let prevMode = mode;

    $: {
        if (open && (prevOpen !== open || prevTarget !== target || prevMode !== mode)) {
            console.log('Reactive statement triggered');
            if (mode === 'edit' && target) {
                targetName = target.name;
                addresses = target.targets.join('\n');
                selectedClient = target.client;
            } else {
                targetName = '';
                addresses = '';
                selectedClient = '';
            }
        }
        prevOpen = open;
        prevTarget = target;
        prevMode = mode;
    }

    async function handleSave() {
        const targetsList = addresses.split('\n').filter(line => line.trim() !== '');
        
        const targetData: TargetData = {
            name: targetName,
            targets: targetsList,
            count: targetsList.length,
            client: selectedClient,
            ...(target?.id ? { id: target.id } : {})
        };

        if (onSave) {
            onSave(targetData);
        }
        dispatch('save', targetData);
        open = false;
    }
</script>

<Modal bind:open size="lg" title={mode === 'edit' ? "Edit Target" : "Add New Target"}>
    <form on:submit|preventDefault={handleSave} class="space-y-6">
        <Label class="space-y-2">
            <span class="text-gray-700 dark:text-gray-300">Target Name</span>
            <Input bind:value={targetName} placeholder="Enter target name" class="mt-1 block w-full border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white" required />
        </Label>
        <Label class="space-y-2">
            <span class="text-gray-700 dark:text-gray-300">Addresses</span>
            <textarea bind:value={addresses} rows="10" class="mt-1 block w-full border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white p-2" placeholder="Enter addresses, one per line" required></textarea>
        </Label>
        <Label class="space-y-2">
            <span class="text-gray-700 dark:text-gray-300">Client</span>
            <Select bind:value={selectedClient} placeholder="Select a client" class="mt-1 block w-full border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-white" required>
                {#each clients as client}
                    <option value={client.id}>{client.name}</option>
                {/each}
            </Select>
        </Label>
        <Button type="submit" class="w-full text-white py-2">{mode === 'edit' ? "Update" : "Save"}</Button>
    </form>
</Modal>
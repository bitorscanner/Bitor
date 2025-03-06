<!-- Archive.svelte -->
<script lang="ts">
    import { Modal, Button } from 'flowbite-svelte';
    export let open: boolean = false;
    export let onArchive: () => void;
</script>

<Modal bind:open size="sm">
    <div class="text-center">
        <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
            Are you sure you want to archive this scan?
        </h3>
        <div class="flex justify-center gap-4">
            <Button color="red" on:click={onArchive}>
                Yes, archive it
            </Button>
            <Button color="alternative" on:click={() => open = false}>
                No, cancel
            </Button>
        </div>
    </div>
</Modal> 
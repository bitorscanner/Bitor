<script lang="ts">
    import { Button, Modal } from 'flowbite-svelte';
    export let open: boolean = false;
    export let onDestroy: () => void;

    function handleDestroy() {
        onDestroy();
        open = false;
    }
</script>

<Modal bind:open size="sm">
    <h3 class="mb-6 text-center text-lg text-gray-500 dark:text-gray-400">
        Are you sure you want to destroy this scan? This action cannot be undone.
    </h3>
    <div class="flex items-center justify-center">
        <Button color="red" class="mr-2" on:click={handleDestroy}>Yes, destroy it</Button>
        <Button color="alternative" on:click={() => (open = false)}>No, cancel</Button>
    </div>
</Modal> 
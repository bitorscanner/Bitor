<script lang="ts">
    import { Button, Modal } from 'flowbite-svelte';
    export let open: boolean = false;
    export let onStart: () => void;

    function handleStart() {
        onStart();
        open = false;
    }
</script>

<Modal bind:open size="sm">
    <h3 class="mb-6 text-center text-lg text-gray-500 dark:text-gray-400">
        Are you sure you want to start this scan?
    </h3>
    <div class="flex items-center justify-center">
        <Button class="mr-2" on:click={handleStart}>Yes, start it</Button>
        <Button color="alternative" on:click={() => (open = false)}>No, cancel</Button>
    </div>
</Modal>


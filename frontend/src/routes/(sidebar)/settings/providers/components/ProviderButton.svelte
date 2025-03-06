<!-- Full file path: frontend/src/routes/(sidebar)/settings/providers/components/ProviderButton.svelte -->
<script lang="ts">
  import { Button } from 'flowbite-svelte';

  export let icon: any = null;
</script>

<div class="w-full">
  <Button color="light" on:click>
    <div class="flex items-center w-full">
      {#if icon}
        <svelte:component this={icon} class="mr-2" />
      {/if}
      <span class="flex-1 text-left">
        <slot />
      </span>
    </div>
  </Button>
</div> 
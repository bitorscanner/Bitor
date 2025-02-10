{#if target}
  <div bind:this={portalContainer} class={className}>
    <slot />
  </div>
{/if}

<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  export let target: HTMLElement | null = null;
  export let className: string = '';
  
  let portalContainer: HTMLElement;

  onMount(() => {
    if (target && portalContainer) {
      target.appendChild(portalContainer);
    }
  });

  onDestroy(() => {
    if (portalContainer && portalContainer.parentNode) {
      portalContainer.parentNode.removeChild(portalContainer);
    }
  });
</script> 
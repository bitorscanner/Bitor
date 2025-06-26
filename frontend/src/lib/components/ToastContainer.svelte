<script lang="ts">
  import { toasts } from '$lib/stores/toasts';
  import { Toast } from 'flowbite-svelte';
  import { CheckCircleSolid, ExclamationCircleSolid, ExclamationCircleOutline, InfoCircleOutline } from 'flowbite-svelte-icons';
  import { fade } from 'svelte/transition';

  function getIcon(type: string) {
    switch (type) {
      case 'success':
        return CheckCircleSolid;
      case 'error':
        return ExclamationCircleSolid;
      case 'warning':
        return ExclamationCircleOutline;
      case 'info':
      default:
        return InfoCircleOutline;
    }
  }

  function getColor(type: string) {
    switch (type) {
      case 'success':
        return 'green';
      case 'error':
        return 'red';
      case 'warning':
        return 'yellow';
      case 'info':
      default:
        return 'blue';
    }
  }
</script>

<div class="fixed top-20 right-4 z-50 space-y-2">
  {#each $toasts as toast (toast.id)}
    <div transition:fade={{ duration: 300 }}>
      <Toast
        color={getColor(toast.type)}
        on:close={() => toasts.remove(toast.id || '')}
      >
        <svelte:fragment slot="icon">
          <svelte:component this={getIcon(toast.type)} class="w-5 h-5" />
        </svelte:fragment>
        {toast.message}
      </Toast>
    </div>
  {/each}
</div> 
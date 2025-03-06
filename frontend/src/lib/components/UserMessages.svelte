<!-- /frontend/src/lib/components/UserMessages.svelte -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Toast } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { fade } from 'svelte/transition';

  let messages: any[] = [];
  let unsubscribe: any;

  async function fetchMessages() {
    try {
      const resultList = await $pocketbase.collection('user_messages').getList(1, 50, {
        filter: 'read = false',
        sort: '-created'
      });
      messages = resultList.items;
    } catch (error) {
      console.error('Error fetching messages:', error);
    }
  }

  async function markAsRead(messageId: string) {
    try {
      await $pocketbase.collection('user_messages').update(messageId, {
        read: true
      });
      messages = messages.filter(m => m.id !== messageId);
    } catch (error) {
      console.error('Error marking message as read:', error);
    }
  }

  function getColorClass(type: string) {
    switch (type) {
      case 'success':
        return 'bg-green-100 text-green-800 dark:bg-green-800 dark:text-green-200';
      case 'error':
        return 'bg-red-100 text-red-800 dark:bg-red-800 dark:text-red-200';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-800 dark:text-yellow-200';
      default:
        return 'bg-blue-100 text-blue-800 dark:bg-blue-800 dark:text-blue-200';
    }
  }

  function formatDate(date: string) {
    const d = new Date(date);
    return d.toLocaleString();
  }

  onMount(async () => {
    console.log('UserMessages component mounted');
    await fetchMessages();
    console.log('Initial messages:', messages);

    // Subscribe to realtime messages
    unsubscribe = await $pocketbase.collection('user_messages').subscribe('*', async ({ action, record }) => {
      console.log('Received realtime message:', { action, record });
      if (action === 'create') {
        const userId = $pocketbase.authStore.model?.id;
        console.log('Current user ID:', userId);
        console.log('Message user ID:', record.user);
        console.log('Message admin ID:', record.admin_id);
        if (record.user === userId || record.admin_id === userId) {
          console.log('Adding new message to display');
          messages = [record, ...messages];
        } else {
          console.log('Message not for current user');
        }
      } else if (action === 'delete') {
        messages = messages.filter(m => m.id !== record.id);
      }
    });
    console.log('Subscribed to messages');
  });

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe();
    }
  });
</script>

<div class="max-h-[400px] overflow-y-auto">
  {#if messages.length === 0}
    <div class="p-4 text-center text-gray-500 dark:text-gray-400">
      No new notifications
    </div>
  {:else}
    {#each messages as message (message.id)}
      <div transition:fade class="border-b dark:border-gray-700 last:border-b-0">
        <div class={`p-4 ${getColorClass(message.type)}`}>
          <div class="flex flex-col gap-2">
            <p class="flex-1">{message.message}</p>
            <div class="flex items-center justify-between text-sm">
              <span class="text-gray-500 dark:text-gray-400">{formatDate(message.created)}</span>
              <button
                class="text-gray-600 dark:text-gray-300 hover:underline"
                on:click={() => markAsRead(message.id)}
              >
                Dismiss
              </button>
            </div>
          </div>
        </div>
      </div>
    {/each}
  {/if}
</div> 
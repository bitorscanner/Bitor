<script lang="ts">
    import { onMount } from 'svelte';
    import { currentUser } from '$lib/stores/auth';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { defaultAvatarPath } from '$lib/utils/variables.js';

    $: avatarUrl = $currentUser?.avatar ? 
        $pocketbase.files.getUrl($currentUser, $currentUser.avatar) : 
        defaultAvatarPath;

    onMount(() => {
        console.log('Header mounted, avatarUrl:', avatarUrl);
    });

    function handleError(event: Event) {
        const img = event.target as HTMLImageElement;
        console.error('Avatar failed to load:', img.src);
    }
</script>

<div class="header-avatar">
    {#if $currentUser?.avatar}
        <img 
            src={avatarUrl}
            class="cursor-pointer rounded-full w-8 h-8 bg-gray-100 dark:bg-gray-600 text-gray-600 dark:text-gray-300 object-cover"
            alt="User avatar"
            on:error={handleError}
        />
    {/if}
</div> 
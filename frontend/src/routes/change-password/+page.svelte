<script lang="ts">
    import { onMount } from 'svelte';
    import { goto } from '$app/navigation';
    import { pocketbase } from '@lib/stores/pocketbase';
    import ChangePasswordRequired from '../utils/authentication/ChangePasswordRequired.svelte';

    let userId = '';

    onMount(() => {
        // Check if user is logged in and requires password change
        const user = $pocketbase.authStore.model;
        console.log('Checking user:', user);
        
        if (!user) {
            goto('/login');
            return;
        }

        if (!user.requirePasswordChange) {
            goto('/');
            return;
        }

        userId = user.id;
    });
</script>

{#if userId}
    <ChangePasswordRequired {userId} />
{/if} 
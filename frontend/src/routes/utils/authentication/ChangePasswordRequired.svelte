<script lang="ts">
    import { Label, Input, Button, Card } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { goto } from '$app/navigation';

    export let userId: string;
    let error = '';
    let loading = false;
    let currentPassword = '';
    let password = '';
    let passwordConfirm = '';

    async function handleSubmit() {
        if (!currentPassword || !password || !passwordConfirm) {
            error = 'Please fill in all fields';
            return;
        }

        if (password.length < 10) {
            error = 'Password must be at least 10 characters long';
            return;
        }

        if (password !== passwordConfirm) {
            error = 'Passwords do not match';
            return;
        }

        try {
            loading = true;
            error = '';

            // Get current user's email before password change
            const user = await $pocketbase.collection('users').getOne(userId);
            const email = user.email;

            // Update password
            await $pocketbase.collection('users').update(userId, {
                oldPassword: currentPassword,
                password,
                passwordConfirm,
                requirePasswordChange: false
            });

            // Re-authenticate with new password
            await $pocketbase.collection('users').authWithPassword(email, password);

            // Redirect to dashboard after successful password change
            goto('/');
        } catch (err) {
            console.error('Password change failed:', err);
            error = err instanceof Error ? err.message : 'Failed to update password';
        } finally {
            loading = false;
        }
    }
</script>

<main class="bg-gray-50 dark:bg-gray-900 w-full">
    <div class="flex flex-col items-center justify-center px-6 pt-8 mx-auto md:h-screen pt:mt-0 dark:bg-gray-900">
        <Card class="w-full max-w-md" size="md" border={false}>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white mb-6">
                Change Password Required
            </h1>
            <p class="text-gray-600 dark:text-gray-400 mb-6">
                For security reasons, you must change your password before continuing.
            </p>

            <form class="space-y-6" on:submit|preventDefault={handleSubmit}>
                <div>
                    <Label for="currentPassword">Current Password *</Label>
                    <Input 
                        id="currentPassword" 
                        type="password" 
                        bind:value={currentPassword} 
                        required 
                    />
                </div>

                <div>
                    <Label for="password">New Password *</Label>
                    <Input 
                        id="password" 
                        type="password" 
                        bind:value={password} 
                        required 
                        minlength={10}
                    />
                    <p class="text-sm text-gray-500 mt-1">Password must be at least 10 characters long</p>
                </div>

                <div>
                    <Label for="passwordConfirm">Confirm New Password *</Label>
                    <Input 
                        id="passwordConfirm" 
                        type="password" 
                        bind:value={passwordConfirm} 
                        required
                    />
                </div>

                {#if error}
                    <p class="text-red-500">{error}</p>
                {/if}

                <Button type="submit" size="lg" disabled={loading}>
                    {loading ? 'Changing Password...' : 'Change Password'}
                </Button>
            </form>
        </Card>
    </div>
</main> 
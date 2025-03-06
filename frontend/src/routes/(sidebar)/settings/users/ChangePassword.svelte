<script lang="ts">
    import { Button, Input, Label, Modal, Toast } from 'flowbite-svelte';
    import { slide } from 'svelte/transition';
    import { CheckCircleSolid, ExclamationCircleSolid } from 'flowbite-svelte-icons';
    import { pocketbase } from '@lib/stores/pocketbase';

    export let open: boolean = false;
    export let userId: string;
    let newPassword = '';
    let confirmPassword = '';
    let errorMessage = '';
    let successMessage = '';
    let loading = false;

    async function changePassword() {
        if (!newPassword || newPassword.length < 10) {
            errorMessage = 'Password must be at least 10 characters long.';
            return;
        }

        if (newPassword !== confirmPassword) {
            errorMessage = 'Passwords do not match.';
            return;
        }

        loading = true;
        errorMessage = '';
        successMessage = '';

        try {
            await $pocketbase.collection('users').update(userId, {
                password: newPassword,
                passwordConfirm: confirmPassword
            });
            successMessage = 'Password changed successfully';
            setTimeout(() => {
                open = false;
                // Reset form
                newPassword = '';
                confirmPassword = '';
                successMessage = '';
            }, 2000);
        } catch (error) {
            errorMessage = error.message || 'Error changing password';
        } finally {
            loading = false;
        }
    }

    // Reset form when modal is opened
    $: if (open) {
        newPassword = '';
        confirmPassword = '';
        errorMessage = '';
        successMessage = '';
        loading = false;
    }
</script>

<Modal bind:open title="Reset User Password">
    <form class="space-y-4" on:submit|preventDefault={changePassword}>
        <div class="space-y-2">
            <Label for="newPassword">New Password</Label>
            <Input
                id="newPassword"
                type="password"
                bind:value={newPassword}
                placeholder="Enter new password"
                required
                minlength="10"
            />
            <p class="text-sm text-gray-500">Password must be at least 10 characters long</p>
        </div>

        <div class="space-y-2">
            <Label for="confirmPassword">Confirm Password</Label>
            <Input
                id="confirmPassword"
                type="password"
                bind:value={confirmPassword}
                placeholder="Confirm new password"
                required
            />
        </div>

        {#if errorMessage}
            <Toast color="red" transition={slide}>
                <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
                {errorMessage}
            </Toast>
        {/if}

        {#if successMessage}
            <Toast color="green" transition={slide}>
                <CheckCircleSolid slot="icon" class="w-5 h-5" />
                {successMessage}
            </Toast>
        {/if}

        <div class="flex justify-end space-x-2">
            <Button color="alternative" on:click={() => (open = false)}>Cancel</Button>
            <Button type="submit" disabled={loading}>
                {loading ? 'Changing Password...' : 'Change Password'}
            </Button>
        </div>
    </form>
</Modal>

<script lang="ts">
	import { Button, Modal } from 'flowbite-svelte';
	import { ExclamationCircleOutline } from 'flowbite-svelte-icons';
	import { pocketbase } from '@lib/stores/pocketbase'; // Import the pocketbase instance
	import { goto } from '$app/navigation'; // Import the goto function
	export let open: boolean = false; // modal control
	export let userId: string; // The ID of the user to delete
	export let userRole: string; // Add a prop to receive the user's role

	async function deleteUser() {
		if (userRole === 'Superadmin') {
			return; // Prevent deletion if the user is a superadmin
		}

		try {
			await $pocketbase.collection('users').delete(userId);
			open = false; // Close the modal after deletion
			goto('/settings/users'); // Refresh the page by navigating to the same path
		} catch (error) {
			console.error('Error deleting user:', error);
		}
	}
</script>

<Modal bind:open size="sm">
	<ExclamationCircleOutline class="mx-auto mb-4 mt-8 h-10 w-10 text-red-600" />

	<h3 class="mb-6 text-center text-lg text-gray-500 dark:text-gray-400">
		{#if userRole === 'Superadmin'}
			Superadmin cannot be deleted.
		{:else}
			Are you sure you want to delete this user?
		{/if}
	</h3>

	<div class="flex items-center justify-center">
		<Button on:click={deleteUser} color="red" class="mr-2" disabled={userRole === 'Superadmin'}>
			Yes, I'm sure
		</Button>
		<Button color="alternative" on:click={() => (open = false)}>No, cancel</Button>
	</div>
</Modal>

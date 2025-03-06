<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { Modal, Label, Input, Button, Select } from 'flowbite-svelte';
	import { onMount } from 'svelte';

	export let open = false;

	const dispatch = createEventDispatcher();

	let email = '';
	let groupId = '';
	let groups: any[] = [];
	let loading = false;
	let error = '';

	onMount(async () => {
		await fetchGroups();
	});

	async function fetchGroups() {
		try {
			const response = await $pocketbase.collection('groups').getFullList();
			groups = response;
			console.log('Fetched groups:', groups);
		} catch (err) {
			console.error('Error fetching groups:', err);
			error = 'Failed to fetch groups';
		}
	}

	async function handleSubmit() {
		if (!email) {
			error = 'Email is required';
			return;
		}

		if (!groupId) {
			error = 'Group is required';
			return;
		}

		loading = true;
		error = '';

		const requestData = {
			email,
			group: groupId
		};

		console.log('Sending invitation request:', requestData);

		try {
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/invitations/invite`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
						Authorization: `Bearer ${$pocketbase.authStore.token}`,
					},
					body: JSON.stringify(requestData),
				});

			if (!response.ok) {
				const errorData = await response.json();
				console.error('Server error:', errorData);
				throw new Error(errorData.message || 'Failed to send invitation');
			}

			const data = await response.json();
			console.log('Invitation response:', data);
			
			dispatch('invite', { token: data.token });
			email = '';
			groupId = '';
			open = false;
		} catch (err) {
			console.error('Error sending invitation:', err);
			error = typeof err === 'string' ? err : 'Failed to send invitation';
		} finally {
			loading = false;
		}
	}

	function handleClose() {
		email = '';
		groupId = '';
		error = '';
		open = false;
	}

	$: console.log('Current groupId:', groupId);
</script>

<Modal bind:open={open} title="Invite User" on:close={handleClose}>
	<form on:submit|preventDefault={handleSubmit} class="space-y-6">
		{#if error}
			<div class="text-red-500 text-sm">{error}</div>
		{/if}

		<Label class="space-y-2">
			<span>Email</span>
			<Input
				type="email"
				bind:value={email}
				placeholder="Enter email address"
				required
			/>
		</Label>

		<Label class="space-y-2">
			<span>Group</span>
			<Select bind:value={groupId} required>
				<option value="">Select a group</option>
				{#each groups as group}
					<option value={group.id}>{group.name}</option>
				{/each}
			</Select>
		</Label>

		<div class="flex justify-end gap-2">
			<Button color="gray" on:click={handleClose}>Cancel</Button>
			<Button type="submit" disabled={loading}>
				{loading ? 'Sending...' : 'Send Invitation'}
			</Button>
		</div>
	</form>
</Modal> 
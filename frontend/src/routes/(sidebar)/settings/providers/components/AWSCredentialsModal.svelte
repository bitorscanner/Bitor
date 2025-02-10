<script lang="ts">
	import { Button, Input, Label, Modal, Alert } from 'flowbite-svelte';
	import { pocketbase } from '$lib/stores/pocketbase';
	import type { Provider } from '../types';

	export let provider: Provider;
	export let open = false;
	export let onClose: () => void;
	export let onSave: () => void;

	let accessKeyId = '';
	let secretAccessKey = '';
	let isLoading = false;
	let isValidating = false;
	let errorMessage = '';

	async function loadCredentials() {
		try {
			const accessKeyRecord = await $pocketbase.collection('api_keys').getFirstListItem(
				`provider = "${provider.id}" && key_type = "access_key"`
			);
			const secretKeyRecord = await $pocketbase.collection('api_keys').getFirstListItem(
				`provider = "${provider.id}" && key_type = "secret_key"`
			);

			if (accessKeyRecord) {
				accessKeyId = accessKeyRecord.key;
			}
			if (secretKeyRecord) {
				secretAccessKey = secretKeyRecord.key;
			}
		} catch (error) {
			console.error('Error loading credentials:', error);
		}
	}

	async function validateCredentials() {
		isValidating = true;
		errorMessage = '';
		try {
			// Create temporary records for validation
			const tempAccessKey = await $pocketbase.collection('api_keys').create({
				provider: provider.id,
				key_type: 'access_key',
				key: accessKeyId,
				is_temporary: true
			});

			const tempSecretKey = await $pocketbase.collection('api_keys').create({
				provider: provider.id,
				key_type: 'secret_key',
				key: secretAccessKey,
				is_temporary: true
			});

			try {
				// Validate credentials
				const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
				const response = await fetch(`${baseUrl}/api/aws/validate?provider=${provider.id}`);
				const data = await response.json();
				
				if (!response.ok) {
					throw new Error(data.message || 'Invalid credentials');
				}

				// If validation successful, update the temporary records to be permanent
				await $pocketbase.collection('api_keys').update(tempAccessKey.id, { is_temporary: false });
				await $pocketbase.collection('api_keys').update(tempSecretKey.id, { is_temporary: false });

				// Delete any old keys
				try {
					const oldAccessKey = await $pocketbase.collection('api_keys').getFirstListItem(
						`provider = "${provider.id}" && key_type = "access_key" && id != "${tempAccessKey.id}"`
					);
					await $pocketbase.collection('api_keys').delete(oldAccessKey.id);
				} catch {}

				try {
					const oldSecretKey = await $pocketbase.collection('api_keys').getFirstListItem(
						`provider = "${provider.id}" && key_type = "secret_key" && id != "${tempSecretKey.id}"`
					);
					await $pocketbase.collection('api_keys').delete(oldSecretKey.id);
				} catch {}

				onSave();
				onClose();
			} catch (error: any) {
				// If validation fails, delete the temporary records
				await $pocketbase.collection('api_keys').delete(tempAccessKey.id);
				await $pocketbase.collection('api_keys').delete(tempSecretKey.id);
				throw error;
			}
		} catch (error: any) {
			console.error('Error validating credentials:', error);
			errorMessage = error.message || 'Failed to validate credentials. Please check your access key and secret key.';
		} finally {
			isValidating = false;
		}
	}

	function handleClose() {
		errorMessage = '';
		onClose();
	}

	$: if (open) {
		loadCredentials();
		errorMessage = '';
	}
</script>

<Modal bind:open={open} size="md" autoclose={false} class="w-full">
	<div class="flex justify-between items-center mb-6">
		<h3 class="text-xl font-medium text-gray-900 dark:text-white">AWS Credentials</h3>
		<Button color="alternative" pill={true} size="xs" on:click={handleClose}>✕</Button>
	</div>

	<form class="space-y-6" on:submit|preventDefault={validateCredentials}>
		{#if errorMessage}
			<Alert color="red" class="mb-4">
				<span class="font-medium">Error:</span> {errorMessage}
			</Alert>
		{/if}

		<div>
			<Label for="access-key-id" class="mb-2">Access Key ID</Label>
			<Input
				id="access-key-id"
				type="text"
				placeholder="Enter your AWS Access Key ID"
				required
				bind:value={accessKeyId}
			/>
		</div>

		<div>
			<Label for="secret-access-key" class="mb-2">Secret Access Key</Label>
			<Input
				id="secret-access-key"
				type="password"
				placeholder="Enter your AWS Secret Access Key"
				required
				bind:value={secretAccessKey}
			/>
		</div>

		<div class="flex justify-end space-x-2">
			<Button color="alternative" on:click={handleClose}>Cancel</Button>
			<Button
				type="submit"
				disabled={!accessKeyId || !secretAccessKey || isValidating}
			>
				{#if isValidating}
					Validating...
				{:else}
					Save & Validate
				{/if}
			</Button>
		</div>
	</form>
</Modal> 
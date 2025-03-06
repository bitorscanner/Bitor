<script lang="ts">
	import { onMount } from 'svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { Button, Label, Select, Toast, Spinner, Badge, Input } from 'flowbite-svelte';
	import { CheckCircleSolid, ExclamationCircleSolid } from 'flowbite-svelte-icons';
	import type { Provider, AWSSettings } from '../types';
	import AWSCredentialsModal from './AWSCredentialsModal.svelte';
	import { createEventDispatcher } from 'svelte';
	import { addToast } from '$lib/stores/toasts';

	export let provider: Provider;
	export let onSave: () => void;

	const dispatch = createEventDispatcher();

	let selectedRegion = '';
	let selectedVpc = '';
	let selectedSubnet = '';
	let selectedInstanceType = '';
	let newTag = '';

	// Ensure settings has tags array
	if (!provider.settings) {
		provider.settings = {} as AWSSettings;
	}
	const settings = provider.settings as AWSSettings;
	if (!Array.isArray(settings.tags)) {
		settings.tags = [];
	}

	let regions: Array<{ value: string; name: string }> = [];
	let vpcs: Array<{ value: string; name: string }> = [];
	let subnets: Array<{ value: string; name: string }> = [];
	let instanceTypes: Array<{ value: string; name: string }> = [];

	let isLoadingRegions = false;
	let isLoadingVpcs = false;
	let isLoadingSubnets = false;
	let isLoadingInstanceTypes = false;
	let showCredentialsModal = false;
	let hasValidCredentials = false;
	let isInitialLoading = false;

	onMount(async () => {
		// Load existing settings if available
		const awsSettings = provider.settings as AWSSettings;
		if (awsSettings) {
			selectedRegion = awsSettings.region || '';
			selectedVpc = awsSettings.vpc || '';
			selectedSubnet = awsSettings.subnet || '';
			selectedInstanceType = awsSettings.instance_type || '';
		}

		// Check if we have credentials and load resources
		try {
			isInitialLoading = true;
			const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
			const response = await fetch(`${baseUrl}/api/aws/validate?provider=${provider.id}`);
			if (response.ok) {
				hasValidCredentials = true;
				await loadRegions();
				
				// If we have a region selected, load the VPCs and instance types
				if (selectedRegion) {
					await loadVpcs();
					await loadInstanceTypes();
					
					// If we have a VPC selected, load subnets
					if (selectedVpc) {
						await loadSubnets();
					}
				}
			}
		} catch (error) {
			console.error('Error checking credentials:', error);
			addToast({
				message: 'Failed to load AWS configuration. Please try again.',
				type: 'error'
			});
		} finally {
			isInitialLoading = false;
		}
	});

	async function loadRegions() {
		isLoadingRegions = true;
		try {
			const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
			const response = await fetch(`${baseUrl}/api/aws/regions?provider=${provider.id}`);
			const data = await response.json();
			regions = data.map((region: { id: string; name: string }) => ({
				value: region.id,
				name: region.name
			}));
		} catch (error) {
			console.error('Error loading regions:', error);
			addToast({
				message: 'Failed to load AWS regions. Please check your credentials.',
				type: 'error'
			});
		} finally {
			isLoadingRegions = false;
		}
	}

	async function loadVpcs() {
		if (!selectedRegion) return;
		isLoadingVpcs = true;
		try {
			const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
			const response = await fetch(
				`${baseUrl}/api/aws/vpcs?provider=${provider.id}&region=${selectedRegion}`
			);
			const data = await response.json();
			vpcs = data.map((vpc: { id: string; name: string }) => ({
				value: vpc.id,
				name: vpc.name || vpc.id
			}));
		} catch (error) {
			console.error('Error loading VPCs:', error);
			addToast({
				message: 'Failed to load AWS VPCs. Please check your credentials and region selection.',
				type: 'error'
			});
		} finally {
			isLoadingVpcs = false;
		}
	}

	async function loadSubnets() {
		if (!selectedRegion || !selectedVpc) return;
		isLoadingSubnets = true;
		try {
			const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
			const response = await fetch(
				`${baseUrl}/api/aws/subnets?provider=${provider.id}&region=${selectedRegion}&vpc=${selectedVpc}`
			);
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}`);
			}
			const data = await response.json();
			if (!data || !Array.isArray(data)) {
				throw new Error('Invalid response format from server');
			}
			subnets = data.map((subnet: { id: string; name: string }) => ({
				value: subnet.id,
				name: subnet.name || subnet.id
			}));
		} catch (error) {
			console.error('Error loading subnets:', error);
			subnets = [];
			addToast({
				message: 'Failed to load AWS subnets. Please check your VPC selection.',
				type: 'error'
			});
		} finally {
			isLoadingSubnets = false;
		}
	}

	async function loadInstanceTypes() {
		if (!selectedRegion) {
			console.log('Skipping instance types load - no region selected');
			return;
		}
		console.log('Loading instance types for region:', selectedRegion);
		isLoadingInstanceTypes = true;
		try {
			const baseUrl = import.meta.env.VITE_API_BASE_URL || '';
			const url = `${baseUrl}/api/aws/instance-types?provider=${provider.id}&region=${selectedRegion}`;
			console.log('Fetching instance types from:', url);
			
			const response = await fetch(url);
			console.log('Response status:', response.status);
			
			const data = await response.json();
			console.log('Raw response data:', data);
			
			if (!response.ok) {
				throw new Error(`HTTP error! status: ${response.status}, message: ${JSON.stringify(data)}`);
			}
			
			if (!data || !Array.isArray(data)) {
				console.error('Invalid data format received:', data);
				throw new Error('Invalid response format from server');
			}
			
			instanceTypes = data.map((type: { id: string; name: string; description: string }) => ({
				value: type.id,
				name: `${type.name} - ${type.description}`
			}));
			console.log('Processed instance types:', instanceTypes);
		} catch (error) {
			console.error('Error loading instance types:', error);
			console.error('Full error object:', JSON.stringify(error, null, 2));
			instanceTypes = [];
			addToast({
				message: 'Failed to load AWS instance types.',
				type: 'error'
			});
		} finally {
			isLoadingInstanceTypes = false;
		}
	}

	async function handleCredentialsSave() {
		hasValidCredentials = true;
		await loadRegions();
		addToast({
			message: 'AWS credentials saved successfully',
			type: 'success'
		});
	}

	function handleSelectChange(event: Event, key: string) {
		const select = event.target as HTMLSelectElement;
		if (select) {
			updateSettings(key, select.value);
		}
	}

	async function updateSettings(key: string, value: string | string[]) {
		const newSettings = { ...provider.settings, [key]: value };
		
		// Update local state
		if (key === 'region') selectedRegion = value as string;
		if (key === 'vpc') selectedVpc = value as string;
		if (key === 'subnet') selectedSubnet = value as string;
		if (key === 'instance_type') selectedInstanceType = value as string;
		if (key === 'tags') settings.tags = value as string[];

		// Update provider settings
		provider.settings = newSettings;
		
		// Save the settings to the database
		try {
			console.log('Saving settings to database:', newSettings);
			await $pocketbase.collection('providers').update(provider.id, {
				settings: newSettings
			});
			addToast({
				message: 'Settings saved successfully',
				type: 'success'
			});
			dispatch('update', { settings: newSettings });
		} catch (error) {
			console.error('Error saving settings:', error);
			addToast({
				message: 'Failed to save settings. Please try again.',
				type: 'error'
			});
		}
	}

	function addTag(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ',') {
			event.preventDefault();
			const tag = newTag.trim();
			if (tag && !settings.tags.includes(tag)) {
				settings.tags = [...settings.tags, tag];
				updateSettings('tags', settings.tags);
			}
			newTag = '';
		}
	}

	function removeTag(tagToRemove: string) {
		settings.tags = settings.tags.filter(tag => tag !== tagToRemove);
		updateSettings('tags', settings.tags);
	}

	$: if (selectedRegion) {
		loadVpcs();
		loadInstanceTypes();
	}

	$: if (selectedVpc) {
		loadSubnets();
	}
</script>

<div class="p-4">
	<div class="flex justify-between items-center mb-6">
		<h3 class="text-lg font-medium text-gray-900 dark:text-white">AWS Configuration</h3>
		<Button color="blue" on:click={() => (showCredentialsModal = true)}>
			{hasValidCredentials ? 'Update Credentials' : 'Add Credentials'}
		</Button>
	</div>

	{#if isInitialLoading}
		<div class="flex flex-col items-center justify-center py-8">
			<Spinner size="8" />
			<p class="mt-4 text-gray-600 dark:text-gray-400">Loading AWS Configuration...</p>
		</div>
	{:else if hasValidCredentials && regions.length > 0}
		<form class="space-y-6" on:submit|preventDefault>
			<div>
				<Label for="region" class="mb-2">Region</Label>
				<div class="flex items-center gap-2">
					<Select
						id="region"
						class="flex-1"
						items={regions}
						value={selectedRegion}
						on:change={(e) => handleSelectChange(e, 'region')}
					>
						<option value="">Select a region</option>
					</Select>
					{#if isLoadingRegions}
						<Spinner size="5" />
					{/if}
				</div>
			</div>

			{#if vpcs.length > 0}
				<div>
					<Label for="vpc" class="mb-2">VPC</Label>
					<div class="flex items-center gap-2">
						<Select
							id="vpc"
							class="flex-1"
							items={vpcs}
							value={selectedVpc}
							on:change={(e) => handleSelectChange(e, 'vpc')}
							disabled={!selectedRegion}
						>
							<option value="">Select a VPC</option>
						</Select>
						{#if isLoadingVpcs}
							<Spinner size="5" />
						{/if}
					</div>
				</div>

				{#if subnets.length > 0}
					<div>
						<Label for="subnet" class="mb-2">Subnet</Label>
						<div class="flex items-center gap-2">
							<Select
								id="subnet"
								class="flex-1"
								items={subnets}
								value={selectedSubnet}
								on:change={(e) => handleSelectChange(e, 'subnet')}
								disabled={!selectedVpc}
							>
								<option value="">Select a subnet</option>
							</Select>
							{#if isLoadingSubnets}
								<Spinner size="5" />
							{/if}
						</div>
					</div>
				{/if}
			{/if}

			<div>
				<Label for="instance-type" class="mb-2">Instance Type</Label>
				<div class="flex items-center gap-2">
					<Select
						id="instance-type"
						class="flex-1"
						items={instanceTypes}
						value={selectedInstanceType}
						on:change={(e) => handleSelectChange(e, 'instance_type')}
					>
						<option value="">Select an instance type</option>
					</Select>
					{#if isLoadingInstanceTypes}
						<Spinner size="5" />
					{/if}
				</div>
			</div>

			<div>
				<Label for="tags" class="mb-2">
					Tags <span class="text-gray-500 text-sm">(optional)</span>
				</Label>
				<div class="flex flex-wrap gap-2 mb-2">
					{#each settings.tags as tag}
						<Badge
							color="blue"
							class="flex items-center gap-1"
						>
							{tag}
							<button
								type="button"
								class="ml-1 hover:text-red-500"
								on:click={() => removeTag(tag)}
							>
								×
							</button>
						</Badge>
					{/each}
				</div>
				<Input
					id="tags"
					type="text"
					placeholder="Type a tag and press Enter (optional)"
					bind:value={newTag}
					on:keydown={addTag}
				/>
			</div>
		</form>
	{:else if !hasValidCredentials}
		<div class="text-center py-4">
			<p class="text-gray-600 dark:text-gray-400">Please add your AWS credentials to continue.</p>
		</div>
	{/if}
</div>

<AWSCredentialsModal
	bind:open={showCredentialsModal}
	{provider}
	onClose={() => (showCredentialsModal = false)}
	onSave={handleCredentialsSave}
/> 
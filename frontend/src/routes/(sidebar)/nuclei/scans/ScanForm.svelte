<script lang="ts">
	import { onMount } from 'svelte';
	import { createEventDispatcher } from 'svelte';
	import {
		Button,
		Input,
		Label,
		Select,
		Modal,
		Toggle
	} from 'flowbite-svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import type { ScanData, ScanFormData, Client } from './types';
	import TemplateSelector from './TemplateSelector.svelte';

	export let open: boolean = false; // Modal control
	export let onSave: (scanData: ScanFormData) => void; // Function to call when saving scan
	export let scan: ScanData | null = null;

	const dispatch = createEventDispatcher();

	// Stepper state
	let currentStep = 1;
	const totalSteps = 3;

	// Form fields
	let scanName = '';
	let targetId = '';
	let scanProfileId = '';
	let nucleiProfileId = ''; // Placeholder for now
	let clientId = '';
	let frequency: 'one-time' | 'scheduled' = 'one-time'; // 'one-time' or 'scheduled'
	let cronSchedule = '';
	let startImmediately = false;
	let preserveVM = false;

	// Data collections with proper types
	interface Target {
		id: string;
		name: string;
	}

	interface ScanProfile {
		id: string;
		name: string;
		nuclei_interact: string;
		vm_provider: string;
		state_bucket: string;
		scan_bucket: string;
		expand?: {
			nuclei_interact?: {
				id: string;
				name: string;
			}
			vm_provider?: {
				id: string;
				name: string;
			}
			state_bucket?: {
				id: string;
				name: string;
			}
			scan_bucket?: {
				id: string;
				name: string;
			}
		}
	}

	interface NucleiProfile {
		id: string;
		name: string;
	}

	interface Client {
		id: string;
		name: string;
	}

	let targets: Target[] = [];
	let scanProfiles: ScanProfile[] = [];
	let clients: Client[] = [];
	let nucleiProfiles: NucleiProfile[] = [];

	let selectedTemplates: string[] = [];
	let useAllTemplates: boolean = false;

	// Lifecycle methods
	onMount(async () => {
		try {
			await fetchRelations();
			if (scan) {
				assignScanData();
			}
		} catch (error) {
			console.error('Error fetching data:', error);
		}
	});

	async function fetchRelations() {
		try {
			console.log('Fetching relations...');
			const [targetsData, scanProfilesData, nucleiProfilesData, clientsData] = await Promise.all([
				$pocketbase.collection('nuclei_targets').getFullList(),
				$pocketbase.collection('scan_profiles').getFullList({
					expand: 'nuclei_interact,vm_provider,state_bucket,scan_bucket'
				}),
				$pocketbase.collection('nuclei_profiles').getFullList(),
				$pocketbase.collection('clients').getFullList()
			]);

			targets = targetsData.map(item => ({ id: item.id, name: item.name }));
			scanProfiles = scanProfilesData.map(item => ({
				id: item.id,
				name: item.name,
				nuclei_interact: item.nuclei_interact,
				vm_provider: item.vm_provider,
				state_bucket: item.state_bucket,
				scan_bucket: item.scan_bucket,
				expand: item.expand
			}));
			nucleiProfiles = nucleiProfilesData.map(item => ({
				id: item.id,
				name: item.name
			}));
			clients = clientsData.map(item => ({ id: item.id, name: item.name }));

			console.log('Fetched scan profiles with all relations:', scanProfiles);
		} catch (error) {
			console.error('Error fetching relations:', error);
			throw error;
		}
	}

	function assignScanData() {
		if (!scan) return;
		// After null check, we can safely assert scan is not null
		const s = scan!;
		
		scanName = s.name;
		targetId = s.nuclei_targets;
		scanProfileId = s.scan_profile;
		nucleiProfileId = s.nuclei_profile;
		clientId = s.client;
		frequency = s.frequency;
		cronSchedule = s.cron || '';
		startImmediately = s.startImmediately;
		preserveVM = s.preserve_vm || false;
	}

	function nextStep() {
		if (currentStep < totalSteps) {
			currentStep += 1;
		}
	}

	function previousStep() {
		if (currentStep > 1) {
			currentStep -= 1;
		}
	}

	function resetForm() {
		scanName = '';
		targetId = '';
		scanProfileId = '';
		nucleiProfileId = '';
		clientId = '';
		frequency = 'one-time';
		cronSchedule = '';
		startImmediately = false;
		preserveVM = false;
		currentStep = 1;
	}

	async function handleSave(shouldStart = false) {
		try {
			console.log('Starting handleSave with shouldStart:', shouldStart);
			// Validate required fields
			if (!scanName || !targetId || !scanProfileId || !nucleiProfileId || !clientId) {
				console.error('Missing required fields:', {
					scanName: !scanName,
					targetId: !targetId,
					scanProfileId: !scanProfileId,
					nucleiProfileId: !nucleiProfileId,
					clientId: !clientId
				});
				return;
			}
			console.log('All required fields present');

			// Get selected scan profile with all its relations
			const selectedScanProfile = scanProfiles.find(p => p.id === scanProfileId);
			if (!selectedScanProfile) {
				console.error('Selected scan profile not found');
				return;
			}
			console.log('Found selected scan profile:', selectedScanProfile);

			// Validate required relations (excluding nuclei_interact which is optional)
			if (!selectedScanProfile.vm_provider || 
				!selectedScanProfile.state_bucket || !selectedScanProfile.scan_bucket) {
				console.error('Selected scan profile missing required relations:', {
					vm_provider: !selectedScanProfile.vm_provider,
					state_bucket: !selectedScanProfile.state_bucket,
					scan_bucket: !selectedScanProfile.scan_bucket
				});
				return;
			}
			console.log('All required relations present');

			// Create scan data with all necessary relations
			const scanData: ScanFormData = {
				name: scanName,
				nuclei_targets: targetId,
				scan_profile: scanProfileId,
				nuclei_profile: nucleiProfileId,
				nuclei_interact: selectedScanProfile.nuclei_interact || '',
				vm_provider: selectedScanProfile.vm_provider,
				state_bucket: selectedScanProfile.state_bucket,
				scan_bucket: selectedScanProfile.scan_bucket,
				client: clientId,
				frequency: frequency,
				cron: frequency === 'scheduled' ? cronSchedule : null,
				startImmediately: shouldStart,
				status: 'Created',
				use_all_templates: useAllTemplates,
				selected_templates: selectedTemplates,
				preserve_vm: preserveVM
			};

			console.log('Scan data prepared:', scanData);
			console.log('onSave function present:', !!onSave);

			if (typeof onSave !== 'function') {
				console.error('onSave is not a function:', onSave);
				return;
			}

			try {
				console.log('Calling onSave with scan data...');
				await onSave(scanData);
				console.log('Save completed successfully');
				resetForm();
				open = false; // Close modal regardless of start option
				dispatch('saved');
			} catch (error) {
				console.error('Error saving scan:', error);
				throw error;
			}
		} catch (error) {
			console.error('Error in handleSave:', error);
			throw error;
		}
	}
</script>

<Modal bind:open size="lg">
	<div class="p-6 space-y-6">
		<!-- Stepper -->
		<ol class="flex items-center w-full text-sm font-medium text-center text-gray-500 dark:text-gray-400 sm:text-base">
			<!-- Step 1 -->
			<li class="flex md:w-full items-center">
				<span class="flex items-center">
					<span
						class="flex items-center justify-center w-6 h-6 mr-2 rounded-full"
						class:bg-blue-600={currentStep === 1}
						class:bg-green-600={currentStep > 1}
						class:text-white={currentStep >= 1}
						class:text-gray-500={currentStep < 1}
					>
						1
					</span>
					Basic Info
				</span>
			</li>
			<!-- Line between steps -->
			<li class="flex-auto border-t-2 transition duration-500 ease-in-out border-gray-300"></li>
			<!-- Step 2 -->
			<li class="flex md:w-full items-center">
				<span class="flex items-center">
					<span
						class="flex items-center justify-center w-6 h-6 mr-2 rounded-full"
						class:bg-blue-600={currentStep === 2}
						class:bg-green-600={currentStep > 2}
						class:bg-gray-300={currentStep < 2}
						class:text-white={currentStep >= 2}
					>
						2
					</span>
					Scan Profile
				</span>
			</li>
			<!-- Line between steps -->
			<li class="flex-auto border-t-2 transition duration-500 ease-in-out border-gray-300"></li>
			<!-- Step 3 -->
			<li class="flex md:w-full items-center">
				<span class="flex items-center">
					<span
						class="flex items-center justify-center w-6 h-6 mr-2 rounded-full"
						class:bg-blue-600={currentStep === 3}
						class:bg-gray-300={currentStep < 3}
						class:text-white={currentStep >= 3}
					>
						3
					</span>
					Client & Frequency
				</span>
			</li>
		</ol>

		<!-- Step Content -->
		{#if currentStep === 1}
			<!-- Step 1: Basic Info -->
			<div class="space-y-4 mt-6">
				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Scan Name <span class="text-red-500">*</span></span>
					<Input bind:value={scanName} placeholder="Enter Scan Name" required />
					{#if !scanName}
						<span class="text-sm text-red-500">Please enter a scan name</span>
					{/if}
				</Label>
				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Choose Target <span class="text-red-500">*</span></span>
					<Select bind:value={targetId} required class="w-full">
						<option value="">Select Target</option>
						{#each targets as target}
							<option value={target.id}>{target.name}</option>
						{/each}
					</Select>
					{#if !targetId}
						<span class="text-sm text-red-500">Please select a target</span>
					{/if}
				</Label>
				<div class="flex justify-end">
					<Button color="primary" on:click={nextStep} disabled={!scanName || !targetId}>Next</Button>
				</div>
			</div>
		{:else if currentStep === 2}
			<!-- Step 2: Scan Profile -->
			<div class="space-y-4 mt-6">
				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Scan Profile <span class="text-red-500">*</span></span>
					<Select bind:value={scanProfileId} required class="w-full">
						<option value="">Select Scan Profile</option>
						{#each scanProfiles as profile}
							<option value={profile.id}>{profile.name}</option>
						{/each}
					</Select>
					{#if !scanProfileId}
						<span class="text-sm text-red-500">Please select a scan profile</span>
					{/if}
				</Label>

				<!-- Template Selection -->
				<div class="mt-4">
					<Label class="block mb-2">
						<span class="text-gray-700 dark:text-gray-400">Templates <span class="text-red-500">*</span></span>
					</Label>
					<TemplateSelector
						bind:selectedTemplates
						bind:useAllTemplates
						on:change={({ detail }) => {
							selectedTemplates = detail.selectedTemplates;
							useAllTemplates = detail.useAllTemplates;
						}}
					/>
				</div>

				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Nuclei Profile <span class="text-red-500">*</span></span>
					<Select bind:value={nucleiProfileId} required class="w-full">
						<option value="">Select Nuclei Profile</option>
						{#each nucleiProfiles as profile}
							<option value={profile.id}>{profile.name}</option>
						{/each}
					</Select>
					{#if !nucleiProfileId}
						<span class="text-sm text-red-500">Please select a nuclei profile</span>
					{/if}
				</Label>
				<div class="flex justify-between">
					<Button color="alternative" on:click={previousStep}>Previous</Button>
					<Button 
						color="primary" 
						on:click={nextStep} 
						disabled={!scanProfileId || (!useAllTemplates && selectedTemplates.length === 0)}
					>
						Next
					</Button>
				</div>
			</div>
		{:else if currentStep === 3}
			<!-- Step 3: Client & Frequency -->
			<div class="space-y-4 mt-6">
				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Client <span class="text-red-500">*</span></span>
					<Select bind:value={clientId} required class="w-full">
						<option value="">Select Client</option>
						{#each clients as client}
							<option value={client.id}>{client.name}</option>
						{/each}
					</Select>
					{#if !clientId}
						<span class="text-sm text-red-500">Please select a client</span>
					{/if}
				</Label>
				<Label class="block">
					<span class="text-gray-700 dark:text-gray-400">Frequency</span>
					<Select bind:value={frequency} required class="w-full">
						<option value="one-time">One-Time</option>
						<option value="scheduled">Scheduled</option>
					</Select>
				</Label>
				{#if frequency === 'scheduled'}
					<Label class="block">
						<span class="text-gray-700 dark:text-gray-400">Cron Schedule</span>
						<Input bind:value={cronSchedule} placeholder="e.g., 0 0 * * 0" required />
						<p class="text-sm text-gray-500">Specify a cron expression for scheduling the scan.</p>
					</Label>
				{/if}
				
				<!-- Preserve VM Option -->
				<div class="flex items-center space-x-3 p-3 border border-yellow-200 bg-yellow-50 dark:bg-yellow-900/20 dark:border-yellow-800 rounded-lg">
					<Toggle bind:checked={preserveVM} color="yellow" />
					<div>
						<div class="text-sm font-medium text-gray-900 dark:text-white">
							Preserve VM for Testing
						</div>
						<div class="text-xs text-gray-500 dark:text-gray-400">
							Keep the VM running after scan completion for investigation. 
							<span class="font-semibold text-yellow-600 dark:text-yellow-400">
								Remember to manually destroy it when done!
							</span>
						</div>
					</div>
				</div>
				<div class="flex justify-between mt-6">
					<Button color="alternative" on:click={previousStep}>Previous</Button>
					<div class="space-x-2">
						{#if frequency === 'one-time'}
							<Button color="alternative" on:click={() => handleSave(false)}>Create Scan</Button>
							<Button color="primary" on:click={() => handleSave(true)}>Create & Start Scan</Button>
						{:else}
							<Button color="primary" on:click={() => handleSave(false)}>Create Scheduled Scan</Button>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>
</Modal>

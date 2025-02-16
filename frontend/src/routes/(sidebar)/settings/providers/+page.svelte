<script lang="ts">
	import { onMount } from 'svelte';
	import { pocketbase } from '$lib/stores/pocketbase';
	import { page } from '$app/stores';
	import { 
		Card, 
		Table, 
		TableBody, 
		TableBodyCell, 
		TableBodyRow, 
		TableHead, 
		TableHeadCell, 
		Toggle, 
		Button, 
		Dropdown, 
		DropdownItem, 
		Select, 
		Label, 
		Input, 
		Badge,
		Button as BadgeButton,
		MultiSelect,
		Modal,
		Toast,
		Alert,
		Accordion,
		AccordionItem
	} from 'flowbite-svelte';
	import { 
		ArrowDownOutline,
		ArrowUpOutline,
		CheckCircleSolid,
		ExclamationCircleSolid,
		PenSolid,
		TrashBinSolid,
		CloudArrowUpSolid
	} from 'flowbite-svelte-icons';
	import {
		SiGmail,
		SiSlack,
		SiDiscord,
		SiJira,
		SiAmazon,
		SiDigitalocean,
		SiAmazons3,
		SiTelegram
	} from '@icons-pack/svelte-simple-icons';
	import type { ComponentType, SvelteComponent } from 'svelte';
	import type { RecordModel } from 'pocketbase';
	import type { Provider, ProviderType, ApiKey, JiraSettings } from './types';
	import UseMultiSelect from './UseMultiSelect.svelte';
	import AWSProvider from './components/AWSProvider.svelte';
	import DigitalOceanProvider from './components/DigitalOceanProvider.svelte';
	import S3Provider from './components/S3Provider.svelte';
	import NotificationProvider from './components/NotificationProvider.svelte';
	import JiraProvider from './components/JiraProvider.svelte';

	interface ProviderApiKeys {
		[key: string]: ApiKey[];
	}

	let providers: Provider[] = [];
	let sortField: keyof Provider = 'name';
	let sortDirection: 'asc' | 'desc' = 'asc';
	let expandedProvider: string | null = null;
	let showApiKeyModal = false;
	let showSmtpModal = false;
	let showDeleteModal = false;
	let showEditModal = false;
	let selectedProvider: Provider | null = null;
	let providerApiKeys: ProviderApiKeys = {};
	let dropdownTriggers: Record<string, HTMLElement> = {};
	let loading = false;
	let error = '';
	let showSuccessToast = false;
	let successMessage = '';
	let showS3Modal = false;
	let apiKeys: ApiKey[] = [];
	let modalTimeout: NodeJS.Timeout;
	let success = '';
	let editingName: string | null = null;

	function createProvider(type: ProviderType): Provider {
		const uses = ['email', 'slack', 'teams', 'discord', 'telegram', 'jira'].includes(type) 
			? ['notification'] 
			: [];
			
		return {
			provider_type: type,
			name: `New ${type} Provider`,
			enabled: true,
			uses,
			settings: {}
		} as Provider;
	}

	async function loadProviders() {
		try {
			const result = await $pocketbase.collection('providers').getFullList({
				sort: '-created'
			});
			console.log('Raw providers from PocketBase:', result);
			providers = result.map(record => {
				console.log('Processing record:', record);
				let defaultUses: string[] = [];
				if (['email', 'slack', 'teams', 'discord', 'telegram', 'jira'].includes(record.provider_type)) {
					defaultUses = ['notification'];
				} else if (record.provider_type === 'digitalocean') {
					defaultUses = ['compute'];
				} else if (record.provider_type === 's3') {
					defaultUses = ['terraform_storage', 'scan_storage'];
				} else if (record.provider_type === 'aws') {
					defaultUses = ['compute'];
				}

				// Initialize default settings based on provider type if not set
				let settings = record.settings || {};
				if (Object.keys(settings).length === 0) {
					if (record.provider_type === 'digitalocean') {
						settings = {
							region: '',
							do_project: '',
							size: '',
							tags: []
						};
					} else if (record.provider_type === 'aws') {
						settings = {
							region: '',
							account_id: ''
						};
					} else if (record.provider_type === 's3') {
						settings = {
							endpoint: '',
							bucket: '',
							region: '',
							use_path_style: false,
							statefile_path: '/statefile',
							scans_path: '/scans'
						};
					} else if (record.provider_type === 'email') {
						settings = {
							smtp_host: '',
							smtp_port: 587,
							from_address: '',
							encryption: 'tls'
						};
					} else if (['slack', 'teams', 'discord'].includes(record.provider_type)) {
						settings = {
							webhook_url: ''
						};
					} else if (record.provider_type === 'telegram') {
						settings = {
							bot_token: '',
							chat_id: ''
						};
					} else if (record.provider_type === 'jira') {
						settings = {
							jira_url: '',
							project_key: '',
							issue_type: 'Task',
							client_mappings: []
						} as JiraSettings;
					}
				}
					
				return {
					id: record.id,
					name: record.name,
					provider_type: record.provider_type,
					enabled: record.enabled,
					uses: record.use || defaultUses,
					settings: settings,
					created: record.created,
					updated: record.updated
				};
			}) as Provider[];
			console.log('Processed providers:', providers);
			error = '';
		} catch (e: any) {
			console.error('Error loading providers:', e);
			error = e.message || 'Failed to load providers';
			providers = [];
		}
	}

	async function deleteProvider() {
		if (!selectedProvider?.id) {
			error = 'No provider selected for deletion';
			return;
		}
		
		try {
			const id = selectedProvider.id;
			loading = true;
			error = '';
			success = '';

			console.log('Starting provider deletion process for:', id);

			// First, get and delete all API keys for this provider
			const apiKeys = await $pocketbase.collection('api_keys').getList(1, 100, {
				filter: `provider = "${id}"`
			});

			if (apiKeys.items.length > 0) {
				console.log(`Deleting ${apiKeys.items.length} associated API keys`);
				for (const apiKey of apiKeys.items) {
					await $pocketbase.collection('api_keys').delete(apiKey.id);
				}
				console.log('Successfully deleted all associated API keys');
			}

			// Now that API keys are deleted, delete the provider
			console.log('Deleting provider');
			await $pocketbase.collection('providers').delete(id);
			console.log('Provider deleted successfully');

			// Update UI
			providers = providers.filter(p => p.id !== id);
			showDeleteModal = false;
			selectedProvider = null;
			success = 'Provider and associated API keys deleted successfully';
			error = '';
		} catch (e: any) {
			console.error('Error in deletion process:', e);
			error = e.message || 'Failed to delete provider';
			success = '';
		} finally {
			loading = false;
		}
	}

	function handleProviderSave(provider: Provider) {
		const index = providers.findIndex(p => p.id === provider.id);
		if (index !== -1) {
			providers[index] = provider;
			providers = [...providers];
		}
		success = 'Provider settings saved successfully';
		error = '';
	}

	async function handleToggleChange(provider: Provider) {
		try {
			await $pocketbase.collection('providers').update(provider.id, {
				enabled: provider.enabled
			});
			success = `Provider ${provider.enabled ? 'enabled' : 'disabled'} successfully`;
			error = '';
		} catch (e: any) {
			console.error('Error updating provider:', e);
			error = e.message || 'Failed to update provider';
			success = '';
			// Revert the toggle if the update failed
			provider.enabled = !provider.enabled;
		}
	}

	async function addProvider(type: ProviderType) {
		try {
			// Set default uses based on provider type
			let uses: string[] = [];
			if (['email', 'slack', 'teams', 'discord', 'telegram', 'jira'].includes(type)) {
				uses = ['notification'];
			} else if (type === 'digitalocean') {
				uses = ['compute'];
			} else if (type === 's3') {
				uses = ['terraform_storage', 'scan_storage'];
			} else if (type === 'aws') {
				uses = ['compute'];
			}

			// Initialize default settings based on provider type
			let settings = {};
			if (type === 'digitalocean') {
				settings = {
					region: '',
					do_project: '',
					size: '',
					tags: []
				};
			} else if (type === 'aws') {
				settings = {
					region: '',
					account_id: ''
				};
			} else if (type === 's3') {
				settings = {
					endpoint: '',
					bucket: '',
					region: '',
					use_path_style: false,
					statefile_path: '/statefile',
					scans_path: '/scans'
				};
			} else if (type === 'email') {
				settings = {
					smtp_host: '',
					smtp_port: 587,
					from_address: '',
					encryption: 'tls'
				};
			} else if (['slack', 'teams', 'discord'].includes(type)) {
				settings = {
					webhook_url: ''
				};
			} else if (type === 'telegram') {
				settings = {
					bot_token: '',
					chat_id: ''
				};
			} else if (type === 'jira') {
				settings = {
					jira_url: '',
					project_key: '',
					issue_type: 'Task',
					client_mappings: []
				} as JiraSettings;
			}

			const newProvider = {
				provider_type: type,
				name: `New ${type} Provider`,
				enabled: true,
				use: uses,
				settings: settings
			};
			const result = await $pocketbase.collection('providers').create(newProvider);
			const createdProvider: Provider = {
				id: result.id,
				name: result.name || `New ${type} Provider`,
				provider_type: result.provider_type,
				enabled: result.enabled,
				uses: result.use || uses,
				settings: result.settings || settings,
				created: result.created,
				updated: result.updated
			};
			providers = [createdProvider, ...providers];
			editingName = result.id;
			success = 'Provider added successfully';
			error = '';
		} catch (e: any) {
			console.error('Error adding provider:', e);
			error = e.message || 'Failed to add provider';
			success = '';
		}
	}

	async function updateProviderName(provider: Provider, newName: string) {
		try {
			await $pocketbase.collection('providers').update(provider.id, {
				name: newName
			});
			provider.name = newName;
			providers = [...providers];
			editingName = null;
			success = 'Provider name updated successfully';
			error = '';
		} catch (e: any) {
			console.error('Error updating provider name:', e);
			error = e.message || 'Failed to update provider name';
			success = '';
		}
	}

	function getInputValue(providerId: string): string {
		const input = document.getElementById(`name-${providerId}`) as HTMLInputElement;
		return input?.value || '';
	}

	function toggleExpand(providerId: string) {
		expandedProvider = expandedProvider === providerId ? null : providerId;
	}

	async function handleUseChange(provider: Provider, event: { detail: string[] }) {
		if (!provider.id) return;
		
		try {
			const uses = event.detail;
			console.log('Updating provider uses:', { providerId: provider.id, uses });
			await $pocketbase.collection('providers').update(provider.id, {
				use: uses
			});
			provider.uses = uses;
			providers = [...providers];
			console.log('Updated provider:', provider);
			success = 'Provider uses updated successfully';
			error = '';
		} catch (e: any) {
			console.error('Error updating provider uses:', e);
			error = e.message || 'Failed to update provider uses';
			success = '';
		}
	}

	onMount(() => {
		loadProviders();
	});
</script>

<div class="container mx-auto px-4 py-8">
	<div class="flex justify-between items-center mb-6">
		<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Providers</h1>
		<div class="relative">
			<Button>Add Provider</Button>
			<Dropdown>
				<DropdownItem on:click={() => addProvider('aws')}>AWS</DropdownItem>
				<DropdownItem on:click={() => addProvider('digitalocean')}>DigitalOcean</DropdownItem>
				<DropdownItem on:click={() => addProvider('s3')}>S3</DropdownItem>
				<DropdownItem on:click={() => addProvider('email')}>Email</DropdownItem>
				<DropdownItem on:click={() => addProvider('slack')}>Slack</DropdownItem>
				<DropdownItem on:click={() => addProvider('discord')}>Discord</DropdownItem>
				<DropdownItem on:click={() => addProvider('telegram')}>Telegram</DropdownItem>
				<DropdownItem on:click={() => addProvider('jira')}>Jira</DropdownItem>
			</Dropdown>
		</div>
	</div>

	{#if error}
		<Alert color="red" class="mb-4">
			<span class="font-medium">Error!</span> {error}
		</Alert>
	{/if}

	{#if success}
		<Alert color="green" class="mb-4">
			<span class="font-medium">Success!</span> {success}
		</Alert>
	{/if}

	<Table hoverable={true}>
		<TableHead>
			<TableHeadCell>Name</TableHeadCell>
			<TableHeadCell>Type</TableHeadCell>
			<TableHeadCell>Uses</TableHeadCell>
			<TableHeadCell>Status</TableHeadCell>
			<TableHeadCell>Actions</TableHeadCell>
		</TableHead>
		<TableBody>
			{#each providers as provider}
				<TableBodyRow>
					<TableBodyCell>
						<div 
							class="cursor-pointer hover:text-blue-600 w-full"
							on:click={(e) => {
								e.stopPropagation();
								if (!editingName) {
									editingName = provider.id;
									setTimeout(() => {
										const input = document.getElementById(`name-${provider.id}`);
										if (input instanceof HTMLInputElement) {
											input.focus();
											input.select();
										}
									}, 0);
								}
							}}
						>
							{#if editingName === provider.id}
								<form 
									class="flex w-full"
									on:submit|preventDefault={() => {
										const input = document.getElementById(`name-${provider.id}`);
										if (input instanceof HTMLInputElement) {
											updateProviderName(provider, input.value);
										}
									}}
								>
									<Input
										id="name-{provider.id}"
										type="text"
										value={provider.name}
										class="w-full"
										on:blur={(e) => {
											if (e.currentTarget instanceof HTMLInputElement) {
												updateProviderName(provider, e.currentTarget.value);
											}
										}}
										on:keydown={(e) => {
											if (e.key === 'Escape') {
												editingName = null;
											}
										}}
									/>
								</form>
							{:else}
								<span class="w-full block">{provider.name}</span>
							{/if}
						</div>
					</TableBodyCell>
					<TableBodyCell>
						<div class="flex items-center space-x-2">
							{#if provider.provider_type === 'aws'}
								<SiAmazon />
							{:else if provider.provider_type === 'digitalocean'}
								<SiDigitalocean />
							{:else if provider.provider_type === 's3'}
								<SiAmazons3 />
							{:else if provider.provider_type === 'email'}
								<SiGmail />
							{:else if provider.provider_type === 'slack'}
								<SiSlack />
							{:else if provider.provider_type === 'discord'}
								<SiDiscord />
							{:else if provider.provider_type === 'jira'}
								<SiJira />
							{:else if provider.provider_type === 'telegram'}
								<SiTelegram />
							{/if}
							<span>{provider.provider_type}</span>
						</div>
					</TableBodyCell>
					<TableBodyCell>
						{#if ['email', 'slack', 'teams', 'discord', 'telegram', 'jira'].includes(provider.provider_type)}
							<div class="text-gray-600">Notification</div>
						{:else if provider.provider_type === 'digitalocean'}
							<UseMultiSelect
								value={provider.uses || []}
								useDescriptions={{
									compute: 'Compute resources (VMs, etc)',
									dns: 'DNS Management'
								}}
								onChange={(uses) => handleUseChange(provider, { detail: uses })}
							/>
						{:else if provider.provider_type === 's3'}
							<UseMultiSelect
								value={provider.uses || []}
								useDescriptions={{
									terraform_storage: 'Terraform State Storage',
									scan_storage: 'Scan Results Storage'
								}}
								onChange={(uses) => handleUseChange(provider, { detail: uses })}
							/>
						{:else}
							<UseMultiSelect
								value={provider.uses || []}
								useDescriptions={{
									compute: 'Compute resources (VMs, etc)',
									dns: 'DNS Management',
									notification: 'Notifications'
								}}
								onChange={(uses) => handleUseChange(provider, { detail: uses })}
							/>
						{/if}
					</TableBodyCell>
					<TableBodyCell>
						<Toggle bind:checked={provider.enabled} on:change={() => handleToggleChange(provider)} />
					</TableBodyCell>
					<TableBodyCell>
						<div class="flex items-center space-x-2">
							<Button size="xs" on:click={() => toggleExpand(provider.id)}>Configure</Button>
							<button
								class="text-red-500 hover:text-red-700"
								on:click={() => {
									selectedProvider = provider;
									showDeleteModal = true;
								}}
							>
								<TrashBinSolid size="sm" class="w-5 h-5" />
							</button>
						</div>
					</TableBodyCell>
				</TableBodyRow>
				{#if expandedProvider === provider.id}
					<TableBodyRow>
						<TableBodyCell colspan={5} class="p-4 bg-gray-50 dark:bg-gray-800">
							{#if provider.provider_type === 'aws'}
								<AWSProvider {provider} onSave={handleProviderSave} />
							{:else if provider.provider_type === 'digitalocean'}
								<DigitalOceanProvider {provider} onSave={handleProviderSave} />
							{:else if provider.provider_type === 's3'}
								<S3Provider {provider} onSave={handleProviderSave} />
							{:else if provider.provider_type === 'jira'}
								<JiraProvider {provider} onSave={handleProviderSave} />
							{:else if ['email', 'slack', 'teams', 'discord', 'telegram'].includes(provider.provider_type)}
								<NotificationProvider {provider} onSave={handleProviderSave} />
							{/if}
						</TableBodyCell>
					</TableBodyRow>
				{/if}
			{/each}
		</TableBody>
	</Table>
</div>

<Modal bind:open={showDeleteModal} size="xs" autoclose={false}>
	<div class="text-center">
		<CloudArrowUpSolid size="xl" class="mx-auto mb-4 text-gray-400" />
		<h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
			Are you sure you want to delete this provider?
		</h3>
		<div class="flex justify-center gap-4">
			<Button color="red" disabled={loading} on:click={async () => {
				await deleteProvider();
			}}>
				{loading ? 'Deleting...' : 'Yes, I\'m sure'}
			</Button>
			<Button color="alternative" disabled={loading} on:click={() => {
				if (!loading) {
					showDeleteModal = false;
					selectedProvider = null;
				}
			}}>
				No, cancel
			</Button>
		</div>
	</div>
</Modal>

<script>
	import { onMount } from 'svelte';
	import { pocketbase } from '$lib/stores/pocketbase';
	import { currentUser } from '$lib/stores/auth';
	import { Button, Input, Card, Badge, Spinner, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Modal, Label, Alert } from 'flowbite-svelte';
	import { PlusOutline, SearchOutline, DownloadOutline, ExclamationCircleOutline, ChevronUpOutline, ChevronDownOutline } from 'flowbite-svelte-icons';

	let loading = false;
	let scanLoading = false;
	let tldResults = [];
	let stats = {};
	let newTLDModal = false;
	let newDomain = '';
	let tldOptions = {
		save_results: true
	};

	let clients = [];
	let selectedClient = '';
	let clientId = '';

	// Table filtering and search
	let searchTerm = '';
	let sortBy = 'domain'; // domain, tenant_name, discovered_at
	let sortOrder = 'asc'; // asc, desc
	let currentPage = 1;
	let itemsPerPage = 50;

	let errorMessage = '';
	let successMessage = '';

	// Get current client ID
	$: if ($currentUser) {
		if ($pocketbase.authStore.isAdmin) {
			clientId = '';
		} else {
			clientId = $currentUser.client || $currentUser.id || '';
		}
	}

	// Reload data when selectedClient changes
	$: if (selectedClient && typeof window !== 'undefined') {
		loadTLDResults();
		loadStats();
	}

	onMount(async () => {
		await loadTLDResults();
		await loadStats();
		await loadClients();
	});

	async function loadTLDResults() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		loading = true;
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/tld?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.success) {
				tldResults = data.tlds || [];
			}
		} catch (error) {
			console.error('Failed to load TLD results:', error);
		} finally {
			loading = false;
		}
	}

	async function loadStats() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/subdomains/stats?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.success) {
				stats = data.stats || {};
			}
		} catch (error) {
			console.error('Failed to load stats:', error);
		}
	}

	async function loadClients() {
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/collections/clients/records`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.items) {
				clients = data.items;
				if (clientId && clients.find(c => c.id === clientId)) {
					selectedClient = clientId;
				} else if (clients.length > 0) {
					selectedClient = clients[0].id;
				}
			}
		} catch (error) {
			console.error('Failed to load clients:', error);
		}
	}

	async function startTLDDiscovery() {
		if (!newDomain.trim() || !selectedClient) {
			if (!selectedClient) {
				errorMessage = 'Please select a client to scan for';
			}
			return;
		}

		scanLoading = true;
		errorMessage = '';
		successMessage = '';

		try {
			const requestBody = {
				domain: newDomain.trim(),
				client_id: selectedClient,
				save_results: tldOptions.save_results
			};

			const token = $pocketbase.authStore.token;
			
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/tld/discover`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${token}`
				},
				body: JSON.stringify(requestBody)
			});

			const data = await response.json();
			
			if (data.success && data.result) {
				const result = data.result;
				if (result.tenant_info && result.tenant_info.federation_brand_name) {
					successMessage = `TLD Discovery completed! Found ${result.total_domains} domains for Microsoft tenant: ${result.tenant_info.federation_brand_name} (${result.duration})`;
				} else {
					successMessage = `TLD Discovery completed! Found ${result.total_domains} domains in ${result.duration}`;
				}
				newDomain = '';
				newTLDModal = false;
				await loadTLDResults();
				await loadStats();
			} else {
				errorMessage = data.error || 'TLD Discovery failed';
				if (data.result?.error) {
					errorMessage += ': ' + data.result.error;
				}
			}
		} catch (error) {
			errorMessage = 'Failed to start TLD discovery: ' + error.message;
		} finally {
			scanLoading = false;
		}
	}

	// Computed filtered and sorted TLD results
	$: filteredTLDs = tldResults
		.filter(tld => {
			if (searchTerm && !tld.domain.toLowerCase().includes(searchTerm.toLowerCase()) && 
				!(tld.metadata?.tenant_name || '').toLowerCase().includes(searchTerm.toLowerCase())) {
				return false;
			}
			return true;
		})
		.sort((a, b) => {
			let aVal, bVal;
			
			switch (sortBy) {
				case 'domain':
					aVal = a.domain || '';
					bVal = b.domain || '';
					break;
				case 'tenant_name':
					aVal = a.metadata?.tenant_name || '';
					bVal = b.metadata?.tenant_name || '';
					break;
				case 'discovered_at':
					aVal = new Date(a.discovered_at || a.created);
					bVal = new Date(b.discovered_at || b.created);
					break;
				default:
					aVal = a.domain || '';
					bVal = b.domain || '';
			}
			
			if (sortOrder === 'asc') {
				return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
			} else {
				return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
			}
		});

	// Pagination
	$: totalPages = Math.ceil(filteredTLDs.length / itemsPerPage);
	$: paginatedTLDs = filteredTLDs.slice(
		(currentPage - 1) * itemsPerPage, 
		currentPage * itemsPerPage
	);

	// Reset page when filters change
	$: if (searchTerm) {
		currentPage = 1;
	}

	function handleSort(column) {
		if (sortBy === column) {
			sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
		} else {
			sortBy = column;
			sortOrder = 'asc';
		}
	}

	function exportTLDs() {
		const csvContent = [
			['Domain', 'Parent Domain', 'Tenant Name', 'Tenant ID', 'Discovered At'],
			...filteredTLDs.map(tld => [
				tld.domain,
				tld.parent_domain || '',
				tld.metadata?.tenant_name || '',
				tld.metadata?.tenant_id || '',
				new Date(tld.discovered_at || tld.created).toLocaleString()
			])
		].map(row => row.join(',')).join('\n');

		const blob = new Blob([csvContent], { type: 'text/csv' });
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `tld-discovery-${new Date().toISOString().split('T')[0]}.csv`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		window.URL.revokeObjectURL(url);
	}

	// Clear messages after 5 seconds
	$: if (successMessage || errorMessage) {
		setTimeout(() => {
			successMessage = '';
			errorMessage = '';
		}, 5000);
	}
</script>

<svelte:head>
	<title>Attack Surface - TLD Discovery</title>
</svelte:head>

<div class="p-0 max-w-none w-full min-w-0">
	<div class="px-6 py-4">
		<div class="flex justify-between items-center mb-6">
			<div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Attack Surface - TLD Discovery</h1>
				<p class="text-gray-600 dark:text-gray-400">Discover top-level domains owned by the organization using Microsoft tenant enumeration</p>
			</div>
			<div class="flex gap-2 items-center">
				<!-- Client Selector -->
				{#if clients.length > 0}
					<div class="flex items-center gap-2">
						<Label for="client-selector" class="text-sm font-medium">Client:</Label>
						<select
							id="client-selector"
							bind:value={selectedClient}
							class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
						>
							<option value="">Select a client...</option>
							{#each clients as client}
								<option value={client.id}>{client.name || client.id}</option>
							{/each}
						</select>
					</div>
				{/if}
				<Button color="alternative" href="/attack-surface/domains">
					View Subdomains
				</Button>
				<Button on:click={() => newTLDModal = true}>
					<PlusOutline class="w-5 h-5 mr-2" />
					Discover TLDs
				</Button>
			</div>
		</div>

		<!-- Success/Error Messages -->
		{#if successMessage}
			<Alert color="green" class="mb-4">
				<ExclamationCircleOutline slot="icon" class="w-4 h-4" />
				{successMessage}
			</Alert>
		{/if}

		{#if errorMessage}
			<Alert color="red" class="mb-4">
				<ExclamationCircleOutline slot="icon" class="w-4 h-4" />
				{errorMessage}
			</Alert>
		{/if}

		<!-- Statistics Cards -->
		<div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-blue-600">{filteredTLDs.length}</div>
					<div class="text-sm text-gray-600">Discovered TLDs</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-green-600">{[...new Set(filteredTLDs.map(t => t.metadata?.tenant_id).filter(Boolean))].length}</div>
					<div class="text-sm text-gray-600">Unique Tenants</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-purple-600">{stats.total_subdomains || 0}</div>
					<div class="text-sm text-gray-600">Total Subdomains</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-orange-600">{stats.resolved_count || 0}</div>
					<div class="text-sm text-gray-600">Resolved Subdomains</div>
				</div>
			</Card>
		</div>
	</div>

	<!-- TLD Results Table - FULL WIDTH -->
	<div class="w-full min-h-screen bg-white dark:bg-gray-900">
		<div class="px-6 py-8">
			<div class="flex justify-between items-center mb-6">
				<h2 class="text-2xl font-bold text-gray-900 dark:text-white">Discovered Top-Level Domains</h2>
				<div class="flex gap-3">
					<Button color="alternative" size="sm" on:click={exportTLDs} disabled={filteredTLDs.length === 0}>
						<DownloadOutline class="w-4 h-4 mr-2" />
						Export CSV
					</Button>
					<Button color="alternative" size="sm" on:click={loadTLDResults}>
						<SearchOutline class="w-4 h-4 mr-2" />
						Refresh
					</Button>
				</div>
			</div>

			<!-- Filters and Search -->
			<div class="grid grid-cols-1 xl:grid-cols-3 gap-8 mb-10 p-8 bg-gray-50 dark:bg-gray-800 rounded-xl shadow-lg">
				<!-- Search -->
				<div>
					<Label for="search" class="text-base mb-3 block font-semibold">Search Domains</Label>
					<Input 
						id="search"
						type="text" 
						placeholder="Search domains or tenant names..." 
						bind:value={searchTerm}
						class="w-full h-12 text-base"
					/>
				</div>

				<!-- Items Per Page -->
				<div>
					<Label for="items-per-page" class="text-base mb-3 block font-semibold">Items Per Page</Label>
					<select 
						id="items-per-page"
						bind:value={itemsPerPage}
						class="w-full h-12 px-4 py-3 text-base border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
					>
						<option value={25}>25</option>
						<option value={50}>50</option>
						<option value={100}>100</option>
						<option value={200}>200</option>
					</select>
				</div>

				<!-- Placeholder for future filters -->
				<div>
					<Label class="text-base mb-3 block font-semibold">Quick Actions</Label>
					<Button color="light" class="w-full h-12" on:click={() => newTLDModal = true}>
						<PlusOutline class="w-4 h-4 mr-2" />
						Discover More TLDs
					</Button>
				</div>
			</div>

			<!-- Results Summary -->
			<div class="flex justify-between items-center mb-8 px-2">
				<div class="text-lg text-gray-600 dark:text-gray-400">
					Showing {Math.min((currentPage - 1) * itemsPerPage + 1, filteredTLDs.length)} - 
					{Math.min(currentPage * itemsPerPage, filteredTLDs.length)} of {filteredTLDs.length} results
					{#if filteredTLDs.length !== tldResults.length}
						(filtered from {tldResults.length} total)
					{/if}
				</div>
				{#if filteredTLDs.length > 0}
					<div class="flex gap-8 text-lg">
						<span class="text-blue-600 font-bold">Microsoft Tenants: {[...new Set(filteredTLDs.map(t => t.metadata?.tenant_id).filter(Boolean))].length}</span>
					</div>
				{/if}
			</div>

			{#if loading}
				<div class="flex justify-center py-12">
					<Spinner size="8" />
				</div>
			{:else if filteredTLDs.length === 0}
				<div class="text-center py-12 text-gray-500">
					{#if tldResults.length === 0}
						No TLD results found. Start by discovering domains for a Microsoft tenant.
					{:else}
						No TLDs match your current filters.
					{/if}
				</div>
			{:else}
				<!-- Table -->
				<div class="w-full overflow-x-auto bg-white dark:bg-gray-800 rounded-lg shadow-sm">
					<table class="w-full text-base text-left text-gray-500 dark:text-gray-400">
						<thead class="text-sm text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400 border-b-2 border-gray-200 dark:border-gray-600">
							<tr>
								<th scope="col" class="px-8 py-6 w-2/5">
									<button 
										class="flex items-center gap-2 font-semibold hover:text-blue-600 focus:outline-none"
										on:click={() => handleSort('domain')}
									>
										Domain
										{#if sortBy === 'domain'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
								<th scope="col" class="px-8 py-6 w-1/5">Parent Domain</th>
								<th scope="col" class="px-8 py-6 w-1/5">
									<button 
										class="flex items-center gap-2 font-semibold hover:text-blue-600 focus:outline-none"
										on:click={() => handleSort('tenant_name')}
									>
										Microsoft Tenant
										{#if sortBy === 'tenant_name'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
								<th scope="col" class="px-8 py-6 w-1/6">
									<button 
										class="flex items-center gap-2 font-semibold hover:text-blue-600 focus:outline-none"
										on:click={() => handleSort('discovered_at')}
									>
										Discovered At
										{#if sortBy === 'discovered_at'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
							</tr>
						</thead>
						<tbody>
							{#each paginatedTLDs as tld}
								<tr class="bg-white border-b dark:bg-gray-800 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors">
									<td class="px-8 py-6">
										<div class="flex items-center gap-4">
											<a href="https://{tld.domain}" target="_blank" class="text-blue-600 hover:underline font-medium break-all text-base">
												{tld.domain}
											</a>
											<button 
												class="text-xs text-gray-500 hover:text-blue-600 px-3 py-1 rounded border border-gray-300 hover:border-blue-500 transition-colors flex-shrink-0"
												on:click={() => navigator.clipboard.writeText(tld.domain)}
												title="Copy to clipboard"
											>
												copy
											</button>
										</div>
									</td>
									<td class="px-8 py-6 text-gray-900 dark:text-white text-base">
										{tld.parent_domain || '-'}
									</td>
									<td class="px-8 py-6">
										{#if tld.metadata?.tenant_name}
											<div class="space-y-1">
												<div class="font-medium text-gray-900 dark:text-white">{tld.metadata.tenant_name}</div>
												{#if tld.metadata.tenant_id}
													<div class="text-xs text-gray-500 font-mono">{tld.metadata.tenant_id}</div>
												{/if}
												{#if tld.metadata.namespace_type}
													<Badge color="blue" class="text-xs">
														{tld.metadata.namespace_type}
													</Badge>
												{/if}
											</div>
										{:else}
											<span class="text-gray-400">-</span>
										{/if}
									</td>
									<td class="px-8 py-6 text-gray-900 dark:text-white text-base">
										{new Date(tld.discovered_at || tld.created).toLocaleString()}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination -->
				{#if totalPages > 1}
					<div class="flex justify-center items-center gap-2 mt-6 pb-4">
						<Button 
							size="sm" 
							color="alternative"
							disabled={currentPage === 1}
							on:click={() => currentPage = 1}
						>
							First
						</Button>
						<Button 
							size="sm" 
							color="alternative"
							disabled={currentPage === 1}
							on:click={() => currentPage -= 1}
						>
							Previous
						</Button>
						
						<div class="flex items-center gap-1">
							{#each Array.from({length: Math.min(5, totalPages)}, (_, i) => {
								const start = Math.max(1, currentPage - 2);
								const end = Math.min(totalPages, start + 4);
								return start + i <= end ? start + i : null;
							}).filter(Boolean) as page}
								<Button 
									size="sm" 
									color={currentPage === page ? 'blue' : 'alternative'}
									on:click={() => currentPage = page}
								>
									{page}
								</Button>
							{/each}
						</div>

						<Button 
							size="sm" 
							color="alternative"
							disabled={currentPage === totalPages}
							on:click={() => currentPage += 1}
						>
							Next
						</Button>
						<Button 
							size="sm" 
							color="alternative"
							disabled={currentPage === totalPages}
							on:click={() => currentPage = totalPages}
						>
							Last
						</Button>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>

<!-- New TLD Discovery Modal -->
<Modal bind:open={newTLDModal} autoclose={false}>
	<div slot="header">
		<h3 class="text-xl font-semibold">Discover Top-Level Domains</h3>
	</div>
	
	<div class="space-y-4">
		<div>
			<Label for="domain">Domain</Label>
			<Input 
				id="domain"
				bind:value={newDomain}
				placeholder="example.com"
				disabled={scanLoading}
			/>
			<p class="text-sm text-gray-600 dark:text-gray-400 mt-1">
				Enter any domain belonging to a Microsoft tenant to discover all domains owned by that organization.
			</p>
		</div>

		<!-- Client Selection -->
		<div>
			<Label for="client">Client</Label>
			<select
				id="client"
				bind:value={selectedClient}
				class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
				disabled={scanLoading}
				required
			>
				<option value="">Select a client...</option>
				{#each clients as client}
					<option value={client.id}>{client.name || client.id}</option>
				{/each}
			</select>
		</div>

		<div>
			<Label>Discovery Options</Label>
			<div class="space-y-2 mt-2">
				<label class="flex items-center">
					<input type="checkbox" bind:checked={tldOptions.save_results} class="mr-2" disabled={scanLoading} />
					Save results to database
				</label>
			</div>
		</div>

		<div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
			<p class="text-sm text-blue-800 dark:text-blue-200">
				<strong>💡 Microsoft Tenant Discovery:</strong> This feature uses Microsoft's federation endpoints 
				to discover all domains associated with a tenant. It works by:
			</p>
			<ul class="text-sm text-blue-800 dark:text-blue-200 mt-2 list-disc list-inside">
				<li>Checking if the domain is part of a Microsoft tenant</li>
				<li>Querying Microsoft's autodiscover service for tenant domains</li>
				<li>Filtering out generic Microsoft domains (*.onmicrosoft.com)</li>
				<li>Returning verified domains owned by the organization</li>
			</ul>
		</div>
	</div>

	<div slot="footer" class="flex justify-end gap-2">
		<Button color="alternative" on:click={() => newTLDModal = false} disabled={scanLoading}>
			Cancel
		</Button>
		<Button on:click={startTLDDiscovery} disabled={!newDomain.trim() || !selectedClient || scanLoading}>
			{#if scanLoading}
				<Spinner class="mr-2" size="4" />
				Discovering...
			{:else}
				<SearchOutline class="w-4 h-4 mr-2" />
				Start Discovery
			{/if}
		</Button>
	</div>
</Modal> 
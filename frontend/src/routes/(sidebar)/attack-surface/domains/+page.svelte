<script>
	import { onMount } from 'svelte';
	import { pocketbase } from '$lib/stores/pocketbase';
	import { currentUser } from '$lib/stores/auth';
	import { Button, Input, Card, Badge, Spinner, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Modal, Select, Label, Alert, Checkbox, Helper } from 'flowbite-svelte';
	import { PlusOutline, SearchOutline, DownloadOutline, ExclamationCircleOutline, ChevronUpOutline, ChevronDownOutline } from 'flowbite-svelte-icons';

	let loading = false;
	let scanLoading = false;
	let subdomains = [];
	let stats = {
		total_domains: 0,
		total_resolved: 0,
		total_unique_ips: 0,
		recent_scans: 0,
		tld_domains: 0
	};
	let newDomainModal = false;
	let newDomain = '';
	let scanOptions = {
		all_sources: false,
		timeout: 30,
		max_time: 10,
		rate_limit: 50,
		recursive: false,
		save_results: true
	};
	let availableSources = [];
	let selectedSources = [];
	let errorMessage = '';
	let successMessage = '';
	let clientId = '';
	let clients = [];
	let selectedClient = '';

	// Table filtering and search
	let searchTerm = '';
	let statusFilter = 'all'; // all, resolved, unresolved
	let sourceFilter = 'all';
	let sortBy = 'subdomain'; // subdomain, parent_domain, source, resolved
	let sortOrder = 'asc'; // asc, desc
	let currentPage = 1;
	let itemsPerPage = 50;

	// Reactive variables
	let domain = '';
	let includeTldDomains = false;
	let scanModal = false;
	let search = '';

	// Get current client ID - but we'll also let user select from available clients
	$: if ($currentUser) {
		console.log('DEBUG: currentUser object:', $currentUser);
		console.log('DEBUG: currentUser.client:', $currentUser.client);
		console.log('DEBUG: currentUser.id:', $currentUser.id);
		
		// For admin users, we need to handle differently
		if ($pocketbase.authStore.isAdmin) {
			// Admin users don't have a client - we might need to get a default client or handle this differently
			console.log('DEBUG: User is admin - no client assigned');
			clientId = ''; // Admins don't have a specific client
		} else {
			// For regular users, the client should be in the client field or we need to use the user ID
			clientId = $currentUser.client || $currentUser.id || '';
		}
		console.log('DEBUG: Set clientId to:', clientId);
	}

	// Reload data when selectedClient changes
	$: if (selectedClient && typeof window !== 'undefined') {
		loadSubdomains();
		loadStats();
	}

	onMount(async () => {
		await loadSubdomains();
		await loadStats();
		await loadAvailableSources();
		await loadClients();
	});

	async function loadSubdomains() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		loading = true;
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/subdomains?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.success) {
				subdomains = data.subdomains || [];
			}
		} catch (error) {
			console.error('Failed to load subdomains:', error);
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
			
			// Also get TLD domain count
			const tldResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/tld?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			
			if (tldResponse.ok) {
				const tldData = await tldResponse.json();
				console.log('DEBUG: TLD response:', tldData);
				stats.tld_domains = tldData.tlds ? tldData.tlds.length : 0;
			} else {
				console.log('DEBUG: TLD response failed:', tldResponse.status);
				stats.tld_domains = 0;
			}
		} catch (error) {
			console.error('Failed to load stats:', error);
		}
	}

	async function loadAvailableSources() {
		try {
			const token = $pocketbase.authStore.token;
			
			// Load available sources from subfinder
			const sourcesResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/sources`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const sourcesData = await sourcesResponse.json();
			
			// Load configured discovery providers
			const providersResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/collections/providers/records?filter=enabled=true&&use:each~'discovery'`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const providersData = await providersResponse.json();
			
			// Get provider types that have API keys configured
			const configuredProviders = new Set();
			if (providersData.items) {
				for (const provider of providersData.items) {
					// Check if this provider has API keys
					const apiKeysResponse = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/collections/api_keys/records?filter=provider='${provider.id}'&&key_type='api_key'`, {
						headers: {
							'Authorization': `Bearer ${token}`
						}
					});
					const apiKeysData = await apiKeysResponse.json();
					
					if (apiKeysData.items && apiKeysData.items.length > 0) {
						configuredProviders.add(provider.provider_type);
					}
				}
			}
			
			// Filter sources to only show free ones or those with configured API keys
			if (sourcesData.success && sourcesData.sources) {
				availableSources = sourcesData.sources.filter(source => {
					// Always show free sources
					if (!source.requires_key) {
						return true;
					}
					
					// For sources that require keys, only show if we have them configured
					return configuredProviders.has(source.name);
				});
			}
		} catch (error) {
			console.error('Failed to load sources:', error);
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
				// Set default selected client if user has one
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

	async function startScan() {
		console.log('DEBUG: startScan called');
		console.log('DEBUG: newDomain:', newDomain);
		console.log('DEBUG: selectedClient:', selectedClient);
		
		// Validate inputs - domain is only required if not including TLD domains
		if ((!newDomain.trim() && !includeTldDomains) || !selectedClient) {
			console.log('DEBUG: Early return - missing domain or client selection');
			if (!selectedClient) {
				errorMessage = 'Please select a client to scan for';
			}
			if (!newDomain.trim() && !includeTldDomains) {
				errorMessage = 'Please enter a domain to scan or enable TLD domain inclusion';
			}
			if (includeTldDomains && stats.tld_domains === 0) {
				errorMessage = 'No TLD domains found. Please run TLD discovery first or enter a domain to scan.';
			}
			return;
		}

		console.log('DEBUG: Starting scan...');
		scanLoading = true;
		errorMessage = '';
		successMessage = '';

		try {
			// Ensure selectedSources is an array
			const sourcesArray = Array.isArray(selectedSources) ? selectedSources : [];
			console.log('DEBUG: selectedSources before request:', selectedSources);
			console.log('DEBUG: sourcesArray:', sourcesArray);
			
			const requestBody = {
				domain: newDomain.trim() || (includeTldDomains ? "tld-only-scan" : ""),
				client_id: selectedClient,
				all_sources: scanOptions.all_sources,
				timeout: scanOptions.timeout,
				max_time: scanOptions.max_time,
				rate_limit: scanOptions.rate_limit,
				recursive: scanOptions.recursive,
				save_results: scanOptions.save_results,
				include_tlds: includeTldDomains,
				sources: sourcesArray.length > 0 ? sourcesArray : undefined
			};

			console.log('DEBUG: Request body:', requestBody);

			const token = $pocketbase.authStore.token;
			console.log('DEBUG: Making request to backend...');
			
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/subdomains/scan`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${token}`
				},
				body: JSON.stringify(requestBody)
			});

			console.log('DEBUG: Response status:', response.status);
			const data = await response.json();
			console.log('DEBUG: Response data:', data);
			
			if (data.success && data.result) {
				const subdomainCount = data.result.subdomains?.length || 0;
				const domainCount = data.result.stats?.domains_scanned || 1;
				if (includeTldDomains && !newDomain.trim()) {
					successMessage = `TLD scan completed! Found ${subdomainCount} subdomains across ${domainCount} TLD domains in ${data.result.duration}`;
				} else {
					successMessage = `Scan completed! Found ${subdomainCount} subdomains in ${data.result.duration}`;
				}
				newDomain = '';
				newDomainModal = false;
				await loadSubdomains();
				await loadStats();
			} else {
				errorMessage = data.error || 'Scan failed';
				if (data.result?.error) {
					errorMessage += ': ' + data.result.error;
				}
				console.log('DEBUG: Scan failed:', errorMessage);
			}
		} catch (error) {
			console.log('DEBUG: Exception caught:', error);
			errorMessage = 'Failed to start scan: ' + error.message;
		} finally {
			console.log('DEBUG: Scan loading set to false');
			scanLoading = false;
		}
	}

	function exportSubdomains() {
		const csvContent = [
			['Subdomain', 'Parent Domain', 'Source', 'Resolved', 'IP Address'],
			...subdomains.map(sub => [
				sub.subdomain,
				sub.parent_domain || '',
				sub.source || '',
				sub.resolved ? 'Yes' : 'No',
				sub.ip || ''
			])
		].map(row => row.join(',')).join('\n');

		const blob = new Blob([csvContent], { type: 'text/csv' });
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `subdomains-${new Date().toISOString().split('T')[0]}.csv`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		window.URL.revokeObjectURL(url);
	}

	// Helper function to get badge color for source
	function getSourceBadgeColor(source) {
		const colors = {
			'subfinder': 'blue',
			'manual': 'green',
			'imported': 'yellow'
		};
		return colors[source] || 'gray';
	}

	// Clear messages after 5 seconds
	$: if (successMessage || errorMessage) {
		setTimeout(() => {
			successMessage = '';
			errorMessage = '';
		}, 5000);
	}

	// Computed filtered and sorted subdomains
	$: filteredSubdomains = subdomains
		.filter(subdomain => {
			// Search filter
			if (searchTerm && !subdomain.subdomain.toLowerCase().includes(searchTerm.toLowerCase()) && 
				!(subdomain.parent_domain || '').toLowerCase().includes(searchTerm.toLowerCase())) {
				return false;
			}
			
			// Status filter
			if (statusFilter === 'resolved' && !subdomain.resolved) return false;
			if (statusFilter === 'unresolved' && subdomain.resolved) return false;
			
			// Source filter
			if (sourceFilter !== 'all' && subdomain.source !== sourceFilter) return false;
			
			return true;
		})
		.sort((a, b) => {
			let aVal, bVal;
			
			switch (sortBy) {
				case 'subdomain':
					aVal = a.subdomain || '';
					bVal = b.subdomain || '';
					break;
				case 'parent_domain':
					aVal = a.parent_domain || '';
					bVal = b.parent_domain || '';
					break;
				case 'source':
					aVal = a.source || '';
					bVal = b.source || '';
					break;
				case 'resolved':
					aVal = a.resolved ? 1 : 0;
					bVal = b.resolved ? 1 : 0;
					break;
				default:
					aVal = a.subdomain || '';
					bVal = b.subdomain || '';
			}
			
			if (sortOrder === 'asc') {
				return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
			} else {
				return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
			}
		});

	// Pagination
	$: totalPages = Math.ceil(filteredSubdomains.length / itemsPerPage);
	$: paginatedSubdomains = filteredSubdomains.slice(
		(currentPage - 1) * itemsPerPage, 
		currentPage * itemsPerPage
	);

	// Get unique sources for filter dropdown
	$: uniqueSources = [...new Set(subdomains.map(s => s.source).filter(Boolean))].sort();

	// Reset page when filters change
	$: if (searchTerm || statusFilter || sourceFilter) {
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
</script>

<svelte:head>
	<title>Attack Surface - Domains</title>
</svelte:head>

<div class="p-0 max-w-none w-full min-w-0">
	<div class="px-6 py-4">
		<div class="flex justify-between items-center mb-6">
			<div>
				<h1 class="text-2xl font-bold text-gray-900 dark:text-white">Attack Surface - Domains</h1>
				<p class="text-gray-600 dark:text-gray-400">Manage and discover subdomains for your attack surface</p>
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
				<Button color="alternative" href="/settings/providers">
					Configure API Keys
				</Button>
				<Button on:click={() => newDomainModal = true}>
					<PlusOutline class="w-5 h-5 mr-2" />
					Scan Domain
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
					<div class="text-3xl font-bold text-blue-600">{stats.total_domains || 0}</div>
					<div class="text-sm text-gray-600">Total Domains</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-green-600">{stats.total_resolved || 0}</div>
					<div class="text-sm text-gray-600">Resolved</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-purple-600">{stats.total_unique_ips || 0}</div>
					<div class="text-sm text-gray-600">Unique IPs</div>
				</div>
			</Card>
			<Card class="p-6">
				<div class="text-center">
					<div class="text-3xl font-bold text-orange-600">{stats.recent_scans || 0}</div>
					<div class="text-sm text-gray-600">Recent Scans</div>
				</div>
			</Card>
		</div>
	</div>

	<!-- Subdomains Table - FULL WIDTH -->
	<div class="w-full min-h-screen bg-white dark:bg-gray-900">
		<div class="px-6 py-8">
			<div class="flex justify-between items-center mb-6">
				<h2 class="text-2xl font-bold text-gray-900 dark:text-white">Discovered Subdomains</h2>
				<div class="flex gap-3">
					<Button color="alternative" size="sm" on:click={exportSubdomains} disabled={subdomains.length === 0}>
						<DownloadOutline class="w-4 h-4 mr-2" />
						Export CSV
					</Button>
					<Button color="alternative" size="sm" on:click={loadSubdomains}>
						<SearchOutline class="w-4 h-4 mr-2" />
						Refresh
					</Button>
				</div>
			</div>

			<!-- Filters and Search -->
			<div class="grid grid-cols-1 xl:grid-cols-4 gap-8 mb-10 p-8 bg-gray-50 dark:bg-gray-800 rounded-xl shadow-lg">
				<!-- Search -->
				<div>
					<Label for="search" class="text-base mb-3 block font-semibold">Search Subdomains</Label>
					<Input 
						id="search"
						type="text" 
						placeholder="Search subdomains or domains..." 
						bind:value={searchTerm}
						class="w-full h-12 text-base"
					/>
				</div>

				<!-- Status Filter -->
				<div>
					<Label for="status-filter" class="text-base mb-3 block font-semibold">Status</Label>
					<select 
						id="status-filter"
						bind:value={statusFilter}
						class="w-full h-12 px-4 py-3 text-base border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
					>
						<option value="all">All Status</option>
						<option value="resolved">Resolved Only</option>
						<option value="unresolved">Unresolved Only</option>
					</select>
				</div>

				<!-- Source Filter -->
				<div>
					<Label for="source-filter" class="text-base mb-3 block font-semibold">Source</Label>
					<select 
						id="source-filter"
						bind:value={sourceFilter}
						class="w-full h-12 px-4 py-3 text-base border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
					>
						<option value="all">All Sources</option>
						{#each uniqueSources as source}
							<option value={source}>{source}</option>
						{/each}
					</select>
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
			</div>

			<!-- Results Summary -->
			<div class="flex justify-between items-center mb-8 px-2">
				<div class="text-lg text-gray-600 dark:text-gray-400">
					Showing {Math.min((currentPage - 1) * itemsPerPage + 1, filteredSubdomains.length)} - 
					{Math.min(currentPage * itemsPerPage, filteredSubdomains.length)} of {filteredSubdomains.length} results
					{#if filteredSubdomains.length !== subdomains.length}
						(filtered from {subdomains.length} total)
					{/if}
				</div>
				{#if filteredSubdomains.length > 0}
					<div class="flex gap-8 text-lg">
						<span class="text-green-600 font-bold">Resolved: {filteredSubdomains.filter(s => s.resolved).length}</span>
						<span class="text-red-600 font-bold">Unresolved: {filteredSubdomains.filter(s => !s.resolved).length}</span>
					</div>
				{/if}
			</div>

			{#if loading}
				<div class="flex justify-center py-12">
					<Spinner size="8" />
				</div>
			{:else if filteredSubdomains.length === 0}
				<div class="text-center py-12 text-gray-500">
					{#if subdomains.length === 0}
						No subdomains found. Start by scanning a domain.
					{:else}
						No subdomains match your current filters.
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
										on:click={() => handleSort('subdomain')}
									>
										Subdomain
										{#if sortBy === 'subdomain'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
								<th scope="col" class="px-8 py-6 w-1/5">
									<button 
										class="flex items-center gap-2 font-semibold hover:text-blue-600 focus:outline-none"
										on:click={() => handleSort('parent_domain')}
									>
										Parent Domain
										{#if sortBy === 'parent_domain'}
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
										on:click={() => handleSort('source')}
									>
										Source
										{#if sortBy === 'source'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
								<th scope="col" class="px-8 py-6 w-1/8">
									<button 
										class="flex items-center gap-2 font-semibold hover:text-blue-600 focus:outline-none"
										on:click={() => handleSort('resolved')}
									>
										Status
										{#if sortBy === 'resolved'}
											{#if sortOrder === 'asc'}
												<ChevronUpOutline class="w-4 h-4" />
											{:else}
												<ChevronDownOutline class="w-4 h-4" />
											{/if}
										{/if}
									</button>
								</th>
								<th scope="col" class="px-8 py-6 w-1/6">IP Address</th>
							</tr>
						</thead>
						<tbody>
							{#each paginatedSubdomains as subdomain}
								<tr class="bg-white border-b dark:bg-gray-800 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors">
									<td class="px-8 py-6">
										<div class="flex items-center gap-4">
											<a href="https://{subdomain.subdomain}" target="_blank" class="text-blue-600 hover:underline font-medium break-all text-base">
												{subdomain.subdomain}
											</a>
											<button 
												class="text-xs text-gray-500 hover:text-blue-600 px-3 py-1 rounded border border-gray-300 hover:border-blue-500 transition-colors flex-shrink-0"
												on:click={() => navigator.clipboard.writeText(subdomain.subdomain)}
												title="Copy to clipboard"
											>
												copy
											</button>
										</div>
									</td>
									<td class="px-8 py-6 text-gray-900 dark:text-white text-base">
										{subdomain.parent_domain || '-'}
									</td>
									<td class="px-8 py-6">
										<Badge color={getSourceBadgeColor(subdomain.source)} class="text-sm px-3 py-1">
											{subdomain.source || 'unknown'}
										</Badge>
									</td>
									<td class="px-8 py-6">
										<Badge color={subdomain.resolved ? 'green' : 'red'} class="text-sm px-3 py-1">
											{subdomain.resolved ? 'Resolved' : 'Unresolved'}
										</Badge>
									</td>
									<td class="px-8 py-6">
										{#if subdomain.ip}
											<div class="flex items-center gap-4">
												<span class="font-mono text-base text-gray-900 dark:text-white">{subdomain.ip}</span>
												<button 
													class="text-xs text-gray-500 hover:text-blue-600 px-3 py-1 rounded border border-gray-300 hover:border-blue-500 transition-colors flex-shrink-0"
													on:click={() => navigator.clipboard.writeText(subdomain.ip)}
													title="Copy IP to clipboard"
												>
													copy
												</button>
											</div>
										{:else}
											<span class="text-gray-400 text-base">-</span>
										{/if}
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

<!-- New Domain Scan Modal -->
<Modal bind:open={newDomainModal} autoclose={false}>
	<div slot="header">
		<h3 class="text-xl font-semibold">Scan Domain for Subdomains</h3>
	</div>
	
	<div class="space-y-4">
		<div>
			<Label for="domain">Domain</Label>
			<Input 
				id="domain"
				bind:value={newDomain}
				placeholder={includeTldDomains ? "Optional - leave empty to scan all TLD domains" : "example.com"}
				disabled={scanLoading}
			/>
			<Helper class="text-xs">
				{#if includeTldDomains}
					Optional: Enter a specific domain to scan along with TLD domains, or leave empty to scan only TLD domains
				{:else}
					Enter the primary domain to scan for subdomains
				{/if}
			</Helper>
			
			<!-- Include TLD domains option -->
			<div class="flex items-center space-x-2 mt-4">
				<Checkbox bind:checked={includeTldDomains} />
				<Label class="text-sm">Include discovered TLD domains in scan</Label>
			</div>
			<Helper class="text-xs">
				When enabled, subfinder will also scan all domains discovered through TLD discovery
				{#if stats.tld_domains > 0}
					({stats.tld_domains} TLD domains available)
				{:else}
					(no TLD domains found - run TLD discovery first)
				{/if}
			</Helper>
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
			<Label>Scan Options</Label>
			<div class="space-y-2 mt-2">
				<label class="flex items-center">
					<input type="checkbox" bind:checked={scanOptions.all_sources} class="mr-2" disabled={scanLoading} />
					Use all available sources (slower but more comprehensive)
				</label>
				<label class="flex items-center">
					<input type="checkbox" bind:checked={scanOptions.recursive} class="mr-2" disabled={scanLoading} />
					Recursive scanning
				</label>
			</div>
		</div>

		<div class="grid grid-cols-2 gap-4">
			<div>
				<Label for="timeout">Timeout (seconds)</Label>
				<Input 
					id="timeout" 
					type="number" 
					bind:value={scanOptions.timeout} 
					min="10" 
					max="300"
					disabled={scanLoading}
				/>
			</div>
			<div>
				<Label for="rate_limit">Rate Limit (req/sec)</Label>
				<Input 
					id="rate_limit" 
					type="number" 
					bind:value={scanOptions.rate_limit} 
					min="1" 
					max="100"
					disabled={scanLoading}
				/>
			</div>
		</div>

		{#if !scanOptions.all_sources}
			<div>
				<Label>Select Specific Sources (optional)</Label>
				<div class="max-h-40 overflow-y-auto border border-gray-300 rounded-md p-2 space-y-2">
					{#each availableSources as source}
						<label class="flex items-center">
							<input 
								type="checkbox" 
								value={source.name}
								checked={selectedSources.includes(source.name)}
								on:change={(e) => {
									if (e.target.checked) {
										selectedSources = [...selectedSources, source.name];
									} else {
										selectedSources = selectedSources.filter(s => s !== source.name);
									}
								}}
								disabled={scanLoading}
								class="mr-2"
							/>
							<span class="text-sm">
								{source.name} - {source.description}
								{source.requires_key ? '(API Key Configured ✓)' : '(Free)'}
							</span>
						</label>
					{/each}
				</div>
				<p class="text-sm text-gray-600 dark:text-gray-400 mt-2">
					💡 Only showing sources that are available: free sources and those with API keys configured in 
					<a href="/settings/providers" class="text-blue-600 underline">Settings > Providers</a>
				</p>
			</div>
		{/if}

		<div class="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-lg">
			<p class="text-sm text-blue-800 dark:text-blue-200">
				<strong>💡 Sources Available:</strong> Currently showing {availableSources.filter(s => !s.requires_key).length} free sources
				{#if availableSources.filter(s => s.requires_key).length > 0}
					and {availableSources.filter(s => s.requires_key).length} premium sources with configured API keys
				{/if}.
				Add more API keys in <a href="/settings/providers" class="underline">Settings > Providers</a> to unlock additional data sources.
			</p>
		</div>
	</div>

	<div slot="footer" class="flex flex-col gap-2">
		<!-- Debug info -->
		<div class="text-xs text-gray-500 p-2 bg-gray-100 rounded">
			Debug: domain="{newDomain}" | includeTldDomains={includeTldDomains} | selectedClient="{selectedClient}" | tld_domains={stats.tld_domains || 0}
		</div>
		
		<div class="flex justify-end gap-2">
			<Button color="alternative" on:click={() => newDomainModal = false} disabled={scanLoading}>
				Cancel
			</Button>
		<Button on:click={startScan} disabled={(!newDomain.trim() && !includeTldDomains) || !selectedClient || scanLoading}>
			{#if scanLoading}
				<Spinner class="mr-2" size="4" />
				Scanning...
			{:else}
				<SearchOutline class="w-4 h-4 mr-2" />
				{#if includeTldDomains && !newDomain.trim()}
					Scan TLD Domains ({stats.tld_domains || 0})
				{:else}
					Start Scan
				{/if}
			{/if}
		</Button>
		</div>
	</div>
</Modal> 
<script lang="ts">
	import { onMount } from 'svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { currentUser } from '@lib/stores/auth';
	import {
		Button,
		Card,
		Input,
		Label,
		Helper,
		Checkbox,
		Modal,
		Spinner,
		Alert,
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell,
		Badge,
		ButtonGroup,
		Search,
		Select,
		Textarea,
		Dropdown,
		DropdownItem
	} from 'flowbite-svelte';
	import {
		SearchOutline,
		ChevronDownOutline,
		DownloadOutline,
		ExclamationCircleOutline,
		InfoCircleOutline,
		ServerOutline,
		GlobeOutline
	} from 'flowbite-svelte-icons';

	// Reactive variables
	let orgNames = [''];
	let customRanges = [''];
	let useDomainIPs = true;
	let filterCloud = true;
	let discoveryLoading = false;
	let discoveryModal = false;
	let currentPage = 1;
	let itemsPerPage = 25;
	let search = '';
	let sortBy = 'confidence';
	let sortOrder = 'desc';
	let sourceFilter = 'all';
	let confidenceFilter = 'all';

	// Client selection
	let clients = [];
	let selectedClient = '';
	let clientId = '';

	// Results data
	let netblocks = [];
	let ips = [];
	let filteredNetblocks = [];
	let filteredIPs = [];

	// Statistics
	let stats = {
		total_netblocks: 0,
		total_ips: 0,
		high_confidence: 0,
		unique_asns: 0
	};

	// UI state
	let errorMessage = '';
	let successMessage = '';
	let activeTab = 'netblocks'; // 'netblocks' or 'ips'

	// Load initial data
	onMount(async () => {
		await loadClients();
		if ($currentUser?.client) {
			clientId = $currentUser.client;
			selectedClient = clientId;
			await Promise.all([
				loadNetblocks(),
				loadIPs(),
				loadStats()
			]);
		}
	});

	// Load clients for selection
	async function loadClients() {
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/collections/clients/records`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			clients = data.items || [];
		} catch (error) {
			console.error('Failed to load clients:', error);
		}
	}

	// Load netblocks for client
	async function loadNetblocks() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/netblocks?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.success) {
				netblocks = data.netblocks || [];
				applyNetblockFilters();
			}
		} catch (error) {
			console.error('Failed to load netblocks:', error);
		}
	}

	// Load IPs for client
	async function loadIPs() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/ips?client_id=${targetClientId}`, {
				headers: {
					'Authorization': `Bearer ${token}`
				}
			});
			const data = await response.json();
			if (data.success) {
				ips = data.ips || [];
				applyIPFilters();
			}
		} catch (error) {
			console.error('Failed to load IPs:', error);
		}
	}

	// Load statistics
	async function loadStats() {
		const targetClientId = selectedClient || clientId;
		if (!targetClientId) return;
		
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/netblocks/stats?client_id=${targetClientId}`, {
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

	// Apply filters to netblocks
	function applyNetblockFilters() {
		filteredNetblocks = netblocks.filter(nb => {
			// Search filter
			if (search && !nb.cidr.toLowerCase().includes(search.toLowerCase()) && 
				!nb.org_name?.toLowerCase().includes(search.toLowerCase()) &&
				!nb.organization?.toLowerCase().includes(search.toLowerCase())) {
				return false;
			}

			// Source filter
			if (sourceFilter !== 'all' && nb.source !== sourceFilter) {
				return false;
			}

			// Confidence filter
			if (confidenceFilter === 'high' && nb.confidence < 0.8) {
				return false;
			} else if (confidenceFilter === 'medium' && (nb.confidence < 0.5 || nb.confidence >= 0.8)) {
				return false;
			} else if (confidenceFilter === 'low' && nb.confidence >= 0.5) {
				return false;
			}

			return true;
		});

		// Sort
		filteredNetblocks.sort((a, b) => {
			let aVal = a[sortBy];
			let bVal = b[sortBy];
			
			if (typeof aVal === 'string') {
				aVal = aVal.toLowerCase();
				bVal = bVal.toLowerCase();
			}
			
			if (sortOrder === 'asc') {
				return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
			} else {
				return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
			}
		});
	}

	// Apply filters to IPs
	function applyIPFilters() {
		filteredIPs = ips.filter(ip => {
			// Search filter
			if (search && !ip.ip.includes(search) && 
				!ip.source_domain?.toLowerCase().includes(search.toLowerCase())) {
				return false;
			}

			// Source filter
			if (sourceFilter !== 'all' && ip.source !== sourceFilter) {
				return false;
			}

			return true;
		});

		// Sort
		filteredIPs.sort((a, b) => {
			let aVal = a[sortBy];
			let bVal = b[sortBy];
			
			if (typeof aVal === 'string') {
				aVal = aVal.toLowerCase();
				bVal = bVal.toLowerCase();
			}
			
			if (sortOrder === 'asc') {
				return aVal < bVal ? -1 : aVal > bVal ? 1 : 0;
			} else {
				return aVal > bVal ? -1 : aVal < bVal ? 1 : 0;
			}
		});
	}

	// Start netblock discovery
	async function startDiscovery() {
		if (!selectedClient) {
			errorMessage = 'Please select a client to discover netblocks for';
			return;
		}

		// Validate input
		const validOrgNames = orgNames.filter(name => name.trim()).map(name => name.trim());
		const validRanges = customRanges.filter(range => range.trim()).map(range => range.trim());

		if (validOrgNames.length === 0 && validRanges.length === 0 && !useDomainIPs) {
			errorMessage = 'Please specify at least one discovery method';
			return;
		}

		try {
			discoveryLoading = true;
			errorMessage = '';
			successMessage = '';

			const token = $pocketbase.authStore.token;
			const requestBody = {
				client_id: selectedClient,
				org_names: validOrgNames,
				custom_ranges: validRanges,
				use_domain_ips: useDomainIPs,
				filter_cloud: filterCloud
			};

			console.log('DEBUG: Starting netblock discovery with:', requestBody);

			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/attack-surface/netblock/discover`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${token}`
				},
				body: JSON.stringify(requestBody)
			});

			const data = await response.json();
			console.log('DEBUG: Discovery response:', data);

			if (data.success && data.result) {
				const result = data.result;
				successMessage = `Discovery completed! Found ${result.netblock_results?.length || 0} netblocks and ${result.filtered_ips?.length || 0} IPs in ${result.duration}`;
				
				// Reset form
				orgNames = [''];
				customRanges = [''];
				discoveryModal = false;
				
				// Reload data
				await Promise.all([
					loadNetblocks(),
					loadIPs(),
					loadStats()
				]);
			} else {
				errorMessage = data.message || 'Discovery failed';
			}
		} catch (error) {
			console.error('Discovery failed:', error);
			errorMessage = `Discovery failed: ${error.message}`;
		} finally {
			discoveryLoading = false;
		}
	}

	// Add organization name field
	function addOrgName() {
		orgNames = [...orgNames, ''];
	}

	// Remove organization name field
	function removeOrgName(index) {
		orgNames = orgNames.filter((_, i) => i !== index);
	}

	// Add custom range field
	function addCustomRange() {
		customRanges = [...customRanges, ''];
	}

	// Remove custom range field
	function removeCustomRange(index) {
		customRanges = customRanges.filter((_, i) => i !== index);
	}

	// Export data as CSV
	function exportCSV() {
		if (activeTab === 'netblocks') {
			const csvData = [
				['CIDR', 'IP', 'ASN', 'ASN Name', 'Organization', 'Confidence', 'Source', 'Country', 'Discovered At'],
				...filteredNetblocks.map(nb => [
					nb.cidr,
					nb.ip,
					nb.asn || '',
					nb.asn_name || '',
					nb.organization || '',
					nb.confidence,
					nb.source,
					nb.country || '',
					new Date(nb.discovered_at).toLocaleString()
				])
			];

			const csvContent = csvData.map(row => 
				row.map(cell => `"${cell}"`).join(',')
			).join('\n');

			const blob = new Blob([csvContent], { type: 'text/csv' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `netblocks-${selectedClient || clientId}-${new Date().toISOString().split('T')[0]}.csv`;
			a.click();
			URL.revokeObjectURL(url);
		} else {
			const csvData = [
				['IP', 'Source', 'Source Domain', 'Discovered At'],
				...filteredIPs.map(ip => [
					ip.ip,
					ip.source,
					ip.source_domain || '',
					new Date(ip.discovered_at).toLocaleString()
				])
			];

			const csvContent = csvData.map(row => 
				row.map(cell => `"${cell}"`).join(',')
			).join('\n');

			const blob = new Blob([csvContent], { type: 'text/csv' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `ips-${selectedClient || clientId}-${new Date().toISOString().split('T')[0]}.csv`;
			a.click();
			URL.revokeObjectURL(url);
		}
	}

	// Get confidence color
	function getConfidenceColor(confidence) {
		if (confidence >= 0.8) return 'green';
		if (confidence >= 0.5) return 'yellow';
		return 'red';
	}

	// Reactive filters
	$: {
		if (activeTab === 'netblocks') {
			applyNetblockFilters();
		} else {
			applyIPFilters();
		}
	}

	// Reactive client change
	$: if (selectedClient) {
		loadNetblocks();
		loadIPs();
		loadStats();
	}

	// Pagination
	$: totalPages = Math.ceil((activeTab === 'netblocks' ? filteredNetblocks.length : filteredIPs.length) / itemsPerPage);
	$: paginatedData = (activeTab === 'netblocks' ? filteredNetblocks : filteredIPs).slice(
		(currentPage - 1) * itemsPerPage,
		currentPage * itemsPerPage
	);
</script>

<svelte:head>
	<title>Netblock Discovery - Attack Surface</title>
</svelte:head>

<div class="container mx-auto px-4 py-6 max-w-none w-full min-h-screen">
	<!-- Header -->
	<div class="mb-8">
		<div class="flex justify-between items-start mb-4">
			<div>
				<h1 class="text-3xl font-bold text-gray-900 mb-2">Netblock Discovery</h1>
				<p class="text-gray-600">Discover IP netblocks owned by organizations using WhoisXML API and domain analysis</p>
			</div>
			<div class="flex gap-3">
				<Button on:click={() => discoveryModal = true} color="primary">
					<SearchOutline class="w-4 h-4 mr-2" />
					Start Discovery
				</Button>
				<Button on:click={exportCSV} color="alternative">
					<DownloadOutline class="w-4 h-4 mr-2" />
					Export CSV
				</Button>
			</div>
		</div>

		<!-- Client Selection -->
		{#if clients.length > 1}
			<div class="mb-4">
				<Label for="client-select" class="mb-2">Client</Label>
				<Select id="client-select" bind:value={selectedClient} class="max-w-sm">
					{#each clients as client}
						<option value={client.id}>{client.name}</option>
					{/each}
				</Select>
			</div>
		{/if}

		<!-- Statistics Cards -->
		<div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-6">
			<Card class="p-6">
				<div class="flex items-center">
					<ServerOutline class="w-8 h-8 text-blue-500 mr-3" />
					<div>
						<p class="text-2xl font-bold text-gray-900">{stats.total_netblocks || 0}</p>
						<p class="text-sm text-gray-600">Total Netblocks</p>
					</div>
				</div>
			</Card>

			<Card class="p-6">
				<div class="flex items-center">
					<GlobeOutline class="w-8 h-8 text-green-500 mr-3" />
					<div>
						<p class="text-2xl font-bold text-gray-900">{stats.total_ips || 0}</p>
						<p class="text-sm text-gray-600">Collected IPs</p>
					</div>
				</div>
			</Card>

			<Card class="p-6">
				<div class="flex items-center">
					<ExclamationCircleOutline class="w-8 h-8 text-orange-500 mr-3" />
					<div>
						<p class="text-2xl font-bold text-gray-900">{stats.high_confidence || 0}</p>
						<p class="text-sm text-gray-600">High Confidence</p>
					</div>
				</div>
			</Card>

			<Card class="p-6">
				<div class="flex items-center">
					<InfoCircleOutline class="w-8 h-8 text-purple-500 mr-3" />
					<div>
						<p class="text-2xl font-bold text-gray-900">{stats.unique_asns || 0}</p>
						<p class="text-sm text-gray-600">Unique ASNs</p>
					</div>
				</div>
			</Card>
		</div>
	</div>

	<!-- Messages -->
	{#if errorMessage}
		<div class="mb-4">
			<Alert color="red">{errorMessage}</Alert>
		</div>
	{/if}

	{#if successMessage}
		<div class="mb-4">
			<Alert color="green">{successMessage}</Alert>
		</div>
	{/if}

	<!-- Tab Navigation and Table Section -->
	<div class="w-full">
		<!-- Tab Navigation -->
		<div class="flex border-b border-gray-200 mb-6">
			<button
				class="px-6 py-3 text-sm font-medium border-b-2 {activeTab === 'netblocks' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
				on:click={() => { activeTab = 'netblocks'; currentPage = 1; }}
			>
				Netblocks ({filteredNetblocks.length})
			</button>
			<button
				class="px-6 py-3 text-sm font-medium border-b-2 {activeTab === 'ips' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
				on:click={() => { activeTab = 'ips'; currentPage = 1; }}
			>
				IPs ({filteredIPs.length})
			</button>
		</div>

		<!-- Filters -->
		<div class="flex flex-wrap gap-4 items-center mb-6">
			<div class="flex-1 min-w-64">
				<Search placeholder="Search {activeTab}..." bind:value={search} />
			</div>

			<Select bind:value={sourceFilter} class="w-48">
				<option value="all">All Sources</option>
				<option value="whoisxml">WhoisXML API</option>
				<option value="manual">Manual</option>
				<option value="dns">DNS</option>
				<option value="mx">MX Records</option>
				<option value="ns">NS Records</option>
			</Select>

			{#if activeTab === 'netblocks'}
				<Select bind:value={confidenceFilter} class="w-48">
					<option value="all">All Confidence</option>
					<option value="high">High (≥80%)</option>
					<option value="medium">Medium (50-79%)</option>
					<option value="low">Low (&lt;50%)</option>
				</Select>
			{/if}

			<Select bind:value={itemsPerPage} class="w-32">
				<option value={10}>10</option>
				<option value={25}>25</option>
				<option value={50}>50</option>
				<option value={100}>100</option>
			</Select>
		</div>

		<!-- Table -->
		<div class="w-full overflow-x-auto bg-white shadow-sm rounded-lg">
			<Table divClass="overflow-x-auto">
				<TableHead>
					{#if activeTab === 'netblocks'}
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'cidr'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								CIDR <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'confidence'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								Confidence <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">Organization</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'asn'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								ASN <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">Source</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">Country</TableHeadCell>
					{:else}
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'ip'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								IP Address <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'source'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								Source <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">Source Domain</TableHeadCell>
						<TableHeadCell class="px-8 py-6 text-lg font-semibold">
							<button on:click={() => { sortBy = 'discovered_at'; sortOrder = sortOrder === 'asc' ? 'desc' : 'asc'; }} class="flex items-center">
								Discovered <ChevronDownOutline class="w-4 h-4 ml-1" />
							</button>
						</TableHeadCell>
					{/if}
				</TableHead>
				<TableBody>
					{#each paginatedData as item}
						<TableBodyRow>
							{#if activeTab === 'netblocks'}
								<TableBodyCell class="px-8 py-6 text-base font-mono">
									{item.cidr}
									<div class="text-sm text-gray-500">{item.ip}</div>
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6">
									<Badge color={getConfidenceColor(item.confidence)} class="text-sm">
										{Math.round(item.confidence * 100)}%
									</Badge>
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									<div>{item.org_name || item.organization || '-'}</div>
									{#if item.matched_criteria && item.matched_criteria.length > 0}
										<div class="text-sm text-gray-500 mt-1">
											{item.matched_criteria[0]}
											{#if item.matched_criteria.length > 1}
												<span class="text-xs">+{item.matched_criteria.length - 1} more</span>
											{/if}
										</div>
									{/if}
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									{#if item.asn}
										<div class="font-mono">AS{item.asn}</div>
										{#if item.asn_name}
											<div class="text-sm text-gray-500">{item.asn_name}</div>
										{/if}
									{:else}
										<span class="text-gray-400">-</span>
									{/if}
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									<Badge color="blue" variant="outline">{item.source}</Badge>
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									{item.country || '-'}
								</TableBodyCell>
							{:else}
								<TableBodyCell class="px-8 py-6 text-base font-mono">
									{item.ip}
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									<Badge color="blue" variant="outline">{item.source}</Badge>
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									{item.source_domain || '-'}
								</TableBodyCell>
								<TableBodyCell class="px-8 py-6 text-base">
									{new Date(item.discovered_at).toLocaleString()}
								</TableBodyCell>
							{/if}
						</TableBodyRow>
					{/each}
				</TableBody>
			</Table>

			{#if paginatedData.length === 0}
				<div class="text-center py-12 text-gray-500">
					<ServerOutline class="w-12 h-12 mx-auto mb-3 text-gray-300" />
					<p class="text-lg">No {activeTab} found</p>
					<p class="text-sm">Start discovery to find netblocks and IPs</p>
				</div>
			{/if}
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<div class="flex justify-between items-center mt-6">
				<div class="text-sm text-gray-600">
					Showing {(currentPage - 1) * itemsPerPage + 1} to {Math.min(currentPage * itemsPerPage, activeTab === 'netblocks' ? filteredNetblocks.length : filteredIPs.length)} of {activeTab === 'netblocks' ? filteredNetblocks.length : filteredIPs.length} results
				</div>
				<ButtonGroup>
					<Button disabled={currentPage === 1} on:click={() => currentPage--}>Previous</Button>
					<Button disabled={currentPage === totalPages} on:click={() => currentPage++}>Next</Button>
				</ButtonGroup>
			</div>
		{/if}
	</div>
</div>

<!-- Discovery Modal -->
<Modal bind:open={discoveryModal} size="lg" title="Start Netblock Discovery">
	<div class="space-y-6">
		<!-- Client Selection -->
		<div>
			<Label for="client">Client</Label>
			<Select id="client" bind:value={selectedClient} required>
				{#each clients as client}
					<option value={client.id}>{client.name}</option>
				{/each}
			</Select>
			<Helper class="text-xs">Select which client to discover netblocks for</Helper>
		</div>

		<!-- Organization Names -->
		<div>
			<Label class="text-sm font-medium">Organization Names</Label>
			<Helper class="text-xs mb-3">Enter organization names to search for in WhoisXML API (e.g., "Acme Corp", "Example Inc")</Helper>
			{#each orgNames as orgName, index}
				<div class="flex gap-2 mb-2">
					<Input 
						bind:value={orgNames[index]} 
						placeholder="Organization name" 
						class="flex-1"
					/>
					{#if orgNames.length > 1}
						<Button color="red" size="sm" on:click={() => removeOrgName(index)}>Remove</Button>
					{/if}
				</div>
			{/each}
			<Button color="alternative" size="sm" on:click={addOrgName}>Add Organization</Button>
		</div>

		<!-- Custom IP Ranges -->
		<div>
			<Label class="text-sm font-medium">Custom IP Ranges</Label>
			<Helper class="text-xs mb-3">Enter IP addresses, CIDR ranges, or IP ranges (e.g., "192.168.1.0/24", "10.0.0.1-10.0.0.100")</Helper>
			{#each customRanges as range, index}
				<div class="flex gap-2 mb-2">
					<Input 
						bind:value={customRanges[index]} 
						placeholder="IP range or CIDR" 
						class="flex-1"
					/>
					{#if customRanges.length > 1}
						<Button color="red" size="sm" on:click={() => removeCustomRange(index)}>Remove</Button>
					{/if}
				</div>
			{/each}
			<Button color="alternative" size="sm" on:click={addCustomRange}>Add Range</Button>
		</div>

		<!-- Options -->
		<div class="space-y-4">
			<div class="flex items-center space-x-2">
				<Checkbox bind:checked={useDomainIPs} />
				<Label class="text-sm">Collect IPs from discovered domains</Label>
			</div>
			<Helper class="text-xs">
				When enabled, will collect IP addresses from all domains discovered through previous subdomain and TLD scans
			</Helper>

			<div class="flex items-center space-x-2">
				<Checkbox bind:checked={filterCloud} />
				<Label class="text-sm">Filter out cloud provider IPs</Label>
			</div>
			<Helper class="text-xs">
				Remove IPs belonging to major cloud providers (AWS, Azure, Cloudflare, etc.) from results
			</Helper>
		</div>
	</div>

	<div slot="footer" class="flex justify-end gap-2">
		<Button color="alternative" on:click={() => discoveryModal = false} disabled={discoveryLoading}>
			Cancel
		</Button>
		<Button on:click={startDiscovery} disabled={discoveryLoading}>
			{#if discoveryLoading}
				<Spinner class="mr-2" size="4" />
				Discovering...
			{:else}
				<SearchOutline class="w-4 h-4 mr-2" />
				Start Discovery
			{/if}
		</Button>
	</div>
</Modal> 
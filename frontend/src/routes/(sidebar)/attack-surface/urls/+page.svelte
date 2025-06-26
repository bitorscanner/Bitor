<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/stores';
	import { pocketbase } from '$lib/stores/pocketbase';
	import { 
		Button, 
		Card, 
		Table, 
		TableBody, 
		TableBodyCell, 
		TableBodyRow, 
		TableHead, 
		TableHeadCell, 
		Input, 
		Label, 
		Select, 
		Checkbox, 
		Badge, 
		Modal, 
		Alert, 
		Spinner,
		Helper,
		Textarea,
		NumberInput
	} from 'flowbite-svelte';
	import { 
		SearchOutline, 
		ExclamationCircleOutline, 
		CheckCircleOutline,
		GlobeOutline,
		ClockOutline,
		ServerOutline,
		CodeBranchOutline,
		DownloadOutline
	} from 'flowbite-svelte-icons';
	import { urlScanService, type URLScanRequest, type URLScanResult, type URLScanStats } from '$lib/services/URLScanService';

	// Page state
	let loading = true;
	let scanLoading = false;
	let errorMessage = '';
	let successMessage = '';
	
	// Data
	let urls: URLScanResult[] = [];
	let stats: URLScanStats | null = null;
	let clients: any[] = [];
	let selectedClient = '';
	
	// Scan tracking
	let activeScanId = '';
	let progressInterval: any = null;
	
	// Scan configuration
	let newScanModal = false;
	let scanConfig: Partial<URLScanRequest> = {
		include_ports: true,
		include_domains: true,
		include_subdomains: true,
		only_web_ports: true,
		schemes: ['http', 'https'],
		threads: 10,
		timeout: 10,
		retries: 2,
		follow_redirects: true,
		tech_detection: true,
		status_code: true,
		content_length: true,
		response_time: true,
		execution_mode: 'local'
	};
	let manualUrls = '';
	let customPorts = '';
	
	// Filtering and display
	let searchQuery = '';
	let statusFilter = '';
	let schemeFilter = '';
	let sourceFilter = '';
	let hostFilter = '';
	
	// Computed filtered and sorted URLs
	$: filteredUrls = (urls || []).filter(url => {
		const matchesSearch = !searchQuery || 
			url.url.toLowerCase().includes(searchQuery.toLowerCase()) ||
			url.host.toLowerCase().includes(searchQuery.toLowerCase()) ||
			url.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
			url.server?.toLowerCase().includes(searchQuery.toLowerCase());
		
		const matchesStatus = !statusFilter || url.status_code.toString() === statusFilter;
		const matchesScheme = !schemeFilter || url.scheme === schemeFilter;
		const matchesSource = !sourceFilter || url.source === sourceFilter;
		const matchesHost = !hostFilter || url.host.toLowerCase().includes(hostFilter.toLowerCase());
		
		return matchesSearch && matchesStatus && matchesScheme && matchesSource && matchesHost;
	});

	function startProgressTracking(scanId: string) {
		if (!scanId) return;
		
		// Clear any existing interval
		if (progressInterval) {
			clearInterval(progressInterval);
		}
		
		// Poll for progress every 10 seconds (less frequent since we have notifications)
		progressInterval = setInterval(async () => {
			try {
				const response = await urlScanService.getScanProgress(scanId);
				if (response.success && response.progress) {
					// If completed or failed, stop polling and refresh data
					if (response.progress.status === 'completed' || response.progress.status === 'failed') {
						clearInterval(progressInterval);
						activeScanId = '';
						
						// Refresh the data after scan completion
						await loadURLs();
						await loadStats();
					}
				} else {
					// If we can't get progress, assume scan is done and clear tracking
					clearInterval(progressInterval);
					activeScanId = '';
					await loadURLs();
					await loadStats();
				}
			} catch (error) {
				console.error('Error getting scan progress:', error);
				// Clear on error
				clearInterval(progressInterval);
				activeScanId = '';
			}
		}, 10000);
		
		// Stop polling after 10 minutes as a safety measure
		setTimeout(() => {
			if (progressInterval) {
				clearInterval(progressInterval);
				activeScanId = '';
			}
		}, 600000);
	}
	
	function stopProgressTracking() {
		if (progressInterval) {
			clearInterval(progressInterval);
			progressInterval = null;
		}
		activeScanId = '';
	}

	onMount(async () => {
		await Promise.all([
			loadClients(),
			loadURLs(),
			loadStats()
		]);
		loading = false;
	});
	
	// Cleanup on destroy
	onDestroy(() => {
		stopProgressTracking();
	});

	async function loadClients() {
		try {
			const records = await $pocketbase.collection('clients').getFullList();
			clients = records;
			if (clients.length > 0 && !selectedClient) {
				selectedClient = clients[0].id;
			}
		} catch (error) {
			console.error('Failed to load clients:', error);
			errorMessage = 'Failed to load clients';
		}
	}

	async function loadURLs() {
		if (!selectedClient) return;
		
		try {
			const response = await urlScanService.getURLs(selectedClient);
			if (response.success) {
				urls = response.urls || [];
			} else {
				urls = [];
				errorMessage = response.message || 'Failed to load URLs';
			}
		} catch (error) {
			console.error('Failed to load URLs:', error);
			urls = [];
			errorMessage = 'Failed to load URLs';
		}
	}

	async function loadStats() {
		if (!selectedClient) return;
		
		try {
			const response = await urlScanService.getStats(selectedClient);
			if (response.success) {
				stats = response.stats;
			} else {
				console.error('Failed to load stats:', response.message);
			}
		} catch (error) {
			console.error('Failed to load stats:', error);
		}
	}

	async function startURLScan() {
		if (!selectedClient) {
			errorMessage = 'Please select a client to scan for';
			return;
		}

		// Validate that at least one source is selected
		if (!scanConfig.include_ports && !scanConfig.include_domains && !scanConfig.include_subdomains && !manualUrls.trim()) {
			errorMessage = 'Please select at least one URL source or provide manual URLs';
			return;
		}

		scanLoading = true;
		errorMessage = '';
		successMessage = '';

		try {
			// Parse manual URLs
			const targetUrls = manualUrls.trim() 
				? manualUrls.split('\n').map(url => url.trim()).filter(url => url)
				: [];

			// Parse custom ports
			const ports = customPorts.trim()
				? customPorts.split(',').map(p => parseInt(p.trim())).filter(p => !isNaN(p))
				: undefined;

			const requestBody: URLScanRequest = {
				client_id: selectedClient,
				target_urls: targetUrls.length > 0 ? targetUrls : undefined,
				include_ports: scanConfig.include_ports || false,
				include_domains: scanConfig.include_domains || false,
				include_subdomains: scanConfig.include_subdomains || false,
				schemes: scanConfig.schemes || ['http', 'https'],
				ports: ports,
				only_web_ports: scanConfig.only_web_ports || false,
				threads: scanConfig.threads || 10,
				timeout: scanConfig.timeout || 10,
				retries: scanConfig.retries || 2,
				follow_redirects: scanConfig.follow_redirects || false,
				tech_detection: scanConfig.tech_detection || false,
				status_code: scanConfig.status_code || false,
				content_length: scanConfig.content_length || false,
				response_time: scanConfig.response_time || false,
				execution_mode: scanConfig.execution_mode || 'local'
			};

			const response = await urlScanService.startScan(requestBody);
			
			if (response.success) {
				successMessage = `URL scan started successfully! Scan ID: ${response.scan_id}`;
				newScanModal = false;
				
				// Start tracking the scan progress
				activeScanId = response.scan_id || '';
				startProgressTracking(activeScanId);
			} else {
				errorMessage = response.error || 'URL scan failed';
			}
		} catch (error: any) {
			errorMessage = 'Failed to start URL scan: ' + error.message;
		} finally {
			scanLoading = false;
		}
	}

	function exportURLs() {
		const csvContent = [
			['URL', 'Host', 'Port', 'Status Code', 'Title', 'Server', 'Content Type', 'Response Time', 'Content Length', 'Technologies', 'Source', 'IP'],
			...filteredUrls.map(url => [
				url.url,
				url.host,
				url.port.toString(),
				url.status_code.toString(),
				url.title || '',
				url.server || '',
				url.content_type || '',
				url.response_time || '',
				url.content_length.toString(),
				url.technologies?.join(';') || '',
				url.source,
				url.ip || ''
			])
		].map(row => row.map(cell => `"${cell}"`).join(',')).join('\n');

		const blob = new Blob([csvContent], { type: 'text/csv' });
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `urls-${new Date().toISOString().split('T')[0]}.csv`;
		document.body.appendChild(a);
		a.click();
		document.body.removeChild(a);
		window.URL.revokeObjectURL(url);
	}

	// Helper function to get badge color for scheme
	function getSchemeBadgeColor(scheme: string) {
		const colors = {
			'https': 'green',
			'http': 'yellow'
		};
		return colors[scheme] || 'gray';
	}

	// Helper function to get badge color for source
	function getSourceBadgeColor(source: string) {
		const colors = {
			'ports': 'blue',
			'domains': 'green', 
			'subdomains': 'purple',
			'manual': 'yellow'
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

	// Watch for client changes
	$: if (selectedClient) {
		loadURLs();
		loadStats();
	}

	// Get unique values for filters
	$: uniqueStatuses = [...new Set(urls.map(url => url.status_code.toString()))].sort();
	$: uniqueSchemes = [...new Set(urls.map(url => url.scheme))].sort();
	$: uniqueSources = [...new Set(urls.map(url => url.source))].sort();
</script>

<svelte:head>
	<title>URL Discovery - Attack Surface</title>
</svelte:head>

<div class="p-6 space-y-6">
	<!-- Page Header -->
	<div class="flex justify-between items-center">
		<div>
			<h1 class="text-3xl font-bold text-gray-900 dark:text-white">URL Discovery</h1>
			<p class="text-gray-600 dark:text-gray-400 mt-2">
				Discover and analyze web interfaces from discovered assets
			</p>
		</div>
		<div class="flex items-center gap-3">
			{#if activeScanId}
				<div class="flex items-center gap-2 px-3 py-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
					<Spinner size="4" />
					<div class="text-sm">
						<div class="font-medium text-blue-800 dark:text-blue-200">
							URL Scan Running
						</div>
						<div class="text-blue-600 dark:text-blue-300">
							Scan ID: {activeScanId}
						</div>
					</div>
				</div>
			{/if}
			<Button on:click={() => newScanModal = true} disabled={!selectedClient || !!activeScanId}>
				<SearchOutline class="w-4 h-4 mr-2" />
				{activeScanId ? 'Scan Running...' : 'New URL Scan'}
			</Button>
		</div>
	</div>

	<!-- Messages -->
	{#if errorMessage}
		<Alert color="red" dismissable on:close={() => errorMessage = ''}>
							<ExclamationCircleOutline slot="icon" class="w-4 h-4" />
			{errorMessage}
		</Alert>
	{/if}

	{#if successMessage}
		<Alert color="green" dismissable on:close={() => successMessage = ''}>
			<CheckCircleOutline slot="icon" class="w-4 h-4" />
			{successMessage}
		</Alert>
	{/if}

	<!-- Stats Cards -->
	{#if stats}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
			<Card>
				<div class="flex items-center">
					<div class="p-2 bg-blue-100 rounded-lg dark:bg-blue-900">
						<GlobeOutline class="w-6 h-6 text-blue-600 dark:text-blue-400" />
					</div>
					<div class="ml-4">
						<p class="text-sm font-medium text-gray-600 dark:text-gray-400">Total URLs</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_urls.toLocaleString()}</p>
					</div>
				</div>
			</Card>

			<Card>
				<div class="flex items-center">
					<div class="p-2 bg-green-100 rounded-lg dark:bg-green-900">
						<ServerOutline class="w-6 h-6 text-green-600 dark:text-green-400" />
					</div>
					<div class="ml-4">
						<p class="text-sm font-medium text-gray-600 dark:text-gray-400">Unique Hosts</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.unique_hosts.toLocaleString()}</p>
					</div>
				</div>
			</Card>

			<Card>
				<div class="flex items-center">
					<div class="p-2 bg-purple-100 rounded-lg dark:bg-purple-900">
						<SearchOutline class="w-6 h-6 text-purple-600 dark:text-purple-400" />
					</div>
					<div class="ml-4">
						<p class="text-sm font-medium text-gray-600 dark:text-gray-400">Total Scans</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_scans.toLocaleString()}</p>
					</div>
				</div>
			</Card>

			<Card>
				<div class="flex items-center">
					<div class="p-2 bg-yellow-100 rounded-lg dark:bg-yellow-900">
						<CodeBranchOutline class="w-6 h-6 text-yellow-600 dark:text-yellow-400" />
					</div>
					<div class="ml-4">
						<p class="text-sm font-medium text-gray-600 dark:text-gray-400">2xx Responses</p>
						<p class="text-2xl font-bold text-gray-900 dark:text-white">{(stats.status_breakdown['2xx'] || 0).toLocaleString()}</p>
					</div>
				</div>
			</Card>
		</div>
	{/if}

	<!-- Filters and Controls -->
	<div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-6 space-y-4">
		<h2 class="text-lg font-semibold text-gray-900 dark:text-white mb-4">Filters & Search</h2>
		
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
			<div>
				<Label for="client-select" class="mb-2">Client</Label>
				<Select id="client-select" bind:value={selectedClient} disabled={loading}>
					{#each clients as client}
						<option value={client.id}>{client.name}</option>
					{/each}
				</Select>
			</div>
			<div>
				<Label for="search" class="mb-2">Search URLs</Label>
				<Input 
					id="search"
					placeholder="Search by URL, host, title, or server..." 
					bind:value={searchQuery}
				/>
			</div>
			<div>
				<Label for="status-filter" class="mb-2">Status Code</Label>
				<Select id="status-filter" bind:value={statusFilter}>
					<option value="">All Status Codes</option>
					{#each uniqueStatuses as status}
						<option value={status}>{status}</option>
					{/each}
				</Select>
			</div>
			<div>
				<Label for="scheme-filter" class="mb-2">Scheme</Label>
				<Select id="scheme-filter" bind:value={schemeFilter}>
					<option value="">All Schemes</option>
					{#each uniqueSchemes as scheme}
						<option value={scheme}>{scheme}</option>
					{/each}
				</Select>
			</div>
		</div>
		
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
			<div>
				<Label for="source-filter" class="mb-2">Source</Label>
				<Select id="source-filter" bind:value={sourceFilter}>
					<option value="">All Sources</option>
					{#each uniqueSources as source}
						<option value={source}>{source}</option>
					{/each}
				</Select>
			</div>
			<div>
				<Label for="host-filter" class="mb-2">Host Filter</Label>
				<Input 
					id="host-filter"
					placeholder="Filter by host..." 
					bind:value={hostFilter}
				/>
			</div>
			<div class="flex items-end">
				<Button on:click={exportURLs} disabled={filteredUrls.length === 0} class="w-full">
					<DownloadOutline class="w-4 h-4 mr-2" />
					Export CSV
				</Button>
			</div>
		</div>
	</div>

	<!-- URLs Table -->
	<div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
		<div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
			<h2 class="text-xl font-semibold text-gray-900 dark:text-white">
				Discovered URLs ({filteredUrls.length.toLocaleString()})
			</h2>
		</div>

		{#if loading}
			<div class="flex justify-center py-8">
				<Spinner size="8" />
			</div>
		{:else if filteredUrls.length === 0}
			<div class="text-center py-8">
				<GlobeOutline class="w-12 h-12 text-gray-400 mx-auto mb-4" />
				<p class="text-gray-500 dark:text-gray-400">No URLs found</p>
				{#if !selectedClient}
					<p class="text-sm text-gray-400 mt-2">Please select a client to view URLs</p>
				{:else}
					<p class="text-sm text-gray-400 mt-2">Start a URL scan to discover web interfaces</p>
				{/if}
			</div>
		{:else}
			<div class="overflow-x-auto">
				<Table hoverable={true} striped={true} class="w-full">
					<TableHead>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider min-w-0">URL</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">Status</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-48">Title</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-32">Server</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">Response Time</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-28">Content Length</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-48">Technologies</TableHeadCell>
						<TableHeadCell class="px-6 py-4 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-24">Source</TableHeadCell>
					</TableHead>
					<TableBody>
						{#each filteredUrls as url}
							<TableBodyRow>
								<TableBodyCell class="px-6 py-4">
									<div class="space-y-1">
										<a 
											href={url.url} 
											target="_blank" 
											rel="noopener noreferrer"
											class="text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 break-all font-medium"
										>
											{url.url}
										</a>
										<div class="flex space-x-2 items-center">
											<Badge color={getSchemeBadgeColor(url.scheme)}>{url.scheme}</Badge>
											<span class="text-xs text-gray-500">{url.host}:{url.port}</span>
											{#if url.ip}
												<span class="text-xs text-gray-400">({url.ip})</span>
											{/if}
										</div>
									</div>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<Badge color={urlScanService.getStatusCodeColor(url.status_code).includes('green') ? 'green' : 
										urlScanService.getStatusCodeColor(url.status_code).includes('yellow') ? 'yellow' : 'red'}>
										{url.status_code}
									</Badge>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<span class="text-sm text-gray-900 dark:text-gray-100" title={url.title}>
										{url.title ? (url.title.length > 60 ? url.title.substring(0, 60) + '...' : url.title) : 'N/A'}
									</span>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<span class="text-sm text-gray-600 dark:text-gray-300">{url.server || 'N/A'}</span>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<span class="text-sm text-gray-600 dark:text-gray-300">{urlScanService.formatResponseTime(url.response_time)}</span>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<span class="text-sm text-gray-600 dark:text-gray-300">{urlScanService.formatContentLength(url.content_length)}</span>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<div class="flex flex-wrap gap-1">
										{#each (url.technologies || []).slice(0, 2) as tech}
											<Badge color="blue" class="text-xs">{tech}</Badge>
										{/each}
										{#if (url.technologies || []).length > 2}
											<Badge color="gray" class="text-xs">+{(url.technologies || []).length - 2}</Badge>
										{/if}
									</div>
								</TableBodyCell>
								<TableBodyCell class="px-6 py-4">
									<Badge color={getSourceBadgeColor(url.source)}>{url.source}</Badge>
								</TableBodyCell>
							</TableBodyRow>
						{/each}
					</TableBody>
				</Table>
			</div>
		{/if}
	</div>
</div>

<!-- New URL Scan Modal -->
<Modal bind:open={newScanModal} size="xl" autoclose={false} class="w-full">
	<div slot="header">
		<h3 class="text-xl font-semibold text-gray-900 dark:text-white">
			Start New URL Scan
		</h3>
	</div>

	<div class="space-y-6">
		<!-- Client Selection -->
		<div>
			<Label for="scan-client-select" class="text-base font-medium mb-2">Target Client</Label>
			<Select id="scan-client-select" bind:value={selectedClient} required>
				<option value="">Select a client to scan for...</option>
				{#each clients as client}
					<option value={client.id}>{client.name}</option>
				{/each}
			</Select>
			<Helper class="text-xs">Choose which client to run the URL scan for</Helper>
		</div>

		<!-- URL Sources -->
		<div>
			<Label class="text-base font-medium mb-3">URL Sources</Label>
			<div class="space-y-3">
				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.include_ports} />
					<Label class="text-sm">Include URLs from port scan results</Label>
				</div>
				<Helper class="text-xs">
					Generate URLs from discovered open ports (e.g., 192.168.1.1:8080 → http://192.168.1.1:8080)
				</Helper>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.include_domains} />
					<Label class="text-sm">Include URLs from discovered domains</Label>
				</div>
				<Helper class="text-xs">
					Generate URLs from parent domains discovered through TLD discovery
				</Helper>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.include_subdomains} />
					<Label class="text-sm">Include URLs from discovered subdomains</Label>
				</div>
				<Helper class="text-xs">
					Generate URLs from subdomains discovered through subfinder
				</Helper>
			</div>
		</div>

		<!-- Manual URLs -->
		<div>
			<Label for="manual-urls" class="text-base font-medium mb-2">Manual URLs</Label>
			<Textarea 
				id="manual-urls"
				placeholder="Enter URLs (one per line)&#10;https://example.com&#10;http://test.com:8080&#10;https://api.example.com/v1"
				bind:value={manualUrls}
				rows="4"
			/>
			<Helper class="text-xs">Enter additional URLs to scan manually, one per line</Helper>
		</div>

		<!-- URL Generation Options -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<div>
				<Label class="text-base font-medium mb-2">URL Schemes</Label>
				<div class="space-y-2">
					{#each urlScanService.getDefaultSchemes() as scheme}
						<div class="flex items-center space-x-2">
							<Checkbox 
								checked={scanConfig.schemes?.includes(scheme)}
								on:change={(e) => {
									if (e.target.checked) {
										scanConfig.schemes = [...(scanConfig.schemes || []), scheme];
									} else {
										scanConfig.schemes = (scanConfig.schemes || []).filter(s => s !== scheme);
									}
								}}
							/>
							<Label class="text-sm">{scheme}</Label>
						</div>
					{/each}
				</div>
			</div>

			<div>
				<Label class="text-base font-medium mb-2">Port Options</Label>
				<div class="space-y-3">
					<div class="flex items-center space-x-2">
						<Checkbox bind:checked={scanConfig.only_web_ports} />
						<Label class="text-sm">Only scan common web ports</Label>
					</div>
					<Helper class="text-xs">
						Limit scanning to common web ports: 80, 443, 8080, 8443, 8000, 8888, 9000, 9001, 3000, 5000
					</Helper>

					<div>
						<Label for="custom-ports" class="text-sm">Custom Ports (comma-separated)</Label>
						<Input 
							id="custom-ports"
							placeholder="80,443,8080,9000"
							bind:value={customPorts}
							disabled={scanConfig.only_web_ports}
						/>
						<Helper class="text-xs">
							Specify custom ports to scan (overrides web ports option)
						</Helper>
					</div>
				</div>
			</div>
		</div>

		<!-- Scan Options -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			<div class="space-y-3">
				<Label class="text-base font-medium">Performance Options</Label>
				
				<div>
					<Label for="threads">Threads</Label>
					<NumberInput id="threads" bind:value={scanConfig.threads} min="1" max="100" />
				</div>

				<div>
					<Label for="timeout">Timeout (seconds)</Label>
					<NumberInput id="timeout" bind:value={scanConfig.timeout} min="1" max="300" />
				</div>

				<div>
					<Label for="retries">Retries</Label>
					<NumberInput id="retries" bind:value={scanConfig.retries} min="0" max="10" />
				</div>
			</div>

			<div class="space-y-3">
				<Label class="text-base font-medium">Detection Options</Label>
				
				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.follow_redirects} />
					<Label class="text-sm">Follow redirects</Label>
				</div>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.tech_detection} />
					<Label class="text-sm">Enable technology detection</Label>
				</div>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.status_code} />
					<Label class="text-sm">Include status codes</Label>
				</div>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.content_length} />
					<Label class="text-sm">Include content length</Label>
				</div>

				<div class="flex items-center space-x-2">
					<Checkbox bind:checked={scanConfig.response_time} />
					<Label class="text-sm">Include response time</Label>
				</div>
			</div>
		</div>

		<!-- Execution Mode -->
		<div>
			<Label for="execution-mode" class="text-base font-medium mb-2">Execution Mode</Label>
			<Select id="execution-mode" bind:value={scanConfig.execution_mode}>
				{#each urlScanService.getExecutionModes() as mode}
					<option value={mode.value}>{mode.label}</option>
				{/each}
			</Select>
			<Helper class="text-xs">
				Local: Run httpx on this server. Cloud: Run httpx on a cloud instance (coming soon)
			</Helper>
		</div>
	</div>

	<div slot="footer" class="flex justify-end gap-2">
		<Button color="alternative" on:click={() => newScanModal = false} disabled={scanLoading}>
			Cancel
		</Button>
		<Button on:click={startURLScan} disabled={scanLoading}>
			{#if scanLoading}
				<Spinner class="mr-2" size="4" />
				Starting...
			{:else}
				<SearchOutline class="w-4 h-4 mr-2" />
				Start URL Scan
			{/if}
		</Button>
	</div>
</Modal> 
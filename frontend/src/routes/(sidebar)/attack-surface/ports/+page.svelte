<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge, Button, ButtonGroup, Card, Input, Table, TableBody, TableBodyCell, TableBodyRow, TableHead, TableHeadCell, Modal, Label, Select, Textarea, Toggle, Alert, Spinner, Progressbar } from 'flowbite-svelte';
	import { PlaySolid, DownloadSolid, SearchSolid, ServerSolid, ShieldCheckSolid, ExclamationCircleOutline, ClockSolid, ChartPieSolid } from 'flowbite-svelte-icons';
	import { PortScanService, type PortScanRequest, type Port, type PortScan, type PortStats } from '$lib/services/PortScanService';
	import { currentUser } from '$lib/stores/auth';
	import { pocketbase } from '$lib/stores/pocketbase';

	// Service instances
	const portScanService = new PortScanService();

	// State management
	let selectedClientId = '';
	let clients: Array<{ id: string; name: string }> = [];
	let loading = false;
	let error = '';

	// Data
	let ports: Port[] = [];
	let scans: PortScan[] = [];
	let stats: PortStats | null = null;

	// UI state
	let searchTerm = '';
	let selectedSource = '';
	let showScanModal = false;
	let activeTab = 'ports';

	// Scan configuration
	let scanConfig: PortScanRequest = {
		client_id: '',
		target_ips: [],
		include_domains: true,
		include_netblocks: true,
		ports: '',
		top_ports: '100',
		exclude_ports: '',
		scan_type: 'CONNECT',
		rate: 1000,
		threads: 25,
		timeout: 1000,
		retries: 3,
		host_discovery: false,
		exclude_cdn: true,
		verify: false,
		execution_mode: 'local',
		cloud_provider: '',
		nmap_integration: false,
		nmap_command: ''
	};

	// Form state
	let manualIpsText = '';
	let portPreset = '100';
	let customPorts = '';
	let isScanning = false;
	let scanProgress = 0;
	let activeScanId = '';
	let backgroundScanProgress = 0;
	let showBackgroundProgress = false;

	// Computed values
	$: filteredPorts = (ports || []).filter(port => {
		const matchesSearch = !searchTerm || 
			port.ip.toLowerCase().includes(searchTerm.toLowerCase()) ||
			port.port.toString().includes(searchTerm) ||
			(port.service && port.service.toLowerCase().includes(searchTerm.toLowerCase()));
		
		const matchesSource = !selectedSource || port.source === selectedSource;
		
		return matchesSearch && matchesSource;
	});

	$: uniqueSources = [...new Set((ports || []).map(p => p.source))];

	// Load initial data
	onMount(async () => {
		await loadClients();
		if ($currentUser?.client) {
			selectedClientId = $currentUser.client;
			await loadData();
		}
	});

	// Reactive statement to reload data when selectedClientId changes
	$: if (selectedClientId && typeof window !== 'undefined') {
		loadData();
	}

	// Load clients
	async function loadClients() {
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/collections/clients/records`, {
				headers: {
					'Content-Type': 'application/json',
					'Authorization': `Bearer ${token}`
				}
			});

			if (response.ok) {
				const data = await response.json();
				clients = data.items || [];
				
				// Set default selected client if user has one
				if ($currentUser?.client && clients.find(c => c.id === $currentUser.client)) {
					selectedClientId = $currentUser.client;
				} else if (clients.length > 0) {
					selectedClientId = clients[0].id;
				}
			}
		} catch (err) {
			console.error('Failed to load clients:', err);
		}
	}

	// Load all data for selected client
	async function loadData() {
		if (!selectedClientId) return;
		
		loading = true;
		error = '';

		try {
			await Promise.all([
				loadPorts(),
				loadScans(),
				loadStats()
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load data';
		} finally {
			loading = false;
		}
	}

	// Load ports
	async function loadPorts() {
		const response = await portScanService.getPorts(selectedClientId);
		if (response.success) {
			ports = response.ports || [];
		} else {
			ports = [];
		}
	}

	// Load scan history
	async function loadScans() {
		const response = await portScanService.getPortScans(selectedClientId);
		if (response.success) {
			scans = response.scans || [];
		} else {
			scans = [];
		}
	}

	// Load statistics
	async function loadStats() {
		const response = await portScanService.getPortStats(selectedClientId);
		if (response.success) {
			stats = response.stats;
		} else {
			stats = null;
		}
	}

	// Start port scan
	async function startPortScan() {
		if (!selectedClientId) {
			error = 'Please select a client';
			return;
		}

		// Prepare target IPs from manual input
		const targetIPs = manualIpsText
			.split('\n')
			.map(ip => ip.trim())
			.filter(ip => ip.length > 0);

		// Validate that at least one target source is selected
		if (targetIPs.length === 0 && !scanConfig.include_domains && !scanConfig.include_netblocks) {
			error = 'Please specify at least one target source';
			return;
		}

		// Set ports based on preset
		if (portPreset === 'custom') {
			const validation = portScanService.validatePorts(customPorts);
			if (!validation.valid) {
				error = validation.error || 'Invalid ports specified';
				return;
			}
			scanConfig.ports = customPorts;
			scanConfig.top_ports = '';
		} else {
			scanConfig.top_ports = portPreset;
			scanConfig.ports = '';
		}

		// Prepare scan request
		const request: PortScanRequest = {
			...scanConfig,
			client_id: selectedClientId,
			target_ips: targetIPs
		};

		isScanning = true;
		error = '';
		scanProgress = 0;

		try {
			const response = await portScanService.startPortScan(request);
			
			if (response.success && response.scan_id) {
				// Close modal immediately and start background scanning
				activeScanId = response.scan_id;
				showScanModal = false;
				showBackgroundProgress = true;
				backgroundScanProgress = 0;
				
				// Reset form
				manualIpsText = '';
				portPreset = '100';
				customPorts = '';
				isScanning = false;
				
				// Start background polling for progress
				const progressInterval = setInterval(async () => {
					try {
						const progressResponse = await portScanService.getScanProgress(activeScanId);
						if (progressResponse.success && progressResponse.progress) {
							backgroundScanProgress = progressResponse.progress.progress;
							
							if (progressResponse.progress.status === 'completed') {
								clearInterval(progressInterval);
								showBackgroundProgress = false;
								await loadData(); // Refresh data
								
								// Show success notification
								error = ''; // Clear any errors
								// You could add a toast notification here
								console.log('Port scan completed successfully!');
								
							} else if (progressResponse.progress.status === 'failed') {
								clearInterval(progressInterval);
								showBackgroundProgress = false;
								error = progressResponse.progress.error || 'Port scan failed';
							}
						}
					} catch (progressError) {
						console.error('Failed to get progress:', progressError);
					}
				}, 2000); // Poll every 2 seconds

				// Set a timeout to stop polling after 30 minutes
				setTimeout(() => {
					clearInterval(progressInterval);
					if (showBackgroundProgress) {
						showBackgroundProgress = false;
						error = 'Scan timed out';
					}
				}, 30 * 60 * 1000);
				
			} else {
				error = response.message || 'Failed to start port scan';
				isScanning = false;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Port scan failed';
			isScanning = false;
			scanProgress = 0;
		}
	}

	// Export ports to CSV
	function exportPorts() {
		const csv = portScanService.exportPortsToCSV(filteredPorts);
		const blob = new Blob([csv], { type: 'text/csv' });
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `ports_${selectedClientId}_${new Date().toISOString().split('T')[0]}.csv`;
		a.click();
		window.URL.revokeObjectURL(url);
	}

	// Get badge color for port state
	function getStateBadgeColor(state: string): string {
		switch (state.toLowerCase()) {
			case 'open': return 'green';
			case 'closed': return 'red';
			case 'filtered': return 'yellow';
			default: return 'gray';
		}
	}

	// Get badge color for source
	function getSourceBadgeColor(source: string): string {
		switch (source.toLowerCase()) {
			case 'domains': return 'blue';
			case 'netblocks': return 'purple';
			case 'manual': return 'gray';
			default: return 'gray';
		}
	}

	// Get badge color for execution mode
	function getExecutionModeBadgeColor(mode: string): string {
		switch (mode.toLowerCase()) {
			case 'local': return 'blue';
			case 'cloud': return 'green';
			default: return 'gray';
		}
	}

	// Format duration
	function formatDuration(duration: string): string {
		if (!duration) return 'N/A';
		// Convert duration like "2m30.5s" to a more readable format
		return duration.replace(/(\d+)m/g, '$1min ').replace(/(\d+\.?\d*)s/g, '$1sec');
	}

	// Handle client selection change
	async function handleClientChange() {
		if (selectedClientId) {
			await loadData();
		}
	}
</script>

<div class="p-0 max-w-none w-full min-w-0">
	<div class="px-6 py-4">
			
			<!-- Header -->
			<header class="mb-4 lg:mb-6 not-format">
				<div class="flex items-center justify-between">
					<div>
						<h1 class="mb-4 text-3xl font-extrabold leading-tight text-gray-900 lg:mb-6 lg:text-4xl dark:text-white">
							Port Scanning
							{#if showBackgroundProgress}
								<span class="ml-3 inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800">
									<svg class="animate-spin -ml-1 mr-2 h-4 w-4 text-blue-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
										<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
										<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
									</svg>
									Scanning... {backgroundScanProgress}%
								</span>
							{/if}
						</h1>
						<p class="text-gray-500 dark:text-gray-400">
							Discover open ports on your attack surface using naabu
							{#if showBackgroundProgress}
								<br><span class="text-blue-600 font-medium">Port scan running in background - you can continue using the site</span>
							{/if}
						</p>
					</div>
					<div class="flex items-center space-x-3">
						<div class="flex items-center space-x-2">
							<Label for="client-select" class="text-sm font-medium">Client:</Label>
							<Select id="client-select" bind:value={selectedClientId} on:change={handleClientChange} class="w-48">
								<option value="">Select a client</option>
								{#each clients as client}
									<option value={client.id}>{client.name}</option>
								{/each}
							</Select>
						</div>
						<Button on:click={() => showScanModal = true} disabled={!selectedClientId || isScanning}>
							<PlaySolid class="w-4 h-4 mr-2" />
							Start Scan
						</Button>
					</div>
				</div>
			</header>

			<!-- Error Alert -->
			{#if error}
				<Alert color="red" class="mb-4">
					<ExclamationCircleOutline slot="icon" class="w-4 h-4" />
					{error}
				</Alert>
			{/if}

			<!-- Statistics Cards -->
			{#if stats && selectedClientId}
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
					<Card>
						<div class="flex items-center">
							<ShieldCheckSolid class="w-8 h-8 text-green-500 mr-3" />
							<div>
								<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_open_ports}</p>
								<p class="text-sm text-gray-500 dark:text-gray-400">Open Ports</p>
							</div>
						</div>
					</Card>
					<Card>
						<div class="flex items-center">
							<ServerSolid class="w-8 h-8 text-blue-500 mr-3" />
							<div>
								<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.unique_hosts}</p>
								<p class="text-sm text-gray-500 dark:text-gray-400">Unique Hosts</p>
							</div>
						</div>
					</Card>
					<Card>
						<div class="flex items-center">
							<SearchSolid class="w-8 h-8 text-purple-500 mr-3" />
							<div>
								<p class="text-2xl font-bold text-gray-900 dark:text-white">{stats.total_scans}</p>
								<p class="text-sm text-gray-500 dark:text-gray-400">Total Scans</p>
							</div>
						</div>
					</Card>
					<Card>
						<div class="flex items-center">
							<ChartPieSolid class="w-8 h-8 text-orange-500 mr-3" />
							<div>
								<p class="text-2xl font-bold text-gray-900 dark:text-white">
									{(stats.top_ports && stats.top_ports.length > 0) ? stats.top_ports[0].port : 'N/A'}
								</p>
								<p class="text-sm text-gray-500 dark:text-gray-400">Top Port</p>
							</div>
						</div>
					</Card>
				</div>
			{/if}

			<!-- Tabs -->
			<div class="border-b border-gray-200 dark:border-gray-700 mb-6">
				<ul class="flex flex-wrap -mb-px text-sm font-medium text-center">
					<li class="mr-2">
						<button
							class="px-6 py-3 text-sm font-medium border-b-2 {activeTab === 'ports' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
							on:click={() => { activeTab = 'ports'; }}
						>
							Open Ports
						</button>
					</li>
					<li class="mr-2">
						<button
							class="px-6 py-3 text-sm font-medium border-b-2 {activeTab === 'scans' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'}"
							on:click={() => { activeTab = 'scans'; }}
						>
							Scan History
						</button>
					</li>
				</ul>
			</div>



	</div>

	<!-- Ports Table - FULL WIDTH -->
	{#if activeTab === 'ports'}
		<div class="w-full min-h-screen bg-white dark:bg-gray-900">
			<div class="px-6 py-8">
				<div class="flex justify-between items-center mb-6">
					<h2 class="text-2xl font-bold text-gray-900 dark:text-white">Open Ports</h2>
					<div class="flex gap-3">
						<Button color="alternative" size="sm" on:click={exportPorts} disabled={filteredPorts.length === 0}>
							<DownloadSolid class="w-4 h-4 mr-2" />
							Export CSV
						</Button>
						<span class="text-sm text-gray-500 dark:text-gray-400">
							{filteredPorts.length} of {ports.length} ports
						</span>
					</div>
				</div>

				<!-- Filters and Search -->
				<div class="grid grid-cols-1 xl:grid-cols-4 gap-8 mb-10 p-8 bg-gray-50 dark:bg-gray-800 rounded-xl shadow-lg">
					<!-- Search -->
					<div>
						<Label for="search" class="text-base mb-3 block font-semibold">Search Ports</Label>
						<Input 
							id="search"
							type="text" 
							placeholder="Search IP, port, or service..." 
							bind:value={searchTerm}
							class="w-full h-12 text-base"
						/>
					</div>

					<!-- Source Filter -->
					<div>
						<Label for="source-filter" class="text-base mb-3 block font-semibold">Source</Label>
						<select 
							id="source-filter"
							bind:value={selectedSource}
							class="w-full h-12 px-4 py-3 text-base border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 dark:bg-gray-700 dark:border-gray-600"
						>
							<option value="">All Sources</option>
							{#each uniqueSources as source}
								<option value={source}>{source}</option>
							{/each}
						</select>
					</div>

					<div></div> <!-- Empty div for spacing -->
					<div></div> <!-- Empty div for spacing -->
				</div>

				<!-- Results Summary -->
				<div class="flex justify-between items-center mb-8 px-2">
					<div class="text-lg text-gray-600 dark:text-gray-400">
						Showing {filteredPorts.length} results
						{#if filteredPorts.length !== (ports || []).length}
							(filtered from {(ports || []).length} total)
						{/if}
					</div>
				</div>

				{#if loading}
					<div class="flex justify-center py-12">
						<Spinner size="8" />
					</div>
				{:else if filteredPorts.length === 0}
					<div class="text-center py-12 text-gray-500">
						{#if ports.length === 0}
							No ports found. Start a port scan to discover open ports.
						{:else}
							No ports match your current filters.
						{/if}
					</div>
				{:else}
					<!-- Table -->
					<div class="w-full overflow-x-auto bg-white dark:bg-gray-800 rounded-lg shadow-sm">
						<table class="w-full text-base text-left text-gray-500 dark:text-gray-400">
							<thead class="text-sm text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400 border-b-2 border-gray-200 dark:border-gray-600">
								<tr>
									<th scope="col" class="px-8 py-6 w-1/4">IP Address</th>
									<th scope="col" class="px-8 py-6 w-1/8">Port</th>
									<th scope="col" class="px-8 py-6 w-1/4">Service</th>
									<th scope="col" class="px-8 py-6 w-1/8">State</th>
									<th scope="col" class="px-8 py-6 w-1/8">Source</th>
									<th scope="col" class="px-8 py-6 w-1/4">Discovered</th>
								</tr>
							</thead>
							<tbody>
								{#each filteredPorts as port}
									<tr class="bg-white border-b dark:bg-gray-800 dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600 transition-colors">
										<td class="px-8 py-6">
											<div class="flex items-center gap-4">
												<span class="font-mono text-base text-gray-900 dark:text-white">{port.ip}</span>
												<button 
													class="text-xs text-gray-500 hover:text-blue-600 px-3 py-1 rounded border border-gray-300 hover:border-blue-500 transition-colors flex-shrink-0"
													on:click={() => navigator.clipboard.writeText(port.ip)}
													title="Copy to clipboard"
												>
													copy
												</button>
											</div>
										</td>
										<td class="px-8 py-6 text-gray-900 dark:text-white text-base font-mono">
											{port.port}
										</td>
										<td class="px-8 py-6">
											{#if port.service}
												<span class="font-medium text-base">{port.service}</span>
												{#if port.protocol}
													<span class="text-gray-500">/{port.protocol}</span>
												{/if}
											{:else}
												<span class="text-gray-400 text-base">Unknown</span>
											{/if}
										</td>
										<td class="px-8 py-6">
											<Badge color={getStateBadgeColor(port.state)} class="text-sm px-3 py-1">
												{port.state}
											</Badge>
										</td>
										<td class="px-8 py-6">
											<Badge color={getSourceBadgeColor(port.source)} class="text-sm px-3 py-1">
												{port.source}
											</Badge>
										</td>
										<td class="px-8 py-6">
											<span class="text-gray-400 text-base">
												{new Date(port.discovered_at).toLocaleString()}
											</span>
										</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Port Scan Configuration Modal -->
<Modal bind:open={showScanModal} size="lg" title="Configure Port Scan" class="w-full">
	<form on:submit|preventDefault={startPortScan} class="space-y-6">
		<!-- Target Configuration -->
		<div>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Target Configuration</h3>
			
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
				<div>
					<Label class="mb-2">
						<Toggle bind:checked={scanConfig.include_domains} />
						Include Domain IPs
					</Label>
					<p class="text-xs text-gray-500">Include IPs resolved from discovered domains</p>
				</div>
				<div>
					<Label class="mb-2">
						<Toggle bind:checked={scanConfig.include_netblocks} />
						Include Netblock IPs
					</Label>
					<p class="text-xs text-gray-500">Include IPs from discovered netblocks</p>
				</div>
			</div>

			<div>
				<Label for="manual-ips" class="mb-2">Manual IP Addresses (optional)</Label>
				<Textarea 
					id="manual-ips" 
					bind:value={manualIpsText}
					placeholder="192.168.1.1&#10;10.0.0.0/24&#10;example.com"
					rows="4"
					class="text-sm font-mono"
				/>
				<p class="text-xs text-gray-500 mt-1">One IP, CIDR, or hostname per line</p>
			</div>
		</div>

		<!-- Port Configuration -->
		<div>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Port Configuration</h3>
			
			<div class="mb-4">
				<Label for="port-preset" class="mb-2">Port Preset</Label>
				<Select id="port-preset" bind:value={portPreset}>
					{#each portScanService.getPortPresets() as preset}
						<option value={preset.value}>{preset.label} - {preset.description}</option>
					{/each}
				</Select>
			</div>

			{#if portPreset === 'custom'}
				<div class="mb-4">
					<Label for="custom-ports" class="mb-2">Custom Ports</Label>
					<Input 
						id="custom-ports" 
						bind:value={customPorts}
						placeholder="80,443,8080-8090"
						class="font-mono"
					/>
					<p class="text-xs text-gray-500 mt-1">Format: 80,443,8080-8090</p>
				</div>
			{/if}

			<div>
				<Label for="exclude-ports" class="mb-2">Exclude Ports (optional)</Label>
				<Input 
					id="exclude-ports" 
					bind:value={scanConfig.exclude_ports}
					placeholder="22,3389"
					class="font-mono"
				/>
			</div>
		</div>

		<!-- Scan Options -->
		<div>
			<h3 class="text-lg font-medium text-gray-900 dark:text-white mb-4">Scan Options</h3>
			
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
				<div>
					<Label for="scan-type" class="mb-2">Scan Type</Label>
					<Select id="scan-type" bind:value={scanConfig.scan_type}>
						{#each portScanService.getScanTypes() as scanType}
							<option value={scanType.value}>
								{scanType.label}
								{scanType.requiresRoot ? ' (requires root)' : ''}
							</option>
						{/each}
					</Select>
				</div>
				<div>
					<Label for="execution-mode" class="mb-2">Execution Mode</Label>
					<Select id="execution-mode" bind:value={scanConfig.execution_mode}>
						{#each portScanService.getExecutionModes() as mode}
							<option value={mode.value}>
								{mode.label}
								{mode.recommended ? ' (recommended)' : ''}
							</option>
						{/each}
					</Select>
				</div>
			</div>

			{#if scanConfig.execution_mode === 'cloud'}
				<div class="mb-4">
					<Label for="cloud-provider" class="mb-2">Cloud Provider</Label>
					<Select id="cloud-provider" bind:value={scanConfig.cloud_provider}>
						<option value="">Select provider</option>
						{#each portScanService.getCloudProviders() as provider}
							<option value={provider.value}>{provider.label}</option>
						{/each}
					</Select>
				</div>
			{/if}

			<div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
				<div>
					<Label for="rate" class="mb-2">Rate (packets/sec)</Label>
					<Input 
						id="rate" 
						type="number" 
						bind:value={scanConfig.rate}
						min="1"
						max="10000"
					/>
				</div>
				<div>
					<Label for="threads" class="mb-2">Threads</Label>
					<Input 
						id="threads" 
						type="number" 
						bind:value={scanConfig.threads}
						min="1"
						max="100"
					/>
				</div>
				<div>
					<Label for="timeout" class="mb-2">Timeout (ms)</Label>
					<Input 
						id="timeout" 
						type="number" 
						bind:value={scanConfig.timeout}
						min="100"
						max="10000"
					/>
				</div>
			</div>

			<div class="grid grid-cols-1 md:grid-cols-3 gap-4">
				<div>
					<Label class="mb-2">
						<Toggle bind:checked={scanConfig.host_discovery} />
						Host Discovery
					</Label>
				</div>
				<div>
					<Label class="mb-2">
						<Toggle bind:checked={scanConfig.exclude_cdn} />
						Exclude CDN/WAF
					</Label>
				</div>
				<div>
					<Label class="mb-2">
						<Toggle bind:checked={scanConfig.verify} />
						Verify Results
					</Label>
				</div>
			</div>
		</div>

		<!-- Nmap Integration -->
		<div>
			<Label class="mb-2">
				<Toggle bind:checked={scanConfig.nmap_integration} />
				Enable Nmap Integration
			</Label>
			<p class="text-xs text-gray-500 mb-2">Run nmap on discovered ports for service detection</p>
			
			{#if scanConfig.nmap_integration}
				<div>
					<Label for="nmap-command" class="mb-2">Nmap Command</Label>
					<Input 
						id="nmap-command" 
						bind:value={scanConfig.nmap_command}
						placeholder="nmap -sV -sC"
						class="font-mono"
					/>
					<p class="text-xs text-gray-500 mt-1">Custom nmap command (optional)</p>
				</div>
			{/if}
		</div>

		{#if isScanning}
			<div class="space-y-3">
				<div class="flex items-center justify-between">
					<span class="text-sm font-medium text-gray-700 dark:text-gray-300">Starting scan...</span>
					<span class="text-sm text-gray-500">Please wait</span>
				</div>
				<Progressbar progress={20} />
			</div>
		{/if}

		<div class="flex justify-end space-x-3 pt-4 border-t">
			<Button color="alternative" on:click={() => showScanModal = false} disabled={isScanning}>
				Cancel
			</Button>
			<Button type="submit" disabled={isScanning || !selectedClientId}>
				{#if isScanning}
					<Spinner class="w-4 h-4 mr-2" />
					Scanning...
				{:else}
					<PlaySolid class="w-4 h-4 mr-2" />
					Start Scan
				{/if}
			</Button>
		</div>
	</form>
</Modal> 
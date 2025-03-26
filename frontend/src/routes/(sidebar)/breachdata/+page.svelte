<!-- frontend/src/routes/(sidebar)/breachdata/+page.svelte -->
<script lang="ts">
    import { page } from '$app/stores';
    import { 
        Heading, P, Table, TableBody, TableBodyCell, TableBodyRow, 
        TableHead, TableHeadCell, Card, Button, Badge, Spinner,
        Select, Input, Label, Alert, Tabs, TabItem, Accordion, AccordionItem
    } from 'flowbite-svelte';
    import { 
        ShieldCheckSolid, 
        ExclamationCircleSolid, 
        SearchSolid, 
        DesktopPcSolid, 
        CreditCardSolid, 
        FingerprintOutline
    } from 'flowbite-svelte-icons';
    import { enhance } from '$app/forms';
    
    export let data;
    export let form;
    
    let loading = false;
    
    // Define the type for breach data
    type BreachData = {
        id: number;
        client: string;
        email: string;
        breachDate: string;
        source: string;
        severity: string;
        details: string;
    };

    // Initialize with proper type
    let breachData: BreachData[] = data.breachData || [];
    
    // Mock clients data - in real implementation, this would come from the client list endpoint
    let clients = [
        { id: 1, name: 'Acme Corporation' },
        { id: 2, name: 'TechStart Inc.' },
        { id: 3, name: 'Globex Industries' },
        { id: 4, name: 'Example Company' }
    ];
    
    let selectedClient = '';
    let manualDomain = '';
    let scanInProgress = false;
    let showSuccessAlert = false;
    let alertMessage = '';
    let activeTab = 0;
    
    // Mock stealer logs data - this would be replaced with actual API data
    type StealerLog = {
        id: string;
        timestamp: string;
        ipAddress: string;
        country: string;
        browser: string;
        os: string;
        deviceType: string;
        sessionData: {
            cookies?: { domain: string; name: string; value: string }[];
            credentials?: { url: string; username: string; password: string }[];
            cards?: { type: string; number: string; expiry: string; name: string }[];
        };
    };
    
    let stealerLogs: StealerLog[] = [
        {
            id: '1',
            timestamp: '2023-11-15T14:32:45Z',
            ipAddress: '192.168.1.1',
            country: 'United States',
            browser: 'Chrome 98.0',
            os: 'Windows 10',
            deviceType: 'Desktop',
            sessionData: {
                cookies: [
                    { domain: 'example.com', name: 'session_id', value: 'abc123xyz' },
                    { domain: 'mail.example.com', name: 'auth', value: 'token456' }
                ],
                credentials: [
                    { url: 'https://example.com/login', username: 'user@example.com', password: '********' }
                ],
                cards: [
                    { type: 'Visa', number: '**** **** **** 1234', expiry: '05/25', name: 'John Doe' }
                ]
            }
        },
        {
            id: '2',
            timestamp: '2023-11-14T09:15:22Z',
            ipAddress: '10.0.0.15',
            country: 'Canada',
            browser: 'Firefox 97.0',
            os: 'macOS 12.2',
            deviceType: 'Desktop',
            sessionData: {
                cookies: [
                    { domain: 'store.example.com', name: 'cart_id', value: 'cart789' }
                ],
                credentials: [
                    { url: 'https://store.example.com', username: 'customer123', password: '********' }
                ]
            }
        },
        {
            id: '3',
            timestamp: '2023-11-13T18:45:11Z',
            ipAddress: '172.16.0.10',
            country: 'United Kingdom',
            browser: 'Safari 15.2',
            os: 'iOS 15.3',
            deviceType: 'Mobile',
            sessionData: {
                cookies: [
                    { domain: 'api.example.com', name: 'api_token', value: 'tkn987654' }
                ],
                cards: [
                    { type: 'Mastercard', number: '**** **** **** 5678', expiry: '12/24', name: 'Jane Smith' }
                ]
            }
        }
    ];
    
    let selectedLog: StealerLog | null = null;
    let showLogDetails = false;
    
    function viewLogDetails(log: StealerLog) {
        selectedLog = log;
        showLogDetails = true;
    }
    
    // This would be replaced with actual data from flare.io in the future
    let mockBreachData: BreachData[] = [
        {
            id: 1,
            client: 'Acme Corporation',
            email: 'admin@acmecorp.com',
            breachDate: '2023-06-15',
            source: 'LinkedIn',
            severity: 'High',
            details: 'Password and personal information exposed'
        },
        {
            id: 2,
            client: 'TechStart Inc.',
            email: 'support@techstart.io',
            breachDate: '2023-08-22',
            source: 'Adobe',
            severity: 'Medium',
            details: 'Email and hashed passwords exposed'
        },
        {
            id: 3,
            client: 'Globex Industries',
            email: 'info@globex.com',
            breachDate: '2023-09-10',
            source: 'Dropbox',
            severity: 'Low',
            details: 'Email addresses exposed'
        }
    ];
    
    // Simulate data loading
    breachData = mockBreachData;
    
    function getSeverityBadge(severity: string): 'red' | 'yellow' | 'blue' | 'dark' {
        switch(severity.toLowerCase()) {
            case 'high':
                return 'red';
            case 'medium':
                return 'yellow';
            case 'low':
                return 'blue';
            default:
                return 'dark';
        }
    }
    
    // Handle form submission success
    $: if (form?.success) {
        scanInProgress = false;
        showSuccessAlert = true;
        alertMessage = form.message || 'Operation completed successfully';
        
        // Hide the alert after 5 seconds
        setTimeout(() => {
            showSuccessAlert = false;
        }, 5000);
    }
    
    // Format date for display
    function formatDate(dateString: string): string {
        const date = new Date(dateString);
        return date.toLocaleString();
    }
</script>

<!-- Main container that fits within the sidebar layout -->
<div>
    <!-- Page header with title and description -->
    <Card size="xl" class="shadow-sm max-w-none mb-6">
        <div class="mb-4">
            <Heading tag="h3" class="text-xl font-semibold dark:text-white">
                Breach Data
            </Heading>
            <P class="text-gray-500 dark:text-gray-400 mt-2">
                View breach data from flare.io for your clients. This section provides information about potential data breaches.
            </P>
        </div>

        {#if showSuccessAlert}
            <Alert color="green" class="mb-4" dismissable>
                <span class="font-medium">Success!</span> {alertMessage}
            </Alert>
        {/if}
        
        {#if form?.error}
            <Alert color="red" class="mb-4" dismissable>
                <span class="font-medium">Error:</span> {form.error}
            </Alert>
        {/if}
    </Card>
    
    <!-- Scan Options Card -->
    <Card size="xl" class="shadow-sm max-w-none mb-6">
        <Heading tag="h3" class="text-xl font-semibold dark:text-white mb-4">
            Run Breach Scan
        </Heading>
        
        <Tabs style="underline" activeClasses="text-primary-600 border-primary-600 dark:text-primary-500 dark:border-primary-500">
            <TabItem open={activeTab === 0} on:click={() => activeTab = 0} title="Select Client">
                <div class="p-4">
                    <form method="POST" action="?/scanClient" use:enhance={() => {
                        scanInProgress = true;
                        
                        return async ({ result }) => {
                            scanInProgress = false;
                            if (result.type === 'success') {
                                selectedClient = '';
                            }
                        };
                    }}>
                        <div class="mb-4">
                            <Label for="client" class="mb-2 text-base font-medium">Select a client to scan for breached credentials</Label>
                            <Select id="client" name="clientId" bind:value={selectedClient} class="w-full lg:w-1/2">
                                <option value="" selected disabled>Select a client</option>
                                {#each clients as client}
                                    <option value={client.id}>{client.name}</option>
                                {/each}
                            </Select>
                        </div>
                        
                        <Button type="submit" color="primary" class="mt-2" size="lg" disabled={!selectedClient || scanInProgress}>
                            {#if scanInProgress}
                                <Spinner class="mr-3" size="5" color="white" />
                            {:else}
                                <ShieldCheckSolid class="mr-2 h-5 w-5" />
                            {/if}
                            Scan Selected Client
                        </Button>
                    </form>
                </div>
            </TabItem>
            
            <TabItem open={activeTab === 1} on:click={() => activeTab = 1} title="Enter Domain">
                <div class="p-4">
                    <form method="POST" action="?/scanDomain" use:enhance={() => {
                        scanInProgress = true;
                        
                        return async ({ result }) => {
                            scanInProgress = false;
                            if (result.type === 'success') {
                                manualDomain = '';
                            }
                        };
                    }}>
                        <div class="mb-4">
                            <Label for="manual-domain" class="mb-2 text-base font-medium">Manually enter a domain to scan for breached credentials</Label>
                            <div class="flex w-full lg:w-1/2">
                                <Input id="manual-domain" name="domain" placeholder="example.com" bind:value={manualDomain} />
                            </div>
                        </div>
                        
                        <Button type="submit" color="primary" class="mt-2" size="lg" disabled={!manualDomain || scanInProgress}>
                            {#if scanInProgress}
                                <Spinner class="mr-3" size="5" color="white" />
                            {:else}
                                <SearchSolid class="mr-2 h-5 w-5" />
                            {/if}
                            Scan Domain
                        </Button>
                    </form>
                </div>
            </TabItem>
            
            <TabItem open={activeTab === 2} on:click={() => activeTab = 2} title="Stealer Logs">
                <div class="p-4">
                    <div class="mb-4">
                        <P>View session data from stealer logs including cookies, saved credentials, and payment information.</P>
                    </div>
                    
                    <div class="overflow-x-auto">
                        <Table striped={true} class="w-full text-left">
                            <TableHead class="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                                <TableHeadCell class="px-6 py-3">Timestamp</TableHeadCell>
                                <TableHeadCell class="px-6 py-3">IP Address</TableHeadCell>
                                <TableHeadCell class="px-6 py-3">Location</TableHeadCell>
                                <TableHeadCell class="px-6 py-3">Browser</TableHeadCell>
                                <TableHeadCell class="px-6 py-3">OS</TableHeadCell>
                                <TableHeadCell class="px-6 py-3">Actions</TableHeadCell>
                            </TableHead>
                            <TableBody>
                                {#each stealerLogs as log}
                                    <TableBodyRow class="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                                        <TableBodyCell class="px-6 py-4 font-medium text-gray-900 whitespace-nowrap dark:text-white">
                                            {formatDate(log.timestamp)}
                                        </TableBodyCell>
                                        <TableBodyCell class="px-6 py-4">
                                            {log.ipAddress}
                                        </TableBodyCell>
                                        <TableBodyCell class="px-6 py-4">
                                            {log.country}
                                        </TableBodyCell>
                                        <TableBodyCell class="px-6 py-4">
                                            {log.browser}
                                        </TableBodyCell>
                                        <TableBodyCell class="px-6 py-4">
                                            {log.os}
                                        </TableBodyCell>
                                        <TableBodyCell class="px-6 py-4">
                                            <Button size="xs" color="primary" on:click={() => viewLogDetails(log)}>
                                                View Details
                                            </Button>
                                        </TableBodyCell>
                                    </TableBodyRow>
                                {/each}
                            </TableBody>
                        </Table>
                    </div>
                    
                    <!-- Log Details Modal -->
                    {#if showLogDetails && selectedLog}
                        <div class="fixed inset-0 bg-gray-900 bg-opacity-50 z-40 flex items-center justify-center p-4">
                            <Card size="xl" class="w-full max-w-4xl max-h-[90vh] overflow-y-auto">
                                <div class="flex justify-between items-center mb-4">
                                    <Heading tag="h4" class="text-lg font-semibold">
                                        Stealer Log Details
                                    </Heading>
                                    <Button color="alternative" size="xs" on:click={() => showLogDetails = false}>
                                        ✕
                                    </Button>
                                </div>
                                
                                <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
                                    <div>
                                        <p class="text-sm text-gray-500">Timestamp</p>
                                        <p class="font-medium">{formatDate(selectedLog.timestamp)}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm text-gray-500">IP Address</p>
                                        <p class="font-medium">{selectedLog.ipAddress}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm text-gray-500">Location</p>
                                        <p class="font-medium">{selectedLog.country}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm text-gray-500">Browser</p>
                                        <p class="font-medium">{selectedLog.browser}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm text-gray-500">Operating System</p>
                                        <p class="font-medium">{selectedLog.os}</p>
                                    </div>
                                    <div>
                                        <p class="text-sm text-gray-500">Device Type</p>
                                        <p class="font-medium">{selectedLog.deviceType}</p>
                                    </div>
                                </div>
                                
                                <Accordion>
                                    {#if selectedLog.sessionData.cookies && selectedLog.sessionData.cookies.length > 0}
                                        <AccordionItem>
                                            <span slot="header" class="flex items-center">
                                                <FingerprintOutline class="mr-2 h-5 w-5 text-gray-500" />
                                                Cookies ({selectedLog.sessionData.cookies.length})
                                            </span>
                                            <div class="overflow-x-auto">
                                                <Table striped={true} class="w-full text-left text-sm">
                                                    <TableHead class="bg-gray-50 dark:bg-gray-700">
                                                        <TableHeadCell>Domain</TableHeadCell>
                                                        <TableHeadCell>Name</TableHeadCell>
                                                        <TableHeadCell>Value</TableHeadCell>
                                                    </TableHead>
                                                    <TableBody>
                                                        {#each selectedLog.sessionData.cookies as cookie}
                                                            <TableBodyRow>
                                                                <TableBodyCell class="font-medium">{cookie.domain}</TableBodyCell>
                                                                <TableBodyCell>{cookie.name}</TableBodyCell>
                                                                <TableBodyCell class="truncate max-w-xs">{cookie.value}</TableBodyCell>
                                                            </TableBodyRow>
                                                        {/each}
                                                    </TableBody>
                                                </Table>
                                            </div>
                                        </AccordionItem>
                                    {/if}
                                    
                                    {#if selectedLog.sessionData.credentials && selectedLog.sessionData.credentials.length > 0}
                                        <AccordionItem>
                                            <span slot="header" class="flex items-center">
                                                <DesktopPcSolid class="mr-2 h-5 w-5 text-gray-500" />
                                                Saved Credentials ({selectedLog.sessionData.credentials.length})
                                            </span>
                                            <div class="overflow-x-auto">
                                                <Table striped={true} class="w-full text-left text-sm">
                                                    <TableHead class="bg-gray-50 dark:bg-gray-700">
                                                        <TableHeadCell>Website</TableHeadCell>
                                                        <TableHeadCell>Username</TableHeadCell>
                                                        <TableHeadCell>Password</TableHeadCell>
                                                    </TableHead>
                                                    <TableBody>
                                                        {#each selectedLog.sessionData.credentials as credential}
                                                            <TableBodyRow>
                                                                <TableBodyCell class="font-medium">{credential.url}</TableBodyCell>
                                                                <TableBodyCell>{credential.username}</TableBodyCell>
                                                                <TableBodyCell>{credential.password}</TableBodyCell>
                                                            </TableBodyRow>
                                                        {/each}
                                                    </TableBody>
                                                </Table>
                                            </div>
                                        </AccordionItem>
                                    {/if}
                                    
                                    {#if selectedLog.sessionData.cards && selectedLog.sessionData.cards.length > 0}
                                        <AccordionItem>
                                            <span slot="header" class="flex items-center">
                                                <CreditCardSolid class="mr-2 h-5 w-5 text-gray-500" />
                                                Payment Information ({selectedLog.sessionData.cards.length})
                                            </span>
                                            <div class="overflow-x-auto">
                                                <Table striped={true} class="w-full text-left text-sm">
                                                    <TableHead class="bg-gray-50 dark:bg-gray-700">
                                                        <TableHeadCell>Card Type</TableHeadCell>
                                                        <TableHeadCell>Number</TableHeadCell>
                                                        <TableHeadCell>Expiry</TableHeadCell>
                                                        <TableHeadCell>Name</TableHeadCell>
                                                    </TableHead>
                                                    <TableBody>
                                                        {#each selectedLog.sessionData.cards as card}
                                                            <TableBodyRow>
                                                                <TableBodyCell class="font-medium">{card.type}</TableBodyCell>
                                                                <TableBodyCell>{card.number}</TableBodyCell>
                                                                <TableBodyCell>{card.expiry}</TableBodyCell>
                                                                <TableBodyCell>{card.name}</TableBodyCell>
                                                            </TableBodyRow>
                                                        {/each}
                                                    </TableBody>
                                                </Table>
                                            </div>
                                        </AccordionItem>
                                    {/if}
                                </Accordion>
                            </Card>
                        </div>
                    {/if}
                </div>
            </TabItem>
        </Tabs>
    </Card>
    
    <!-- Results Card -->
    <Card size="xl" class="shadow-sm max-w-none">
        <Heading tag="h3" class="text-xl font-semibold dark:text-white mb-4">
            Breach Data Results
        </Heading>
        
        {#if loading}
            <div class="flex justify-center items-center p-8">
                <Spinner size="8" />
                <p class="ml-4 text-lg">Loading breach data...</p>
            </div>
        {:else if breachData.length === 0}
            <div class="text-center p-8">
                <ExclamationCircleSolid class="mx-auto mb-4 h-16 w-16 text-gray-400" />
                <p class="text-gray-500 dark:text-gray-400 text-lg">No breach data available at this time.</p>
            </div>
        {:else}
            <div class="overflow-x-auto">
                <Table striped={true} class="w-full text-left">
                    <TableHead class="text-xs text-gray-700 uppercase bg-gray-50 dark:bg-gray-700 dark:text-gray-400">
                        <TableHeadCell class="px-6 py-3">Client</TableHeadCell>
                        <TableHeadCell class="px-6 py-3">Email</TableHeadCell>
                        <TableHeadCell class="px-6 py-3">Breach Date</TableHeadCell>
                        <TableHeadCell class="px-6 py-3">Source</TableHeadCell>
                        <TableHeadCell class="px-6 py-3">Severity</TableHeadCell>
                        <TableHeadCell class="px-6 py-3">Details</TableHeadCell>
                    </TableHead>
                    <TableBody>
                        {#each breachData as item}
                            <TableBodyRow class="bg-white border-b dark:bg-gray-800 dark:border-gray-700">
                                <TableBodyCell class="px-6 py-4 font-medium text-gray-900 whitespace-nowrap dark:text-white">
                                    {item.client}
                                </TableBodyCell>
                                <TableBodyCell class="px-6 py-4">
                                    {item.email}
                                </TableBodyCell>
                                <TableBodyCell class="px-6 py-4">
                                    {new Date(item.breachDate).toLocaleDateString()}
                                </TableBodyCell>
                                <TableBodyCell class="px-6 py-4">
                                    {item.source}
                                </TableBodyCell>
                                <TableBodyCell class="px-6 py-4">
                                    <Badge color={getSeverityBadge(item.severity)} class="px-2.5 py-0.5 text-xs font-semibold">
                                        {item.severity}
                                    </Badge>
                                </TableBodyCell>
                                <TableBodyCell class="px-6 py-4">
                                    {item.details}
                                </TableBodyCell>
                            </TableBodyRow>
                        {/each}
                    </TableBody>
                </Table>
            </div>
        {/if}
    </Card>
</div> 
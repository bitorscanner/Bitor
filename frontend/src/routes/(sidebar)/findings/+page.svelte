<script lang="ts">
    import { onMount } from 'svelte';
    import { pocketbase } from '$lib/stores/pocketbase';
    import {
      Card,
      Heading,
      Button,
      Accordion,
      AccordionItem,
      Skeleton,
      Checkbox,
      MultiSelect,
      Input,
      Select,
      Label,
    } from 'flowbite-svelte';
    import FindingModal from './FindingModal.svelte';
    import BulkCommentModal from './BulkCommentModal.svelte';
  
    interface Finding {
      id: string;
      info: {
        name: string;
        tags: string[];
        description?: string;
        reference?: string[];
      };
      host: string;
      ip: string;
      template_id: string;
      severity: string;
      timestamp: string;
      last_seen: string;
      acknowledged: boolean;
      false_positive: boolean;
      remediated: boolean;
      notes?: string;
      client: {
        id: string;
        name: string;
        favicon?: string;
      } | null;
      scan: {
        name: string;
        id: string;
      } | null;
      severity_order: number;
      info_name: string;
      client_name: string;
      scan_name: string;
      request?: string;
      response?: string;
      selected?: boolean;
      matched_at?: string;
      extracted_results?: string[];
      severity_override?: string;
    }
  
    interface GroupedFindings {
      template_id: string;
      count: number;
      findings: Finding[];
      selected?: boolean;
      indeterminate?: boolean;
      severity_order?: number;
    }
  
    interface ClientOption {
      value: string;
      name: string;
    }
  
    interface SeverityOption {
      value: string;
      name: string;
    }
  
    let groupedFindings: GroupedFindings[] = [];
    let selectedFinding: Finding | null = null;
    let showModal = false;
    let currentPage = 1;
    let perPage = 10; // Number of clusters per page
    let totalPages = 1;
    let totalItems = 0;
  
    // Sorting variables
    let sortField = 'severity_order';
    let sortDirection: 'asc' | 'desc' = 'asc';
  
    // Filter variables
    let severityFilter: string[] = [];
    let clientFilter: string[] = [];
    let searchTerm = '';
    let searchField = '';
  
    // Clients list for filtering
    let clients: { id: string; name: string }[] = [];
  
    let isLoading = false;
  
    let selectedFindings: Finding[] = [];
  
    let isUpdating = false;
    let updateError = '';
  
    // Keep track of selected findings
    $: selectedFindings = groupedFindings
      ? groupedFindings.flatMap((group) => group.findings.filter((f) => f.selected))
      : [];
  
    // Update group selection when findings are selected
    $: if (groupedFindings) {
      groupedFindings.forEach((group) => {
        // If all findings in a group are selected, mark the group as selected
        group.selected = group.findings.every((f) => f.selected);
      });
    }
  
    // Update finding selections when a group is selected
    $: if (groupedFindings) {
      groupedFindings.forEach((group) => {
        if (group.selected) {
          group.findings.forEach((finding) => (finding.selected = true));
        }
      });
    }
  
    // Define severity options with 'name' instead of 'label'
    const severityOptions: SeverityOption[] = [
      { value: 'critical', name: 'Critical' },
      { value: 'high', name: 'High' },
      { value: 'medium', name: 'Medium' },
      { value: 'low', name: 'Low' },
      { value: 'info', name: 'Info' },
      { value: 'unknown', name: 'Unknown' },
    ];

    // Define client options (after fetching clients)
    let clientOptions: ClientOption[] = [];

    // Update the clientOptions after fetching clients
    async function fetchClients() {
      try {
        const result = await $pocketbase.collection('clients').getFullList();
        clients = result.map((client) => ({ id: client.id, name: client.name }));

        // Map clients to options for MultiSelect using 'name'
        clientOptions = clients.map((client) => ({
          value: client.id,
          name: client.name,
        }));
      } catch (error) {
        console.error('Error fetching clients:', error);
      }
    }
  
    async function fetchGroupedFindings() {
      try {
        isLoading = true;
  
        // Build query parameters
        const params = new URLSearchParams();
        params.append('page', currentPage.toString());
        params.append('perPage', perPage.toString());
  
        // Include filters in the query parameters
        if (severityFilter.length > 0) {
          severityFilter.forEach((severity) => {
            params.append('severity', severity);
          });
        }
        if (clientFilter.length > 0) {
          clientFilter.forEach((clientId) => {
            params.append('client', clientId);
          });
        }
        if (searchTerm && searchField) {
          params.append('search', searchTerm);
          params.append('searchField', searchField);
        }
  
        // Include status filters
        if (statusFilter.length > 0) {
          statusFilter.forEach((status) => {
            params.append('status', status);
          });
        }
  
        // Include sorting parameters if your backend supports them
        if (sortField) {
          params.append('sortField', sortField);
          params.append('sortDirection', sortDirection);
        }
  
        // Get the auth token from PocketBase
        const token = $pocketbase.authStore.token;
  
        // Make the API request
        const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/findings/grouped?${params.toString()}`, {
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
          },
        });
  
        if (!response.ok) {
          const errorData = await response.json();
          console.error('Error fetching grouped findings:', errorData);
          throw new Error(`Error fetching grouped findings: ${response.statusText}`);
        }
  
        const data = await response.json();
  
        // Extract pagination metadata
        currentPage = data.page;
        perPage = data.perPage;
        totalPages = data.totalPages;
        totalItems = data.totalItems;

        // First, get all unique client IDs from the findings
        const clientIds = new Set();
        data.items.forEach(group => {
          group.findings.forEach(finding => {
            if (finding.client?.id) {
              clientIds.add(finding.client.id);
            }
          });
        });

        // Fetch full client data for each unique client ID
        const clientsData = new Map();
        await Promise.all(Array.from(clientIds).map(async (clientId) => {
          try {
            const client = await $pocketbase.collection('clients').getOne(clientId);
            clientsData.set(clientId, {
              ...client,
              favicon: client.favicon ? $pocketbase.getFileUrl(client, client.favicon) : null
            });
          } catch (error) {
            console.error(`Error fetching client ${clientId}:`, error);
          }
        }));

        // Map the findings data and include the full client data
        const mappedData = {
          ...data,
          items: data.items.map(group => ({
            ...group,
            findings: group.findings.map(finding => ({
              ...finding,
              client: finding.client?.id ? {
                ...finding.client,
                ...clientsData.get(finding.client.id)
              } : null
            }))
          }))
        };
  
        groupedFindings = initializeSelections(mappedData);
      } catch (error) {
        console.error('Error fetching grouped findings:', error);
      } finally {
        isLoading = false;
      }
    }
  
    function initializeSelections(data) {
      return data.items.map((group) => ({
        ...group,
        selected: false,
        indeterminate: false,
        findings: group.findings.map((finding) => ({
          ...finding,
          selected: false,
        })),
      }));
    }
  
    function openModal(finding: Finding) {
      selectedFinding = finding;
      showModal = true;
    }
  
    function changePage(direction: 'prev' | 'next') {
      if (direction === 'prev' && currentPage > 1) {
        currentPage--;
        fetchGroupedFindings();
      } else if (direction === 'next' && currentPage < totalPages) {
        currentPage++;
        fetchGroupedFindings();
      }
    }
  
    async function applyFilters() {
      // Reset to the first page when filters are applied
      currentPage = 1;
      await fetchGroupedFindings();
    }
  
    function getSeverityColor(severity: string): string {
      switch (severity.toLowerCase()) {
        case 'critical':
          return 'bg-red-600 text-white';
        case 'high':
          return 'bg-orange-500 text-white';
        case 'medium':
          return 'bg-yellow-500 text-white';
        case 'low':
          return 'bg-green-500 text-white';
        case 'info':
          return 'bg-blue-500 text-white';
        default:
          return 'bg-gray-500 text-white';
      }
    }
  
    async function markSelectedAsAcknowledged() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const ids = selectedFindings.map((finding) => finding.id);

      try {
        // Call backend API to update findings in bulk
        await updateFindingsBulk(ids, { acknowledged: true });

        // Update the local state
        selectedFindings.forEach((finding) => {
          finding.acknowledged = true;
        });

        // Clear selection
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }
  
    async function markSelectedAsFalsePositive() {
      const ids = selectedFindings.map((finding) => finding.id);

      try {
        // Call backend API to update findings in bulk
        await updateFindingsBulk(ids, { false_positive: true });

        // Update the local state
        selectedFindings.forEach((finding) => {
          finding.false_positive = true;
        });

        // Clear selection
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
      }
    }
  
    async function updateFindingsBulk(ids: string[], updateData: Record<string, any>) {
      const token = $pocketbase.authStore.token;

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/nuclei_results/bulk-update`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          ids,
          updateData,
        }),
      });

      if (!response.ok) {
        throw new Error(`Error updating findings: ${response.statusText}`);
      }
    }

    function clearSelections() {
      groupedFindings.forEach(group => {
        group.selected = false;
        group.indeterminate = false;
        group.findings.forEach(finding => {
          finding.selected = false;
        });
      });
      selectedFindings = [];
    }
  
    let staleThresholdDays = 30;

    onMount(async () => {
      try {
        // Get stale threshold from system settings
        const systemSettings = await $pocketbase.collection('system_settings').getFirstListItem('');
        if (systemSettings) {
          staleThresholdDays = systemSettings.stale_threshold_days || 30;
        }
      } catch (error) {
        console.error('Error loading system settings:', error);
      }

      // Load other data
      await fetchClients();
      await fetchGroupedFindings();
    });

    // Function to map severity_order back to severity string
    function severityOrderToString(order: number): string {
      switch (order) {
        case 1:
          return 'Critical';
        case 2:
          return 'High';
        case 3:
          return 'Medium';
        case 4:
          return 'Low';
        case 5:
          return 'Info';
        default:
          return 'Unknown';
      }
    }

    let falsePositiveFilter: boolean = false;
    let acknowledgedFilter: boolean = false;

    // Status filter variables
    let statusFilter: string[] = [];

    const statusOptions = [
      { value: 'acknowledged', name: 'Acknowledged' },
      { value: 'false_positive', name: 'False Positive' },
      { value: 'remediated', name: 'Remediated' },
      { value: 'no_status', name: 'No Status' },
    ];

    // Computed properties for acknowledgment status
    $: allSelectedAcknowledged = selectedFindings.length > 0 && selectedFindings.every(f => f.acknowledged);
    $: noneSelectedAcknowledged = selectedFindings.length > 0 && selectedFindings.every(f => !f.acknowledged);
    $: someSelectedAcknowledged = selectedFindings.length > 0 && !allSelectedAcknowledged && !noneSelectedAcknowledged;

    // Computed properties for false positive status
    $: allSelectedFalsePositive = selectedFindings.length > 0 && selectedFindings.every(f => f.false_positive);
    $: noneSelectedFalsePositive = selectedFindings.length > 0 && selectedFindings.every(f => !f.false_positive);
    $: someSelectedFalsePositive = selectedFindings.length > 0 && !allSelectedFalsePositive && !noneSelectedFalsePositive;

    // Add these computed properties for remediated status
    $: allSelectedRemediated = selectedFindings.length > 0 && selectedFindings.every(f => f.remediated);
    $: noneSelectedRemediated = selectedFindings.length > 0 && selectedFindings.every(f => !f.remediated);
    $: someSelectedRemediated = selectedFindings.length > 0 && !allSelectedRemediated && !noneSelectedRemediated;

    async function unmarkSelectedAsAcknowledged() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const ids = selectedFindings.map((finding) => finding.id);

      try {
        await updateFindingsBulk(ids, { acknowledged: false });

        // Update the local state
        selectedFindings.forEach((finding) => {
          finding.acknowledged = false;
        });

        // Clear selection
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }

    async function unmarkSelectedAsFalsePositive() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const ids = selectedFindings.map((finding) => finding.id);

      try {
        await updateFindingsBulk(ids, { false_positive: false });

        // Update the local state
        selectedFindings.forEach((finding) => {
          finding.false_positive = false;
        });

        // Clear selection
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }

    async function toggleSelectedAcknowledged() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const idsToAcknowledge = selectedFindings.filter(f => !f.acknowledged).map(f => f.id);
      const idsToUnacknowledge = selectedFindings.filter(f => f.acknowledged).map(f => f.id);

      try {
        // Acknowledge findings
        if (idsToAcknowledge.length > 0) {
          await updateFindingsBulk(idsToAcknowledge, { acknowledged: true });
          idsToAcknowledge.forEach(id => {
            const finding = selectedFindings.find(f => f.id === id);
            if (finding) finding.acknowledged = true;
          });
        }

        // Unacknowledge findings
        if (idsToUnacknowledge.length > 0) {
          await updateFindingsBulk(idsToUnacknowledge, { acknowledged: false });
          idsToUnacknowledge.forEach(id => {
            const finding = selectedFindings.find(f => f.id === id);
            if (finding) finding.acknowledged = false;
          });
        }

        // Clear selection
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }

    function toggleGroupSelection(groupIndex) {
      groupedFindings[groupIndex] = {
        ...groupedFindings[groupIndex],
        selected: !groupedFindings[groupIndex].selected,
        indeterminate: false,
      };

      groupedFindings[groupIndex].findings = groupedFindings[groupIndex].findings.map(finding => ({
        ...finding,
        selected: groupedFindings[groupIndex].selected,
      }));

      groupedFindings = [...groupedFindings];
      updateSelectedFindings();
    }

    function handleFindingSelectionChange(groupIndex: number) {
      const group = groupedFindings[groupIndex];
      if (!group) return;

      // Check if all findings in the group are selected
      const allSelected = group.findings.every((finding) => finding.selected);
      // Check if some findings in the group are selected
      const someSelected = group.findings.some((finding) => finding.selected);

      group.selected = allSelected;
      group.indeterminate = !allSelected && someSelected;

      // Update selectedFindings
      selectedFindings = groupedFindings.flatMap((group) => 
        group.findings.filter((finding) => finding.selected)
      );
    }

    function updateSelectedFindings() {
      selectedFindings = groupedFindings.flatMap(group =>
        group.findings.filter(finding => finding.selected)
      );
    }

    // Add these functions for remediated status
    async function markSelectedAsRemediated() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const ids = selectedFindings.map((finding) => finding.id);

      try {
        await updateFindingsBulk(ids, { remediated: true });
        selectedFindings.forEach((finding) => {
          finding.remediated = true;
        });
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }

    async function unmarkSelectedAsRemediated() {
      if (isUpdating) return;
      isUpdating = true;
      updateError = '';

      const ids = selectedFindings.map((finding) => finding.id);

      try {
        await updateFindingsBulk(ids, { remediated: false });
        selectedFindings.forEach((finding) => {
          finding.remediated = false;
        });
        clearSelections();
      } catch (error) {
        console.error('Error updating findings:', error);
        updateError = error.message;
      } finally {
        isUpdating = false;
      }
    }

    let showBulkCommentModal = false;

    // Update the helper function to use timestamp if last_seen is not available
    function getLastSeenStatus(lastSeen: string | undefined, timestamp: string): { isStale: boolean; daysAgo: number } {
      // Use lastSeen if available, otherwise use timestamp
      const dateToCheck = new Date(lastSeen || timestamp);
      const now = new Date();
      const diffTime = Math.abs(now.getTime() - dateToCheck.getTime());
      const daysAgo = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
      return {
        isStale: daysAgo > staleThresholdDays,
        daysAgo
      };
    }
  </script>
  
  <Card size="xl" class="shadow-sm max-w-none">
    <Heading tag="h3" class="-ml-0.25 mb-2 text-xl font-semibold dark:text-white">
      Nuclei Findings
    </Heading>
  
    <!-- Filter and Sort Form -->
    <form on:submit|preventDefault={applyFilters} class="flex flex-wrap gap-4 mb-4">
      <!-- Severity Filter -->
      <div class="flex flex-col">
        <label for="severityFilter">Severity:</label>
        <MultiSelect
          id="severityFilter"
          bind:value={severityFilter}
          items={severityOptions}
          placeholder="Select Severities"
          class="w-64"
        />
      </div>
  
      <!-- Client Filter -->
      <div class="flex flex-col">
        <label for="clientFilter">Client:</label>
        <MultiSelect
          id="clientFilter"
          bind:value={clientFilter}
          items={clientOptions}
          placeholder="Select Clients"
          class="w-64"
        />
      </div>

      <!-- Status Filter -->
      <div class="flex flex-col">
        <label for="statusFilter">Status:</label>
        <MultiSelect
          id="statusFilter"
          bind:value={statusFilter}
          items={statusOptions}
          placeholder="Select Status"
          class="w-64"
        />
      </div>
  
  
      <!-- Search Input -->
      <div class="flex flex-col">
        <label for="searchTerm">Search:</label>
        <Input
          id="searchTerm"
          bind:value={searchTerm}
          placeholder="Enter search term"
          class="w-64"
        />
      </div>
  
      <!-- Fields to Search -->
      <div class="flex flex-col">
        <label for="searchField">Search Field:</label>
        <Select
          id="searchField"
          bind:value={searchField}
          placeholder="Search In:"
          class="w-64"
        >
          <option value="">Select Field</option>
          <option value="name">Name</option>
          <option value="host">Host</option>
          <option value="ip">IP Address</option>
          <option value="template_id">Template ID</option>
          <option value="scan_name">Scan Name</option>
          <!-- Add other fields as needed -->
        </Select>
      </div>
      <!-- Apply Filters Button -->
      <div class="flex items-end">
        <Button type="submit">Apply Filters</Button>
      </div>
    </form>
  
    <!-- Bulk Action Buttons -->
    {#if selectedFindings.length > 0}
      <div class="bulk-actions flex space-x-4 mb-4">
        <!-- Acknowledged Button -->
        {#if allSelectedAcknowledged}
          <Button on:click={unmarkSelectedAsAcknowledged}>
            Unmark Selected as Acknowledged
          </Button>
        {:else if noneSelectedAcknowledged || someSelectedAcknowledged}
          <Button on:click={markSelectedAsAcknowledged}>
            Mark Selected as Acknowledged
          </Button>
        {/if}

        <!-- False Positive Button -->
        {#if allSelectedFalsePositive}
          <Button on:click={unmarkSelectedAsFalsePositive}>
            Unmark Selected as False Positive
          </Button>
        {:else if noneSelectedFalsePositive || someSelectedFalsePositive}
          <Button on:click={markSelectedAsFalsePositive}>
            Mark Selected as False Positive
          </Button>
        {/if}

        <!-- Remediated Button -->
        {#if allSelectedRemediated}
          <Button on:click={unmarkSelectedAsRemediated}>
            Unmark Selected as Remediated
          </Button>
        {:else if noneSelectedRemediated || someSelectedRemediated}
          <Button on:click={markSelectedAsRemediated}>
            Mark Selected as Remediated
          </Button>
        {/if}

        <!-- Add Comment Button -->
        <Button on:click={() => showBulkCommentModal = true}>
          Add Comment
        </Button>
      </div>
    {/if}
  
    <!-- Findings List with Grouped Results -->
    <div>
      {#if isLoading}
        <div class="space-y-4">
          {#each Array(perPage) as _, index}
            <div class="p-4 bg-white dark:bg-gray-800 rounded shadow" key={index}>
              <Skeleton class="h-4 w-1/4 mb-2 dark:bg-gray-700" />
              <Skeleton class="h-4 w-3/4 dark:bg-gray-700" />
            </div>
          {/each}
        </div>
      {:else if groupedFindings && groupedFindings.length > 0}
        {#each groupedFindings as group, groupIndex (group.template_id)}
          <Accordion flush={true}>
            <AccordionItem>
              <!-- Accordion Header -->
              <div slot="header" class="flex justify-between items-center">
                <!-- Group Selection Checkbox -->
                <Checkbox
                  bind:checked={group.selected}
                  indeterminate={group.indeterminate}
                  on:change={() => toggleGroupSelection(groupIndex)}
                  class="mr-2"
                />

                <!-- Severity and Template Name -->
                <div class="flex items-center space-x-2">
                  <!-- Severity Badge -->
                  <span
                    class={`inline-block px-2 py-1 rounded ${getSeverityColor(group.findings[0].severity_override || severityOrderToString(group.severity_order))}`}
                  >
                    {group.findings[0].severity_override || severityOrderToString(group.severity_order)}
                  </span>
                  <!-- Template Name -->
                  <span>{group.findings[0].info.name}</span>
                  
                  <!-- Status Badges at Group Level -->
                  {#if group.findings.every(f => f.false_positive)}
                    <div class="status-badge-container">
                      <span class="status-badge false-positive" title="All False Positive">❌
                        <span class="tooltip">All False Positive</span>
                      </span>
                    </div>
                  {:else if group.findings.some(f => f.false_positive)}
                    <div class="status-badge-container">
                      <span class="status-badge false-positive" title="Some False Positive">❌
                        <span class="tooltip">Some False Positive</span>
                      </span>
                    </div>
                  {/if}

                  {#if group.findings.every(f => f.acknowledged)}
                    <div class="status-badge-container">
                      <span class="status-badge acknowledged" title="All Acknowledged">✓
                        <span class="tooltip">All Acknowledged</span>
                      </span>
                    </div>
                  {:else if group.findings.some(f => f.acknowledged)}
                    <div class="status-badge-container">
                      <span class="status-badge acknowledged" title="Some Acknowledged">✓
                        <span class="tooltip">Some Acknowledged</span>
                      </span>
                    </div>
                  {/if}

                  {#if group.findings.every(f => f.remediated)}
                    <div class="status-badge-container">
                      <span class="status-badge remediated" title="All Remediated">🛠️
                        <span class="tooltip">All Remediated</span>
                      </span>
                    </div>
                  {:else if group.findings.some(f => f.remediated)}
                    <div class="status-badge-container">
                      <span class="status-badge remediated" title="Some Remediated">🛠️
                        <span class="tooltip">Some Remediated</span>
                      </span>
                    </div>
                  {/if}

                  {#if group.findings.every(f => f.notes)}
                    <div class="status-badge-container">
                      <span class="status-badge has-comments" title="All Have Notes">📝
                        <span class="tooltip">All Have Notes</span>
                      </span>
                    </div>
                  {:else if group.findings.some(f => f.notes)}
                    <div class="status-badge-container">
                      <span class="status-badge has-comments" title="Some Have Notes">📝
                        <span class="tooltip">Some Have Notes</span>
                      </span>
                    </div>
                  {/if}
                </div>
                <!-- Findings Count, Template ID, Client Name -->
                <div class="flex items-center space-x-4">
                  <!-- Findings Count Badge -->
                  <span class="inline-block bg-gray-200 text-gray-800 text-xs px-2 py-1 rounded-full">
                    {group.count}
                  </span>
                  <!-- Template ID -->
                  <span>Template ID: {group.template_id}</span>
                  <!-- Client Name -->
                  <span>
                    Client: 
                    {#if group.findings[0].client}
                      <div class="flex items-center gap-2 inline-flex">
                        {#if group.findings[0].client.favicon}
                          <img src={group.findings[0].client.favicon} alt="{group.findings[0].client.name} Favicon" class="h-4 w-4" />
                        {/if}
                        {group.findings[0].client.name}
                      </div>
                    {:else}
                      N/A
                    {/if}
                  </span>
                </div>
              </div>
  
              <!-- Accordion Content: List of Findings -->
              <ul class="list-none p-4">
                {#each group.findings as finding, findingIndex (finding.id)}
                  <li class="border-b py-2 flex justify-between items-center">
                    <div class="flex items-center">
                      <!-- Finding Selection Checkbox -->
                      <Checkbox
                        bind:checked={finding.selected}
                        on:change={() => handleFindingSelectionChange(groupIndex)}
                        class="mr-2"
                      />

                      <!-- Host, IP, and Status Badges -->
                      <div>
                        <div class="font-medium flex items-center space-x-2">
                          <span>{finding.host} ({finding.ip})</span>
                          {#if finding.false_positive}
                            <div class="status-badge-container">
                              <span class="status-badge false-positive" title="False Positive">❌
                                <span class="tooltip">False Positive</span>
                              </span>
                            </div>
                          {/if}
                          {#if finding.acknowledged}
                            <div class="status-badge-container">
                              <span class="status-badge acknowledged" title="Acknowledged">✓
                                <span class="tooltip">Acknowledged</span>
                              </span>
                            </div>
                          {/if}
                          {#if finding.remediated}
                            <div class="status-badge-container">
                              <span class="status-badge remediated" title="Remediated">🛠️
                                <span class="tooltip">Remediated</span>
                              </span>
                            </div>
                          {/if}
                          {#if finding.notes}
                            <div class="status-badge-container">
                              <span class="status-badge has-comments" title="Has Notes">📝
                                <span class="tooltip">Has Notes</span>
                              </span>
                            </div>
                          {/if}
                        </div>
                        <div class="text-sm text-gray-500">
                          Timestamp: {finding.timestamp ? new Date(finding.timestamp).toLocaleString() : 'N/A'}
                          {#if finding.last_seen && finding.last_seen !== finding.timestamp}
                            <br>
                            Last Seen: {new Date(finding.last_seen).toLocaleString()}
                            {#if getLastSeenStatus(finding.last_seen, finding.timestamp).isStale}
                              <div class="mt-1 text-amber-600 dark:text-amber-500 flex items-center gap-1">
                                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                                </svg>
                                <span>Not seen in {getLastSeenStatus(finding.last_seen, finding.timestamp).daysAgo} days</span>
                              </div>
                            {/if}
                          {:else if getLastSeenStatus(undefined, finding.timestamp).isStale}
                            <div class="mt-1 text-amber-600 dark:text-amber-500 flex items-center gap-1">
                              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                              </svg>
                              <span>Only seen once, {getLastSeenStatus(undefined, finding.timestamp).daysAgo} days ago</span>
                            </div>
                          {/if}
                        </div>
                      </div>
                    </div>

                    <!-- View Details Button -->
                    <Button size="xs" on:click={() => openModal(finding)}>
                      View Details
                    </Button>
                  </li>
                {/each}
              </ul>
            </AccordionItem>
          </Accordion>
        {/each}
      {:else}
        <p>No findings available.</p>
      {/if}
    </div>
  
    <!-- Pagination Controls -->
    <div class="pagination-controls flex justify-between mt-4">
      <Button on:click={() => changePage('prev')} disabled={currentPage === 1}>
        Previous
      </Button>
      <span>Page {currentPage} of {totalPages}</span>
      <Button on:click={() => changePage('next')} disabled={currentPage >= totalPages}>
        Next
      </Button>
    </div>
  </Card>
  
  <!-- Finding Modal -->
  {#if showModal && selectedFinding}
    <FindingModal 
      bind:open={showModal} 
      finding={selectedFinding} 
      on:findingUpdated={async () => {
        // Refresh the findings list
        await loadFindings();
      }}
    />
  {/if}
  
  <!-- Bulk Comment Modal -->
  {#if showBulkCommentModal && selectedFindings.length > 0}
    <BulkCommentModal 
      bind:open={showBulkCommentModal}
      findings={selectedFindings}
    />
  {/if}
  
  <style>
    .status-badge {
      @apply text-xs px-2 py-1 rounded cursor-help;
    }
    .false-positive {
      @apply bg-red-200 text-red-800;
    }
    .acknowledged {
      @apply bg-green-200 text-green-800;
    }
    .remediated {
      @apply bg-blue-200 text-blue-800;
    }
    .has-comments {
      @apply bg-purple-200 text-purple-800;
    }
    .tooltip {
      visibility: hidden;
      position: absolute;
      z-index: 50;
      padding: 0.5rem 0.75rem;
      font-size: 0.875rem;
      font-weight: 500;
      color: #ffffff;
      background-color: #1f2937;
      border-radius: 0.5rem;
      box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
      opacity: 0;
      transition-property: opacity;
      transition-duration: 300ms;
      white-space: nowrap;
      top: 100%;
      left: 50%;
      transform: translateX(-50%);
      margin-top: 0.25rem;
    }
    .status-badge:hover .tooltip {
      visibility: visible;
      opacity: 1;
    }
    .status-badge-container {
      @apply relative inline-block;
    }
  </style>
  
  <!-- Display loading indicator or error message -->
  {#if isUpdating}
    <p>Updating findings...</p>
  {/if}
  {#if updateError}
    <p class="text-red-500">Error: {updateError}</p>
  {/if}
  
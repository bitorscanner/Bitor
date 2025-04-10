<script lang="ts">
    import { onMount } from 'svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
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
      Alert,
      Toast,
      Dropdown,
      DropdownItem,
    } from 'flowbite-svelte';
    import FindingModal from './FindingModal.svelte';
    import BulkCommentModal from './BulkCommentModal.svelte';
    import { CheckCircleSolid, DotsVerticalOutline } from 'flowbite-svelte-icons';
    import AdvancedSearch from './AdvancedSearch.svelte';
  
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
    let statusFilter: string[] = [];
    let isAdmin = $pocketbase.authStore.isAdmin;
    let showMyFindingsOnly = !isAdmin; // Default to true for non-admins
    
    // User filter dropdown state
    let userFilterValue = showMyFindingsOnly ? 'mine' : 'all';
    
    // Function to handle user filter dropdown changes
    function handleUserFilterChange() {
      // Non-admins can now see all findings if they choose
      showMyFindingsOnly = userFilterValue === 'mine';
      applyFilters();
    }
  
    // Current user ID for filtering
    let currentUserId = $pocketbase.authStore.model?.id ?? '';
  
    // Clients list for filtering
    let clients: { id: string; name: string }[] = [];
  
    let isLoading = false;
  
    let selectedFindings: Finding[] = [];
  
    let isUpdating = false;
    let updateError = '';
  
    // Keep track of selected findings
    $: selectedFindings = groupedFindings && Array.isArray(groupedFindings)
      ? groupedFindings.flatMap((group) => group.findings.filter((f) => f.selected))
      : [];
  
    // Update group selection when findings are selected
    $: if (groupedFindings && Array.isArray(groupedFindings)) {
      groupedFindings.forEach((group) => {
        // If all findings in a group are selected, mark the group as selected
        group.selected = group.findings.every((f) => f.selected);
      });
    }
  
    // Update finding selections when a group is selected
    $: if (groupedFindings && Array.isArray(groupedFindings)) {
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
  
    // Add interface for cached data
    interface CachedData {
      items: GroupedFindings[];
      page: number;
      perPage: number;
      totalPages: number;
      totalItems: number;
      timestamp: number;
    }

    // Add interface for API response
    interface APIResponse {
      page: number;
      perPage: number;
      totalPages: number;
      totalItems: number;
      items: GroupedFindings[];
    }

    // Add these variables for caching
    let cachedFindings = new Map<string, CachedData>();
    let lastFetchTimestamp = 0;
    let cacheTimeout = 30000; // 30 seconds cache
  
    // Add advanced filters
    let advancedFilters: Array<{
      field: string;
      operator: string;
      value: string | string[];
      id: string;
    }> = [];
  
    // Modify the fetchGroupedFindings function
    async function fetchGroupedFindings() {
      try {
        isLoading = true;
  
        // Check cache first
        const cacheKey = `${currentPage}-${sortField}-${sortDirection}-${JSON.stringify(severityFilter)}-${JSON.stringify(clientFilter)}-${searchTerm}-${searchField}-${JSON.stringify(statusFilter)}-${showMyFindingsOnly}`;
        const now = Date.now();
        
        if (cachedFindings.has(cacheKey) && (now - lastFetchTimestamp) < cacheTimeout) {
          const cachedData = cachedFindings.get(cacheKey);
          if (cachedData) {
            groupedFindings = cachedData.items;
            currentPage = cachedData.page;
            perPage = cachedData.perPage;
            totalPages = cachedData.totalPages;
            totalItems = cachedData.totalItems;
            isLoading = false;
            return;
          }
        }
  
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

        // Filter by current user if selected and not super admin
        if (showMyFindingsOnly && !isAdmin && currentUserId) {
          params.append('created_by', currentUserId);
        }
  
        // Include sorting parameters if your backend supports them
        if (sortField) {
          params.append('sortField', sortField);
          params.append('sortDirection', sortDirection);
        }
  
        // Get the auth token from PocketBase
        const token = $pocketbase.authStore.token;
  
        // Use AbortController for request cancellation
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), 30000); // 30 second timeout
  
        try {
          const response = await fetch(
            `${import.meta.env.VITE_API_BASE_URL}/api/findings/grouped?${params.toString()}`,
            {
              headers: {
                Authorization: `Bearer ${token}`,
                'Content-Type': 'application/json',
              },
              signal: controller.signal
            }
          );
  
          clearTimeout(timeoutId);
  
          if (!response.ok) {
            const errorData = await response.json();
            console.error('Error fetching grouped findings:', errorData);
            throw new Error(`Error fetching grouped findings: ${response.statusText}`);
          }
  
          const data = await response.json() as APIResponse;
  
          // Update cache
          cachedFindings.set(cacheKey, {
            ...data,
            timestamp: now
          });
          lastFetchTimestamp = now;
  
          // Clean up old cache entries
          const cacheEntries = Array.from(cachedFindings.entries());
          if (cacheEntries.length > 20) { // Keep last 20 pages in cache
            const oldestEntries = cacheEntries
              .sort(([, a], [, b]) => a.timestamp - b.timestamp)
              .slice(0, cacheEntries.length - 20);
            oldestEntries.forEach(([key]) => cachedFindings.delete(key));
          }
  
          // Extract pagination metadata
          currentPage = data.page;
          perPage = data.perPage;
          totalPages = data.totalPages;
          totalItems = data.totalItems;
  
          // Optimize client data fetching
          const clientIds = new Set<string>();
          if (data.items && Array.isArray(data.items)) {
            data.items.forEach((group: GroupedFindings) => {
              group.findings.forEach((finding: Finding) => {
                if (finding.client?.id) {
                  clientIds.add(finding.client.id);
                }
              });
            });
          }
  
          // Batch fetch client data
          const clientsData = new Map<string, any>();
          const batchSize = 50;
          const clientIdArray = Array.from(clientIds);
          
          for (let i = 0; i < clientIdArray.length; i += batchSize) {
            const batch = clientIdArray.slice(i, i + batchSize);
            const batchFilter = batch.map(id => `id="${id}"`).join('||');
            
            try {
              const clientsBatch = await $pocketbase.collection('clients').getList(1, batchSize, {
                filter: batchFilter,
                fields: 'id,name,favicon,collectionId,collectionName'
              });
              
              clientsBatch.items.forEach(client => {
                const faviconUrl = client.favicon ? $pocketbase.files.getUrl(client, client.favicon) : null;
                clientsData.set(client.id, {
                  id: client.id,
                  name: client.name,
                  favicon: faviconUrl
                });
              });
            } catch (error: unknown) {
              const errorMessage = error instanceof Error ? error.message : 'Unknown error';
              console.error(`Error fetching clients batch ${i}:`, errorMessage);
            }
          }
  
          // Map the findings data with optimized client data
          groupedFindings = data.items && Array.isArray(data.items) ? data.items.map((group: GroupedFindings) => ({
            ...group,
            findings: group.findings.map((finding: Finding) => {
              const clientData = finding.client?.id ? clientsData.get(finding.client.id) : null;
              return {
                ...finding,
                client: clientData || finding.client
              };
            })
          })) : [];
  
        } finally {
          clearTimeout(timeoutId);
        }
  
      } catch (error: unknown) {
        if (error instanceof Error && error.name === 'AbortError') {
          console.log('Request was aborted due to timeout');
        } else {
          const errorMessage = error instanceof Error ? error.message : 'Unknown error';
          console.error('Error fetching grouped findings:', errorMessage);
        }
      } finally {
        isLoading = false;
      }
    }
  
    function initializeSelections(data: APIResponse) {
      return data && data.items && Array.isArray(data.items) 
        ? data.items.map((group: GroupedFindings) => ({
            ...group,
            selected: false,
            indeterminate: false,
            findings: group.findings.map((finding: Finding) => ({
              ...finding,
              selected: false,
            })),
          }))
        : [];
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
      isLoading = true;
      try {
        let filter = '';
        let conditions = [];

        // Add severity filter
        if (Array.isArray(severityFilter) && severityFilter.length > 0) {
          conditions.push(`severity in [${severityFilter.map(s => `"${s}"`).join(',')}]`);
        }

        // Add client filter
        if (Array.isArray(clientFilter) && clientFilter.length > 0) {
          conditions.push(`client in [${clientFilter.join(',')}]`);
        }

        // Add status filters
        if (Array.isArray(statusFilter) && statusFilter.length > 0) {
          const statusConditions = statusFilter.map(status => {
            switch (status) {
              case 'acknowledged':
                return 'acknowledged = true';
              case 'false_positive':
                return 'false_positive = true';
              case 'remediated':
                return 'remediated = true';
              default:
                return '';
            }
          }).filter(Boolean);
          if (statusConditions.length > 0) {
            conditions.push(`(${statusConditions.join(' || ')})`);
          }
        }

        // Add search term
        if (searchTerm) {
          if (searchField) {
            conditions.push(`${searchField} ~ "${searchTerm}"`);
          } else {
            conditions.push(`(host ~ "${searchTerm}" || ip ~ "${searchTerm}" || template_id ~ "${searchTerm}" || info_name ~ "${searchTerm}")`);
          }
        }

        // Add user filter
        if (showMyFindingsOnly && currentUserId) {
          conditions.push(`created_by = "${currentUserId}"`);
        }

        // Add advanced filters
        if (Array.isArray(advancedFilters)) {
          advancedFilters.forEach(filter => {
            switch (filter.operator) {
              case 'equals':
                conditions.push(`${filter.field} = "${filter.value}"`);
                break;
              case 'contains':
                conditions.push(`${filter.field} ~ "${filter.value}"`);
                break;
              case 'in':
                if (Array.isArray(filter.value)) {
                  conditions.push(`${filter.field} in [${filter.value.map(v => `"${v}"`).join(',')}]`);
                }
                break;
              case 'not_equals':
                conditions.push(`${filter.field} != "${filter.value}"`);
                break;
            }
          });
        }

        filter = conditions.join(' && ');

        // Reset to the first page when filters are applied
        currentPage = 1;
        await fetchGroupedFindings();
      } catch (error) {
        console.error('Error applying filters:', error);
        // Set groupedFindings to empty array in case of error
        groupedFindings = [];
      } finally {
        isLoading = false;
      }
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

      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/findings/bulk-update`, {
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
      if (groupedFindings && Array.isArray(groupedFindings)) {
        groupedFindings.forEach(group => {
          group.selected = false;
          group.indeterminate = false;
          group.findings.forEach(finding => {
            finding.selected = false;
          });
        });
      }
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
      
      // Try to load saved filters first
      try {
        await loadDefaultFilters();
      } catch (error) {
        console.error('Error loading default filters:', error);
        // If loading saved filters fails, use default values
        if (!isAdmin) {
          showMyFindingsOnly = true;
          userFilterValue = 'mine';
        }
      }
      
      // Fetch findings with current filter settings
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

    // Status options - removed duplicate statusFilter declaration
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

    function toggleGroupSelection(groupIndex: number) {
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

    interface UserPreferences {
      findings_filters: {
        severity: string[];
        client: string[];
        status: string[];
        myFindingsOnly?: boolean;
        sortField?: string;
        sortDirection?: 'asc' | 'desc';
      };
      admin_id?: string;
      users_relation?: string;
    }

    let showToast = false;
    let toastMessage = '';

    async function loadDefaultFilters() {
      try {
        const userId = $pocketbase.authStore.model?.id;
        if (!userId) return;

        let filter = '';
        if (isAdmin) {
          filter = `admin_id="${userId}"`;
        } else {
          filter = `users_relation="${userId}"`;
        }

        const record = await $pocketbase.collection('user_preferences').getFirstListItem(filter);
        if (record?.findings_filters) {
          const filters = record.findings_filters;
          if (filters.severity) severityFilter = filters.severity;
          if (filters.client) clientFilter = filters.client;
          if (filters.status) statusFilter = filters.status;
          if (filters.myFindingsOnly !== undefined) {
            // For non-admins, always default to showing only their findings
            showMyFindingsOnly = isAdmin ? filters.myFindingsOnly : true;
            userFilterValue = showMyFindingsOnly ? 'mine' : 'all';
          }
        } else if (!isAdmin) {
          // If no saved preferences and not admin, default to user's findings
          showMyFindingsOnly = true;
          userFilterValue = 'mine';
        }
      } catch (error: unknown) {
        if (error && typeof error === 'object' && 'status' in error && error.status !== 404) {
          console.error('Error loading default filters:', error);
        }
        // If there's an error and user is not admin, default to user's findings
        if (!isAdmin) {
          showMyFindingsOnly = true;
          userFilterValue = 'mine';
        }
      }
    }

    async function saveAsDefaultFilters() {
      try {
        const isAdmin = $pocketbase.authStore.isAdmin;
        const userId = $pocketbase.authStore.model?.id;
        if (!userId) return;

        let filter = '';
        if (isAdmin) {
          filter = `admin_id="${userId}"`;
        } else {
          filter = `users_relation="${userId}"`;
        }

        const filters: UserPreferences['findings_filters'] = {
          severity: severityFilter,
          client: clientFilter,
          status: statusFilter,
          myFindingsOnly: showMyFindingsOnly,
          sortField,
          sortDirection
        };

        let record;
        try {
          // Try to find existing preferences
          record = await $pocketbase.collection('user_preferences').getFirstListItem(filter);
          // Update existing record
          await $pocketbase.collection('user_preferences').update(record.id, {
            findings_filters: filters
          });
        } catch (error: unknown) {
          if (error && typeof error === 'object' && 'status' in error && error.status === 404) {
            // Create new preferences record
            const data: UserPreferences = {
              findings_filters: filters
            };
            if (isAdmin) {
              data.admin_id = userId;
            } else {
              data.users_relation = userId;
            }
            await $pocketbase.collection('user_preferences').create(data);
          } else {
            throw error;
          }
        }

        showToast = true;
        toastMessage = 'Filters saved successfully';
        setTimeout(() => {
          showToast = false;
        }, 3000);
      } catch (error: unknown) {
        console.error('Error saving default filters:', error);
        showToast = true;
        toastMessage = 'Error saving filters';
        setTimeout(() => {
          showToast = false;
        }, 3000);
      }
    }

    function exportHostsFromGroup(group: GroupedFindings) {
      // Extract unique host, IP, and client combinations from the group
      const hostData = group.findings.map(finding => ({
        host: finding.host,
        ip: finding.ip,
        client: finding.client?.name || 'N/A'
      }));

      // Remove duplicates by creating a unique string for each combination
      const uniqueHostData = Array.from(new Set(
        hostData.map(data => `${data.host},${data.ip},${data.client}`)
      ));
      
      // Create CSV content with headers
      const csvContent = 'Host,IP,Client\n' + uniqueHostData.join('\n');
      
      // Create blob and download
      const blob = new Blob([csvContent], { type: 'text/csv' });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `hosts-${group.template_id}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      window.URL.revokeObjectURL(url);
    }

    // Fix the loadFindings reference
    async function refreshFindings() {
      await fetchGroupedFindings();
    }

    // Add this helper function before the onMount function
    function getUniqueClientsInfo(findings: Finding[]) {
      const uniqueClients = new Set(findings.map(f => f.client?.id).filter(Boolean));
      const firstClient = findings[0]?.client;
      return {
        count: uniqueClients.size,
        firstClient
      };
    }

    // Handle advanced filter changes
    function handleAdvancedFilterChange(event: CustomEvent<{ filters: typeof advancedFilters }>) {
      advancedFilters = event.detail.filters;
      applyFilters();
    }
  </script>
  
  <Card size="xl" class="shadow-sm max-w-none">
    <Heading tag="h3" class="-ml-0.25 mb-2 text-xl font-semibold dark:text-white">
      Nuclei Findings
    </Heading>
  
    <!-- Filter and Sort Form -->
    <form on:submit|preventDefault={applyFilters} class="flex flex-wrap gap-4 mb-4">
      <AdvancedSearch
        {severityOptions}
        {clientOptions}
        {statusOptions}
        {isAdmin}
        bind:showMyFindingsOnly
        on:change={({ detail }) => {
          severityFilter = detail.severityFilter;
          clientFilter = detail.clientFilter;
          statusFilter = detail.statusFilter;
          showMyFindingsOnly = detail.showMyFindingsOnly;
          userFilterValue = detail.showMyFindingsOnly ? 'mine' : 'all';
          searchTerm = detail.searchTerm;
          searchField = detail.searchField;
          advancedFilters = detail.filters;
          applyFilters();
        }}
      />

      <!-- Save as Default Button -->
      <div class="w-full flex justify-end">
        <Button color="alternative" on:click={saveAsDefaultFilters}>Save as Default</Button>
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
                      {@const clientInfo = getUniqueClientsInfo(group.findings)}
                      <div class="flex items-center gap-2 inline-flex">
                        {#if clientInfo.firstClient.favicon}
                          <img src={clientInfo.firstClient.favicon} alt="{clientInfo.firstClient.name} Favicon" class="h-4 w-4" />
                        {/if}
                        {clientInfo.firstClient.name}
                        {#if clientInfo.count > 1}
                          <span class="text-xs text-gray-500 dark:text-gray-400">(+{clientInfo.count - 1} other {clientInfo.count === 2 ? 'client' : 'clients'})</span>
                        {/if}
                      </div>
                    {:else}
                      N/A
                    {/if}
                  </span>
                  <!-- Dropdown menu for actions -->
                  <div class="relative">
                    <Button size="xs" class="!p-1">
                      <DotsVerticalOutline class="w-4 h-4" />
                    </Button>
                    <Dropdown class="w-48" placement="bottom">
                      <DropdownItem on:click={() => exportHostsFromGroup(group)}>
                        Export Hosts
                      </DropdownItem>
                      <!-- Add more dropdown items here as needed -->
                    </Dropdown>
                  </div>
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
                    <div class="flex items-center gap-2">
                      {#if finding.client}
                        <div class="flex items-center gap-2">
                          {#if finding.client.favicon}
                            <img src={finding.client.favicon} alt="{finding.client.name} Favicon" class="h-4 w-4" />
                          {/if}
                          <span class="text-sm text-gray-500 dark:text-gray-400">
                            {finding.client.name}
                          </span>
                        </div>
                      {/if}
                      <Button size="xs" on:click={() => openModal(finding)}>
                        View Details
                      </Button>
                    </div>
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
        await refreshFindings();
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
  
  {#if showToast}
    <div class="fixed bottom-4 right-4">
      <Toast>
        <div class="flex items-center gap-2">
          <CheckCircleSolid class="w-5 h-5 text-green-500" />
          <span class="text-sm font-semibold">{toastMessage}</span>
        </div>
      </Toast>
    </div>
  {/if}
  
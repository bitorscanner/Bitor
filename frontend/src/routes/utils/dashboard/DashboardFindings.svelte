<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import {
    Table,
    TableBody,
    TableBodyCell,
    TableBodyRow,
    TableHead,
    TableHeadCell,
    Card,
    Heading,
    Button
  } from 'flowbite-svelte';
  import FindingModal from './FindingModal.svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';

  interface Finding {
    id: string;
    name: string;
    description: string;
    severity: string;
    type: string;
    tool: string;
    host: string;
    status: string;
    client: string;
    hash: string;
    scan_ids: string[];
  }

  let findings: Finding[] = [];
  let selectedFinding: Finding | null = null;
  let showModal = false;
  let currentPage = 1;
  const itemsPerPage = 10;
  let sortField = '';
  let sortDirection = 'asc';
  let totalPages = 1;
  let totalItems = 0;
  let loading = true;
  let error = '';

  // Fetch data from PocketBase
  async function fetchFindings(page = 1, sortField = '', sortDirection = 'asc') {
    try {
      const result = await $pocketbase.collection('nuclei_findings').getList(page, itemsPerPage, {
        sort: `${sortDirection === 'asc' ? '' : '-'}${sortField}`
      });

      findings = result.items.map(item => ({
        id: item.id,
        name: item.name,
        description: item.description,
        severity: item.severity,
        type: item.type,
        tool: item.tool,
        host: item.host,
        status: item.status,
        client: item.client,
        hash: item.hash,
        scan_ids: item.scan_ids || []
      }));

      totalPages = result.totalPages;
      totalItems = result.totalItems;
      currentPage = page;
      loading = false;
    } catch (e) {
      error = e.message;
      loading = false;
    }
  }

  function sortData(field: string) {
    if (sortField === field) {
      sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortDirection = 'asc';
    }

    // Fetch sorted data from the server
    fetchFindings(currentPage, sortField, sortDirection);
  }

  function openModal(finding: Finding) {
    selectedFinding = finding;
    showModal = true;
  }

  function changePage(direction: string) {
    if (direction === 'next' && currentPage * itemsPerPage < findings.length) {
      currentPage++;
      fetchFindings(currentPage);
    } else if (direction === 'prev' && currentPage > 1) {
      currentPage--;
    }
  }

  function getSeverityColor(severity: string) {
    switch (severity.toLowerCase()) {
      case 'high':
        return 'bg-red-500 text-white';
      case 'medium':
        return 'bg-yellow-500 text-white';
      case 'low':
        return 'bg-green-500 text-white';
      default:
        return 'bg-gray-500 text-white';
    }
  }

  onMount(() => {
    fetchFindings();
  });
</script>

<Card size="xl" class="shadow-sm max-w-none">
  <Heading tag="h3" class="-ml-0.25 mb-2 text-xl font-semibold dark:text-white">
    Nuclei Findings
  </Heading>
  <Table
    hoverable={true}
    noborder
    striped
    class="mt-6 min-w-full divide-y divide-gray-200 dark:divide-gray-600"
  >
  <TableHead class="bg-gray-50 dark:bg-gray-700">
    <TableHeadCell on:click={() => sortData('name')}>
      Name {#if sortField === 'name'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell on:click={() => sortData('host')}>
      Host {#if sortField === 'host'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell on:click={() => sortData('id')}>
      ID {#if sortField === 'id'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell on:click={() => sortData('type')}>
      Type {#if sortField === 'type'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell on:click={() => sortData('severity')}>
      Severity {#if sortField === 'severity'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell on:click={() => sortData('scan_ids')}>
      Scan IDs {#if sortField === 'scan_ids'}{sortDirection === 'asc' ? '▲' : '▼'}{/if}
    </TableHeadCell>
    <TableHeadCell>
        Client
    </TableHeadCell>
    <TableHeadCell>
        Timestamp
    </TableHeadCell>
  </TableHead>
    <TableBody>
      {#each findings.slice((currentPage - 1) * itemsPerPage, currentPage * itemsPerPage) as finding}
        <TableBodyRow on:click={() => openModal(finding)}>
          <TableBodyCell class="px-4 font-normal">{finding.name}</TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {finding.host}
          </TableBodyCell>
          <TableBodyCell class="px-4">{finding.id}</TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {finding.type}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            <span class={`inline-block px-2 py-1 rounded ${getSeverityColor(finding.severity)}`}>
              {finding.severity}
            </span>
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            {#each finding.scan_ids as scanId}
              <span class="inline-block bg-gray-200 text-gray-800 text-xs px-2 py-1 rounded-full mr-1">{scanId}</span>
            {/each}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            {#if finding.client}
              {finding.client}
            {:else}
              N/A
            {/if}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            {finding.timestamp || 'N/A'}
          </TableBodyCell>
        </TableBodyRow>
      {/each}
    </TableBody>
  </Table>
  <div class="pagination-controls">
    <Button on:click={() => changePage('prev')} disabled={currentPage === 1}>Previous</Button>
    <Button on:click={() => changePage('next')}>Next</Button>
  </div>
</Card>

<FindingModal bind:open={showModal} finding={selectedFinding} />

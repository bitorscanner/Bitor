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
    Badge
  } from 'flowbite-svelte';
  import { format, differenceInSeconds } from 'date-fns';

  interface ScanData {
    id: string;
    name: string;
    status: string;
    start_time: string;
  }

  let runningScans: ScanData[] = [];
  let currentUserId = $pocketbase.authStore.model?.id ?? '';

  async function fetchRunningScans() {
    try {
      let filter = 'status="Running" || status="Generating" || status="Deploying"';
      
      // Always apply user filter for non-admin users
      if (!$pocketbase.authStore.isAdmin) {
        filter += ` && created_by = "${currentUserId}"`;
      }

      const result = await $pocketbase.collection('nuclei_scans').getList(1, 50, {
        filter: filter,
        sort: '-start_time',
      });
      runningScans = result.items;
    } catch (error) {
      console.error('Error fetching running scans:', error);
    }
  }

  function formatDateToLocal(dateString: string): string {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return format(date, 'yyyy-MM-dd HH:mm:ss');
  }

  function computeElapsedTime(start: string): string {
    if (!start) return 'N/A';
    const startDate = new Date(start);
    const now = new Date();

    if (isNaN(startDate.getTime())) {
      return 'Invalid Date';
    }

    const diffInSeconds = differenceInSeconds(now, startDate);

    if (diffInSeconds < 60) {
      return `${diffInSeconds} seconds`;
    } else if (diffInSeconds < 3600) {
      const minutes = Math.floor(diffInSeconds / 60);
      const seconds = diffInSeconds % 60;
      return `${minutes} min ${seconds} sec`;
    } else {
      const hours = Math.floor(diffInSeconds / 3600);
      const minutes = Math.floor((diffInSeconds % 3600) / 60);
      return `${hours} hr ${minutes} min`;
    }
  }

  onMount(() => {
    fetchRunningScans();
  });
</script>

<Card size="xl" class="shadow-sm max-w-none">
  <Heading tag="h3" class="-ml-0.25 mb-2 text-xl font-semibold dark:text-white">
    Running Scans
  </Heading>
  <Table
    hoverable={true}
    noborder
    striped
    class="mt-6 min-w-full divide-y divide-gray-200 dark:divide-gray-600"
  >
    <TableHead class="bg-gray-50 dark:bg-gray-700">
      <TableHeadCell>Name</TableHeadCell>
      <TableHeadCell>Start Time</TableHeadCell>
      <TableHeadCell>Elapsed Time</TableHeadCell>
      <TableHeadCell>Status</TableHeadCell>
    </TableHead>
    <TableBody>
      {#each runningScans as scan}
        <TableBodyRow>
          <TableBodyCell class="px-4 font-normal">{scan.name}</TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {formatDateToLocal(scan.start_time)}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {computeElapsedTime(scan.start_time)}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            <Badge color="yellow">{scan.status}</Badge>
          </TableBodyCell>
        </TableBodyRow>
      {/each}
    </TableBody>
  </Table>
</Card>
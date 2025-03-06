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
  import { format, differenceInSeconds, parseISO } from 'date-fns';

  let completedScans = [];

  async function fetchCompletedScans() {
    try {
      const result = await $pocketbase.collection('nuclei_scans').getList(1, 50, {
        filter: 'status="Finished"',
        sort: '-end_time', // Optional: Sort by most recent
      });
      completedScans = result.items;
    } catch (error) {
      console.error('Error fetching completed scans:', error);
    }
  }

  function formatDateToLocal(dateString: string): string {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return format(date, 'yyyy-MM-dd HH:mm:ss');
  }

  function computeDuration(start: string, end: string): string {
    if (!start || !end) return 'N/A';
    const startDate = new Date(start);
    const endDate = new Date(end);

    if (isNaN(startDate.getTime()) || isNaN(endDate.getTime())) {
      return 'Invalid Date';
    }

    const diffInSeconds = differenceInSeconds(endDate, startDate);

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
    fetchCompletedScans();
  });
</script>

<Card size="xl" class="shadow-sm max-w-none">
  <Heading tag="h3" class="-ml-0.25 mb-2 text-xl font-semibold dark:text-white">
    Completed Scans
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
      <TableHeadCell>End Time</TableHeadCell>
      <TableHeadCell>Duration</TableHeadCell>
      <TableHeadCell>Status</TableHeadCell>
    </TableHead>
    <TableBody>
      {#each completedScans as scan}
        <TableBodyRow>
          <TableBodyCell class="px-4 font-normal">{scan.name}</TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {formatDateToLocal(scan.start_time)}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {formatDateToLocal(scan.end_time)}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal text-gray-500 dark:text-gray-400">
            {computeDuration(scan.start_time, scan.end_time)}
          </TableBodyCell>
          <TableBodyCell class="px-4 font-normal">
            <Badge color="green">{scan.status}</Badge>
          </TableBodyCell>
        </TableBodyRow>
      {/each}
    </TableBody>
  </Table>
</Card> 
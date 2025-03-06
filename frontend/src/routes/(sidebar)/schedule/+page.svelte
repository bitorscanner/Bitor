<script lang="ts">
  import ScheduleCalendar from './ScheduleCalendar.svelte';
  import ScheduleScanModal from './ScheduleScanModal.svelte';
  import { Breadcrumb, BreadcrumbItem, Heading, Button, Card } from 'flowbite-svelte';
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';

  let showScheduleModal = false;
  let calendarComponent: any;
  let mode: 'add' | 'edit' = 'add';
  let selectedScan: any = null;
  let selectedDate: Date | null = null;

  function openScheduleModal(newMode: 'add' | 'edit' = 'add', scan = null, date = null) {
    mode = newMode;
    selectedScan = scan;
    selectedDate = date;
    showScheduleModal = true;
  }

  function handleSave() {
    if (calendarComponent) {
      calendarComponent.refetchEvents();
    }
    showScheduleModal = false;
  }

  function handleNewScan(event: { detail: { date: Date } }) {
    openScheduleModal('add', null, event.detail.date);
  }

  function handleEditScan(event: { detail: any }) {
    openScheduleModal('edit', event.detail);
  }
</script>

<main class="p-4">
  <Breadcrumb class="mb-6">
    <BreadcrumbItem href="/" home>Home</BreadcrumbItem>
    <BreadcrumbItem>Schedule</BreadcrumbItem>
  </Breadcrumb>

  <Card padding="xl" class="min-w-full">
    <div class="flex justify-between items-center mb-6">
      <Heading tag="h1" class="text-xl font-semibold text-gray-900 dark:text-white sm:text-2xl">
        Scan Schedule
      </Heading>
      <Button color="primary" on:click={() => openScheduleModal('add')}>Schedule New Scan</Button>
    </div>

    <div class="h-[calc(100vh-16rem)]">
      <ScheduleCalendar
        bind:this={calendarComponent}
        on:editScan={handleEditScan}
        on:newScan={handleNewScan}
      />
    </div>
  </Card>

  <!-- Modal -->
  <ScheduleScanModal
    bind:open={showScheduleModal}
    {mode}
    existingScan={selectedScan}
    {selectedDate}
    on:save={handleSave}
  />
</main>
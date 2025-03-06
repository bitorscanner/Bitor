<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { browser } from '$app/environment';
  import { Modal, Label, Input, Select, Button, Tabs, TabItem, Radio, Badge } from 'flowbite-svelte';
  import { goto } from '$app/navigation';

  // Exported props
  export let open = false;
  export let mode: 'add' | 'edit' = 'add';
  export let existingScan: any = null;
  export let selectedDate: Date | null = null;

  // Event dispatcher
  const dispatch = createEventDispatcher();

  // Form data
  let scanId = '';
  let scanName = '';
  let scheduleType = 'one-time'; // one-time, recurring
  let frequency = 'daily'; // daily, weekly, monthly
  let cronExpression = '';
  let startDate = '';
  let endDate = '';
  let selectedDays: string[] = [];
  let monthlyType = 'day'; // day, date
  let monthlyDate = 1;
  let monthlyDay = 'monday';
  let monthlyWeek = 'first';

  // Available scans
  let availableScans: any[] = [];

  // Initialize form fields
  $: if (open) {
    if (mode === 'edit' && existingScan) {
      initializeEditMode();
    } else {
      initializeAddMode();
    }
    fetchAvailableScans();
  }

  function initializeEditMode() {
    try {
      if (!existingScan || typeof existingScan !== 'object') {
        console.error('Invalid existing scan:', existingScan);
        initializeAddMode();
        return;
      }

      scanId = existingScan.scan_id || '';
      scanName = existingScan.scan_name || '';
      scheduleType = existingScan.cron_expression ? 'recurring' : 'one-time';
      
      // Initialize frequency and parse cron expression
      if (existingScan.cron_expression && typeof existingScan.cron_expression === 'string') {
        const cronParts = existingScan.cron_expression.split(' ').filter(Boolean);
        if (cronParts.length === 5) {
          if (cronParts[2] !== '*') {
            frequency = 'monthly';
            monthlyType = 'date';
            monthlyDate = parseInt(cronParts[2]) || 1;
          } else if (cronParts[4] !== '*') {
            frequency = 'weekly';
            // Parse selected days from cron expression
            selectedDays = cronParts[4].split(',')
              .map(day => day.trim())
              .filter(day => /^[0-6]$/.test(day))
              .map(day => {
                const dayMap: { [key: string]: string } = {
                  '0': 'sunday',
                  '1': 'monday',
                  '2': 'tuesday',
                  '3': 'wednesday',
                  '4': 'thursday',
                  '5': 'friday',
                  '6': 'saturday'
                };
                return dayMap[day] || 'monday';
              });
          } else {
            frequency = 'daily';
          }
        } else {
          console.warn('Invalid cron expression format:', existingScan.cron_expression);
          frequency = 'daily';
        }
      } else {
        frequency = 'daily';
        selectedDays = [];
      }
      
      cronExpression = existingScan.cron_expression || '';
      
      // Handle dates
      try {
        startDate = existingScan.start_date ? new Date(existingScan.start_date).toISOString().slice(0, 16) : '';
      } catch (error) {
        console.error('Invalid start date:', existingScan.start_date);
        startDate = '';
      }
      
      try {
        endDate = existingScan.end_date ? new Date(existingScan.end_date).toISOString().slice(0, 16) : '';
      } catch (error) {
        console.error('Invalid end date:', existingScan.end_date);
        endDate = '';
      }
    } catch (error) {
      console.error('Error initializing edit mode:', error);
      initializeAddMode();
    }
  }

  function initializeAddMode() {
    scanId = '';
    scanName = '';
    scheduleType = 'one-time';
    frequency = 'daily';
    cronExpression = '';
    startDate = selectedDate ? (selectedDate instanceof Date ? selectedDate : new Date(selectedDate)).toISOString().slice(0, 16) : '';
    endDate = '';
    selectedDays = [];
    monthlyType = 'day';
    monthlyDate = 1;
    monthlyDay = 'monday';
    monthlyWeek = 'first';
  }

  async function fetchAvailableScans() {
    if (!browser) return;
    
    try {
      console.log('Fetching scans from PocketBase');
      const result = await $pocketbase.collection('nuclei_scans').getFullList({
        sort: '-created',
        filter: 'status != "Manual"'
      });
      
      console.log('Raw scans data:', result);
      availableScans = result.map(scan => ({
        id: scan.id,
        name: scan.name || `Scan ${scan.id}`
      }));
      console.log('Processed scans:', availableScans);
    } catch (error) {
      console.error('Error fetching scans:', error);
      availableScans = [];
    }
  }

  function goToCreateScan() {
    open = false;
    goto('/scans');
  }

  function generateCronExpression() {
    if (scheduleType === 'one-time') return '';

    switch (frequency) {
      case 'daily':
        return '0 0 * * *';
      case 'weekly':
        if (!selectedDays || selectedDays.length === 0) {
          // Default to Monday if no days selected
          selectedDays = ['monday'];
        }
        const dayMap: { [key: string]: string } = {
          'sunday': '0',
          'monday': '1',
          'tuesday': '2',
          'wednesday': '3',
          'thursday': '4',
          'friday': '5',
          'saturday': '6'
        };
        const days = selectedDays
          .map(day => dayMap[day])
          .filter(day => day !== undefined)
          .join(',');
        return `0 0 * * ${days}`;
      case 'monthly':
        if (monthlyType === 'date') {
          return `0 0 ${monthlyDate || 1} * *`;
        } else {
          const week = {
            first: '1',
            second: '2',
            third: '3',
            fourth: '4',
            last: 'L'
          }[monthlyWeek] || '1';
          const day = {
            sunday: '0',
            monday: '1',
            tuesday: '2',
            wednesday: '3',
            thursday: '4',
            friday: '5',
            saturday: '6'
          }[monthlyDay] || '1';
          return `0 0 * * ${day}#${week}`;
        }
      default:
        return '0 0 * * *';
    }
  }

  async function handleSave() {
    if (!browser) return;
    const token = $pocketbase.authStore.token;
    if (!token) return;

    try {
      if (!scanId) {
        console.error('Scan ID is required');
        return;
      }

      if (!startDate) {
        console.error('Start date is required');
        return;
      }

      // For weekly frequency, ensure at least one day is selected
      if (scheduleType === 'recurring' && frequency === 'weekly' && selectedDays.length === 0) {
        console.error('Please select at least one day for weekly schedule');
        return;
      }

      const data = {
        scan_id: scanId,
        frequency: scheduleType === 'one-time' ? 'one-time' : frequency,
        cron_expression: generateCronExpression(),
        start_date: startDate ? new Date(startDate).toISOString() : null,
        end_date: endDate ? new Date(endDate).toISOString() : null,
        schedule_details: {
          type: scheduleType,
          frequency,
          selectedDays: frequency === 'weekly' ? selectedDays : [],
          monthlyType: frequency === 'monthly' ? monthlyType : null,
          monthlyDate: frequency === 'monthly' && monthlyType === 'date' ? monthlyDate : null,
          monthlyDay: frequency === 'monthly' && monthlyType === 'day' ? monthlyDay : null,
          monthlyWeek: frequency === 'monthly' && monthlyType === 'day' ? monthlyWeek : null,
        }
      };

      console.log('Sending schedule data:', data);

      let record;
      
      if (mode === 'add') {
        // Create new scheduled scan
        record = await $pocketbase.collection('scheduled_scans').create(data);
      } else {
        if (!existingScan?.id) {
          console.error('Missing scan ID for edit mode');
          return;
        }
        // Update existing scheduled scan
        record = await $pocketbase.collection('scheduled_scans').update(existingScan.id, data);
      }

      console.log('Schedule saved successfully:', record);

      // Close modal first
      open = false;
      // Then dispatch save event to trigger refresh
      dispatch('save', { action: mode, data: record });
    } catch (error) {
      console.error('Error saving scheduled scan:', error);
      throw error;
    }
  }

  async function handleDelete() {
    if (!browser || !existingScan?.id) return;
    const token = $pocketbase.authStore.token;
    if (!token) return;

    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/scheduled/${existingScan.id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to delete scheduled scan');
      }

      console.log('Successfully deleted scheduled scan');
      // Close modal first
      open = false;
      // Then dispatch save event to trigger refresh
      dispatch('save', { action: 'delete', id: existingScan.id });
    } catch (error) {
      console.error('Error deleting scheduled scan:', error);
      throw error;
    }
  }
</script>

<Modal bind:open size="lg" title={mode === 'add' ? 'Schedule New Scan' : 'Edit Scheduled Scan'}>
  <form on:submit|preventDefault={handleSave} class="space-y-6">
    <!-- Scan Selection -->
    <div class="space-y-2">
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Select Scan</span>
        <div class="flex gap-2">
          <Select bind:value={scanId} required class="flex-1">
            <option value="">Select a scan</option>
            {#each availableScans as scan}
              <option value={scan.id}>{scan.name || `Scan ${scan.id}`}</option>
            {/each}
          </Select>
          <Button color="primary" on:click={goToCreateScan}>
            Create New
          </Button>
        </div>
        {#if availableScans.length === 0}
          <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
            No scans available. Create a new scan first.
          </p>
        {/if}
      </Label>
    </div>

    <!-- Schedule Type -->
    <div class="space-y-2">
      <span class="text-gray-700 dark:text-gray-300">Schedule Type</span>
      <div class="flex gap-4">
        <Radio bind:group={scheduleType} value="one-time">One-time</Radio>
        <Radio bind:group={scheduleType} value="recurring">Recurring</Radio>
      </div>
    </div>

    {#if scheduleType === 'recurring'}
      <!-- Frequency Selection -->
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Frequency</span>
        <Select bind:value={frequency} required>
          <option value="daily">Daily</option>
          <option value="weekly">Weekly</option>
          <option value="monthly">Monthly</option>
        </Select>
      </Label>

      {#if frequency === 'weekly'}
        <!-- Weekly Options -->
        <div class="space-y-2">
          <span class="text-gray-700 dark:text-gray-300">Select Days</span>
          <div class="flex flex-wrap gap-2">
            {#each ['sunday', 'monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday'] as day}
              <button
                type="button"
                class="px-3 py-1 rounded-full text-sm font-medium transition-colors {selectedDays.includes(day) 
                  ? 'bg-blue-500 text-white hover:bg-blue-600' 
                  : 'bg-gray-200 text-gray-700 hover:bg-gray-300 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600'}"
                on:click={() => {
                  if (selectedDays.includes(day)) {
                    selectedDays = selectedDays.filter(d => d !== day);
                  } else {
                    selectedDays = [...selectedDays, day];
                  }
                  selectedDays = selectedDays; // Trigger reactivity
                }}
              >
                {day.charAt(0).toUpperCase() + day.slice(1)}
              </button>
            {/each}
          </div>
          {#if selectedDays.length === 0}
            <p class="text-sm text-red-500 mt-1">Please select at least one day</p>
          {/if}
        </div>
      {:else if frequency === 'monthly'}
        <!-- Monthly Options -->
        <div class="space-y-4">
          <div class="flex gap-4">
            <Radio bind:group={monthlyType} value="date">By Date</Radio>
            <Radio bind:group={monthlyType} value="day">By Day</Radio>
          </div>

          {#if monthlyType === 'date'}
            <Label class="space-y-2">
              <span class="text-gray-700 dark:text-gray-300">Date of Month</span>
              <Select bind:value={monthlyDate}>
                {#each Array(31).fill(0).map((_, i) => i + 1) as date}
                  <option value={date}>{date}</option>
                {/each}
              </Select>
            </Label>
          {:else}
            <div class="grid grid-cols-2 gap-4">
              <Label class="space-y-2">
                <span class="text-gray-700 dark:text-gray-300">Week</span>
                <Select bind:value={monthlyWeek}>
                  <option value="first">First</option>
                  <option value="second">Second</option>
                  <option value="third">Third</option>
                  <option value="fourth">Fourth</option>
                  <option value="last">Last</option>
                </Select>
              </Label>
              <Label class="space-y-2">
                <span class="text-gray-700 dark:text-gray-300">Day</span>
                <Select bind:value={monthlyDay}>
                  <option value="sunday">Sunday</option>
                  <option value="monday">Monday</option>
                  <option value="tuesday">Tuesday</option>
                  <option value="wednesday">Wednesday</option>
                  <option value="thursday">Thursday</option>
                  <option value="friday">Friday</option>
                  <option value="saturday">Saturday</option>
                </Select>
              </Label>
            </div>
          {/if}
        </div>
      {/if}
    {/if}

    <!-- Start Date -->
    <Label class="space-y-2">
      <span class="text-gray-700 dark:text-gray-300">Start Date & Time</span>
      <Input type="datetime-local" bind:value={startDate} required />
    </Label>

    <!-- End Date (optional) -->
    <Label class="space-y-2">
      <span class="text-gray-700 dark:text-gray-300">End Date & Time (Optional)</span>
      <Input type="datetime-local" bind:value={endDate} />
    </Label>

    <!-- Submit Button -->
    <div class="flex justify-end gap-2">
      <Button color="alternative" on:click={() => (open = false)}>Cancel</Button>
      {#if mode === 'edit'}
        <Button color="red" on:click={handleDelete}>Delete</Button>
      {/if}
      <Button type="submit" color="primary">{mode === 'edit' ? 'Update' : 'Schedule'}</Button>
    </div>
  </form>
</Modal>

<style>
  /* Additional styles if necessary */
</style>
<script lang="ts">
  import { onMount } from 'svelte';
  import { createEventDispatcher } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { browser } from '$app/environment';
  import { ScheduleXCalendar } from '@schedule-x/svelte';
  import { createCalendar, createViewMonthGrid, createViewWeek, createViewDay } from '@schedule-x/calendar';
  import { createEventRecurrencePlugin, createEventsServicePlugin } from '@schedule-x/event-recurrence';
  import { createDragAndDropPlugin } from '@schedule-x/drag-and-drop';
  import { createResizePlugin } from '@schedule-x/resize';
  import { createCurrentTimePlugin } from '@schedule-x/current-time';
  import { createScrollControllerPlugin } from '@schedule-x/scroll-controller';
  import '@schedule-x/theme-default/dist/index.css';

  // Create plugins
  const recurrencePlugin = createEventRecurrencePlugin();
  const eventsServicePlugin = createEventsServicePlugin();
  const dragAndDropPlugin = createDragAndDropPlugin(30); // 30-minute intervals for dragging
  const resizePlugin = createResizePlugin(30); // 30-minute intervals for resizing
  const currentTimePlugin = createCurrentTimePlugin({
    fullWeekWidth: true,
    timeZoneOffset: new Date().getTimezoneOffset() // Use local timezone offset
  });

  // Get current time for initial scroll
  const now = new Date();
  const currentTime = `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;

  const scrollController = createScrollControllerPlugin({
    initialScroll: currentTime // Set initial scroll to current time
  });

  interface CalendarEvent {
    id: string;
    title: string;
    start: string;
    end: string;
    allDay?: boolean;
    extendedProps?: {
      scan: any;
      startTime?: string | null;
      endTime?: string | null;
    };
    rrule?: string;
  }

  const dispatch = createEventDispatcher();
  let calendarApp: any;
  let events: CalendarEvent[] = [];
  let isLoading = true;

  // Add state for delete confirmation modal
  let selectedEvent: CalendarEvent | null = null;

  // Add debug logging
  const debug = {
    log: (...args: any[]) => {
      console.log('[Calendar]', ...args);
    },
    error: (...args: any[]) => {
      console.error('[Calendar]', ...args);
    },
    warn: (...args: any[]) => {
      console.warn('[Calendar]', ...args);
    }
  };

  let calendarConfig: any;

  // Update the reactive statement to use the events service plugin
  $: if (events && browser && calendarApp) {
    debug.log('Events updated, updating calendar...', events);
    try {
      // Clear existing events
      const currentEvents = eventsServicePlugin.getAll();
      currentEvents.forEach(event => eventsServicePlugin.remove(event.id));

      // Add new events
      events.forEach(event => {
        eventsServicePlugin.add(event);
      });

      debug.log('Calendar updated successfully');
    } catch (error) {
      debug.error('Error updating calendar:', error);
    }
  }

  onMount(async () => {
    if (browser) {
      try {
        isLoading = true;
        debug.log('Initializing calendar...');

        // Initial fetch of events
        events = await fetchEvents();
        debug.log('Initial events loaded:', events);

        // Store the calendar configuration
        calendarConfig = {
          views: [createViewMonthGrid(), createViewWeek(), createViewDay()],
          events,
          defaultView: createViewMonthGrid().name,
          isDark: document.documentElement.classList.contains('dark'),
          locale: 'en-US',
          firstDayOfWeek: 1,
          plugins: [
            recurrencePlugin,
            eventsServicePlugin,
            dragAndDropPlugin,
            resizePlugin,
            currentTimePlugin,
            scrollController
          ],
          weekOptions: {
            gridHeight: 800,
            nDays: 5,
            eventWidth: 95,
            timeAxisFormatOptions: { hour: 'numeric' }
          },
          monthGridOptions: {
            nEventsPerDay: 4
          },
          callbacks: {
            onEventClick: handleEventClick,
            onClickDate: (dateStr: string) => {
              const date = new Date(dateStr);
              handleDateClick(date);
            },
            onEventUpdate: handleEventDrop,
            onEventDragStart: (event: CalendarEvent) => {
              debug.log('Event drag started:', event);
            },
            onEventDragEnd: async (event: CalendarEvent) => {
              debug.log('Event drag ended:', event);
              await handleEventDrop(event);
            },
            onEventResizeStart: (event: CalendarEvent) => {
              debug.log('Event resize started:', event);
            },
            onEventResizeEnd: async (event: CalendarEvent) => {
              debug.log('Event resize ended:', event);
              await handleEventDrop(event);
            }
          }
        };

        // Create the initial calendar instance
        calendarApp = createCalendar(calendarConfig);
        debug.log('Calendar initialization complete');
      } catch (error) {
        debug.error('Error initializing calendar:', error);
      } finally {
        isLoading = false;
      }
    }
  });

  export async function refetchEvents() {
    if (!browser) {
      debug.warn('Cannot refetch events: not in browser environment');
      return;
    }
    
    try {
      debug.log('Refetching events...');
      const newEvents = await fetchEvents();
      // Update events to trigger reactivity
      events = newEvents;
      debug.log('Events refetched successfully:', events);
    } catch (error) {
      debug.error('Error refetching events:', error);
    }
  }

  async function fetchEvents(): Promise<CalendarEvent[]> {
    try {
      debug.log('Fetching scheduled scans...');
      const scheduledScans = await fetchScheduledScans();
      debug.log('Scheduled Scans:', scheduledScans);
      
      if (!Array.isArray(scheduledScans)) {
        debug.warn('Scheduled scans is not an array:', scheduledScans);
        return [];
      }
      
      const validScans = scheduledScans.filter(scan => {
        if (!scan || typeof scan !== 'object') {
          debug.warn('Invalid scan object:', scan);
          return false;
        }
        return true;
      });

      const mappedEvents = validScans.map(scan => {
        try {
          return mapScanToEvent(scan);
        } catch (error) {
          debug.error('Error mapping scan to event:', error);
          return null;
        }
      }).filter((event): event is CalendarEvent => event !== null);

      debug.log('Mapped Events:', mappedEvents);
      return mappedEvents;
    } catch (error) {
      debug.error('Error fetching scheduled scans:', error);
      return [];
    }
  }

  async function fetchScheduledScans() {
    if (!browser) return [];
    const token = $pocketbase.authStore.token;
    if (!token) return [];
    
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/scheduled`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });
      
      if (!response.ok) {
        debug.error('Failed to fetch scheduled scans:', response.statusText);
        return [];
      }
      
      const data = await response.json();
      debug.log('Raw API Response Data:', data);

      // Filter out scans with invalid dates and fetch scan names
      const validScans = [];
      for (const scan of Array.isArray(data) ? data : []) {
        const startDate = new Date(scan.start_date);
        const isValidDate = !isNaN(startDate.getTime()) && startDate.getFullYear() > 1970;
        if (!isValidDate) {
          debug.warn('Filtering out scan with invalid date:', scan);
          continue;
        }

        try {
          // Fetch the scan details to get the name
          const scanDetails = await $pocketbase.collection('nuclei_scans').getOne(scan.scan_id);
          scan.scan_name = scanDetails.name || `Scan ${scan.scan_id}`;
          validScans.push(scan);
        } catch (error) {
          debug.warn(`Could not fetch details for scan ${scan.scan_id}:`, error);
          scan.scan_name = `Scan ${scan.scan_id}`;
          validScans.push(scan);
        }
      }

      debug.log('Filtered valid scans with names:', validScans);
      return validScans;
    } catch (error) {
      debug.error('Error fetching scheduled scans:', error);
      return [];
    }
  }

  function generateRRule(scan: any): string {
    const frequency = scan.frequency || 'one-time';
    const details = scan.schedule_details || {};
    
    if (frequency === 'one-time') return '';

    let rule = '';
    
    switch (frequency) {
      case 'daily':
        rule = 'FREQ=DAILY';
        break;
      case 'weekly':
        rule = 'FREQ=WEEKLY';
        if (details.selectedDays && details.selectedDays.length > 0) {
          const dayMap: { [key: string]: string } = {
            'sunday': 'SU',
            'monday': 'MO',
            'tuesday': 'TU',
            'wednesday': 'WE',
            'thursday': 'TH',
            'friday': 'FR',
            'saturday': 'SA'
          };
          const days = details.selectedDays
            .map((day: string) => dayMap[day])
            .filter(Boolean)
            .join(',');
          if (days) {
            rule += `;BYDAY=${days}`;
          }
        }
        break;
      case 'monthly':
        rule = 'FREQ=MONTHLY';
        if (details.monthlyType === 'date' && details.monthlyDate) {
          rule += `;BYMONTHDAY=${details.monthlyDate}`;
        }
        break;
    }

    // Add INTERVAL=1 to ensure proper recurrence
    rule += ';INTERVAL=1';

    // Add COUNT=52 if no end date (limit to 52 occurrences - 1 year)
    if (!scan.end_date) {
      rule += ';COUNT=52';
    } else {
      // Add end date if specified (YYYYMMDDTHHMMSS format)
      const endDate = new Date(scan.end_date);
      if (!isNaN(endDate.getTime())) {
        const until = endDate.toISOString().replace(/[-:]/g, '').split('.')[0];
        rule += `;UNTIL=${until}`;
      }
    }

    return rule;
  }

  function mapScanToEvent(scan: any): CalendarEvent | null {
    if (!scan || typeof scan !== 'object') {
      debug.warn('Invalid scan object:', scan);
      return null;
    }

    try {
      const startDate = new Date(scan.start_date);
      const endDate = scan.end_date ? new Date(scan.end_date) : startDate;

      if (isNaN(startDate.getTime())) {
        debug.warn('Invalid start date:', scan.start_date);
        return null;
      }

      const eventId = scan.id || `temp-${Date.now()}`;
      const scanName = scan.scan_name || 'Unnamed Scan';
      const frequency = scan.frequency || 'one-time';
      const isRecurring = frequency !== 'one-time';

      // Format dates as required by Schedule-X (YYYY-MM-DD HH:mm)
      const formatDateTime = (date: Date) => {
        return date.toISOString().slice(0, 16).replace('T', ' ');
      };

      // Create event object according to Schedule-X format
      const event: CalendarEvent = {
        id: eventId,
        title: scanName,
        start: formatDateTime(startDate),
        end: formatDateTime(endDate),
        extendedProps: {
          scan
        }
      };

      // Add recurrence rule for recurring events
      if (isRecurring) {
        event.rrule = generateRRule(scan);
        debug.log('Generated recurrence rule:', event.rrule);
      }

      debug.log('Created event:', event);
      return event;
    } catch (error) {
      debug.error('Error mapping scan to event:', error);
      return null;
    }
  }

  // Update event handlers to ensure refresh
  function handleEventClick(event: any) {
    if (!event?.extendedProps?.scan) {
      debug.warn('Invalid event data:', event);
      return;
    }
    const scan = event.extendedProps.scan;
    debug.log('Event clicked:', scan);
    
    dispatch('editScan', scan);
  }

  function handleDateClick(date: Date) {
    if (!date) {
      debug.warn('Invalid date click data:', date);
      return;
    }
    dispatch('newScan', { date });
  }

  async function handleEventDrop(event: any) {
    if (!browser || !event?.id) return;
    const token = $pocketbase.authStore.token;
    if (!token) return;

    try {
      debug.log('Event drop data:', event);

      // Get the original event data
      const originalEvent = event.extendedProps?.scan;
      if (!originalEvent) {
        throw new Error('Original event data not found');
      }

      // Calculate the new dates based on the drag
      const newStartDate = new Date(event.start);
      let newEndDate = event.end ? new Date(event.end) : null;

      // If there was an original duration, maintain it
      if (originalEvent.end_date && !newEndDate) {
        const originalDuration = new Date(originalEvent.end_date).getTime() - new Date(originalEvent.start_date).getTime();
        newEndDate = new Date(newStartDate.getTime() + originalDuration);
      }

      // Prepare the update data
      const updateData = {
        scan_id: originalEvent.scan_id,
        start_date: newStartDate.toISOString(),
        end_date: newEndDate ? newEndDate.toISOString() : null,
        // Preserve other important fields
        frequency: originalEvent.frequency,
        cron_expression: originalEvent.cron_expression,
        schedule_details: originalEvent.schedule_details
      };

      debug.log('Updating event with data:', updateData);

      // Use PocketBase API to update the scheduled scan
      const record = await $pocketbase.collection('scheduled_scans').update(event.id, updateData);
      debug.log('Successfully updated event:', record);

      // Refetch events after successful update
      await refetchEvents();
    } catch (error) {
      debug.error('Error updating scan schedule:', error);
      // Refetch events to revert changes
      await refetchEvents();
    }
  }

  async function handleDeleteConfirm() {
    if (!selectedEvent?.extendedProps?.scan?.id) {
      debug.error('Invalid event for deletion');
      return;
    }

    const token = $pocketbase.authStore.token;
    if (!token) return;

    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/scheduled/${selectedEvent.extendedProps.scan.id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to delete scheduled scan');
      }

      debug.log('Successfully deleted scheduled scan');
      selectedEvent = null;
      
      // Refetch events after successful deletion
      await refetchEvents();
    } catch (error) {
      debug.error('Error deleting scheduled scan:', error);
    }
  }
</script>

{#if browser && !isLoading}
  <div class="h-full w-full relative bg-white dark:bg-gray-800">
    <ScheduleXCalendar calendarApp={calendarApp} />
    
    <!-- Delete Confirmation Modal -->
    {#if selectedEvent}
      <div class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div class="bg-white dark:bg-gray-800 rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
          <h3 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">Delete Scheduled Scan</h3>
          <p class="text-gray-700 dark:text-gray-300 mb-4">
            Are you sure you want to delete this scheduled scan?<br>
            <span class="font-medium">{selectedEvent.title}</span>
          </p>
          <div class="flex justify-end gap-3">
            <button
              class="px-4 py-2 text-gray-700 bg-gray-100 hover:bg-gray-200 dark:bg-gray-700 dark:text-gray-300 dark:hover:bg-gray-600 rounded-lg transition-colors"
              on:click={() => selectedEvent = null}
            >
              Cancel
            </button>
            <button
              class="px-4 py-2 text-white bg-red-500 hover:bg-red-600 rounded-lg transition-colors"
              on:click={handleDeleteConfirm}
            >
              Delete
            </button>
          </div>
        </div>
      </div>
    {/if}
  </div>
{:else if isLoading}
  <div class="h-full w-full flex items-center justify-center bg-white dark:bg-gray-800">
    <div class="animate-pulse">Loading calendar...</div>
  </div>
{:else}
  <div class="h-full w-full flex items-center justify-center bg-white dark:bg-gray-800">
    {#if !browser}
      Browser environment not available
    {:else}
      Calendar initialization failed
    {/if}
  </div>
{/if}

<style>
  :global(.sx__calendar-app) {
    --sx-font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    --sx-color-primary: #3b82f6;
    --sx-color-primary-hover: #2563eb;
    --sx-color-primary-contrast: #ffffff;
    --sx-color-text: #374151;
    --sx-color-text-light: #6b7280;
    --sx-color-gray-100: #f3f4f6;
    --sx-color-gray-200: #e5e7eb;
    --sx-color-gray-300: #d1d5db;
    --sx-color-gray-400: #9ca3af;
    --sx-color-gray-500: #6b7280;
    --sx-color-gray-600: #4b5563;
    --sx-color-gray-700: #374151;
    --sx-color-gray-800: #1f2937;
    --sx-color-gray-900: #111827;
    --sx-border-color: #e5e7eb;
    --sx-border-radius: 0.375rem;
    --sx-shadow: 0 1px 3px 0 rgb(0 0 0 / 0.1), 0 1px 2px -1px rgb(0 0 0 / 0.1);
    --sx-shadow-lg: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
    position: absolute !important;
    top: 0 !important;
    left: 0 !important;
    right: 0 !important;
    bottom: 0 !important;
    height: 100% !important;
    width: 100% !important;
    background-color: inherit !important;
  }

  :global(.dark .sx__calendar-app) {
    --sx-color-text: #f3f4f6;
    --sx-color-text-light: #9ca3af;
    --sx-color-gray-100: #1f2937;
    --sx-color-gray-200: #374151;
    --sx-color-gray-300: #4b5563;
    --sx-color-gray-400: #6b7280;
    --sx-color-gray-500: #9ca3af;
    --sx-color-gray-600: #d1d5db;
    --sx-color-gray-700: #e5e7eb;
    --sx-color-gray-800: #f3f4f6;
    --sx-color-gray-900: #f9fafb;
    --sx-border-color: #374151;
  }

  :global(.sx__calendar-app *) {
    font-family: var(--sx-font-family);
  }

  :global(.sx__calendar-app .sx__event) {
    border-radius: var(--sx-border-radius);
    box-shadow: var(--sx-shadow);
    transition: box-shadow 0.2s ease-in-out;
  }

  :global(.sx__calendar-app .sx__event:hover) {
    box-shadow: var(--sx-shadow-lg);
  }

  :global(.sx__calendar-app .sx__button) {
    border-radius: var(--sx-border-radius);
    padding: 0.5rem 1rem;
    font-weight: 500;
    transition: all 0.2s ease-in-out;
  }

  :global(.sx__calendar-app .sx__button:hover) {
    opacity: 0.9;
  }

  :global(.sx__calendar-app .sx__button--primary) {
    background-color: var(--sx-color-primary);
    color: var(--sx-color-primary-contrast);
  }

  :global(.sx__calendar-app .sx__button--primary:hover) {
    background-color: var(--sx-color-primary-hover);
  }

  :global(.sx__calendar-app .sx__header) {
    padding: 1rem;
    border-bottom: 1px solid var(--sx-border-color);
  }

  :global(.sx__calendar-app .sx__toolbar) {
    padding: 0.5rem;
    gap: 0.5rem;
  }

  :global(.sx__calendar-app .sx__view-selector) {
    gap: 0.25rem;
  }

  :global(.sx__calendar-app .sx__date-selector) {
    gap: 0.5rem;
  }

  :global(.sx__calendar-app .sx__grid) {
    border: 1px solid var(--sx-border-color);
    border-radius: var(--sx-border-radius);
  }

  :global(.sx__calendar-app .sx__grid-cell) {
    min-height: 100px;
    border: 1px solid var(--sx-border-color);
  }

  :global(.sx__calendar-app .sx__grid-cell--today) {
    background-color: var(--sx-color-gray-100);
  }

  :global(.sx__calendar-app .sx__grid-header) {
    background-color: var(--sx-color-gray-100);
    border-bottom: 1px solid var(--sx-border-color);
    padding: 0.5rem;
    font-weight: 500;
  }

  :global(.sx__calendar-app .sx__month-grid),
  :global(.sx__calendar-app .sx__week-grid),
  :global(.sx__calendar-app .sx__day-grid) {
    height: 100% !important;
  }

  :global(.sx__calendar-app .sx__month-grid-wrapper),
  :global(.sx__calendar-app .sx__week-grid-wrapper),
  :global(.sx__calendar-app .sx__day-grid-wrapper) {
    height: calc(100% - 4rem) !important;
  }
</style>

<script lang="ts">
  import { Modal, Button, Toggle, Select } from 'flowbite-svelte';
  import { Accordion, AccordionItem } from 'flowbite-svelte';
  import { pocketbase } from '$lib/stores/pocketbase';
  import { createEventDispatcher, onDestroy, onMount } from 'svelte';
  // Import Xterm components and types
  import { Xterm, XtermAddon } from '@battlefieldduck/xterm-svelte';
  import type { ITerminalOptions, ITerminalInitOnlyOptions, Terminal } from '@battlefieldduck/xterm-svelte';
  import BlockNoteEditor from '$lib/BlockNoteEditor.svelte';

  let showNotes = false;
  let saveMessage = '';
  let copyMessage = '';

  let notesContent: string = '';
  let previousFindingId = '';
  let editorKey = 0;
  const dispatch = createEventDispatcher();

  export let open = false;

  // Flag to track if user has toggled showNotes
  let hasUserToggled = false;

  interface Finding {
    id: string;
    info: {
      name: string;
      tags: string[];
      description?: string;
      reference?: string | string[];
    };
    host: string;
    ip: string;
    template_id: string;
    severity: 'critical' | 'high' | 'medium' | 'low' | 'info' | 'unknown';
    severity_order: number;
    client: {
      name: string;
    } | null;
    scan: {
      name: string;
      id: string;
    } | null;
    timestamp: string;
    last_seen: string;
    acknowledged: boolean;
    false_positive: boolean;
    request?: string;
    response?: string;
    notes?: string;
    updated?: string;
    matched_at?: string;
    type?: string;
    extra_info?: any;
    port?: number;
    url?: string;
    curl_command?: string;
    remediated: boolean;
    extracted_results?: string[];
    severity_override?: string;
    severity_override_order?: number;
  }

  export let finding: Finding | null = null;

  // Terminal options
  const terminalOptions: ITerminalOptions = {
    theme: {
      background: '#1e1e1e',
      foreground: '#ffffff',
      cursor: '#ffffff',
      black: '#000000',
      white: '#ffffff'
    },
    fontFamily: 'Consolas, Monaco, "Courier New", monospace',
    fontSize: 14,
    cursorBlink: true,
    convertEol: true,
    rows: 16,
    allowTransparency: false
  };

  // Terminal instances
  let requestTerminalInstance: Terminal;
  let responseTerminalInstance: Terminal;
  let curlTerminalInstance: Terminal;
  let extractedResultsTerminalInstance: Terminal;

  let requestFitAddon: any;
  let responseFitAddon: any;
  let curlFitAddon: any;
  let extractedResultsFitAddon: any;
  // Reactive variable to hold parsed references
  let parsedReferences: string[] = [];

    // Reactive statement to detect finding changes
    $: if (finding && finding.id !== previousFindingId) {
    console.log('Finding changed from', previousFindingId, 'to', finding.id);
    previousFindingId = finding.id;
    fetchNotes().then(() => {
      editorKey += 1; // Increment key to force re-render
    });
  }

  // Update parsedReferences when finding changes
  $: if (finding) {
    if (typeof finding.info.reference === 'string') {
      try {
        const urlRegex = /https?:\/\/[^\s"]+/g;
        parsedReferences = finding.info.reference.match(urlRegex) || [];
      } catch (error) {
        console.error('Error parsing references from string:', error);
      }
    } else if (Array.isArray(finding.info.reference)) {
      parsedReferences = finding.info.reference;
    } else {
      parsedReferences = [];
    }

    // Initialize notes content if available
    if (finding.notes) {
      notesContent = finding.notes;
    } else {
      notesContent = '';
    }
  }

  async function fetchNotes() {
    if (finding) {
      try {
        console.log('Fetching notes for finding ID:', finding.id);
        const record = await $pocketbase.collection('nuclei_results').getOne(finding.id);
        console.log('Fetched record:', record);
        if (record.notes) {
          notesContent = record.notes;
          if (!hasUserToggled) {
            showNotes = true;
          }
        } else {
          notesContent = '';
          if (!hasUserToggled) {
            showNotes = false;
          }
        }
        console.log('Updated notesContent:', notesContent);
      } catch (error) {
        console.error('Error fetching notes:', error);
        notesContent = '';
        if (!hasUserToggled) {
          showNotes = false;
        }
      }
    }
  }

  // Function to initialize terminals with content
  function onCurlLoad(event: CustomEvent<{ terminal: Terminal }>) {
    const { terminal } = event.detail;
    curlTerminalInstance = terminal;

    // Write curl command data to the terminal
    if (finding && finding.curl_command) {
      curlTerminalInstance.reset();
      curlTerminalInstance.write(finding.curl_command);
    } else {
      console.log('No curl command data available');
    }
  }

  // Function to initialize terminals with content
  async function onRequestLoad(event: CustomEvent<{ terminal: Terminal }>) {
    requestTerminalInstance = event.detail.terminal;

    // Load the FitAddon
    requestFitAddon = new (await XtermAddon.FitAddon()).FitAddon();
    requestTerminalInstance.loadAddon(requestFitAddon);
    requestFitAddon.fit();

    // Write request data to the terminal
    if (finding && finding.request) {
      requestTerminalInstance.reset();
      requestTerminalInstance.write(finding.request);
    } else {
      console.log('No request data available');
    }
  }

  async function onResponseLoad(event: CustomEvent<{ terminal: Terminal }>) {
    responseTerminalInstance = event.detail.terminal;

    // Load the FitAddon
    responseFitAddon = new (await XtermAddon.FitAddon()).FitAddon();
    responseTerminalInstance.loadAddon(responseFitAddon);
    responseFitAddon.fit();

    // Write response data to the terminal
    if (finding && finding.response) {
      responseTerminalInstance.reset();
      responseTerminalInstance.write(finding.response);
    } else {
      console.log('No response data available');
    }
  }

  // Function to initialize terminals with content
  function onExtractedResultsLoad(event: CustomEvent<{ terminal: Terminal }>) {
    const { terminal } = event.detail;
    extractedResultsTerminalInstance = terminal;

    // Write extracted results data to the terminal
    if (finding && finding.extracted_results) {
      extractedResultsTerminalInstance.reset();
      extractedResultsTerminalInstance.write(JSON.stringify(finding.extracted_results, null, 2));
    } else {
      console.log('No extracted results data available');
    }
  }

  // Refit terminals when modal opens
  $: if (open) {
    if (requestFitAddon) {
      requestFitAddon.fit();
    }
    if (responseFitAddon) {
      responseFitAddon.fit();
    }
    if (curlFitAddon) {
      curlFitAddon.fit();
    }
    if (extractedResultsFitAddon) {
      extractedResultsFitAddon.fit();
    }
  }

  // Function to handle acknowledged toggle
  async function toggleAcknowledged() {
    if (finding) {
      try {
        const updatedFinding = await $pocketbase.collection('nuclei_results').update(finding.id, {
          acknowledged: finding.acknowledged,
        });
        console.log('Acknowledged status updated:', updatedFinding);
        saveMessage = 'Acknowledged status updated successfully';
      } catch (error) {
        console.error('Error updating acknowledged status:', error);
        saveMessage = 'Error updating acknowledged status';
      }
    }
  }

  // Function to handle false positive toggle
  async function toggleFalsePositive() {
    if (finding) {
      try {
        const updatedFinding = await $pocketbase.collection('nuclei_results').update(finding.id, {
          false_positive: finding.false_positive,
        });
        console.log('False positive status updated:', updatedFinding);
        saveMessage = 'False positive status updated successfully';
      } catch (error) {
        console.error('Error updating false positive status:', error);
        saveMessage = 'Error updating false positive status';
      }
    }
  }

  // Function to handle remediated toggle
  async function toggleRemediated() {
    if (finding) {
      try {
        await $pocketbase.collection('nuclei_results').update(finding.id, {
          remediated: finding.remediated
        });
        dispatch('findingUpdated', finding);
      } catch (error) {
        console.error('Error updating remediated status:', error);
        // Revert the toggle if the update fails
        finding.remediated = !finding.remediated;
      }
    }
  }

  // Function to handle content updates from the editor
  function handleNotesChange(newContent: string) {
    notesContent = newContent;
    // Additional logic if needed
  }

  // Function to handle manual toggle by user
  function handleShowNotesToggle() {
    hasUserToggled = true;
  }

  // Update notesContent and reset toggle when finding changes
  $: if (finding && finding.id !== previousFindingId) {
    previousFindingId = finding.id;
    hasUserToggled = false;
    fetchNotes();
  }

  // Function to save notes
  async function saveNotes() {
    if (finding) {
      try {
        // Save the notesContent back to the backend
        await $pocketbase.collection('nuclei_results').update(finding.id, {
          notes: notesContent,
        });
        // Update the local finding object
        finding.notes = notesContent;
        saveMessage = 'Notes saved successfully';
        setTimeout(() => {
          saveMessage = '';
        }, 2000);
      } catch (error) {
        console.error('Error saving notes:', error);
        saveMessage = 'Error saving notes';
        setTimeout(() => {
          saveMessage = '';
        }, 2000);
      }
    }
  }

  function closeModal() {
    open = false;
    // Reset messages upon closing
    saveMessage = '';
    copyMessage = '';
    showNotes = false;
  }

  // Get severity color
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

  // Function to copy text to clipboard
  function copyToClipboard(text: string | undefined) {
    if (!text) return;
    navigator.clipboard.writeText(text);
    copyMessage = 'Copied!';
    setTimeout(() => {
      copyMessage = '';
    }, 2000);
  }

  // Add after the reactive statement
  $: {
    console.log('Finding ID:', finding?.id);
    console.log('showNotes:', showNotes);
    console.log('hasUserToggled:', hasUserToggled);
  }

  function handleChange(event: CustomEvent<string>) {
    console.log('Editor content changed:', event.detail);
    notesContent = event.detail;
  }

  // Add this helper function for formatting dates
  function formatDate(dateString: string | undefined): string {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toLocaleString();
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
  });

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

  // Add these functions after the getLastSeenStatus function
  async function updateSeverityOverride(newSeverity: string | undefined) {
    if (!finding) return;
    
    try {
      const severityOrder = newSeverity ? getSeverityOrder(newSeverity) : undefined;
      const data = {
        severity_override: newSeverity,
        severity_override_order: severityOrder
      };
      
      await $pocketbase.collection('nuclei_results').update(finding.id, data);
      finding.severity_override = newSeverity;
      finding.severity_override_order = severityOrder;
      saveMessage = 'Severity override updated successfully';
      setTimeout(() => {
        saveMessage = '';
      }, 2000);
      dispatch('findingUpdated', finding);
    } catch (error) {
      console.error('Error updating severity override:', error);
      saveMessage = 'Error updating severity override';
      setTimeout(() => {
        saveMessage = '';
      }, 2000);
    }
  }

  function getSeverityOrder(severity: string): number {
    switch (severity.toLowerCase()) {
      case 'critical': return 1;
      case 'high': return 2;
      case 'medium': return 3;
      case 'low': return 4;
      case 'info': return 5;
      default: return 6;
    }
  }

  interface SeverityOption {
    value: string;
    name: string;
    styleClass: string;
  }

  const severityOptions: SeverityOption[] = [
    { value: '', name: 'None', styleClass: 'bg-gray-500 text-white' },
    { value: 'critical', name: 'Critical', styleClass: 'bg-red-600 text-white' },
    { value: 'high', name: 'High', styleClass: 'bg-orange-500 text-white' },
    { value: 'medium', name: 'Medium', styleClass: 'bg-yellow-500 text-white' },
    { value: 'low', name: 'Low', styleClass: 'bg-green-500 text-white' },
    { value: 'info', name: 'Info', styleClass: 'bg-blue-500 text-white' }
  ];

</script>

<Modal bind:open={open} size="xl" class="w-full" autoclose={false} dismissable={false}>
  <div class="relative bg-white rounded-lg shadow dark:bg-gray-700">
    <!-- Modal header -->
    <div class="flex items-start justify-between p-4 border-b rounded-t dark:border-gray-600">
      <h3 class="text-xl font-semibold text-gray-900 dark:text-white">Finding Details</h3>
      <button
        type="button"
        class="text-gray-400 bg-transparent hover:bg-gray-200 hover:text-gray-900 rounded-lg text-sm p-1.5 ml-auto inline-flex items-center dark:hover:bg-gray-600 dark:hover:text-white"
        on:click={closeModal}
      >
        <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20" xmlns="http://www.w3.org/2000/svg">
          <path
            fill-rule="evenodd"
            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
            clip-rule="evenodd"
          />
        </svg>
      </button>
    </div>

    <!-- Modal body -->
    <div class="p-6 space-y-6">
      {#if finding}
        <!-- Status Toggles -->
        <div class="flex gap-4 mb-6">
          <Toggle bind:checked={finding.acknowledged} on:change={toggleAcknowledged}>
            Acknowledged
            {#if finding.acknowledged}
              <div class="status-badge-container ml-2">
                <span class="status-badge acknowledged" title="Acknowledged">✓
                  <span class="tooltip">Acknowledged</span>
                </span>
              </div>
            {/if}
          </Toggle>
          <Toggle bind:checked={finding.false_positive} on:change={toggleFalsePositive}>
            False Positive
            {#if finding.false_positive}
              <div class="status-badge-container ml-2">
                <span class="status-badge false-positive" title="False Positive">❌
                  <span class="tooltip">False Positive</span>
                </span>
              </div>
            {/if}
          </Toggle>
          <Toggle bind:checked={finding.remediated} on:change={toggleRemediated}>
            Remediated
            {#if finding.remediated}
              <div class="status-badge-container ml-2">
                <span class="status-badge remediated" title="Remediated">🛠️
                  <span class="tooltip">Remediated</span>
                </span>
              </div>
            {/if}
          </Toggle>
        </div>

        <!-- Add the styles -->
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

        <!-- Basic Information -->
        <div class="space-y-4">
          <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
            <h4 class="text-lg font-semibold mb-2 text-gray-900 dark:text-white">
              {finding.info?.name || 'Unnamed Finding'}
            </h4>
            <div class="flex items-center gap-4 mb-4">
              {#if finding.severity_override}
                <!-- When there's an override, show it as the main severity -->
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Current Severity:</span>
                  <span class={`inline-block px-2 py-1 rounded ${getSeverityColor(finding.severity_override)}`}>
                    {finding.severity_override}
                  </span>
                  <span class="text-sm text-gray-500">(overridden)</span>
                </div>
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Original Severity:</span>
                  <span class={`inline-block px-2 py-1 rounded ${getSeverityColor(finding.severity)}`}>
                    {finding.severity}
                  </span>
                </div>
              {:else}
                <!-- When there's no override, show original severity -->
                <div class="flex items-center gap-2">
                  <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Severity:</span>
                  <span class={`inline-block px-2 py-1 rounded ${getSeverityColor(finding.severity)}`}>
                    {finding.severity}
                  </span>
                </div>
              {/if}
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium text-gray-500 dark:text-gray-400">Override Severity:</span>
                <Select
                  class="w-32"
                  items={severityOptions}
                  bind:value={finding.severity_override}
                  on:change={async (e) => {
                    await updateSeverityOverride(finding.severity_override);
                    dispatch('findingUpdated', finding);
                  }}
                >
                  <svelte:fragment slot="item" let:item>
                    <span class={`px-2 py-1 rounded ${item.styleClass}`}>
                      {item.name}
                    </span>
                  </svelte:fragment>
                  <svelte:fragment slot="selected" let:item>
                    <span class={`px-2 py-1 rounded ${item.styleClass}`}>
                      {item.name}
                    </span>
                  </svelte:fragment>
                </Select>
              </div>
            </div>
            <p class="text-gray-600 dark:text-gray-300 mb-2">{finding.info?.description || 'No description available'}</p>
            {#if finding.info?.tags && finding.info.tags.length > 0}
              <div class="flex flex-wrap gap-2">
                {#each finding.info.tags as tag}
                  <span class="px-2 py-1 bg-blue-100 text-blue-800 text-sm rounded-full dark:bg-blue-900 dark:text-blue-300">
                    {tag}
                  </span>
                {/each}
              </div>
            {/if}
          </div>

          <!-- Location Information -->
          <div class="grid grid-cols-2 gap-4">
            <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Host</p>
              <p class="text-gray-900 dark:text-white">{finding.host || 'N/A'}</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">IP</p>
              <p class="text-gray-900 dark:text-white">{finding.ip || 'N/A'}</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Matched At</p>
              <p class="text-gray-900 dark:text-white">{formatDate(finding.matched_at)}</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">First Seen</p>
              <p class="text-gray-900 dark:text-white">{formatDate(finding.timestamp)}</p>
            </div>
            <div class="bg-gray-50 dark:bg-gray-800 p-4 rounded-lg">
              <p class="text-sm font-medium text-gray-500 dark:text-gray-400">Last Seen</p>
              <p class="text-gray-900 dark:text-white">
                {formatDate(finding.last_seen)}
                {#if getLastSeenStatus(finding.last_seen, finding.timestamp).isStale}
                  <span 
                    class="text-yellow-500 ml-2" 
                    title={`Finding is stale (${getLastSeenStatus(finding.last_seen, finding.timestamp).daysAgo} days - Threshold: ${staleThresholdDays} days)`}
                  >
                    ⚠️
                  </span>
                  <span class="text-sm text-yellow-600 dark:text-yellow-400 ml-2">
                    ({getLastSeenStatus(finding.last_seen, finding.timestamp).daysAgo} days ago)
                  </span>
                {/if}
              </p>
            </div>
          </div>
        </div>

        <!-- Technical Details -->
        <Accordion>
          {#if finding.extracted_results && finding.extracted_results.length > 0}
            <AccordionItem>
              <svelte:fragment slot="header">
                <span>Extracted Results</span>
              </svelte:fragment>
              <div class="terminal-container relative">
                <Button
                  class="absolute top-2 right-2 z-10"
                  size="xs"
                  on:click={() => copyToClipboard(JSON.stringify(finding.extracted_results, null, 2))}
                >
                  Copy
                </Button>
                <Xterm
                  options={terminalOptions}
                  on:load={onExtractedResultsLoad}
                />
              </div>
            </AccordionItem>
          {/if}

          {#if parsedReferences && parsedReferences.length > 0}
            <AccordionItem>
              <svelte:fragment slot="header">
                <span>References</span>
              </svelte:fragment>
              <div class="space-y-2">
                {#each parsedReferences as reference}
                  <a
                    href={reference}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="block text-blue-600 hover:text-blue-800 dark:text-blue-400 dark:hover:text-blue-300 break-all"
                  >
                    {reference}
                  </a>
                {/each}
              </div>
            </AccordionItem>
          {/if}

          {#if finding.curl_command}
            <AccordionItem>
              <svelte:fragment slot="header">
                <span>cURL Command</span>
              </svelte:fragment>
              <div class="terminal-container relative">
                <Button
                  class="absolute top-2 right-2 z-10"
                  size="xs"
                  on:click={() => copyToClipboard(finding.curl_command)}
                >
                  Copy
                </Button>
                <Xterm
                  options={terminalOptions}
                  on:load={onCurlLoad}
                />
              </div>
            </AccordionItem>
          {/if}

          {#if finding.request}
            <AccordionItem>
              <svelte:fragment slot="header">
                <span>Request</span>
              </svelte:fragment>
              <div class="terminal-container relative">
                <Button
                  class="absolute top-2 right-2 z-10"
                  size="xs"
                  on:click={() => copyToClipboard(finding.request)}
                >
                  Copy
                </Button>
                <Xterm
                  options={terminalOptions}
                  on:load={onRequestLoad}
                />
              </div>
            </AccordionItem>
          {/if}

          {#if finding.response}
            <AccordionItem>
              <svelte:fragment slot="header">
                <span>Response</span>
              </svelte:fragment>
              <div class="terminal-container relative">
                <Button
                  class="absolute top-2 right-2 z-10"
                  size="xs"
                  on:click={() => copyToClipboard(finding.response)}
                >
                  Copy
                </Button>
                <Xterm
                  options={terminalOptions}
                  on:load={onResponseLoad}
                />
              </div>
            </AccordionItem>
          {/if}
        </Accordion>

        <!-- Notes Section -->
        <div class="mt-6">
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-lg font-semibold">Notes</h4>
            <Button size="xs" on:click={() => (showNotes = !showNotes)}>
              {showNotes ? 'Hide Notes' : 'Show Notes'}
            </Button>
          </div>
          {#if showNotes}
            <div class="mt-2">
              <BlockNoteEditor
                bind:content={notesContent}
                on:change={(e) => handleNotesChange(e.detail)}
              />
              {#if saveMessage}
                <p class="text-sm text-green-600 mt-2">{saveMessage}</p>
              {/if}
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</Modal>

<style>
  .terminal-container {
    position: relative;
    height: 300px;
    background-color: #1e1e1e;
    border: 1px solid #333;
  }

  :global(.xterm) {
    height: 100%;
    opacity: 1 !important;
  }

  :global(.xterm-viewport) {
    overflow-y: auto !important;
  }

  .copy-message {
    position: absolute;
    top: 40px; /* Adjust as needed */
    right: 10px;
    z-index: 10;
    background-color: #d1fae5; /* Light green background */
    color: #065f46; /* Dark green text */
    padding: 5px 10px;
    border-radius: 5px;
    font-size: 0.875rem; /* text-sm */
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .copy-message-global {
    background-color: #d1fae5; /* Light green background */
    color: #065f46; /* Dark green text */
    padding: 10px;
    border-radius: 5px;
    font-size: 1rem;
    margin-bottom: 10px;
    text-align: center;
  }
</style>
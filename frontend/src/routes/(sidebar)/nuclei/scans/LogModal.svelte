<script lang="ts">
    import { Xterm, XtermAddon } from '@battlefieldduck/xterm-svelte';
    import type { ITerminalOptions, ITerminalInitOnlyOptions, Terminal } from '@battlefieldduck/xterm-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { Button, Heading, Modal } from 'flowbite-svelte';
    import { onMount, onDestroy } from 'svelte';
  
    export let open: boolean;
    export let scanId: string;
  
    let options: ITerminalOptions & ITerminalInitOnlyOptions = {
      fontFamily: 'Consolas, Monaco, "Courier New", monospace',
      fontSize: 14,
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
      },
      cursorBlink: true,
    };
  
    let terminalInstance: Terminal;
    let fitAddon: any;
    let unsubscribe: () => void;
    let lastLogCount = 0;
  
    async function fetchLogs() {
      try {
        const record = await $pocketbase.collection('nuclei_scans').getOne(scanId);
        const logs = record.ansible_logs || [];
        
        // Clear terminal
        terminalInstance.clear();
        
        // Process all logs
        logs.forEach((logEntry: any) => {
          const logContent = logEntry.content || '';
          if (logContent) {
            terminalInstance.writeln(logContent);
          }
        });
  
        lastLogCount = logs.length;
        
        // Scroll to bottom
        terminalInstance.scrollToBottom();
      } catch (error) {
        console.error('Error fetching logs:', error);
        terminalInstance.writeln('\x1b[1;31mError fetching logs. Please try again.\x1b[0m');
      }
    }
  
    async function onLoad(event: CustomEvent<{ terminal: Terminal }>) {
      terminalInstance = event.detail.terminal;
  
      // Load the FitAddon
      fitAddon = new (await XtermAddon.FitAddon()).FitAddon();
      terminalInstance.loadAddon(fitAddon);
      fitAddon.fit();
  
      // Initial fetch
      await fetchLogs();
  
      // Subscribe to realtime updates
      try {
        unsubscribe = await $pocketbase
          .collection('nuclei_scans')
          .subscribe(scanId, async (data) => {
            if (data.record.ansible_logs) {
              const logs = data.record.ansible_logs;
              if (logs.length > lastLogCount) {
                // Process only the new logs
                const newLogs = logs.slice(lastLogCount);
                
                newLogs.forEach((logEntry: any) => {
                  const logContent = logEntry.content || '';
                  if (logContent) {
                    terminalInstance.writeln(logContent);
                  }
                });
  
                lastLogCount = logs.length;
                
                // Scroll to bottom
                terminalInstance.scrollToBottom();
              }
            }
          });
      } catch (error) {
        console.error('Error setting up subscription:', error);
        terminalInstance.writeln('\x1b[1;31mError setting up real-time updates. Please try refreshing.\x1b[0m');
      }
    }
  
    // Clean up subscription when component is destroyed
    onDestroy(() => {
      if (unsubscribe) {
        unsubscribe();
      }
    });
  
    // Reset log count and refetch when modal is opened
    $: if (open) {
      lastLogCount = 0;
      if (terminalInstance) {
        fetchLogs();
      }
    }
  
    // Refit terminal when modal opens
    $: if (open && terminalInstance && fitAddon) {
        fitAddon.fit();
    }
  
    function copyTerminalContent() {
      const content = terminalInstance.buffer.active.getLine(0)?.translateToString(true) || '';
      navigator.clipboard.writeText(content).catch(err => {
        console.error('Could not copy text: ', err);
      });
    }
  </script>
  
  <Modal bind:open={open} size="xl" placement="center">
    <div class="flex flex-col items-center">
      <Heading tag="h3" class="text-lg leading-6 font-medium text-gray-900 text-center mt-4">
        Log Details
      </Heading>
      <div class="terminal-container mt-4">
        <Xterm {options} on:load={onLoad} />
      </div>
      <div class="flex justify-between w-full mt-4 px-6">
        <Button color="red" on:click={() => (open = false)}>Close</Button>
        <Button color="green" on:click={copyTerminalContent}>Copy</Button>
      </div>
    </div>
  </Modal>
  
  <style>
    .terminal-container {
      width: 100%;
      max-height: 500px;
      overflow-y: auto;
      background-color: #1e1e1e;
      border-radius: 5px;
      padding: 10px;
    }
  </style>
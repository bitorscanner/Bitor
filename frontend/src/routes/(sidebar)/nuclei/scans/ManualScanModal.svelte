<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { Modal, Button, Alert } from 'flowbite-svelte';
  import { InfoCircleSolid } from 'flowbite-svelte-icons';
  import { pocketbase } from '@lib/stores/pocketbase';
  import type { RecordModel } from 'pocketbase';

  interface Client {
    id: string;
    name: string;
  }

  export let open = false;
  let clients: Client[] = [];
  let selectedClientId = '';
  let file: File | null = null;
  let uploadProgress = 0;
  let showAlert = false;
  let currentChunk = 0;
  let totalChunks = 0;
  const chunkSize = 5 * 1024 * 1024; // 5MB chunks
  const dispatch = createEventDispatcher();

  async function fetchClients() {
    try {
      const records = await $pocketbase.collection('clients').getFullList();
      clients = records.map((record: RecordModel) => ({
        id: record.id,
        name: record.name
      }));
    } catch (error) {
      console.error('Error fetching clients:', error);
    }
  }

  onMount(() => {
    fetchClients();
  });

  function handleFileChange(event: Event) {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      file = input.files[0];
    }
  }

  async function uploadChunk(file: File, scanId: string, chunk: Blob, chunkIndex: number, totalChunks: number) {
    const formData = new FormData();
    formData.append('file', chunk, file.name);
    formData.append('scan_id', scanId);
    formData.append('chunk_index', chunkIndex.toString());
    formData.append('total_chunks', totalChunks.toString());
    formData.append('client_id', selectedClientId);

    const maxRetries = 3;
    let retryCount = 0;

    while (retryCount < maxRetries) {
      try {
        const token = $pocketbase.authStore.token;
        if (!token) {
          throw new Error('No authentication token found. Please log in again.');
        }

        const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/scan/import-scan-results`, {
          method: 'POST',
          headers: {
            'Authorization': token,
          },
          body: formData
        });

        if (response.ok) {
          return response;
        }

        if (response.status === 401) {
          // Redirect to login if unauthorized
          window.location.href = '/authentication/sign-in';
          throw new Error('Authentication failed. Please log in again.');
        }

        if (response.status === 403) {
          throw new Error('Authentication failed. Please check if you are logged in and try again.');
        }

        throw new Error(`Upload failed with status: ${response.status}`);
      } catch (error) {
        retryCount++;
        if (retryCount === maxRetries) {
          throw error;
        }
        // Wait before retrying (exponential backoff)
        await new Promise(resolve => setTimeout(resolve, Math.pow(2, retryCount) * 1000));
      }
    }
  }

  async function handleImport() {
    if (!file || !selectedClientId) {
      alert('Please select a file and a client.');
      return;
    }

    try {
      const scanId = generateScanId();
      const client = clients.find(c => c.id === selectedClientId);
      const clientName = client ? client.name : 'Unknown Client';

      // Validate file size
      const maxFileSize = 1024 * 1024 * 1024; // 1GB max
      if (file.size > maxFileSize) {
        alert('File size exceeds maximum limit of 1GB');
        return;
      }

      const newScan = await $pocketbase.collection('nuclei_scans').create({
        name: scanId,
        client: selectedClientId,
        status: 'Manual',
        created_by: $pocketbase.authStore.model?.id || ''
      });

      const newScanId = newScan.id;
      
      // Calculate total chunks
      totalChunks = Math.ceil(file.size / chunkSize);
      currentChunk = 0;
      let failedChunks = [];

      // Upload chunks
      for (let start = 0; start < file.size; start += chunkSize) {
        const chunk = file.slice(start, Math.min(start + chunkSize, file.size));
        currentChunk++;
        
        try {
          await uploadChunk(file, newScanId, chunk, currentChunk, totalChunks);
          // Update progress
          uploadProgress = (currentChunk / totalChunks) * 100;
        } catch (error) {
          console.error(`Error uploading chunk ${currentChunk}:`, error);
          failedChunks.push(currentChunk);
          
          // If we have failed chunks, try to retry them
          if (failedChunks.length > 0) {
            const retryResult = await retryFailedChunks(file, newScanId, failedChunks);
            if (!retryResult) {
              throw new Error('Failed to upload some chunks after retries');
            }
          }
        }
      }

      console.log('File upload completed successfully');
      dispatch('import');
      open = false;
      showAlert = true;
    } catch (error: unknown) {
      console.error('Error during file upload:', error);
      alert(error instanceof Error ? error.message : 'Upload failed. Please try again.');
    }
  }

  async function retryFailedChunks(file: File, scanId: string, failedChunks: number[]) {
    for (const chunkIndex of failedChunks) {
      const start = (chunkIndex - 1) * chunkSize;
      const chunk = file.slice(start, Math.min(start + chunkSize, file.size));
      
      try {
        await uploadChunk(file, scanId, chunk, chunkIndex, totalChunks);
        // Remove from failed chunks if successful
        failedChunks = failedChunks.filter(index => index !== chunkIndex);
      } catch (error) {
        console.error(`Failed to retry chunk ${chunkIndex}:`, error);
        return false;
      }
    }
    return failedChunks.length === 0;
  }

  function generateScanId() {
    return 'scan_manual_json_' + Date.now();
  }
</script>

<Modal bind:open size="md">
  <div slot="header">
    <h3 class="text-lg font-medium text-gray-900 dark:text-white">
      Manually Add Scan Results
    </h3>
  </div>
  <div class="flex flex-col gap-4 p-4">
    <div>
      <label for="jsonFile" class="block mb-2 text-sm font-medium text-gray-900 dark:text-gray-300">
        Upload JSON File
      </label>
      <input
        type="file"
        id="jsonFile"
        accept=".json"
        on:change={handleFileChange}
        class="block w-full text-sm text-gray-900 bg-gray-50 rounded-lg border border-gray-300 cursor-pointer"
      />
    </div>
    <div>
      <label for="clientSelect" class="block mb-2 text-sm font-medium text-gray-900 dark:text-gray-300">
        Select Client
      </label>
      <select
        id="clientSelect"
        bind:value={selectedClientId}
        class="block w-full p-2 text-sm text-gray-900 bg-gray-50 rounded-lg border border-gray-300"
      >
        <option value="" disabled>Select a client</option>
        {#each clients as client}
          <option value={client.id}>{client.name}</option>
        {/each}
      </select>
    </div>
    {#if uploadProgress > 0}
      <div class="w-full">
        <div class="flex justify-between mb-1">
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            Uploading... ({currentChunk} of {totalChunks} chunks)
          </span>
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {Math.round(uploadProgress)}%
          </span>
        </div>
        <div class="w-full bg-gray-200 rounded-full h-2.5 dark:bg-gray-700">
          <div 
            class="bg-blue-600 h-2.5 rounded-full transition-all duration-300" 
            style="width: {uploadProgress}%"
          ></div>
        </div>
      </div>
    {/if}
  </div>
  <div slot="footer">
    <Button on:click={handleImport}>Import</Button>
    <Button color="alternative" on:click={() => open = false}>Cancel</Button>
  </div>
</Modal>

{#if showAlert}
  <Alert color="green" on:close={() => showAlert = false}>
    <InfoCircleSolid slot="icon" class="w-5 h-5" />
    <span class="font-medium">Success alert!</span> File uploaded successfully. The server is processing the results.
  </Alert>
{/if}

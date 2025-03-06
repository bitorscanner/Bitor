<script lang="ts">
    import { Modal, Button } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    export let open: boolean;
    export let scanId: string;

    const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;

    // Assuming you have a way to get the session token, e.g., from a store or a global variable
    const sessionToken = $pocketbase.authStore.token;

    async function downloadFile(type: 'full' | 'small') {
        console.log('Downloading file for scan:', scanId);
        try {
            console.log('Making request with body:', JSON.stringify({
                scan_id: scanId,
                file_type: type,
            }));
            
            const response = await fetch(`${API_BASE_URL}/api/scan/signed-url`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${sessionToken}`,
                },
                body: JSON.stringify({
                    scan_id: scanId,
                    file_type: type,
                }),
            });
            
            console.log('Response status:', response.status);
            const data = await response.json();
            console.log('Response data:', data);

            if (response.ok) {
                window.open(data.signedUrl, '_blank');
            } else {
                console.error('Error fetching signed URL:', data.error);
            }
        } catch (error) {
            console.error('Error downloading file:', error);
        }
    }

    // Add reactive statement to log scanId changes
    $: console.log('scanId changed:', scanId);
</script>

<Modal bind:open={open} size="lg" placement="center">
    <div class="bg-white dark:bg-gray-800 px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
        <h3 class="text-lg leading-6 font-medium text-gray-900 dark:text-white">
            Download Results
        </h3>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">
            You can download the results of the scan as a ZIP file.
        </p>
        <Button color="primary" class="mt-4 mr-2" on:click={() => downloadFile('full')}>Download Full ZIP</Button>
        <Button color="primary" class="mt-4" on:click={() => downloadFile('small')}>Download Small ZIP</Button>
    </div>
    <div class="bg-gray-50 dark:bg-gray-700 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
        <Button color="alternative" on:click={() => open = false}>Close</Button>
    </div>
</Modal>

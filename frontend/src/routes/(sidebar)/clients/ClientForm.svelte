<script lang="ts">
    import { createEventDispatcher, onMount } from 'svelte';
    import { Input, Label, Modal, Select, Button } from 'flowbite-svelte';
    import { uniqueNamesGenerator, adjectives, colors, animals } from 'unique-names-generator';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { getFavicon } from '$lib/utils/favicon';
  
    export let open = false;
    export let client: {
      name: string;
      logo?: string;
      hidden_name: string;
      id: string;
      group?: ClientGroup | null;
      homepage?: string;
      favicon?: string;
    } | null = null;

    interface ClientGroup {
      id: string;
      name: string;
    }

    export let mode: 'add' | 'edit' = 'add';
    export let size: string = 'lg';
  
    let clientName = '';
    let hiddenName = '';
    let selectedGroupId = '';
    let homepage = '';
    let clientGroups: ClientGroup[] = [];
  
    const dispatch = createEventDispatcher();
  
    let isInitialized = false;
  
    onMount(async () => {
      try {
        const result = await $pocketbase.collection('client_groups').getList();
        clientGroups = result.items.map(group => ({
          id: group.id,
          name: group.name
        }));
      } catch (error) {
        console.error('Error fetching client groups:', error);
      }
    });
  
    $: if (open && !isInitialized) {
      if (client) {
        console.log('client object:', client);
        clientName = client.name || '';
        hiddenName = client.hidden_name || '';
        selectedGroupId = client.group?.id || '';
        homepage = client.homepage || '';
        console.log('selectedGroupId after setting:', selectedGroupId);
      } else {
        clientName = '';
        hiddenName = '';
        selectedGroupId = '';
        homepage = '';
        console.log('Adding new client - form fields reset');
      }
      isInitialized = true;
    }
  
    $: if (!open) {
      isInitialized = false;
    }
  
    function generateCodeName() {
      hiddenName = uniqueNamesGenerator({
        dictionaries: [adjectives, colors, animals],
        separator: '-',
        length: 3,
      });
    }
  
    async function handleSave() {
      try {
        console.log('Starting save process...');
        const formData = new FormData();
        
        // Add basic fields
        formData.append('name', clientName);
        formData.append('hidden_name', hiddenName);
        if (selectedGroupId) {
          formData.append('group', selectedGroupId);
        }
        if (homepage) {
          formData.append('homepage', homepage);
        }

        // Save to PocketBase
        let savedClient;
        if (mode === 'edit' && client) {
          console.log('Updating existing client...');
          savedClient = await $pocketbase.collection('clients').update(client.id, formData);
        } else {
          console.log('Creating new client...');
          savedClient = await $pocketbase.collection('clients').create(formData);
        }
        
        console.log('Raw saved client response:', savedClient);
        
        // After saving, fetch the favicon if we have a homepage
        if (homepage && savedClient.id) {
          console.log('Fetching favicon for homepage:', homepage);
          const faviconDataUri = await getFavicon(homepage, savedClient.id);
          
          if (faviconDataUri) {
            console.log('Successfully fetched favicon');
          } else {
            console.warn('Failed to fetch favicon');
          }
        }
        
        dispatch('save', {
          name: clientName,
          hidden_name: hiddenName,
          group: selectedGroupId,
          homepage: homepage
        });
        
        open = false;
        clientName = '';
        hiddenName = '';
        selectedGroupId = '';
        homepage = '';
      } catch (error) {
        console.error('Save failed:', error);
        if (error && typeof error === 'object' && 'response' in error && error.response instanceof Response) {
          try {
            const errorData = await error.response.json();
            console.error('Error details:', errorData);
          } catch (e) {
            console.error('Could not parse error response');
          }
        }
      }
    }
  </script>

  <Modal bind:open size="lg" title={mode === 'edit' ? "Edit Client" : "Add New Client"}>
    <form on:submit|preventDefault={handleSave} class="space-y-6">
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Client Name</span>
        <Input bind:value={clientName} placeholder="Enter client name" required />
      </Label>
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Homepage</span>
        <div class="flex items-center space-x-2">
          <Input 
            bind:value={homepage} 
            placeholder="Enter company homepage (e.g., https://example.com)" 
            type="url"
          />
        </div>
      </Label>
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Hidden Name</span>
        <div class="flex items-center space-x-2">
          <Input bind:value={hiddenName} placeholder="Enter or generate hidden name" required />
          <Button on:click={generateCodeName} type="button">Generate</Button>
        </div>
      </Label>
      <Label class="space-y-2">
        <span class="text-gray-700 dark:text-gray-300">Client Group</span>
        <Select
          placeholder="Select a group"
          bind:value={selectedGroupId}
          on:change={() => console.log('Group changed to:', selectedGroupId)}
        >
          {#each clientGroups as group}
            <option value={group.id}>{group.name}</option>
          {/each}
        </Select>
      </Label>
      <Button type="submit" class="w-full text-white py-2">
        {mode === 'edit' ? "Update" : "Save"}
      </Button>
    </form>
  </Modal>
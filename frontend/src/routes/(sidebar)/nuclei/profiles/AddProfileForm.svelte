<!-- AddProfileForm.svelte -->
<script lang="ts">
    import { Button, Input, Label, Modal } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { onMount, createEventDispatcher } from 'svelte';
    import type * as Monaco from 'monaco-editor/esm/vs/editor/editor.api';
    import yaml from 'js-yaml';
  
    let editor: Monaco.editor.IStandaloneCodeEditor;
    let monaco: typeof Monaco;
    let editorContainer: HTMLElement;
  
    export let open: boolean = false;
    export let currentProfileData: Record<string, any> = { name: '', profile: '', raw_yaml: '' };
    export let fetchProfiles: () => void;
  
    const dispatch = createEventDispatcher();
  
    onMount(async () => {
      try {
        monaco = (await import('./monaco')).default;
      } catch (error) {
        console.error('Error loading Monaco:', error);
      }
    });
  
    // Reactive statement to handle editor initialization and updates
    $: if (open && monaco && editorContainer) {
      if (!editor) {
        const yamlData = typeof currentProfileData.raw_yaml === 'string' ? currentProfileData.raw_yaml : '';
        editor = monaco.editor.create(editorContainer, {
          value: yamlData || '',
          language: 'yaml',
          automaticLayout: true,
          fontSize: 16,
          minimap: {
            enabled: false
          }
        });
      } else {
        const yamlData = typeof currentProfileData.raw_yaml === 'string' ? currentProfileData.raw_yaml : '';
        editor.setValue(yamlData || '');
      }
    } else if (!open && editor) {
      editor.dispose();
      editor = null;
    }
  
    let errorMessage: string = '';
  
    async function saveProfile() {
      errorMessage = '';
      try {
        const yamlContent = editor.getValue();
        const jsonData = yaml.load(yamlContent);
  
        const formData = new FormData();
        formData.append('name', currentProfileData.name);
        formData.append('profile', JSON.stringify(jsonData));
        formData.append('raw_yaml', yamlContent);
  
        if (currentProfileData.id) {
          await $pocketbase.collection('nuclei_profiles').update(currentProfileData.id, formData);
        } else {
          await $pocketbase.collection('nuclei_profiles').create(formData);
        }
  
        open = false;
  
        // Dispatch an event to refresh the profiles
        dispatch('refreshProfiles');
      } catch (error) {
        errorMessage = 'Error saving profile: ' + error.message;
      }
    }
  </script>
  
  <Modal bind:open size="2xl" placement="center">
    <div slot="header">
      {#if currentProfileData.id}
        Edit Profile
      {:else}
        Add Profile
      {/if}
    </div>
    <div class="p-4 space-y-4">
      <Label for="name">Profile Name</Label>
      <Input id="name" bind:value={currentProfileData.name} required />
  
      <Label for="data">YAML Data</Label>
      <div id="container" bind:this={editorContainer} style="height: 400px;"></div>
  
      {#if errorMessage}
        <p class="text-red-500">{errorMessage}</p>
      {/if}
    </div>
    <div slot="footer" class="flex justify-end space-x-4">
      <Button on:click={saveProfile}>
        {#if currentProfileData.id}
          Save Changes
        {:else}
          Save Profile
        {/if}
      </Button>
      <Button color="gray" on:click={() => (open = false)}>Cancel</Button>
    </div>
  </Modal>

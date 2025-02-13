<script lang="ts">
    import { onMount } from 'svelte';
    import {
      Button,
      Table,
      TableHead,
      TableHeadCell,
      TableBody,
      TableBodyRow,
      TableBodyCell,
      Heading,
      Modal,
      Input,
      Label,
      Alert
    } from 'flowbite-svelte';
    import Card from '../../utils/Card.svelte';
    import { pocketbase } from '$lib/stores/pocketbase';
    import AddProfileForm from './AddProfileForm.svelte';
    import ProfileBuilderModal from './ProfileBuilderModal.svelte';
    import type { RecordModel } from 'pocketbase';
  
    interface Profile extends RecordModel {
      name: string;
      profile: Record<string, any>;
      raw_yaml: string;
    }
  
    const defaultConfig = {
      'max-host-error': 500,
      'bulk-size': 2000,
      'concurrency': 32,
      'error-log': 'nuclei-errors.log',
      'stats': true,
      'no-color': true
    };
  
    let showProfileForm = false;
    let showProfileBuilder = false;
    let currentProfileData: Record<string, any> = {
      name: '',
      profile: '',
      id: '',
      raw_yaml: ''
    };
    let profiles: Profile[] = [];
  
    async function createDefaultProfile() {
      try {
        // Check if default profile already exists
        const existingDefault = await $pocketbase.collection('nuclei_profiles').getFirstListItem(`name="Default Profile"`).catch(() => null);
        
        if (!existingDefault) {
          await $pocketbase.collection('nuclei_profiles').create({
            name: 'Default Profile',
            profile: defaultConfig,
            raw_yaml: `# Default Nuclei Configuration
# These settings provide a balanced configuration for most scans

max-host-error: 500
bulk-size: 2000
concurrency: 32
error-log: nuclei-errors.log
stats: true
no-color: true`,
            is_default: true
          });
          
          await fetchProfiles();
        }
      } catch (error) {
        console.error('Error creating default profile:', error);
      }
    }
  
    async function fetchProfiles() {
      try {
        const result = await $pocketbase.collection('nuclei_profiles').getList<Profile>();
        profiles = result.items;
      } catch (error) {
        console.error('Error fetching profiles:', error);
      }
    }
  
    onMount(async () => {
      await createDefaultProfile();
      await fetchProfiles();
    });
  
    function openAddProfileModal() {
      currentProfileData = { name: '', profile: '', id: '', raw_yaml: '' };
      showProfileForm = true;
    }
  
    function openEditModal(profile: Profile) {
      currentProfileData = { ...profile };
      showProfileForm = true;
    }
  
    async function deleteProfile(id: string) {
      if (confirm('Are you sure you want to delete this profile?')) {
        try {
          await $pocketbase.collection('nuclei_profiles').delete(id);
          fetchProfiles();
        } catch (error) {
          console.error('Error deleting profile:', error);
        }
      }
    }
  
    async function addOfficialProfiles() {
      try {
        const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/profiles/add-official`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${$pocketbase.authStore.token}`
          }
        });
  
        if (!response.ok) {
          throw new Error('Failed to add official profiles');
        }
  
        await fetchProfiles();
      } catch (error) {
        console.error('Error adding official profiles:', error);
      }
    }
  </script>
  
<main class="p-4">
  <Card>
    <div class="space-y-8">
      <!-- Heading for Nuclei Profiles -->
      <Heading tag="h2" customSize="text-xl font-semibold mb-4">Nuclei Profiles</Heading>

      <!-- Buttons to Add Profile -->
      <div class="flex space-x-4 mb-4">
        <Button on:click={openAddProfileModal}>Add Profile (YAML)</Button>
        <Button on:click={() => (showProfileBuilder = true)}>Profile Builder</Button>
        <Button on:click={addOfficialProfiles} color="alternative">Add Official Profiles</Button>
      </div>

      <!-- Profiles Table -->
      <div>
        <Table>
          <TableHead>
            {#each ['Name', 'Created', 'Actions'] as title}
              <TableHeadCell>{title}</TableHeadCell>
            {/each}
          </TableHead>
          <TableBody>
            {#each profiles as profile}
              <TableBodyRow>
                <TableBodyCell>{profile.name}</TableBodyCell>
                <TableBodyCell>{new Date(profile.created).toLocaleDateString()}</TableBodyCell>
                <TableBodyCell class="space-x-2">
                  <Button size="sm" on:click={() => openEditModal(profile)}>Edit</Button>
                  <Button size="sm" color="red" on:click={() => deleteProfile(profile.id)}>Delete</Button>
                </TableBodyCell>
              </TableBodyRow>
            {/each}
          </TableBody>
        </Table>
      </div>

      <!-- Add/Edit Profile Modal -->
      {#if showProfileForm}
        <AddProfileForm
          bind:open={showProfileForm}
          bind:currentProfileData
          fetchProfiles={fetchProfiles}
        />
      {/if}

      <!-- Profile Builder Modal -->
      {#if showProfileBuilder}
        <ProfileBuilderModal bind:open={showProfileBuilder} />
      {/if}
    </div>
  </Card>
</main>
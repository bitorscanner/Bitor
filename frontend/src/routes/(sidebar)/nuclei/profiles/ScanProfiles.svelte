<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Button,
    Table,
    TableBody,
    TableBodyCell,
    TableBodyRow,
    TableHead,
    TableHeadCell,
    Modal,
    Input,
    Label,
    Heading,
    Toggle,
    Select
  } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import Card from '@utils/Card.svelte';
  import { ExclamationCircleOutline } from 'flowbite-svelte-icons';
  import {
    SiAmazon,
    SiDigitalocean,
    SiAmazons3
  } from '@icons-pack/svelte-simple-icons';
  import type { RecordModel } from 'pocketbase';

  interface Provider {
    id: string;
    name: string;
    provider_type: string;
    use: string[];
    settings?: {
      region?: string;
      size?: string;
      [key: string]: any;
    };
  }

  interface Profile {
    id: string;
    name: string;
    description: string;
    nuclei_interact: string;
    vm_provider: string;
    state_bucket: string;
    scan_bucket: string;
    default: boolean;
    vm_size: string;
    expand?: {
      nuclei_interact?: { name: string };
      vm_provider?: Provider;
      state_bucket?: Provider;
      scan_bucket?: Provider;
    };
  }

  interface InteractServer {
    id: string;
    name: string;
  }

  interface DropletSize {
    value: string;
    name: string;
  }

  const providerIcons = {
    aws: SiAmazon,
    digitalocean: SiDigitalocean,
    s3: SiAmazons3
  } as const;

  function getProviderIcon(type: string) {
    return providerIcons[type.toLowerCase() as keyof typeof providerIcons] || null;
  }

  let profiles: Profile[] = [];
  let newProfile: Profile = {
    id: '',
    name: '',
    description: '',
    nuclei_interact: '',
    vm_provider: '',
    state_bucket: '',
    scan_bucket: '',
    default: false,
    vm_size: ''
  };
  let showModal = false;
  let isEditing = false;

  // Collections for relations
  let interactServers: InteractServer[] = [];
  let vmProviders: Provider[] = [];
  let stateBuckets: Provider[] = [];
  let scanBuckets: Provider[] = [];
  let dropletSizes: DropletSize[] = [];

  async function fetchProfiles() {
    try {
      const result = await $pocketbase.collection('scan_profiles').getFullList({
        expand: 'nuclei_interact,vm_provider,state_bucket,scan_bucket'
      });
      profiles = result as unknown as Profile[];
    } catch (error) {
      console.error('Error fetching profiles:', error);
    }
  }

  async function fetchRelations() {
    try {
      // Fetch interact servers
      const interactResult = await $pocketbase.collection('nuclei_interact').getFullList();
      interactServers = interactResult.map(record => ({
        id: record.id,
        name: record.name
      }));

      // Fetch providers
      const providersResult = await $pocketbase.collection('providers').getFullList();
      const providers = providersResult.map(record => ({
        id: record.id,
        name: record.name,
        provider_type: record.provider_type,
        use: record.use || [],
        settings: record.settings || {}
      }));
      
      // Filter providers based on their use field
      vmProviders = providers.filter(provider => 
        provider.use && provider.use.includes('compute')
      );
      
      stateBuckets = providers.filter(provider => 
        provider.use && provider.use.includes('terraform_storage')
      );
      
      scanBuckets = providers.filter(provider => 
        provider.use && provider.use.includes('scan_storage')
      );

    } catch (error) {
      console.error('Error fetching relations:', error);
    }
  }

  // Function to fetch droplet sizes
  async function fetchDropletSizes(providerId: string) {
    try {
      const provider = await $pocketbase.collection('providers').getOne(providerId) as unknown as Provider;
      if (!provider.settings?.region) {
        console.error('Provider has no region configured');
        return;
      }

      const baseUrl = `${import.meta.env.VITE_API_BASE_URL}/api/providers/digitalocean`;
      const response = await fetch(`${baseUrl}/sizes?providerId=${providerId}&region=${provider.settings.region}`, {
        headers: {
          'Authorization': `Bearer ${$pocketbase.authStore.token}`,
          'Content-Type': 'application/json'
        }
      });

      if (!response.ok) {
        throw new Error('Failed to fetch droplet sizes');
      }

      const data = await response.json();
      dropletSizes = data.map((size: any) => ({
        value: size.slug,
        name: `${size.slug} (${size.vcpus} vCPUs, ${size.memory/1024}GB RAM, ${size.disk}GB SSD) - $${size.price_monthly}/mo`
      }));

      // Set default size from provider settings
      if (provider.settings?.size) {
        newProfile.vm_size = provider.settings.size;
      } else if (dropletSizes.length > 0) {
        newProfile.vm_size = dropletSizes[0].value;
      }
    } catch (error) {
      console.error('Error fetching droplet sizes:', error);
      dropletSizes = [];
    }
  }

  // Watch for changes in vm_provider
  $: {
    if (newProfile.vm_provider) {
      const selectedProvider = vmProviders.find(p => p.id === newProfile.vm_provider);
      if (selectedProvider?.provider_type.toLowerCase() === 'digitalocean') {
        fetchDropletSizes(selectedProvider.id);
      } else {
        dropletSizes = [];
        newProfile.vm_size = '';
      }
    }
  }

  onMount(() => {
    fetchProfiles();
    fetchRelations();
  });

  async function addProfile() {
    try {
      // Handle setting default
      if (newProfile.default) {
        // Unset default on all other profiles
        await Promise.all(
          profiles
            .filter((p) => p.default)
            .map((p) =>
              $pocketbase.collection('scan_profiles').update(p.id, { default: false })
            )
        );
      }

      await $pocketbase.collection('scan_profiles').create(newProfile);
      fetchProfiles();
      resetNewProfile();
      showModal = false;
    } catch (error) {
      console.error('Error adding profile:', error);
    }
  }

  async function removeProfile(id: string) {
    try {
      await $pocketbase.collection('scan_profiles').delete(id);
      fetchProfiles();
    } catch (error) {
      console.error('Error removing profile:', error);
    }
  }

  function resetNewProfile() {
    newProfile = {
      id: '',
      name: '',
      description: '',
      nuclei_interact: '',
      vm_provider: '',
      state_bucket: '',
      scan_bucket: '',
      default: false,
      vm_size: ''
    };
  }

  async function toggleDefault(profile: Profile) {
    try {
      if (!profile.default) {
        // Unset default on all other profiles
        await Promise.all(
          profiles
            .filter((p) => p.default)
            .map((p) =>
              $pocketbase.collection('scan_profiles').update(p.id, { default: false })
            )
        );
        // Set default on selected profile
        await $pocketbase.collection('scan_profiles').update(profile.id, { default: true });
      } else if (profiles.length > 1) {
        // If there are more than one profile, allow unsetting default
        await $pocketbase.collection('scan_profiles').update(profile.id, { default: false });
      }
      fetchProfiles();
    } catch (error) {
      console.error('Error toggling default profile:', error);
    }
  }

  function editProfile(profile: Profile) {
    // Populate newProfile with the selected profile's data
    newProfile = { ...profile };
    showModal = true;
    isEditing = true;
  }

  async function saveProfile() {
    try {
      if (newProfile.default) {
        // Unset default on other profiles
        await Promise.all(
          profiles
            .filter((p) => p.default && p.id !== newProfile.id)
            .map((p) =>
              $pocketbase.collection('scan_profiles').update(p.id, { default: false })
            )
        );
      }

      if (isEditing) {
        // Update existing profile
        await $pocketbase.collection('scan_profiles').update(newProfile.id, newProfile);
      } else {
        // Create new profile
        await $pocketbase.collection('scan_profiles').create(newProfile);
      }

      fetchProfiles();
      resetNewProfile();
      showModal = false;
      isEditing = false;
    } catch (error) {
      console.error('Error saving profile:', error);
    }
  }

  // Add helper function for provider options
  function getProviderOptions(providers: Provider[]) {
    return providers.map(provider => ({
      value: provider.id,
      name: provider.name,
      type: provider.provider_type
    }));
  }
</script>

<main class="p-4">
  <Card>
    <div class="space-y-8">
      <!-- Heading -->
      <Heading tag="h2" class="text-xl font-semibold mb-4">Scan Profiles</Heading>

      <!-- Profiles Table -->
      <div>
        <Table class="w-full border border-gray-200 dark:border-gray-700">
          <TableHead class="bg-gray-100 dark:bg-gray-700">
            {#each [
              { text: 'Name', align: 'left' },
              { text: 'Description', align: 'left' },
              { text: 'Interact Server', align: 'left' },
              { text: 'VM Provider', align: 'center' },
              { text: 'VM Size', align: 'center' },
              { text: 'State Bucket', align: 'center' },
              { text: 'Scan Bucket', align: 'center' },
              { text: 'Default', align: 'center' },
              { text: 'Actions', align: 'left' }
            ] as header}
              <TableHeadCell class="ps-4 font-normal text-{header.align}">{header.text}</TableHeadCell>
            {/each}
          </TableHead>
          <TableBody>
            {#each profiles as profile}
              <TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
                <TableBodyCell class="p-4">{profile.name}</TableBodyCell>
                <TableBodyCell class="p-4">{profile.description}</TableBodyCell>
                <TableBodyCell class="p-4">
                  {profile.expand?.nuclei_interact?.name || 'N/A'}
                </TableBodyCell>
                <TableBodyCell class="p-4 text-center w-40">
                  {#if profile.expand?.vm_provider}
                    <div class="flex flex-col items-center justify-center h-full">
                      <svelte:component 
                        this={getProviderIcon(profile.expand.vm_provider.provider_type)} 
                        size={24}
                      />
                      <span class="text-sm mt-1">{profile.expand.vm_provider.name}</span>
                    </div>
                  {:else}
                    N/A
                  {/if}
                </TableBodyCell>
                <TableBodyCell class="p-4 text-center">
                  {#if profile.expand?.vm_provider?.provider_type.toLowerCase() === 'digitalocean'}
                    {#if profile.vm_size}
                      <div class="text-sm">
                        <span class="font-medium">{profile.vm_size}</span>
                      </div>
                    {:else}
                      <span class="text-gray-500">Configure in Provider Settings</span>
                    {/if}
                  {:else}
                    <span class="text-gray-500">N/A</span>
                  {/if}
                </TableBodyCell>
                <TableBodyCell class="p-4 text-center w-40">
                  {#if profile.expand?.state_bucket}
                    <div class="flex flex-col items-center justify-center h-full">
                      <svelte:component 
                        this={getProviderIcon(profile.expand.state_bucket.provider_type)} 
                        size={24}
                      />
                      <span class="text-sm mt-1">{profile.expand.state_bucket.name}</span>
                    </div>
                  {:else}
                    N/A
                  {/if}
                </TableBodyCell>
                <TableBodyCell class="p-4 text-center w-40">
                  {#if profile.expand?.scan_bucket}
                    <div class="flex flex-col items-center justify-center h-full">
                      <svelte:component 
                        this={getProviderIcon(profile.expand.scan_bucket.provider_type)} 
                        size={24}
                      />
                      <span class="text-sm mt-1">{profile.expand.scan_bucket.name}</span>
                    </div>
                  {:else}
                    N/A
                  {/if}
                </TableBodyCell>
                <TableBodyCell class="p-4">
                  <Toggle
                    checked={profile.default}
                    on:change={() => toggleDefault(profile)}
                    disabled={profiles.length === 1 && profile.default}
                  />
                </TableBodyCell>
                <TableBodyCell class="space-x-2">
                  <Button size="sm" on:click={() => editProfile(profile)}>Edit</Button>
                  <Button size="sm" color="red" on:click={() => removeProfile(profile.id)}>
                    Remove
                  </Button>
                </TableBodyCell>
              </TableBodyRow>
            {/each}
          </TableBody>
        </Table>
      </div>

      <!-- Add Profile Button -->
      <Button class="mt-4" on:click={() => (showModal = true)}>Add Profile</Button>

      <!-- Modal for Adding/Editing a Profile -->
      <Modal bind:open={showModal} size="xl" autoclose={false} class="w-full">
        <div class="flex items-center justify-between p-4 md:p-5 border-b rounded-t dark:border-gray-600">
          <Heading tag="h3" class="text-xl font-semibold text-gray-900 dark:text-white">
            {isEditing ? 'Edit Profile' : 'Add New Profile'}
          </Heading>
        </div>
        <div class="p-4 md:p-5 space-y-4">
          <form class="space-y-4" on:submit|preventDefault={saveProfile}>
            <!-- Name -->
            <div>
              <Label for="name" class="mb-2">Name</Label>
              <Input
                id="name"
                type="text"
                required
                bind:value={newProfile.name}
                placeholder="Enter profile name"
              />
            </div>

            <!-- Description -->
            <div>
              <Label for="description" class="mb-2">Description</Label>
              <Input
                id="description"
                type="text"
                bind:value={newProfile.description}
                placeholder="Enter profile description"
              />
            </div>

            <!-- Interact Server -->
            <div>
              <Label for="nuclei_interact" class="mb-2">Interact Server</Label>
              <select
                id="nuclei_interact"
                class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:border-blue-500 dark:focus:ring-blue-500"
                bind:value={newProfile.nuclei_interact}
              >
                <option value="">Select an interact server</option>
                {#each interactServers as server}
                  <option value={server.id}>{server.name}</option>
                {/each}
              </select>
            </div>

            <!-- VM Provider -->
            <div>
              <Label for="vm_provider" class="mb-2">VM Provider</Label>
              <Select
                id="vm_provider"
                items={getProviderOptions(vmProviders)}
                bind:value={newProfile.vm_provider}
                placeholder="Select a VM provider"
                class="mt-2"
              >
                <div slot="item" let:item class="flex items-center gap-2">
                  <svelte:component this={getProviderIcon(item.type)} size={16} />
                  {item.name}
                </div>
              </Select>
            </div>

            <!-- VM Size (only shown for DigitalOcean) -->
            {#if vmProviders.find(p => p.id === newProfile.vm_provider)?.provider_type.toLowerCase() === 'digitalocean'}
              <div>
                <Label for="vm_size" class="mb-2">VM Size</Label>
                <Select
                  id="vm_size"
                  items={dropletSizes}
                  bind:value={newProfile.vm_size}
                  placeholder="Select a VM size"
                  class="mt-2"
                >
                  <div slot="item" let:item class="flex items-center gap-2">
                    {item.name}
                  </div>
                </Select>
              </div>
            {/if}

            <!-- State Bucket -->
            <div>
              <Label for="state_bucket" class="mb-2">State Bucket</Label>
              <Select
                id="state_bucket"
                items={getProviderOptions(stateBuckets)}
                bind:value={newProfile.state_bucket}
                placeholder="Select a state bucket"
                class="mt-2"
              >
                <div slot="item" let:item class="flex items-center gap-2">
                  <svelte:component this={getProviderIcon(item.type)} size={16} />
                  {item.name}
                </div>
              </Select>
            </div>

            <!-- Scan Bucket -->
            <div>
              <Label for="scan_bucket" class="mb-2">Scan Bucket</Label>
              <Select
                id="scan_bucket"
                items={getProviderOptions(scanBuckets)}
                bind:value={newProfile.scan_bucket}
                placeholder="Select a scan bucket"
                class="mt-2"
              >
                <div slot="item" let:item class="flex items-center gap-2">
                  <svelte:component this={getProviderIcon(item.type)} size={16} />
                  {item.name}
                </div>
              </Select>
            </div>

            <!-- Default Toggle -->
            <div class="flex items-center gap-2">
              <Toggle
                bind:checked={newProfile.default}
                disabled={profiles.length === 1 && profiles[0]?.default}
              >
                <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
                  Set as Default Profile
                </span>
              </Toggle>
            </div>
          </form>
        </div>
        <div class="flex items-center p-4 md:p-5 border-t border-gray-200 rounded-b dark:border-gray-600">
          <Button type="submit" on:click={saveProfile}>{isEditing ? 'Update' : 'Create'} Profile</Button>
          <Button color="alternative" class="ms-3" on:click={() => {
            showModal = false;
            resetNewProfile();
            isEditing = false;
          }}>Cancel</Button>
        </div>
      </Modal>
    </div>
  </Card>
</main>

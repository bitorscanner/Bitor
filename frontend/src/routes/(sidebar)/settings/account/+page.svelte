<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { currentUser } from '$lib/stores/auth';
  import { Button, Card, Label, Input, Toast, Dropzone, Select, Avatar, Breadcrumb, BreadcrumbItem } from 'flowbite-svelte';
  import { slide } from 'svelte/transition';
  import { CheckCircleSolid, ExclamationCircleSolid, DesktopPcOutline } from 'flowbite-svelte-icons';
  import Icon from '@iconify/svelte';
  import { avatarPath, defaultAvatarPath } from '$lib/utils/variables.js';

  type DropzoneChangeEvent = Event & {
    target: HTMLInputElement;
  };

  type DropzoneDropEvent = DragEvent & {
    dataTransfer: DataTransfer;
  };

  let loading = true;
  let showSuccessToast = false;
  let showErrorToast = false;
  let toastMessage = '';
  let avatarFile: File | null = null;
  let avatarPreview: string | null = null;
  let showPasswordModal = false;
  let isUploading = false;

  let userData = {
    first_name: '',
    last_name: '',
    email: '',
    timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
    avatar: ''
  };

  let passwordData = {
    current: '',
    new: '',
    confirm: ''
  };

  const timezones = Intl.supportedValuesOf('timeZone');

  onMount(async () => {
    console.log('Current user:', $currentUser);
    console.log('Is admin:', $pocketbase.authStore.isAdmin);
    
    if ($currentUser && $pocketbase.authStore.isValid) {
      try {
        if ($pocketbase.authStore.isAdmin) {
          console.log('Admin user data:', $currentUser);
          userData = {
            first_name: 'Super',
            last_name: 'Admin',
            email: $currentUser.email || '',
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            avatar: $currentUser.avatar || ''
          };

          if ($currentUser.avatar) {
            avatarPreview = `${$pocketbase.baseUrl}/api/admins/${$currentUser.id}/avatar/${$currentUser.avatar}`;
          }
        } else {
          const user = await $pocketbase.collection('users').getOne($currentUser.id);
          console.log('Regular user data:', user);
          
          userData = {
            first_name: user.first_name || '',
            last_name: user.last_name || '',
            email: user.email || '',
            timezone: user.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
            avatar: user.avatar || ''
          };

          if (user.avatar) {
            avatarPreview = `${import.meta.env.VITE_API_BASE_URL}/api/files/users/${user.id}/${user.avatar}`;
          }
        }
      } catch (error) {
        console.error('Error fetching user data:', error);
        toastMessage = 'Error loading user data';
        showErrorToast = true;
        setTimeout(() => showErrorToast = false, 3000);
      }
    }
    loading = false;
  });

  function handleFileSelect(file: File) {
    if (file && file.type.startsWith('image/')) {
      avatarFile = file;
      avatarPreview = URL.createObjectURL(file);
    }
  }

  function handleDrop(event: DragEvent) {
    if (event.dataTransfer?.files?.[0]) {
      handleFileSelect(event.dataTransfer.files[0]);
    }
  }

  function handleChange(event: Event) {
    const input = event.target as HTMLInputElement;
    if (input?.files?.[0]) {
      handleFileSelect(input.files[0]);
    }
  }

  async function updateProfile() {
    try {
      if (!$currentUser || !$pocketbase.authStore.isValid) {
        throw new Error('No authenticated user');
      }

      const formData = new FormData();
      
      if (userData.first_name) formData.append('firstName', userData.first_name);
      if (userData.last_name) formData.append('lastName', userData.last_name);
      if (userData.timezone) formData.append('timezone', userData.timezone);
      
      if (avatarFile) {
        formData.append('avatar', avatarFile);
      }

      let updatedUser;
      if ($pocketbase.authStore.isAdmin) {
        updatedUser = await $pocketbase.admins.update($currentUser.id, formData);
        if (updatedUser.avatar) {
          avatarPreview = `${$pocketbase.baseUrl}/api/admins/${$currentUser.id}/avatar/${updatedUser.avatar}`;
        }
      } else {
        updatedUser = await $pocketbase.collection('users').update($currentUser.id, formData);
        if (updatedUser.avatar) {
          avatarPreview = $pocketbase.files.getUrl(updatedUser, updatedUser.avatar);
        }
      }
      
      toastMessage = 'Profile updated successfully';
      showSuccessToast = true;
      setTimeout(() => showSuccessToast = false, 3000);
    } catch (error) {
      console.error('Error updating profile:', error);
      toastMessage = 'Error updating profile';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
    }
  }

  async function updatePassword() {
    if (passwordData.new !== passwordData.confirm) {
      toastMessage = 'New passwords do not match';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
      return;
    }

    try {
      if ($pocketbase.authStore.isAdmin) {
        await $pocketbase.admins.update($currentUser.id, {
          oldPassword: passwordData.current,
          password: passwordData.new,
          passwordConfirm: passwordData.confirm
        });
      } else {
        await $pocketbase.collection('users').update($currentUser.id, {
          oldPassword: passwordData.current,
          password: passwordData.new,
          passwordConfirm: passwordData.confirm
        });
      }
      
      toastMessage = 'Password updated successfully';
      showSuccessToast = true;
      setTimeout(() => showSuccessToast = false, 3000);
      showPasswordModal = false;
      passwordData = { current: '', new: '', confirm: '' };
    } catch (error) {
      console.error('Error updating password:', error);
      toastMessage = 'Error updating password';
      showErrorToast = true;
      setTimeout(() => showErrorToast = false, 3000);
    }
  }

  async function handleAvatarUpload(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files?.length) return;

    const file = input.files[0];
    
    // Validate file type
    if (!file.type.startsWith('image/')) {
        toastMessage = 'Please upload an image file';
        showErrorToast = true;
        setTimeout(() => showErrorToast = false, 3000);
        return;
    }

    // Validate file size (e.g., max 5MB)
    if (file.size > 5 * 1024 * 1024) {
        toastMessage = 'File size must be less than 5MB';
        showErrorToast = true;
        setTimeout(() => showErrorToast = false, 3000);
        return;
    }

    isUploading = true;
    
    try {
        const formData = new FormData();
        formData.append('avatar', file);

        const updatedUser = await $pocketbase.collection('users').update($currentUser.id, formData);
        $currentUser = updatedUser;

        // Update the avatar preview with the utility function
        avatarPreview = avatarPath(updatedUser.id, updatedUser.avatar);

        toastMessage = 'Profile photo updated successfully';
        showSuccessToast = true;
        setTimeout(() => showSuccessToast = false, 3000);
    } catch (error) {
        console.error('Error uploading avatar:', error);
        toastMessage = 'Failed to update profile photo';
        showErrorToast = true;
        setTimeout(() => showErrorToast = false, 3000);
    } finally {
        isUploading = false;
    }
  }
</script>

{#if showSuccessToast}
  <Toast transition={slide} color="green" class="mb-4">
    <CheckCircleSolid slot="icon" class="w-5 h-5" />
    {toastMessage}
  </Toast>
{/if}

{#if showErrorToast}
  <Toast transition={slide} color="red" class="mb-4">
    <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
    {toastMessage}
  </Toast>
{/if}

<div class="container mx-auto px-4 py-6">
  <Breadcrumb class="mb-4">
    <BreadcrumbItem href="/">Home</BreadcrumbItem>
    <BreadcrumbItem href="/settings">Settings</BreadcrumbItem>
    <BreadcrumbItem>Account</BreadcrumbItem>
  </Breadcrumb>

  <h1 class="text-2xl font-bold mb-4">Account Settings</h1>

  {#if loading}
    <p>Loading settings...</p>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <!-- Profile Picture -->
      <Card padding="sm">
        <h2 class="text-xl font-semibold mb-3">Profile Picture</h2>
        <div class="space-y-4">
          <div class="flex justify-center">
            {#if avatarPreview}
              <Avatar 
                src={avatarPreview}
                class="w-32 h-32" 
                rounded
                on:error={e => {
                  e.currentTarget.src = defaultAvatarPath;
                }}
              />
            {:else}
              <div class="w-32 h-32 rounded-full bg-gray-200 flex items-center justify-center">
                <Icon icon="mdi:user" class="w-16 h-16 text-gray-400" />
              </div>
            {/if}
          </div>
          <div class="text-center text-sm text-gray-500 mb-2">
            JPG, GIF or PNG. Max size of 800K
          </div>
          <div class="mb-4">
            <label for="avatar" class="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Profile Photo
            </label>
            <input
              type="file"
              id="avatar"
              accept="image/*"
              on:change={handleAvatarUpload}
              disabled={isUploading}
              class="mt-1 block w-full text-sm text-gray-900 border border-gray-300 rounded-lg cursor-pointer bg-gray-50 dark:text-gray-400 focus:outline-none dark:bg-gray-700 dark:border-gray-600 dark:placeholder-gray-400"
            />
            {#if isUploading}
              <p class="text-sm text-gray-500 mt-1">Uploading...</p>
            {/if}
          </div>
        </div>
      </Card>

      <!-- General Information -->
      <Card padding="sm">
        <h2 class="text-xl font-semibold mb-3">General Information</h2>
        <form class="space-y-4" on:submit|preventDefault={updateProfile}>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <Label for="first_name">First Name</Label>
              <Input 
                id="first_name" 
                bind:value={userData.first_name} 
                disabled={$pocketbase.authStore.isAdmin}
              />
            </div>
            
            <div>
              <Label for="last_name">Last Name</Label>
              <Input 
                id="last_name" 
                bind:value={userData.last_name} 
                disabled={$pocketbase.authStore.isAdmin}
              />
            </div>
          </div>

          <div>
            <Label for="email">Email</Label>
            <Input id="email" type="email" value={userData.email} disabled />
            {#if $pocketbase.authStore.isAdmin}
              <p class="text-sm text-gray-500 mt-1">Super admin details cannot be modified through this interface.</p>
            {:else}
              <p class="text-sm text-gray-500 mt-1">Email changes require verification and must be handled through account settings.</p>
            {/if}
          </div>

          <div>
            <Label for="timezone">Timezone</Label>
            <Select 
              id="timezone" 
              bind:value={userData.timezone}
              disabled={$pocketbase.authStore.isAdmin}
            >
              {#each timezones as tz}
                <option value={tz}>{tz}</option>
              {/each}
            </Select>
          </div>

          <div class="flex justify-end">
            <Button 
              type="submit" 
              color="primary"
              disabled={$pocketbase.authStore.isAdmin}
            >
              Save Changes
            </Button>
          </div>
        </form>
      </Card>

      <!-- Security -->
      <Card padding="sm">
        <h2 class="text-xl font-semibold mb-3">Security</h2>
        <div class="space-y-4">
          {#if showPasswordModal}
            <div class="space-y-4">
              <Label class="space-y-2">
                <span>Current password</span>
                <Input type="password" bind:value={passwordData.current} />
              </Label>
              <Label class="space-y-2">
                <span>New password</span>
                <Input type="password" bind:value={passwordData.new} />
              </Label>
              <Label class="space-y-2">
                <span>Confirm password</span>
                <Input type="password" bind:value={passwordData.confirm} />
              </Label>
              <div class="flex gap-2">
                <Button color="primary" on:click={updatePassword}>Update Password</Button>
                <Button color="light" on:click={() => showPasswordModal = false}>Cancel</Button>
              </div>
            </div>
          {:else}
            <div>
              <h3 class="text-lg font-medium mb-2">Password</h3>
              <p class="text-gray-600 mb-2">Change your password to keep your account secure.</p>
              <Button color="light" on:click={() => showPasswordModal = true}>Change Password</Button>
            </div>
          {/if}
          <div>
            <h3 class="text-lg font-medium mb-2">Two-Factor Authentication</h3>
            <p class="text-gray-600 mb-2">Add an extra layer of security to your account.</p>
            <Button color="light" disabled>Coming Soon</Button>
          </div>
        </div>
      </Card>

      <!-- Session Information -->
      <Card padding="sm">
        <h2 class="text-xl font-semibold mb-3">Session Information</h2>
        <div class="space-y-3">
          <div class="flex items-start space-x-3 p-3 bg-gray-50 dark:bg-gray-800 rounded-lg">
            <DesktopPcOutline class="w-5 h-5 text-gray-500 mt-0.5" />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-gray-900 dark:text-white truncate">Current Browser</p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 break-all">
                {navigator.userAgent}
              </p>
              <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">
                Last active: {new Date($currentUser?.updated || '').toLocaleString()}
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>
  {/if}
</div> 
<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { currentUser } from '$lib/stores/auth';
  import { Button, Card, Label, Input, Toast, Dropzone, Select, Avatar } from 'flowbite-svelte';
  import { slide } from 'svelte/transition';
  import { CheckCircleSolid, ExclamationCircleSolid, DesktopPcOutline } from 'flowbite-svelte-icons';
  import Icon from '@iconify/svelte';

  let loading = true;
  let showSuccessToast = false;
  let showErrorToast = false;
  let toastMessage = '';
  let avatarFile: File | null = null;
  let avatarPreview: string | null = null;
  let showPasswordModal = false;

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
    if ($currentUser) {
      try {
        const user = await $pocketbase.collection('users').getOne($currentUser.id);
        userData = {
          first_name: user.first_name || '',
          last_name: user.last_name || '',
          email: user.email || '',
          timezone: user.timezone || Intl.DateTimeFormat().resolvedOptions().timeZone,
          avatar: user.avatar || ''
        };
        if (user.avatar) {
          avatarPreview = $pocketbase.files.getUrl(user, user.avatar);
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

  function handleAvatarDrop(e: DragEvent) {
    const dt = e.dataTransfer;
    if (dt?.files && dt.files[0]) {
      avatarFile = dt.files[0];
      avatarPreview = URL.createObjectURL(avatarFile);
    }
  }

  async function updateProfile() {
    try {
      const formData = new FormData();
      formData.append('first_name', userData.first_name);
      formData.append('last_name', userData.last_name);
      formData.append('timezone', userData.timezone);
      
      if (avatarFile) {
        formData.append('avatar', avatarFile);
      }
      
      await $pocketbase.collection('users').update($currentUser.id, formData);
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
      await $pocketbase.collection('users').update($currentUser.id, {
        oldPassword: passwordData.current,
        password: passwordData.new,
        passwordConfirm: passwordData.confirm
      });
      
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

<div class="container mx-auto px-4 py-8">
  <h1 class="text-2xl font-bold mb-6">Profile Settings</h1>

  {#if loading}
    <p>Loading profile...</p>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <!-- General Information -->
      <Card class="md:col-span-2">
        <h2 class="text-xl font-semibold mb-4">General Information</h2>
        <form class="space-y-4" on:submit|preventDefault={updateProfile}>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <Label for="first_name">First Name</Label>
              <Input id="first_name" bind:value={userData.first_name} />
            </div>
            
            <div>
              <Label for="last_name">Last Name</Label>
              <Input id="last_name" bind:value={userData.last_name} />
            </div>
          </div>

          <div>
            <Label for="email">Email</Label>
            <Input id="email" type="email" value={userData.email} disabled />
            <p class="text-sm text-gray-500 mt-1">Email changes require verification and must be handled through account settings.</p>
          </div>

          <div>
            <Label for="timezone">Timezone</Label>
            <Select id="timezone" bind:value={userData.timezone}>
              {#each timezones as tz}
                <option value={tz}>{tz}</option>
              {/each}
            </Select>
          </div>

          <div class="flex justify-end">
            <Button type="submit" color="primary">Save Changes</Button>
          </div>
        </form>
      </Card>

      <!-- Profile Picture -->
      <Card>
        <h2 class="text-xl font-semibold mb-4">Profile Picture</h2>
        <div class="space-y-4">
          <div class="flex justify-center">
            {#if avatarPreview}
              <Avatar src={avatarPreview} class="w-32 h-32" rounded />
            {:else}
              <div class="w-32 h-32 rounded-full bg-gray-200 flex items-center justify-center">
                <Icon icon="mdi:user" class="w-16 h-16 text-gray-400" />
              </div>
            {/if}
          </div>
          <div class="text-center text-sm text-gray-500 mb-2">
            JPG, GIF or PNG. Max size of 800K
          </div>
          <Dropzone on:drop={handleAvatarDrop} accept="image/*" />
        </div>
      </Card>

      <!-- Security -->
      <Card class="md:col-span-2">
        <h2 class="text-xl font-semibold mb-4">Security</h2>
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
      <Card>
        <h2 class="text-xl font-semibold mb-4">Session Information</h2>
        <div class="space-y-4">
          <div class="flex items-center space-x-4">
            <DesktopPcOutline class="w-6 h-6 text-gray-500" />
            <div class="flex-1">
              <p class="text-sm font-medium">Current Session</p>
              <p class="text-sm text-gray-500">
                {navigator.userAgent}
              </p>
              <p class="text-sm text-gray-500">
                Last active: {new Date($currentUser?.updated || '').toLocaleString()}
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>
  {/if}
</div> 
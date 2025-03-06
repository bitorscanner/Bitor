<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { goto } from '$app/navigation';
  import { writable } from 'svelte/store';
  import { theme } from '$lib/stores/theme';
  import {
    Tabs,
    TabItem,
    Label,
    Input,
    Button,
    Toast,
  } from 'flowbite-svelte';
  import {
    CheckCircleSolid,
    ExclamationCircleSolid,
    SunSolid,
    MoonSolid,
  } from 'flowbite-svelte-icons';

  let activeTab = 0;
  let form = {
    email: '',
    password: '',
    confirm: ''
  };

  const errorMessage = writable('');
  const successMessage = writable('');
  let setupCompleted = false;
  let loading = true;

  // Initialize error message cleanup
  const unsubscribe = errorMessage.subscribe((message) => {
    if (message) {
      setTimeout(() => {
        errorMessage.set('');
      }, 5000);
    }
  });

  onDestroy(() => {
    if (unsubscribe) {
      unsubscribe();
    }
  });

  onMount(async () => {
    // Initialize theme
    theme.initialize();
    
    try {
      // Check if setup is already completed
      const settings = await $pocketbase.collection('settings_public').getFirstListItem('');
      setupCompleted = settings.setup_completed;
      
      if (setupCompleted) {
        // If setup is completed, redirect to login
        goto('/authentication/sign-in');
        return;
      }
    } catch (err) {
      console.error('Error checking setup status:', err);
    } finally {
      loading = false;
    }
  });

  async function register() {
    if (loading) return;
    
    if (form.password !== form.confirm) {
      errorMessage.set('Passwords do not match');
      return;
    }

    try {
      // Check again if setup is completed before proceeding
      const settings = await $pocketbase.collection('settings_public').getFirstListItem('');
      if (settings.setup_completed) {
        errorMessage.set('Setup has already been completed');
        goto('/authentication/sign-in');
        return;
      }

      // Create the first admin using PocketBase's admin API
      await $pocketbase.admins.create({
        email: form.email,
        password: form.password,
        passwordConfirm: form.password
      });

      // Login as the newly created admin
      await $pocketbase.admins.authWithPassword(form.email, form.password);

      // Update settings to mark setup as complete
      const appSettings = await $pocketbase.collection('settings_public').getFirstListItem('');
      await $pocketbase.collection('settings_public').update(appSettings.id, {
        setup_completed: true,
      });

      activeTab = 2;
      successMessage.set('Admin account created successfully!');
    } catch (err: any) {
      errorMessage.set(err.message || 'An unknown error occurred. Please try again.');
    }
  }
</script>

{#if loading}
  <div class="flex min-h-screen items-center justify-center bg-gray-100 dark:bg-gray-900">
    <div class="text-center">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-gray-900 dark:border-white mx-auto"></div>
      <p class="mt-4 text-gray-600 dark:text-gray-400">Loading...</p>
    </div>
  </div>
{:else if !setupCompleted}
  <!-- Main Container -->
  <div class="min-h-screen flex flex-col items-start justify-start bg-gray-100 dark:bg-gray-900 px-4 py-10">
    <!-- Theme Toggle Button -->
    <div class="absolute top-4 right-4 z-50">
      <Button color="light" class="!p-2 dark:!bg-gray-800 dark:hover:!bg-gray-700" on:click={theme.toggle}>
        {#if $theme}
          <SunSolid class="w-5 h-5 text-gray-700 dark:text-gray-300" />
        {:else}
          <MoonSolid class="w-5 h-5 text-gray-700 dark:text-gray-300" />
        {/if}
      </Button>
    </div>
    
    <div class="w-full max-w-md mx-auto">
      <!-- Tabs centered horizontally -->
      <Tabs class="w-full flex justify-center" style="default">
        <TabItem
          title="Welcome"
          open={activeTab === 0}
          on:click={() => (activeTab = 0)}
        >
          <!-- Welcome Panel -->
          <div class="p-6 text-center">
            <!-- Light mode logo -->
            <img
              src="/images/Orbit-Main-Logo.png"
              alt="Orbit Logo"
              class="mx-auto block dark:hidden"
            />
            <!-- Dark mode logo -->
            <img
              src="/images/Orbit_White_Logo.png"
              alt="Orbit Logo"
              class="mx-auto hidden dark:block"
            />
            <h2 class="text-2xl font-semibold mt-4 text-gray-900 dark:text-white">Welcome to Orbit!</h2>
            <p class="mt-2 text-gray-600 dark:text-gray-300">
              Let's get started by setting up your admin account.
            </p>
            <Button
              on:click={() => (activeTab = 1)}
              class="mt-6 w-full"
              size="lg"
              color="primary"
            >
              Next
            </Button>
          </div>
        </TabItem>

        <TabItem
          title="Create Account"
          open={activeTab === 1}
          on:click={() => (activeTab = 1)}
          disabled={activeTab < 1}
        >
          <!-- Create Account Panel -->
          <div class="p-6">
            <h2 class="text-2xl font-semibold text-center text-gray-900 dark:text-white">Create Admin Account</h2>
            <form class="mt-4 space-y-4" on:submit|preventDefault={register}>
              <div>
                <Label for="email" class="mb-2 text-gray-900 dark:text-white">Email</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="name@company.com"
                  required
                  bind:value={form.email}
                  class="dark:bg-gray-800 dark:border-gray-700 dark:text-white"
                />
              </div>
              <div>
                <Label for="password" class="mb-2 text-gray-900 dark:text-white">Password</Label>
                <Input
                  id="password"
                  type="password"
                  placeholder="••••••••"
                  required
                  bind:value={form.password}
                  class="dark:bg-gray-800 dark:border-gray-700 dark:text-white"
                />
              </div>
              <div>
                <Label for="confirm" class="mb-2 text-gray-900 dark:text-white">Confirm Password</Label>
                <Input
                  id="confirm"
                  type="password"
                  placeholder="••••••••"
                  required
                  bind:value={form.confirm}
                  class="dark:bg-gray-800 dark:border-gray-700 dark:text-white"
                />
              </div>
              {#if $errorMessage}
                <Toast color="red" class="mt-4">
                  <ExclamationCircleSolid slot="icon" class="w-5 h-5" />
                  {$errorMessage}
                </Toast>
              {/if}
              <Button type="submit" class="w-full" size="lg" color="primary">
                Create Account
              </Button>
            </form>
          </div>
        </TabItem>

        <TabItem
          title="Complete"
          open={activeTab === 2}
          on:click={() => (activeTab = 2)}
          disabled={activeTab < 2}
        >
          <!-- Complete Panel -->
          <div class="p-6 text-center">
            <!-- Light mode logo -->
            <img
              src="/images/Orbit-Main-Logo.png"
              alt="Orbit Logo"
              class="mx-auto block dark:hidden"
            />
            <!-- Dark mode logo -->
            <img
              src="/images/Orbit_White_Logo.png"
              alt="Orbit Logo"
              class="mx-auto hidden dark:block"
            />
            <h2 class="text-2xl font-semibold mt-4 text-gray-900 dark:text-white">Setup Complete!</h2>
            {#if $successMessage}
              <Toast color="green" class="mt-4">
                <CheckCircleSolid slot="icon" class="w-5 h-5" />
                {$successMessage}
              </Toast>
            {/if}
            <p class="mt-2 text-gray-600 dark:text-gray-300">
              Your admin account has been created. You can now start using Orbit.
            </p>
            <Button
              on:click={() => goto('/')}
              class="mt-6 w-full"
              size="lg"
              color="green"
            >
              Enter App
            </Button>
          </div>
        </TabItem>
      </Tabs>
    </div>
  </div>
{/if}
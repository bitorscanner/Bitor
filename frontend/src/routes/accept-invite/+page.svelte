<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import Logo from '$lib/components/Logo.svelte';
  import { Alert } from 'flowbite-svelte';
  import { ExclamationCircleSolid } from 'flowbite-svelte-icons';
  import { pocketbase } from '@lib/stores/pocketbase';

  let token = '';
  let username = '';
  let password = '';
  let passwordConfirm = '';
  let first_name = '';
  let last_name = '';
  let loading = false;
  let error = '';
  let validatingToken = true;
  let tokenValid = false;

  async function validateToken(token: string) {
    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/invitations/validate?token=${token}`);
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.message || 'Invalid invitation link');
      }

      tokenValid = true;
      return true;
    } catch (err: any) {
      error = err.message;
      tokenValid = false;
      return false;
    } finally {
      validatingToken = false;
    }
  }

  onMount(async () => {
    // Get token from URL query parameter
    const url = new URL(window.location.href);
    token = url.searchParams.get('token') || '';
    if (!token) {
      error = 'Invalid invitation link: No token provided';
      validatingToken = false;
      return;
    }

    await validateToken(token);
  });

  const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

  async function handleSubmit() {
    if (password !== passwordConfirm) {
      error = 'Passwords do not match';
      return;
    }

    loading = true;
    error = '';

    try {
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/invitations/accept`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          token,
          username,
          password,
          first_name,
          last_name
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.message || 'Failed to accept invitation');
      }

      // Add a delay to ensure the user is created in the database
      await delay(1000);

      try {
        // Login directly after accepting invitation
        await $pocketbase.collection('users').authWithPassword(username, password);
        
        // Redirect to dashboard
        goto('/');
      } catch (loginErr: any) {
        console.error('Login error:', loginErr);
        error = 'Account created successfully, but login failed. Please try logging in manually.';
        goto('/authentication/sign-in');
      }
    } catch (err: any) {
      error = err.message;
    } finally {
      loading = false;
    }
  }
</script>

<div class="flex min-h-screen flex-col items-center justify-center bg-gray-50 dark:bg-gray-900">
  <div class="w-full max-w-md space-y-8 rounded-lg bg-white p-6 shadow-lg dark:bg-gray-800">
    <div class="flex flex-col items-center">
      <Logo class="h-12 w-auto" />
      <h2 class="mt-6 text-center text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
        Accept Invitation
      </h2>
      <p class="mt-2 text-center text-sm text-gray-600 dark:text-gray-400">
        Create your account to get started
      </p>
    </div>

    {#if validatingToken}
      <div class="flex justify-center">
        <svg class="animate-spin h-8 w-8 text-primary-600" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
    {:else if error}
      <Alert color="red" class="mt-4">
        <ExclamationCircleSolid slot="icon" class="h-4 w-4" />
        {error}
      </Alert>
    {/if}

    {#if !validatingToken && tokenValid}
      <form class="mt-8 space-y-6" on:submit|preventDefault={handleSubmit}>
        <div class="-space-y-px rounded-md shadow-sm">
          <div>
            <label for="first_name" class="sr-only">First Name</label>
            <input
              id="first_name"
              name="first_name"
              type="text"
              required
              bind:value={first_name}
              class="relative block w-full rounded-t-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-gray-700 dark:text-white dark:ring-gray-600 dark:placeholder:text-gray-400 dark:focus:ring-primary-500 sm:text-sm sm:leading-6"
              placeholder="First Name"
            />
          </div>
          <div>
            <label for="last_name" class="sr-only">Last Name</label>
            <input
              id="last_name"
              name="last_name"
              type="text"
              required
              bind:value={last_name}
              class="relative block w-full border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-gray-700 dark:text-white dark:ring-gray-600 dark:placeholder:text-gray-400 dark:focus:ring-primary-500 sm:text-sm sm:leading-6"
              placeholder="Last Name"
            />
          </div>
          <div>
            <label for="username" class="sr-only">Username</label>
            <input
              id="username"
              name="username"
              type="text"
              required
              bind:value={username}
              class="relative block w-full border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-gray-700 dark:text-white dark:ring-gray-600 dark:placeholder:text-gray-400 dark:focus:ring-primary-500 sm:text-sm sm:leading-6"
              placeholder="Username"
            />
          </div>
          <div>
            <label for="password" class="sr-only">Password</label>
            <input
              id="password"
              name="password"
              type="password"
              required
              bind:value={password}
              class="relative block w-full border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-gray-700 dark:text-white dark:ring-gray-600 dark:placeholder:text-gray-400 dark:focus:ring-primary-500 sm:text-sm sm:leading-6"
              placeholder="Password"
            />
          </div>
          <div>
            <label for="passwordConfirm" class="sr-only">Confirm Password</label>
            <input
              id="passwordConfirm"
              name="passwordConfirm"
              type="password"
              required
              bind:value={passwordConfirm}
              class="relative block w-full rounded-b-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:z-10 focus:ring-2 focus:ring-inset focus:ring-primary-600 dark:bg-gray-700 dark:text-white dark:ring-gray-600 dark:placeholder:text-gray-400 dark:focus:ring-primary-500 sm:text-sm sm:leading-6"
              placeholder="Confirm Password"
            />
          </div>
        </div>

        <div>
          <button
            type="submit"
            disabled={loading}
            class="group relative flex w-full justify-center rounded-md bg-primary-600 px-3 py-2 text-sm font-semibold text-white hover:bg-primary-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary-600 disabled:opacity-50 dark:bg-primary-500 dark:hover:bg-primary-400"
          >
            {#if loading}
              <span class="absolute inset-y-0 left-0 flex items-center pl-3">
                <svg class="h-5 w-5 animate-spin text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
              </span>
            {/if}
            Accept Invitation
          </button>
        </div>
      </form>
    {/if}
  </div>
</div> 
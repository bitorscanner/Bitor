<script lang="ts">
	import { onMount } from 'svelte';
	import { afterNavigate, goto } from '$app/navigation';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { currentUser } from '$lib/stores/auth';
	import { settings } from '$lib/stores/settings';
	import { theme } from '$lib/stores/theme';
	import { writable, get } from 'svelte/store';
	import { page } from '$app/stores';

	const loading = writable(true);
	let isRedirecting = false;

	async function safeGoto(path: string) {
		if (!isRedirecting) {
			isRedirecting = true;
			await goto(path);
			isRedirecting = false;
		}
	}

	async function loadUserData() {
		if ($pocketbase.authStore.isValid && $pocketbase.authStore.model?.id) {
			try {
				// Check if the current user is an admin
				if ($pocketbase.authStore.isAdmin) {
					// For admin users, we don't need to fetch additional data
					currentUser.set($pocketbase.authStore.model);
					return;
				}

				// For regular users, fetch their data with group expansion
				const user = await $pocketbase.collection('users').getOne(
					$pocketbase.authStore.model.id,
					{
						expand: 'group'
					}
				);
				
				const groupName = user.expand?.group?.name;
				const enrichedUser = {
					...user,
					group: {
						...user.group,
						name: groupName
					}
				};
				
				currentUser.set(enrichedUser);
			} catch (error: any) {
				console.error('Error loading user data:', error);
				// If the user record doesn't exist anymore (404), clear the auth store
				if (error.status === 404) {
					$pocketbase.authStore.clear();
					await safeGoto('/authentication/sign-in');
				}
			}
		}
	}

	async function runAuthChecks() {
		if (isRedirecting) {
			return;
		}

		// Get the current path
		const currentPath = get(page).url.pathname;

		// If we're already on the setup page, no need to check further
		if (currentPath === '/setup') {
			loading.set(false);
			return;
		}

		try {
			// Check settings_public for setup_completed
			const settingsRecord = await $pocketbase.collection('settings_public').getFirstListItem('');
			const settingsData = {
				setup_completed: settingsRecord.setup_completed,
			};
			settings.set(settingsData);

			if (settingsData.setup_completed === false) {
				await safeGoto('/setup');
				loading.set(false);
				return;
			}

			// After confirming setup is complete, handle public routes
			const publicRoutes = [
				'/authentication/sign-in',
				'/authentication/forgot-password',
				'/accept-invite'
			];

			const isPublicRoute = publicRoutes.some((route) =>
				currentPath.startsWith(route)
			);

			if (isPublicRoute) {
				loading.set(false);
				return;
			}

			// Finally check auth
			if (!$pocketbase.authStore.isValid) {
				await safeGoto('/authentication/sign-in');
				loading.set(false);
				return;
			}

			// After auth checks, load user data
			await loadUserData();
		} catch (error) {
			console.error('Error in auth checks:', error);
			// If any error occurs in the main try block, redirect to setup as a safe default
			await safeGoto('/setup');
		}

		// Authentication checks are complete
		loading.set(false);
	}

	onMount(() => {
		// Initialize theme
		theme.initialize();
		// Run auth checks
		runAuthChecks();
	});

	afterNavigate(() => {
		// Only run checks if we're not already redirecting
		if (!isRedirecting) {
			loading.set(true);
			runAuthChecks();
		}
	});
</script>

<div class="app">
	<main>
		{#if $loading}
			<div class="flex items-center justify-center h-screen">
				<div class="relative w-16 h-16">
					<!-- Outer orbit -->
					<div class="absolute inset-0 rounded-full border-4 border-gray-200 dark:border-gray-700"></div>
					<!-- Outer orbiting dot -->
					<div class="absolute w-2 h-2 bg-orange-500 rounded-full outer-dot-animation glow-animation"></div>
					<!-- Inner orbiting dot -->
					<div class="absolute w-3 h-3 bg-orange-500 rounded-full inner-dot-animation glow-animation"></div>
					<!-- Inner circle -->
					<div class="absolute inset-4 rounded-full bg-gray-800 dark:bg-gray-200 breathe-animation"></div>
				</div>
			</div>
		{:else}
			<slot />
		{/if}
	</main>
</div>

<style>
	.inner-dot-animation {
		animation: inner-orbit 1.5s linear infinite;
		/* Position the dot at the edge of the inner circle */
		top: 50%;
		left: 50%;
		margin-top: -1.5px;
		margin-left: -1.5px;
		transform-origin: center;
	}

	.outer-dot-animation {
		animation: outer-orbit 3s linear infinite;
		/* Position the dot at the edge of the outer circle */
		top: 50%;
		left: 50%;
		margin-top: -1px;
		margin-left: -1px;
		transform-origin: center;
	}

	.breathe-animation {
		animation: breathe 3s ease-in-out infinite;
	}

	.glow-animation {
		box-shadow: 0 0 10px rgba(249, 115, 22, 0.5);
		animation: glow 2s ease-in-out infinite;
	}

	@keyframes inner-orbit {
		from {
			transform: rotate(0deg) translateX(12px);
		}
		to {
			transform: rotate(360deg) translateX(12px);
		}
	}

	@keyframes outer-orbit {
		from {
			transform: rotate(360deg) translateX(24px);
		}
		to {
			transform: rotate(0deg) translateX(24px);
		}
	}

	@keyframes breathe {
		0%, 100% {
			transform: scale(1);
		}
		50% {
			transform: scale(0.95);
		}
	}

	@keyframes glow {
		0%, 100% {
			box-shadow: 0 0 10px rgba(249, 115, 22, 0.5);
		}
		50% {
			box-shadow: 0 0 20px rgba(249, 115, 22, 0.8);
		}
	}
</style>

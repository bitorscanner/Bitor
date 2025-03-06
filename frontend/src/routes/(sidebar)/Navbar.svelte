<script>
	import UserMenu from '../utils/widgets/UserMenu.svelte';
	import {
		DarkMode,
		Dropdown,
		DropdownItem,
		NavBrand,
		NavHamburger,
		NavLi,
		NavUl,
		Navbar,
		Search,
		Button
	} from 'flowbite-svelte';
	import { ChevronDownOutline } from 'flowbite-svelte-icons';
	import '../../app.pcss';
	import { pocketbase } from '@lib/stores/pocketbase'; // Import the pocketbase store
	import { onMount, onDestroy } from 'svelte'; // Import onMount and onDestroy
	import UserMessages from '$lib/components/UserMessages.svelte';
	import { writable } from 'svelte/store';

	export let fluid = true;
	export let drawerHidden = false;
	export let list = false;

	/**
	 * @typedef {Object} User
	 * @property {string} name
	 * @property {string} email
	 * @property {string} [avatar]
	 * @property {string} [role]
	 */

	/** @type {User | null} */
	let currentUser = null;
	let showNotifications = false;
	let unreadCount = writable(0);

	// Reactive statement to safely assign the current user
	$: if ($pocketbase.authStore.model) {
		currentUser = {
			name: $pocketbase.authStore.model.name || '',
			email: $pocketbase.authStore.model.email,
			avatar: $pocketbase.authStore.model.avatar,
			role: $pocketbase.authStore.model.role
		};
	} else {
		currentUser = null;
	}

	// Function to fetch unread notifications count
	async function fetchUnreadCount() {
		try {
			let filter = 'read = false';
			
			// Add user/admin specific filter
			if ($pocketbase.authStore.isValid && $pocketbase.authStore.model?.id) {
				if ($pocketbase.authStore.isAdmin) {
					// For super admin, check admin_id field
					filter += ` && admin_id = "${$pocketbase.authStore.model.id}"`;
				} else {
					// For regular users, check user field
					filter += ` && user = "${$pocketbase.authStore.model.id}"`;
				}
			}

			const resultList = await $pocketbase.collection('user_messages').getList(1, 1, {
				filter: filter,
				fields: 'id'
			});
			unreadCount.set(resultList.totalItems);
			console.log('Fetched unread count with filter:', filter);
		} catch (error) {
			console.error('Error fetching unread count:', error);
		}
	}

	onMount(() => {
		// Fetch initial unread count
		fetchUnreadCount();

		// Set up interval to refresh unread count
		const interval = setInterval(fetchUnreadCount, 30000); // Check every 30 seconds

		return () => {
			clearInterval(interval);
		};
	});
</script>

<Navbar {fluid} class="text-black" color="default" let:NavContainer>
	<NavHamburger
		onClick={() => (drawerHidden = !drawerHidden)}
		class="m-0 me-3 md:block lg:hidden"
	/>
	<NavBrand href="/" class={list ? 'w-40' : 'lg:w-60'}>
		<!-- Logo for Light Mode -->
		<img
			src="/images/Orbit-Main-Logo.png"
			class="me-2.5 h-6 sm:h-8 block dark:hidden"
			alt="Orbit Logo Light"
		/>

		<!-- Logo for Dark Mode -->
		<img
			src="/images/Orbit_White_Logo.png"
			class="me-2.5 h-6 sm:h-8 hidden dark:block"
			alt="Orbit Logo Dark"
		/>
	</NavBrand>

	<div class="ms-auto flex items-center text-gray-500 dark:text-gray-400 sm:order-2">
		<!-- Notification Bell -->
		<div class="relative mr-2">
			<Button color="none" class="!p-2" on:click={() => showNotifications = !showNotifications}>
				<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9" />
				</svg>
				{#if $unreadCount > 0}
					<span class="absolute -top-1 -right-1 bg-red-500 text-white text-xs rounded-full h-5 w-5 flex items-center justify-center">
						{$unreadCount}
					</span>
				{/if}
			</Button>
			{#if showNotifications}
				<div class="absolute right-0 mt-2 w-96 bg-white dark:bg-gray-800 rounded-lg shadow-lg overflow-hidden z-50">
					<div class="p-4 border-b dark:border-gray-700 flex justify-between items-center">
						<h3 class="text-lg font-semibold text-gray-900 dark:text-white">Notifications</h3>
						<button 
							class="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
							on:click={() => showNotifications = false}
						>
							<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					</div>
					<UserMessages />
				</div>
			{/if}
		</div>
		<DarkMode />
		{#if currentUser}
			<UserMenu user={currentUser} />
		{:else}
			<p>Loading user...</p>
		{/if}
	</div>
</Navbar>

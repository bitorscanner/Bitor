<script>
	import '../../app.pcss';
	import Navbar from './Navbar.svelte';
	import Sidebar from './Sidebar.svelte';
	import ToastContainer from '$lib/components/ToastContainer.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pocketbase } from '@lib/stores/pocketbase';

	let drawerHidden = false;

	onMount(() => {
		// Check if the user is authenticated
		if (!$pocketbase.authStore.isValid) {
			// Redirect to the login page if not authenticated
			goto('/authentication/sign-in');
		}
	});
</script>

<div class="min-h-screen bg-gray-50 dark:bg-gray-900">
	<header class="fixed top-0 z-50 w-full">
		<Navbar bind:drawerHidden />
	</header>

	<div class="flex pt-16">
		<Sidebar bind:drawerHidden />
		<main class="relative h-full w-full overflow-y-auto bg-gray-50 dark:bg-gray-900 lg:ml-64">
			<div class="p-4">
				<slot />
			</div>
		</main>
	</div>
	
	<!-- Toast Notifications -->
	<ToastContainer />
</div>

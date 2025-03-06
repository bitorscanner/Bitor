<script lang="ts">
	import { Avatar, Dropdown, DropdownDivider, DropdownHeader, DropdownItem } from 'flowbite-svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { goto } from '$app/navigation';

	export let user: { name: string; email: string; avatar?: string; role?: string };

	function signOut() {
		$pocketbase.authStore.clear();
		goto('/authentication/sign-in');
	}

	function goToSettings() {
		goto('/settings');
	}

	function goToAccount() {
		goto('/settings/account');
	}

	// Get avatar URL using PocketBase's file URL method
	$: avatarSrc = user?.avatar ? 
		$pocketbase.files.getUrl(user, user.avatar) : 
		'/images/default_avatar.png';

	function handleError(e: Event) {
		if (e.target instanceof HTMLImageElement) {
			e.target.src = '/images/default_avatar.png';
		}
	}
</script>

<button class="ms-3 rounded-full ring-gray-400 focus:ring-4 dark:ring-gray-600">
	<Avatar 
		size="sm" 
		src={avatarSrc} 
		tabindex={0}
		on:error={handleError}
	/>
</button>
<Dropdown placement="bottom-end">
	<DropdownHeader>
		<span class="block text-sm">{user?.name || 'User'}</span>
		<span class="block truncate text-sm font-medium">{user?.email || 'user@example.com'}</span>
	</DropdownHeader>
	<DropdownItem on:click={goToAccount}>Account Settings</DropdownItem>
	{#if $pocketbase.authStore.isAdmin || user?.role === 'admin'}
		<DropdownItem on:click={goToSettings}>System Settings</DropdownItem>
	{/if}
	<DropdownDivider />
	<DropdownItem on:click={signOut}>Sign out</DropdownItem>
</Dropdown>

<!--
@component
[Go to docs](https://flowbite-svelte-admin-dashboard.vercel.app/)
## Props
@prop export let id: number = 0;
@prop export let name: string = '';
@prop export let avatar: string = '';
@prop export let email: string = '';
@prop export let biography: string = '';
@prop export let position: string = '';
@prop export let country: string = '';
@prop export let status: string = '';
-->

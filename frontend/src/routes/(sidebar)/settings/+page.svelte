<script lang="ts">
	import { Breadcrumb, BreadcrumbItem, Card } from 'flowbite-svelte';
	import { 
		UsersGroupSolid, 
		CogSolid, 
		LockSolid,
		BellSolid, 
		UserCircleSolid,
	} from 'flowbite-svelte-icons';
	import { goto } from '$app/navigation';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { currentUser } from '$lib/stores/auth';

	$: userPermissions = $currentUser?.group?.permissions || {};
	$: isAdmin = $pocketbase?.authStore?.isAdmin || userPermissions?.manage_users === true;

	const settingsCategories = [
		{
			title: 'Account Settings',
			description: 'Manage your profile, preferences, and security settings',
			icon: UserCircleSolid,
			href: '/settings/account',
			permission: null // Always visible
		},
		{
			title: 'System Settings',
			description: 'Configure system-wide settings and limits',
			icon: CogSolid,
			href: '/settings/system',
			permission: 'manage_system'
		},
		{
			title: 'User Management',
			description: 'Manage users, roles, and permissions',
			icon: UsersGroupSolid,
			href: '/settings/users',
			permission: 'manage_users'
		},
		{
			title: 'Groups',
			description: 'Manage user groups and their permissions',
			icon: UsersGroupSolid,
			href: '/settings/groups',
			permission: 'manage_groups'
		},
		{
			title: 'Providers',
			description: 'Manage cloud and service providers',
			icon: CogSolid,
			href: '/settings/providers',
			permission: 'manage_providers'
		},
		{
			title: 'API Keys',
			description: 'Manage API keys and access tokens',
			icon: LockSolid,
			href: '/settings/api_keys',
			permission: 'manage_api_keys'
		},
		{
			title: 'Notifications',
			description: 'Configure notification preferences and channels',
			icon: BellSolid,
			href: '/settings/notifications',
			permission: 'manage_notifications'
		}
	];

	function hasPermission(permission: string | null): boolean {
		if (permission === null) return true;
		if ($pocketbase?.authStore?.isAdmin) return true;
		return userPermissions[permission] === true;
	}
</script>

<div class="container mx-auto px-4 py-6">
	<Breadcrumb class="mb-4">
		<BreadcrumbItem href="/">Home</BreadcrumbItem>
		<BreadcrumbItem>Settings</BreadcrumbItem>
	</Breadcrumb>

	<h1 class="text-2xl font-bold mb-6">Settings</h1>

	<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
		{#each settingsCategories as category}
			{#if hasPermission(category.permission)}
				<Card padding="xl" class="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors" 
					  on:click={() => goto(category.href)}>
					<div class="flex items-start">
						<div class="flex-shrink-0">
							<svelte:component 
								this={category.icon} 
								class="w-8 h-8 text-primary-600 dark:text-primary-500" 
							/>
						</div>
						<div class="ml-4">
							<h3 class="text-lg font-semibold text-gray-900 dark:text-white">
								{category.title}
							</h3>
							<p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
								{category.description}
							</p>
						</div>
					</div>
				</Card>
			{/if}
		{/each}
	</div>
</div>
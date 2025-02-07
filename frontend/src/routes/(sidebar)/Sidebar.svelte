<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { pocketbase } from '$lib/stores/pocketbase';
	import { currentUser } from '$lib/stores/auth';

	import {
		Sidebar,
		SidebarDropdownWrapper,
		SidebarGroup,
		SidebarItem,
		SidebarWrapper
	} from 'flowbite-svelte';
	import {
		AngleDownOutline,
		AngleUpOutline,
		ClipboardListSolid,
		CogOutline,
		FileChartBarSolid,
		GithubSolid,
		LayersSolid,
		LifeSaverSolid,
		LockSolid,
		WandMagicSparklesOutline,
		ChartPieOutline,
		RectangleListSolid,
		ArrowRightAltSolid,
		TableColumnSolid,
		ProfileCardSolid,
		SearchSolid,
		InboxFullSolid,
		CalendarMonthSolid,
		UserCircleSolid,
		UsersGroupSolid,
		BellSolid,
		CogSolid,
		ServerSolid
	} from 'flowbite-svelte-icons';

	// Import the Target icon from lucide-svelte
	import { Target, UserRoundSearch } from 'lucide-svelte';

	export let drawerHidden: boolean = false;

	const closeDrawer = () => {
		drawerHidden = true;
	};

	let iconClass =
		'flex-shrink-0 w-6 h-6 text-gray-500 transition duration-75 group-hover:text-gray-900 dark:text-gray-400 dark:group-hover:text-white';
	let itemClass =
		'flex items-center p-2 text-base text-gray-900 transition duration-75 rounded-lg hover:bg-gray-100 group dark:text-gray-200 dark:hover:bg-gray-700';
	let groupClass = 'pt-2 space-y-2';

	$: mainSidebarUrl = $page.url.pathname;
	let activeMainSidebar: string;

	// Initialize dropdowns state
	let dropdowns: { [key: string]: boolean } = {};

	// Update dropdowns when URL changes
	$: {
		const path = $page.url.pathname;
		if (path.startsWith('/settings')) {
			dropdowns['Settings'] = true;
		}
	}

	afterNavigate((navigation) => {
		// this fixes https://github.com/themesberg/flowbite-svelte/issues/364
		document.getElementById('svelte')?.scrollTo({ top: 0 });
		closeDrawer();

		activeMainSidebar = navigation.to?.url.pathname ?? '';
	});

	$: isAdmin = $pocketbase?.authStore?.isAdmin || $currentUser?.group?.name === 'admin';

	$: posts = [
		{ name: 'Dashboard', icon: ChartPieOutline, href: '/dashboard' },
		{ name: 'Findings', icon: InboxFullSolid, href: '/findings' },
		{ name: 'Scans', icon: SearchSolid, href: '/scans' },
		{ name: 'Targets', icon: Target, href: '/targets' },
		{ name: 'Profiles', icon: ProfileCardSolid, href: '/profiles' },
		{ name: 'Templates', icon: ArrowRightAltSolid, href: '/templates' },
		{ name: 'Clients', icon: UserRoundSearch, href: '/clients' },
		// Commenting out schedule for beta release as feature is not complete
		// { name: 'Schedule', icon: CalendarMonthSolid, href: '/schedule' },
		// Show settings for admin users or users with admin group
		...(($pocketbase?.authStore?.isAdmin || $currentUser?.group?.name === 'admin') ? [{
			name: 'Settings',
			icon: CogOutline,
			children: {
				'Account': {
					href: '/settings/account',
					icon: UserCircleSolid
				},
				'Users': {
					href: '/settings/users',
					icon: UsersGroupSolid
				},
				'Groups': {
					href: '/settings/groups',
					icon: UsersGroupSolid
				},
				'Providers': {
					href: '/settings/providers',
					icon: ServerSolid
				},
				// Commenting out notifications for beta release as feature is not complete
				// 'Notifications': {
				// 	href: '/settings/notifications',
				// 	icon: BellSolid
				// },
				'System': {
					href: '/settings/system',
					icon: CogSolid
				}
			}
		}] : [])
	];

	let links = [
		{
			label: 'GitHub Repository',
			href: 'https://github.com/orbitscanner/orbit',
				icon: GithubSolid
		},
		{
			label: 'Documentation',
			href: 'https://orbitscanner.io',
			icon: LifeSaverSolid
		}
	];

	// Initialize dropdowns for each post with children
	onMount(() => {
		posts.forEach(post => {
			if (post.children) {
				dropdowns[post.name] = $page.url.pathname.startsWith(`/${post.name.toLowerCase()}`);
			}
		});
	});

	let version = import.meta.env.DEV ? 'development' : 'unknown';

	onMount(async () => {
		// Only check for version updates in production
		if (!import.meta.env.DEV) {
			try {
				const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/version/check`);
				if (response.ok) {
					const data = await response.json();
					version = data.current_version;
				} else {
					console.error('Failed to fetch version');
				}
			} catch (error) {
				console.error('Error fetching version:', error);
			}
		}
	});
</script>

<Sidebar
	class={drawerHidden ? 'hidden' : ''}
	activeUrl={mainSidebarUrl}
	activeClass="bg-gray-100 dark:bg-gray-700"
	asideClass="fixed inset-0 z-30 flex-none h-full w-64 lg:h-auto border-e border-gray-200 dark:border-gray-600 lg:overflow-y-visible lg:pt-16 lg:block"
>
	<h4 class="sr-only">Main menu</h4>
	<SidebarWrapper
		divClass="overflow-y-auto px-3 pt-20 lg:pt-5 h-full bg-white scrolling-touch max-w-2xs lg:h-[calc(100vh-4rem)] lg:block dark:bg-gray-800 lg:mr-0 lg:sticky top-2"
	>
		<nav class="divide-y divide-gray-200 dark:divide-gray-700 pb-4">
			<SidebarGroup ulClass={groupClass} class="mb-3">
				{#each posts as { name, icon, children, href } (name)}
					{#if children}
						<SidebarDropdownWrapper bind:isOpen={dropdowns[name]} label={name} class="pr-3">
							<AngleDownOutline slot="arrowdown" strokeWidth="3.3" size="sm" />
							<AngleUpOutline slot="arrowup" strokeWidth="3.3" size="sm" />
							<svelte:component this={icon} slot="icon" class={iconClass} />

							{#each Object.entries(children) as [title, item]}
								<SidebarItem
									label={title}
									href={item.href}
									spanClass="ml-9"
									class={itemClass}
								>
									<svelte:component 
										this={item.icon} 
										slot="icon" 
										class={iconClass}
									/>
								</SidebarItem>
							{/each}
						</SidebarDropdownWrapper>
					{:else}
						<SidebarItem
							label={name}
							{href}
								spanClass="ml-3"
								class={itemClass}
						>
							<svelte:component this={icon} slot="icon" class={iconClass} />
						</SidebarItem>
					{/if}
				{/each}
			</SidebarGroup>
			<SidebarGroup ulClass={groupClass}>
				{#each links as { label, href, icon } (label)}
					<SidebarItem
						{label}
						{href}
						spanClass="ml-3"
						class={itemClass}
						target="_blank"
					>
						<svelte:component this={icon} slot="icon" class={iconClass} />
					</SidebarItem>
				{/each}
			</SidebarGroup>
		</nav>

		<div class="mt-4">
			<p class="text-xs text-gray-500 dark:text-gray-400 text-center">
				Version: {version}
			</p>
		</div>
	</SidebarWrapper>
</Sidebar>

{#if !drawerHidden}
	<div
		class="fixed inset-0 z-20 bg-gray-900/50 dark:bg-gray-900/60 lg:hidden"
		on:click={closeDrawer}
		on:keydown={closeDrawer}
		role="presentation"
	/>
{/if}

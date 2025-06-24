<script lang="ts">
	import { dndzone, type DndEvent } from 'svelte-dnd-action';
	import RunningScans from '@utils/dashboard/RunningScans.svelte';
	import CompletedScans from '@utils/dashboard/CompletedScans.svelte';
	import FailedScans from '@utils/dashboard/FailedScans.svelte';
	import OpenVulnerabilitiesChart from '@utils/dashboard/OpenVulnerabilitiesChart.svelte';
	import VulnerabilitiesByClient from '@utils/dashboard/VulnerabilitiesByClient.svelte';
	import RecentFindings from '@utils/dashboard/RecentFindings.svelte';
	import MetaTag from '@utils/MetaTag.svelte';
	import Footer from '../Footer.svelte';
	import { onMount } from 'svelte';
	import { fade } from 'svelte/transition';
	import { flip } from 'svelte/animate';

	const path: string = '/dashboard';
	const description: string = 'Dashboard Bitor';
	const title: string = 'Dashboard Bitor';
	const subtitle: string = 'Dashboard Bitor';

	interface DashboardItem {
		id: string;
		title: string;
		component: keyof typeof componentMap;
	}

	const componentMap = {
		RunningScans,
		CompletedScans,
		FailedScans,
		VulnerabilitiesByClient,
		OpenVulnerabilitiesChart,
		RecentFindings
	} as const;

	let items: DashboardItem[] = [
		{ id: 'running', title: 'Running Scans', component: 'RunningScans' },
		{ id: 'completed', title: 'Completed Scans', component: 'CompletedScans' },
		{ id: 'failed', title: 'Failed Scans', component: 'FailedScans' },
		{ id: 'vulnerabilities', title: 'Vulnerabilities by Client', component: 'VulnerabilitiesByClient' },
		{ id: 'open-vulnerabilities', title: 'Open Vulnerabilities', component: 'OpenVulnerabilitiesChart' },
		{ id: 'recent', title: 'Recent Findings', component: 'RecentFindings' }
	];

	onMount(() => {
		const savedLayout = localStorage.getItem('dashboardLayout');
		if (savedLayout) {
			try {
				const parsed = JSON.parse(savedLayout);
				if (Array.isArray(parsed) && parsed.every((item: DashboardItem) => 
					typeof item === 'object' && 
					'id' in item && 
					'title' in item && 
					'component' in item
				)) {
					items = parsed;
				}
			} catch (e) {
				console.error('Failed to load dashboard layout:', e);
			}
		}
	});

	function handleDndConsider(e: CustomEvent<DndEvent<DashboardItem>>) {
		items = e.detail.items;
	}

	function handleDndFinalize(e: CustomEvent<DndEvent<DashboardItem>>) {
		items = e.detail.items;
		try {
			localStorage.setItem('dashboardLayout', JSON.stringify(items));
		} catch (e) {
			console.error('Failed to save dashboard layout:', e);
		}
	}
</script>

<MetaTag {path} {description} {title} {subtitle} />

<main class="p-4 space-y-6">
	<div
		use:dndzone={{items, flipDurationMs: 300}}
		on:consider={handleDndConsider}
		on:finalize={handleDndFinalize}
		class="grid grid-cols-1 md:grid-cols-2 gap-6"
	>
		{#each items as item (item.id)}
			<div
				class="relative bg-white dark:bg-gray-800 rounded-lg shadow-lg ring-1 ring-black/5 dark:ring-white/10
					   transition-all duration-300 ease-out hover:shadow-xl group"
			>
				<!-- Drag Handle -->
				<div class="absolute top-3 right-3 cursor-move opacity-0 group-hover:opacity-100 transition-opacity">
					⋮⋮
				</div>
				
				<!-- Component Content -->
				<div class="p-4">
					<svelte:component this={componentMap[item.component]} />
				</div>
			</div>
		{/each}
	</div>
</main>

<Footer />

<style>
	:global(.draggable-dropzone--is-dragged-over) {
		@apply bg-primary-50 dark:bg-primary-900/20;
	}

	:global(.draggable-dropzone--active) {
		opacity: 0.6;
		animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
	}

	@keyframes pulse {
		0%, 100% {
			opacity: 0.6;
		}
		50% {
			opacity: 0.8;
		}
	}
</style>
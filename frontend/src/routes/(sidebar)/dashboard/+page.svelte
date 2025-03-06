<script lang="ts">
	import RunningScans from '@utils/dashboard/RunningScans.svelte';
	import CompletedScans from '@utils/dashboard/CompletedScans.svelte';
	import FailedScans from '@utils/dashboard/FailedScans.svelte';
	import OpenVulnerabilitiesChart from '@utils/dashboard/OpenVulnerabilitiesChart.svelte';
	import VulnerabilitiesByClient from '@utils/dashboard/VulnerabilitiesByClient.svelte';
	import RecentFindings from '@utils/dashboard/RecentFindings.svelte';
	import MetaTag from '@utils/MetaTag.svelte';
	import Footer from '../Footer.svelte';

	const path: string = '/dashboard';
  const description: string = 'Dashboard Orbit';
  const title: string = 'Dashboard Orbit';
  const subtitle: string = 'Dashboard Orbit';
</script>

<MetaTag {path} {description} {title} {subtitle} />

<main class="p-4 space-y-6">
	<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
		<RunningScans />
		<CompletedScans />
		<FailedScans />
		<VulnerabilitiesByClient />
		<OpenVulnerabilitiesChart />
	</div>
	<RecentFindings />
</main>
<Footer />
<script>
	import { Breadcrumb, BreadcrumbItem } from 'flowbite-svelte';
	import NucleiSettings from './NucleiSettings.svelte';
	import ScanProfiles from './ScanProfiles.svelte';
</script>

<div class="p-4 space-y-8">
	<Breadcrumb class="mb-6">
		<BreadcrumbItem home>Home</BreadcrumbItem>
		<BreadcrumbItem>Profiles</BreadcrumbItem>
	</Breadcrumb>

	<NucleiSettings />
	<ScanProfiles />
</div>

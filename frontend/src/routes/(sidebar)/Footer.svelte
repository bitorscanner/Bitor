<script lang="ts">
	import { Frame, type LinkType } from 'flowbite-svelte';
	import {
		GithubSolid
	} from 'flowbite-svelte-icons';
	import type { ComponentType } from 'svelte';

	const links: LinkType[] = [
		{ name: 'Terms and conditions', href: '#' },
		{ name: 'Licensing', href: '#' },
	];

	const brands: [ComponentType, string][] = [
		[GithubSolid, ''],
	];
</script>

<Frame
	tag="footer"
	rounded
	shadow
	class="mx-4 my-2 rounded-lg bg-white p-4 shadow dark:bg-gray-800 md:flex md:items-center md:justify-between md:p-6 xl:p-8"
>
	<ul class="mb-6 flex flex-wrap items-center space-y-1 sm:gap-4 md:mb-0 xl:gap-6">
		{#each links as { name, href }}
			<li>
				<a
					{href}
					class="mr-4 text-sm font-normal text-gray-500 hover:underline dark:text-gray-400 sm:mr-6"
				>
					{name}
				</a>
			</li>
		{/each}
	</ul>
	<div class="flex space-x-6 sm:justify-center">
		{#each brands as [component, href]}
			<a {href} class="text-gray-500 hover:text-gray-900 dark:text-gray-400 dark:hover:text-white">
				<svelte:component this={component} size="md" />
			</a>
		{/each}
	</div>
</Frame>

<p class="my-10 text-center text-sm text-gray-500">
	© 2025 <a href="https://bitor.dev/" class="hover:underline" target="_blank">
		Bitor Scanner
	</a>. All rights reserved.
</p>

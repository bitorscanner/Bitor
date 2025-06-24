<script lang="ts">
	import { Button } from 'flowbite-svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { imagesPath } from '$lib/utils/variables';
	import '../app.pcss';
	import { hackerQuotes } from '$lib/utils/quotes';
	import { onMount } from 'svelte';

	let currentQuote = hackerQuotes[0];

	onMount(() => {
		// Sort by rating and get top 5
		const topQuotes = [...hackerQuotes].sort((a, b) => b.rating - a.rating).slice(0, 5);
		currentQuote = topQuotes[Math.floor(Math.random() * topQuotes.length)];
	});

	function goBack() {
		window.history.back();
	}

	function goHome() {
		goto('/');
	}
</script>

<div class="bg-gradient-to-br from-gray-900 via-gray-800 to-gray-900 min-h-screen">
	<div class="flex flex-col items-center justify-center px-6 pt-8 mx-auto h-screen xl:px-0">
		<div class="block md:max-w-lg mb-8 relative">
			<img 
				src="/images/Bitor_White_Logo.png" 
				alt="Bitor Logo"
				class="w-auto h-24 mx-auto mb-4 object-contain animate-pulse drop-shadow-[0_0_15px_rgba(59,130,246,0.5)]" 
			/>
			<div class="absolute inset-0 bg-blue-500/10 blur-2xl rounded-full animate-pulse"></div>
		</div>
		<div class="text-center xl:max-w-4xl relative backdrop-blur-sm bg-white/10 p-8 rounded-lg border border-blue-500/30">
			<div class="cyber-scanner"></div>
			<h1 class="mb-3 text-2xl font-bold leading-tight text-blue-100 sm:text-4xl lg:text-5xl glitch-text">
				{$page?.error?.message || 'Page Not Found'}
			</h1>
			<div class="my-8 p-6 bg-gray-900/60 rounded border border-blue-500/30 quote-container">
				<p class="mb-3 text-lg font-mono text-blue-200">"{currentQuote.text}"</p>
				<p class="text-sm font-mono text-blue-400">// {currentQuote.author}</p>
			</div>
			<div class="flex flex-col sm:flex-row items-center justify-center gap-4">
				<Button size="lg" color="primary" class="cyber-button w-full sm:w-auto" on:click={goBack}>
					<svg
						class="-ml-1 mr-2 h-5 w-5"
						fill="currentColor"
						viewBox="0 0 20 20"
						xmlns="http://www.w3.org/2000/svg"
					>
						<path
							fill-rule="evenodd"
							d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z"
							clip-rule="evenodd"
						/>
					</svg>
					Go Back
				</Button>
				<Button size="lg" color="alternative" class="cyber-button w-full sm:w-auto" on:click={goHome}>
					Go Home
				</Button>
			</div>
		</div>
	</div>
</div>

<style>
	.cyber-scanner {
		position: absolute;
		top: 0;
		left: 0;
		width: 100%;
		height: 2px;
		background: linear-gradient(
			to right,
			transparent,
			#3b82f6,
			transparent
		);
		animation: scan 2s linear infinite;
	}

	@keyframes scan {
		0% {
			transform: translateY(0);
		}
		50% {
			transform: translateY(100%);
		}
		100% {
			transform: translateY(0);
		}
	}

	.quote-container {
		position: relative;
		overflow: hidden;
	}

	.quote-container::before,
	.quote-container::after {
		content: '';
		position: absolute;
		width: 100%;
		height: 1px;
		background: linear-gradient(90deg, transparent, #3b82f6, transparent);
	}

	.quote-container::before {
		top: 0;
		animation: slide-right 3s linear infinite;
	}

	.quote-container::after {
		bottom: 0;
		animation: slide-left 3s linear infinite;
	}

	@keyframes slide-right {
		from { transform: translateX(-100%); }
		to { transform: translateX(100%); }
	}

	@keyframes slide-left {
		from { transform: translateX(100%); }
		to { transform: translateX(-100%); }
	}

	.glitch-text {
		text-shadow: 
			0.05em 0 0 rgba(255, 0, 0, 0.75),
			-0.025em -0.05em 0 rgba(0, 255, 0, 0.75),
			0.025em 0.05em 0 rgba(0, 0, 255, 0.75);
		animation: glitch 500ms infinite;
	}

	@keyframes glitch {
		0% {
			text-shadow: 
				0.05em 0 0 rgba(255, 0, 0, 0.75),
				-0.05em -0.025em 0 rgba(0, 255, 0, 0.75),
				-0.025em 0.05em 0 rgba(0, 0, 255, 0.75);
		}
		15% {
			text-shadow: 
				-0.05em -0.025em 0 rgba(255, 0, 0, 0.75),
				0.025em 0.025em 0 rgba(0, 255, 0, 0.75),
				-0.05em -0.05em 0 rgba(0, 0, 255, 0.75);
		}
		50% {
			text-shadow: 
				0.025em 0.05em 0 rgba(255, 0, 0, 0.75),
				0.05em 0 0 rgba(0, 255, 0, 0.75),
				0 -0.05em 0 rgba(0, 0, 255, 0.75);
		}
	}

	.cyber-button {
		position: relative;
		overflow: hidden;
		transition: all 0.3s ease;
		border: 1px solid rgba(59, 130, 246, 0.3);
	}

	.cyber-button:hover {
		transform: translateY(-2px);
		box-shadow: 0 0 15px rgba(59, 130, 246, 0.5);
	}

	.cyber-button::before {
		content: '';
		position: absolute;
		top: 0;
		left: -100%;
		width: 100%;
		height: 100%;
		background: linear-gradient(
			90deg,
			transparent,
			rgba(59, 130, 246, 0.2),
			transparent
		);
		transition: 0.5s;
	}

	.cyber-button:hover::before {
		left: 100%;
	}
</style>

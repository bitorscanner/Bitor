<script lang="ts">
	import { Label, Input } from 'flowbite-svelte';
	import SignIn from '@utils/authentication/SignIn.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { writable } from 'svelte/store';
	import { Alert } from 'flowbite-svelte';
	import { InfoCircleSolid } from 'flowbite-svelte-icons';

	const errorMessage = writable('');

	// onMount(() => {
	// 	// Check if the user is authenticated
	// 	if ($pocketbase.authStore.isValid) {
	// 		// Redirect to the dashboard
	// 		goto('/dashboard');
	// 	}
	// });

	const title = 'Sign in to Orbit';
	const site = {
		name: 'Orbit',
		img: '/images/Orbit-Main-Logo.png',
		link: '/',
		imgAlt: 'Orbit Logo'
	};
	const rememberMe = false;
	const lostPassword = false;
	const createAccount = false;
	const lostPasswordLink = '';
	const loginTitle = 'Login to your account';
	const registerLink = '';
	const createAccountTitle = 'Create account';

	const onSubmit = async (e: Event) => {
		e.preventDefault();
		const formData = new FormData(e.target as HTMLFormElement);
		const email = formData.get('email') as string;
		const password = formData.get('password') as string;

		try {
			// Attempt to authenticate as a regular user
			let authData = await $pocketbase.collection('users').authWithPassword(email, password);
			console.log('Logged in as user:', authData);

			// Check if password change is required
			if (authData.record.requirePasswordChange) {
				console.log('Password change required, redirecting...');
				goto('/change-password');
				return;
			}

			// Redirect to dashboard
			goto('/dashboard');
		} catch (userError) {
			console.error('User login failed:', userError);

			try {
				// Attempt to authenticate as an admin
				let authData = await $pocketbase.admins.authWithPassword(email, password);
				console.log('Logged in as admin:', authData);

				// Redirect to dashboard
				goto('/dashboard');
			} catch (adminError) {
				console.error('Admin login failed:', adminError);
				// Set a generic error message
				errorMessage.set('Invalid username or password. Please try again.');
			}
		}
	};
</script>

<SignIn
	{title}
	{site}
	{rememberMe}
	{lostPassword}
	{createAccount}
	{lostPasswordLink}
	{loginTitle}
	{registerLink}
	{createAccountTitle}
	on:submit={onSubmit}
>
	<div>
		<Label for="email" class="mb-2 dark:text-white">Your email</Label>
		<Input
			type="email"
			name="email"
			id="email"
			placeholder="name@company.com"
			required
			class="border outline-none dark:border-gray-600 dark:bg-gray-700"
		/>
	</div>
	<div>
		<Label for="password" class="mb-2 dark:text-white">Your password</Label>
		<Input
			type="password"
			name="password"
			id="password"
			placeholder="••••••••"
			required
			class="border outline-none dark:border-gray-600 dark:bg-gray-700"
		/>
	</div>
	{#if $errorMessage}
		<Alert color="red" class="mt-4">
			<InfoCircleSolid slot="icon" class="w-5 h-5" />
			{$errorMessage}
		</Alert>
	{/if}
</SignIn>

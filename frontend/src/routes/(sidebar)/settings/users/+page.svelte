<script lang="ts">
	import { onMount } from 'svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import Icon from '@iconify/svelte';
	import {
		Table,
		TableBody,
		TableBodyCell,
		TableBodyRow,
		TableHead,
		TableHeadCell,
		Button,
		Badge,
		Card,
		Checkbox,
		Breadcrumb,
		BreadcrumbItem
	} from 'flowbite-svelte';
	import { TrashBinSolid, PenSolid, HomeSolid } from 'flowbite-svelte-icons';
	import ChangePassword from './ChangePassword.svelte';
	import AddUser from './AddUser.svelte';
	import InviteUser from './InviteUser.svelte';
	import type { RecordModel } from 'pocketbase';

	let users: RecordModel[] = [];
	let adminUser: any = null;
	let loading = true;
	let error = '';
	let selectedUser: RecordModel | null = null;
	let showPasswordModal = false;
	let showEditModal = false;
	let showInviteModal = false;

	onMount(async () => {
		await loadUsers();
	});

	async function loadUsers() {
		try {
			loading = true;
			// Fetch regular users
			const result = await $pocketbase.collection('users').getList(1, 50, {
				sort: '-created',
				expand: 'group',
				$autoCancel: false
			});
			
			// Only fetch admin if the current user is a super admin
			if ($pocketbase.authStore.isAdmin) {
				try {
					const adminList = await $pocketbase.admins.getFullList();
					adminUser = adminList[0];
					users = [
						{
							...adminUser,
							isAdmin: true,
							email: adminUser.email,
							group: { name: 'Super Admin' }
						},
						...result.items
					];
				} catch (error) {
					console.error('Error fetching admin:', error);
					users = result.items;
				}
			} else {
				users = result.items;
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load users';
		} finally {
			loading = false;
		}
	}

	async function deleteUser(id: string, isAdmin: boolean = false) {
		if (isAdmin) {
			error = 'Cannot delete super admin user';
			return;
		}

		if (!confirm('Are you sure you want to delete this user?')) {
			return;
		}
		
		try {
			await $pocketbase.collection('users').delete(id);
				users = users.filter(user => user.id !== id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to delete user';
		}
	}

	function getUserInitials(user: RecordModel): string {
		if (user.isAdmin) return 'SA';
		const firstName = user.first_name || '';
		const lastName = user.last_name || '';
		if (!firstName && !lastName) return '?';
		return (firstName[0] || '') + (lastName[0] || '');
	}

	function getFullName(user: RecordModel): string {
		if (user.isAdmin) return 'Super Admin';
		const firstName = user.first_name || '';
		const lastName = user.last_name || '';
		const fullName = [firstName, lastName].filter(Boolean).join(' ');
		return fullName || 'No name set';
	}

	function handleUserSave() {
		loadUsers();
		showEditModal = false;
	}

	function handleAvatarError(e: Event) {
		if (e.target instanceof HTMLImageElement) {
			e.target.src = '/images/default_avatar.png';
		}
	}
</script>

<div class="w-full p-4">
	<Breadcrumb class="mb-4">
		<BreadcrumbItem home href="/">Home</BreadcrumbItem>
		<BreadcrumbItem href="/settings">Settings</BreadcrumbItem>
		<BreadcrumbItem>Users</BreadcrumbItem>
	</Breadcrumb>

	<Card padding="xl" class="min-w-full">
		<div class="flex justify-between items-center mb-6">
			<h1 class="text-3xl font-bold">Users</h1>
			<div class="flex gap-2">
				<Button color="primary" on:click={() => {
					selectedUser = null;
					showEditModal = true;
				}}>Add User</Button>
				<Button color="blue" on:click={() => {
					showInviteModal = true;
				}}>Invite User</Button>
			</div>
		</div>

		<Table class="mt-4 border border-gray-200 dark:border-gray-700">
			<TableHead class="bg-gray-100 dark:bg-gray-700">
				<TableHeadCell class="w-4 p-4"><Checkbox /></TableHeadCell>
				{#each ['Name', 'Email', 'Group', 'Actions'] as title}
					<TableHeadCell class="ps-4 font-normal">{title}</TableHeadCell>
				{/each}
			</TableHead>
			<TableBody>
				{#each users as user}
					<TableBodyRow class="text-base hover:bg-gray-50 dark:hover:bg-gray-800">
						<TableBodyCell class="w-4 p-4"><Checkbox /></TableBodyCell>
						<TableBodyCell class="p-4">
							<div class="flex items-center space-x-3">
								{#if user.avatar}
									<img 
										class="w-8 h-8 rounded-full" 
										src={$pocketbase.files.getUrl(user, user.avatar)} 
										alt="User avatar"
										on:error={handleAvatarError}
									/>
								{:else}
									<div class="w-8 h-8 rounded-full bg-blue-600 flex items-center justify-center text-white text-sm font-medium">
										{getUserInitials(user)}
									</div>
								{/if}
								<span class="font-medium text-gray-900">{getFullName(user)}</span>
							</div>
						</TableBodyCell>
						<TableBodyCell class="p-4">{user.email}</TableBodyCell>
						<TableBodyCell class="p-4">
							{#if user.isAdmin}
								<Badge color="red">Super Admin</Badge>
							{:else if user.expand?.group?.name}
								<Badge color="blue">{user.expand.group.name}</Badge>
							{:else}
								<Badge color="dark">No Group</Badge>
							{/if}
						</TableBodyCell>
						<TableBodyCell class="space-x-2">
							<Button size="sm" class="gap-2 px-3" on:click={() => {
								selectedUser = user;
								showEditModal = true;
							}}>
								<PenSolid class="w-4 h-4" />
								Edit
							</Button>
							<Button
								size="sm"
								class="gap-2 px-3"
								color="blue"
								on:click={() => {
									selectedUser = user;
									showPasswordModal = true;
								}}
							>
								<Icon icon="mdi:key" class="w-4 h-4" />
								Reset Password
							</Button>
							{#if !user.isAdmin}
								<Button
									color="red"
									size="sm"
									class="gap-2 px-3"
									on:click={() => deleteUser(user.id, user.isAdmin)}
								>
									<TrashBinSolid class="w-4 h-4" />
									Delete
								</Button>
							{/if}
						</TableBodyCell>
					</TableBodyRow>
				{/each}
			</TableBody>
		</Table>
	</Card>
</div>

<ChangePassword 
	bind:open={showPasswordModal} 
	userId={selectedUser?.id} 
/>

<AddUser
	bind:open={showEditModal}
	data={selectedUser}
	mode={selectedUser ? 'edit' : 'create'}
	onSave={handleUserSave}
/>

<InviteUser
	bind:open={showInviteModal}
	on:save={loadUsers}
/>


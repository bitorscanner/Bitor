<script lang="ts">
    import { Button, Input, Label, Modal, Select, Checkbox } from 'flowbite-svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import type { RecordModel } from 'pocketbase';

    export let open = false;
    export let onSave: (user: RecordModel) => void;
    export let data: RecordModel | null = null;
    export let mode: 'create' | 'edit' = 'create';

    let loading = false;
    let error = '';
    let groups: RecordModel[] = [];
    let formData = {
        email: '',
        username: '',
        password: '',
        passwordConfirm: '',
        group: '',
        first_name: '',
        last_name: '',
        requirePasswordChange: false
    };
    let initialized = false;

    $: if (open && !initialized) {
        initializeForm();
    }

    $: if (!open) {
        initialized = false;
    }

    async function initializeForm() {
        await loadGroups();
        if (mode === 'edit' && data) {
            console.log('Editing user:', data);
            formData = {
                email: data.email || '',
                username: data.username || '',
                password: '',
                passwordConfirm: '',
                group: data.group || '',
                first_name: data.first_name || '',
                last_name: data.last_name || '',
                requirePasswordChange: data.requirePasswordChange || false
            };
            console.log('Form data initialized:', formData);
        } else {
            resetForm();
        }
        initialized = true;
    }

    function resetForm() {
        formData = {
            email: '',
            username: '',
            password: '',
            passwordConfirm: '',
            group: '',
            first_name: '',
            last_name: '',
            requirePasswordChange: false
        };
        error = '';
    }

    async function loadGroups() {
        try {
            const result = await $pocketbase.collection('groups').getList(1, 50);
            groups = result.items;
        } catch (err) {
            console.error('Error loading groups:', err);
            error = 'Failed to load groups';
        }
    }

    function isValidUsername(username: string): boolean {
        // Check if the username contains @ symbol (indicating an email)
        if (username.includes('@')) {
            return false;
        }
        // Add any additional username validation rules here
        return true;
    }

    async function handleSubmit() {
        console.log('Submitting form:', { mode, formData });
        
        if (!formData.email || !formData.group || !formData.username) {
            error = 'Please fill in all required fields';
            return;
        }

        if (!isValidUsername(formData.username)) {
            error = 'Username cannot be an email address. Please use a simple username without @ symbol.';
            return;
        }

        if (mode === 'create') {
            if (!formData.password) {
                error = 'Password is required for new users';
                return;
            }
            if (formData.password.length < 10) {
                error = 'Password must be at least 10 characters long';
                return;
            }
            if (formData.password !== formData.passwordConfirm) {
                error = 'Passwords do not match';
                return;
            }
        }

        try {
            loading = true;
            error = '';
            console.log('Processing submission for mode:', mode);
            
            let userData: Record<string, any> = {
                username: formData.username,
                email: formData.email,
                emailVisibility: true,
                first_name: formData.first_name,
                last_name: formData.last_name,
                group: formData.group,
                requirePasswordChange: formData.requirePasswordChange
            };

            if (mode === 'create') {
                userData.password = formData.password;
                userData.passwordConfirm = formData.passwordConfirm;
            }

            console.log('User data to submit:', userData);

            let user;
            if (mode === 'create') {
                console.log('Creating new user with data:', userData);
                user = await $pocketbase.collection('users').create(userData);
            } else if (data) {
                console.log('Updating user:', data.id);
                console.log('Update data:', userData);
                user = await $pocketbase.collection('users').update(data.id, userData);
            }

            console.log('Operation successful, user:', user);
            if (user) {
                onSave(user);
                open = false;
            }
        } catch (err) {
            console.error('Operation failed:', err);
            error = err instanceof Error ? err.message : `Failed to ${mode} user`;
        } finally {
            loading = false;
        }
    }
</script>

<Modal bind:open title={mode === 'create' ? 'Add New User' : 'Edit User'}>
    <form class="space-y-4" on:submit|preventDefault={handleSubmit}>
        <div>
            <Label for="first_name">First Name</Label>
            <Input id="first_name" type="text" bind:value={formData.first_name} placeholder="First name" />
        </div>

        <div>
            <Label for="last_name">Last Name</Label>
            <Input id="last_name" type="text" bind:value={formData.last_name} placeholder="Last name" />
        </div>

        <div>
            <Label for="username">Username *</Label>
            <Input id="username" type="text" bind:value={formData.username} placeholder="username" required />
            <p class="text-sm text-gray-500 mt-1">Username should be simple and cannot be an email address</p>
        </div>

        <div>
            <Label for="email">Email *</Label>
            <Input id="email" type="email" bind:value={formData.email} placeholder="user@example.com" required />
        </div>

        {#if mode === 'create'}
            <div>
                <Label for="password">Password *</Label>
                <Input 
                    id="password" 
                    type="password" 
                    bind:value={formData.password} 
                    required 
                    minlength={10}
                />
                <p class="text-sm text-gray-500 mt-1">Password must be at least 10 characters long</p>
            </div>

            <div>
                <Label for="passwordConfirm">Confirm Password *</Label>
                <Input 
                    id="passwordConfirm" 
                    type="password" 
                    bind:value={formData.passwordConfirm} 
                    required
                />
            </div>

            <div class="flex items-center gap-2">
                <Checkbox 
                    id="requirePasswordChange" 
                    bind:checked={formData.requirePasswordChange}
                />
                <Label for="requirePasswordChange" class="flex">
                    Require password change on first login
                </Label>
            </div>
        {/if}

        <div>
            <Label for="group">Group *</Label>
            <Select id="group" bind:value={formData.group} required>
                <option value="">Select a group</option>
                {#each groups as group}
                    <option value={group.id}>{group.name}</option>
                {/each}
            </Select>
        </div>

        {#if error}
            <p class="text-red-500">{error}</p>
        {/if}

        <div class="flex justify-end space-x-2">
            <Button color="alternative" on:click={() => (open = false)}>Cancel</Button>
            <Button type="submit" disabled={loading}>
                {loading ? (mode === 'create' ? 'Creating...' : 'Saving...') : (mode === 'create' ? 'Create User' : 'Save Changes')}
            </Button>
        </div>
    </form>
</Modal> 
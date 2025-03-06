<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import FileExplorer from './FileExplorer.svelte';
	import { pocketbase } from '@lib/stores/pocketbase';
	import { Button, Toast } from 'flowbite-svelte';
	import { CheckCircleSolid, ExclamationCircleSolid } from 'flowbite-svelte-icons';

	let monaco: typeof import('monaco-editor');
	let editor: import('monaco-editor').editor.IStandaloneCodeEditor;
	let editorContainer: HTMLDivElement;
	let currentFilePath = '';
	let isCustomFile = false;

	let showToast = false;
	let toastMessage = '';
	let toastColor = 'green'; // 'green' for success, 'red' for error
	let toastIcon = CheckCircleSolid;

	let searchQuery = '';
	let fileExplorerRef;

	let darkMode = false;
	let observer: MutationObserver;

	function onSelectFile(filePath: string, isCustom: boolean) {
		console.log('File selected:', filePath, 'isCustom:', isCustom);
		currentFilePath = filePath;
		isCustomFile = isCustom;
		fetchFileContent(filePath, isCustom);
	}

	async function fetchFileContent(filePath: string, isCustom: boolean) {
		try {
			const token = $pocketbase.authStore.token;
			const response = await fetch(
				`${import.meta.env.VITE_API_BASE_URL}/api/templates/content?path=${encodeURIComponent(filePath)}&custom=${isCustom}`,
				{
					headers: {
						'Authorization': `Bearer ${token}`,
					},
				}
			);
			const content = await response.text();

			// Update editor content
			const language = 'yaml';
			const model = monaco.editor.createModel(content, language);
			editor.setModel(model);
		} catch (error) {
			console.error('Error fetching file content:', error);
		}
	}

	function showToastFunction(message: string, color: string = 'green', icon = CheckCircleSolid) {
		toastMessage = message;
		toastColor = color;
		toastIcon = icon;
		showToast = true;
	}

	async function saveFile() {
		try {
			const token = $pocketbase.authStore.token;
			const content = editor.getValue();
			console.log('Saving content:', content);
			await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/templates/content`, {
				method: 'POST',
				headers: {
					'Authorization': `Bearer ${token}`,
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					path: currentFilePath,
					content: content,
					isCustom: isCustomFile,
				}),
			});
			// Display success toast
			showToastFunction('File saved successfully.', 'green', CheckCircleSolid);

			fileExplorerRef.refreshAllFiles();
		} catch (error) {
			console.error('Error saving file:', error);
			// Display error toast
			showToastFunction('Error saving file.', 'red', ExclamationCircleSolid);
		}
	}

	function createNewTemplate() {
		const baseFileName = 'exploit';
		const extension = '.yaml';
		let counter = 0;
		let fileName = baseFileName + extension;
		let filePath = fileName; // Since 'Custom' is not included in the path

		// Loop to find a unique filename
		while (fileExplorerRef.getFileByPath(filePath, true)) {
			counter++;
			fileName = `${baseFileName}${counter}${extension}`;
			filePath = fileName;
		}

		// Proceed to create the file
		const token = $pocketbase.authStore.token;
		fetch(`${import.meta.env.VITE_API_BASE_URL}/api/templates/content`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
				'Authorization': `Bearer ${token}`,
			},
			body: JSON.stringify({
				path: filePath,
				content: '',
				isCustom: true,
			}),
		})
			.then(() => {
				fileExplorerRef.refreshAllFiles();
				onSelectFile(filePath, true);

				// Display success toast
				showToastFunction(`Template "${fileName}" created successfully!`, 'green', CheckCircleSolid);
			})
			.catch((error) => {
				console.error('Error creating new template:', error);
				// Display error toast
				showToastFunction('Error creating new template.', 'red', ExclamationCircleSolid);
			});
	}

	function onSearchInput(event: Event) {
		const target = event.target as HTMLInputElement;
		searchQuery = target.value;
		fileExplorerRef.setSearchQuery(searchQuery);
	}

	function updateEditorTheme() {
		if (monaco) {
			const theme = darkMode ? 'vs-dark' : 'vs';
			monaco.editor.setTheme(theme);
		}
	}

	function updateDarkMode() {
		darkMode = document.documentElement.classList.contains('dark');
		updateEditorTheme();
	}

	onMount(async () => {
		monaco = await import('monaco-editor');

		editor = monaco.editor.create(editorContainer, {
			value: '',
			language: 'yaml',
			automaticLayout: true,
		});

		updateDarkMode();

		observer = new MutationObserver(() => {
			updateDarkMode();
		});

		observer.observe(document.documentElement, {
			attributes: true,
			attributeFilter: ['class'],
		});
	});

	onDestroy(() => {
		if (observer) {
			observer.disconnect();
		}
	});
</script>

{#if showToast}
	<Toast
		class="fixed top-4 right-4 z-50"
		color={toastColor}
		on:hide={() => showToast = false}
		duration={3000}
	>
		<svelte:fragment slot="icon">
			<svelte:component this={toastIcon} class="w-5 h-5" />
		</svelte:fragment>
		{toastMessage}
	</Toast>
{/if}

<div class="flex items-center p-2 bg-gray-50 dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
	<Button on:click={createNewTemplate}>New Template</Button>
	<div class="flex-grow"></div>
	<input
		type="text"
		placeholder="Search templates..."
		bind:value={searchQuery}
		on:input={onSearchInput}
		class="w-48 p-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 placeholder-gray-500 dark:placeholder-gray-400"
	/>
</div>

<div class="container flex h-screen">
	<FileExplorer bind:this={fileExplorerRef} {onSelectFile} showToast={showToastFunction} />
	<div class="editor-wrapper relative flex-grow h-full">
		<div class="editor h-full" bind:this={editorContainer}></div>
		{#if currentFilePath !== '' && isCustomFile}
			<button
				class="save-button absolute top-2 right-2 z-10 px-2 py-1 bg-blue-600 hover:bg-blue-700 text-white rounded text-sm"
				on:click={saveFile}
			>
				Save
			</button>
		{/if}
	</div>
</div>

<script lang="ts">
    import { onMount } from 'svelte';
    import { writable, get, derived } from 'svelte/store';
    import { pocketbase } from '$lib/stores/pocketbase';
    import TreeItem from './TreeItem.svelte';

    export let onSelectFile;
    export let showToast;

    interface FileItem {
        name: string;
        path: string;
        isDir: boolean;
        isCustom: boolean;
        children?: FileItem[];
        isExpanded?: boolean;
    }

    let allFiles = writable<FileItem[]>([]);
    let searchQuery = writable('');

    // Expose the function to set the search query
    export function setSearchQuery(query: string) {
        searchQuery.set(query);
    }

    // Function to filter files based on search query
    const filteredFiles = derived([allFiles, searchQuery], ([$allFiles, $searchQuery]) => {
        if (!$searchQuery) return $allFiles;

        function filterItems(items: FileItem[]): FileItem[] {
            return items
                .map(item => {
                    if (item.isDir) {
                        const filteredChildren = filterItems(item.children || []);
                        if (filteredChildren.length > 0) {
                            return { ...item, children: filteredChildren };
                        }
                    } else if (item.name.toLowerCase().includes($searchQuery.toLowerCase())) {
                        return item;
                    }
                    return null;
                })
                .filter((item): item is FileItem => item !== null);
        }

        return filterItems($allFiles);
    });

    async function fetchAllFiles() {
        allFiles.set([]); // Clear existing files
        const rootItems: FileItem[] = [];

        try {
            const token = $pocketbase.authStore.token;

            // Fetch Public Templates
            const responsePublic = await fetch(
                `${import.meta.env.VITE_API_BASE_URL}/api/templates/all?custom=false`,
                {
                    headers: {
                        'Authorization': `Bearer ${token}`,
                    },
                }
            );
            const dataPublic = await responsePublic.json();
            const publicDataWithFlags = addIsCustomFlag(dataPublic, false);

            rootItems.push({
                name: 'Public',
                isDir: true,
                isExpanded: true,
                children: publicDataWithFlags,
                isCustom: false,
                path: '', // Root folder 'Public' has no path
            });

            // Fetch Custom Templates
            const responseCustom = await fetch(
                `${import.meta.env.VITE_API_BASE_URL}/api/templates/all?custom=true`,
                {
                    headers: {
                        'Authorization': `Bearer ${token}`,
                    },
                }
            );
            const dataCustom = await responseCustom.json();
            const customDataWithFlags = addIsCustomFlag(dataCustom, true);

            rootItems.push({
                name: 'Custom',
                isDir: true,
                isExpanded: true,
                children: customDataWithFlags,
                isCustom: true,
                path: '', // Root folder 'Custom' has no path
            });

            allFiles.set(rootItems);
        } catch (error) {
            console.error('Error fetching all files:', error);
        }
    }

    function addIsCustomFlag(items: FileItem[], isCustom: boolean, parentPath = ''): FileItem[] {
        if (!items) {
            return []; // Return an empty array if items is null or undefined
        }
        return items.map((item) => {
            const newPath = parentPath ? `${parentPath}/${item.name}` : item.name;
            const newItem: FileItem = {
                ...item,
                isCustom: isCustom,
                path: newPath,
            };
            if (item.children) {
                newItem.children = addIsCustomFlag(item.children, isCustom, newPath);
            }
            return newItem;
        });
    }

    // Expose the refresh function to the parent
    export function refreshAllFiles() {
        fetchAllFiles();
    }

    // Implement and export getFileByPath
    export function getFileByPath(path: string, isCustom: boolean): FileItem | null {
        const files = get(allFiles);

        function recursiveSearch(items: FileItem[]): FileItem | null {
            for (let item of items) {
                if (item.path === path && item.isCustom === isCustom && !item.isDir) {
                    return item;
                }
                if (item.children) {
                    const found = recursiveSearch(item.children);
                    if (found) {
                        return found;
                    }
                }
            }
            return null;
        }

        return recursiveSearch(files);
    }

    onMount(() => {
        fetchAllFiles();
    });
</script>

<div class="file-explorer w-64 overflow-auto border-r border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-2">
    {#each $filteredFiles as item}
        <TreeItem
            {item}
            {onSelectFile}
            showToast={showToast}
            refreshTree={refreshAllFiles}
        />
    {/each}
</div>

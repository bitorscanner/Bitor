<script lang="ts">
  import { pocketbase } from '@lib/stores/pocketbase';
  import { CheckCircleSolid, ExclamationCircleOutline, TrashBinOutline } from 'flowbite-svelte-icons';
  import { Modal, Button } from 'flowbite-svelte';
  
  export let item;
  export let onSelectFile;
  export let showToast; // function passed from parent
  export let refreshTree; // function passed from parent to refresh the file explorer

  let isExpanded = false;
  let isRenaming = false;
  let newName = item.name;

  // State variables for the modal
  let isDeleteModalOpen = false; // Controls modal visibility

  function handleClick(event) {
    event.stopPropagation();
    if (item.isDir) {
      isExpanded = !isExpanded;
    } else {
      onSelectFile(item.path, item.isCustom);
    }
  }

  function startRenaming(event) {
    event.stopPropagation();
    isRenaming = true;
    newName = item.name;
  }

  async function handleRename() {
    if (newName === item.name || !newName.trim()) {
      isRenaming = false;
      newName = item.name; // Reset newName
      return;
    }

    try {
      const token = $pocketbase.authStore.token;
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/templates/rename`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          oldPath: item.path,
          newName: newName,
          isCustom: item.isCustom,
        }),
      });

      if (response.ok) {
        // Update the item's name and path
        item.name = newName;
        const parentPath = item.path.substring(0, item.path.lastIndexOf('/'));
        item.path = parentPath ? `${parentPath}/${newName}` : newName;

        // Call the toast function from the parent
        showToast('File renamed successfully.', 'green', CheckCircleSolid);

        // Optionally refresh the file explorer
        // Depending on your setup, you might need to trigger a refresh
      } else {
        const errorData = await response.json();
        console.error('Error renaming file:', errorData);
        showToast('An error occurred while renaming the file.', 'red', ExclamationCircleSolid);
      }
    } catch (error) {
      console.error('Error renaming file:', error);
      showToast('An error occurred while renaming the file.', 'red', ExclamationCircleSolid);
    } finally {
      isRenaming = false;
    }
  }

  function handleKeyDown(event) {
    if (event.key === 'Enter') {
      handleRename();
    } else if (event.key === 'Escape') {
      isRenaming = false;
      newName = item.name; // Reset to original name
    }
  }

  function deleteItem(event) {
    event.stopPropagation();
    isDeleteModalOpen = true; // Open the modal
  }

  async function confirmDelete() {
    isDeleteModalOpen = false; // Close the modal

    try {
      const token = $pocketbase.authStore.token;
      const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/api/templates/delete`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          path: item.path,
          isCustom: item.isCustom,
        }),
      });

      if (response.ok) {
        showToast('Template deleted successfully.', 'green', CheckCircleSolid);
        refreshTree();
      } else {
        const errorData = await response.json();
        console.error('Error deleting file:', errorData);
        showToast('An error occurred while deleting the template.', 'red', ExclamationCircleSolid);
      }
    } catch (error) {
      console.error('Error deleting file:', error);
      showToast('An error occurred while deleting the template.', 'red', ExclamationCircleSolid);
    }
  }

  function cancelDelete() {
    isDeleteModalOpen = false; // Close the modal without deleting
  }

  function shouldShowDeleteButton(item) {
    // Hide delete button for root folders (e.g., 'Custom' and 'Public')
    // Assuming root folders have an empty path or level 0
    return item.isCustom && item.path !== '' && item.path !== null;
  }
</script>

<div>
  <div
    class="item-container flex items-center cursor-pointer relative hover:bg-gray-100 dark:hover:bg-gray-700 p-1 rounded"
    on:click={handleClick}
    on:dblclick={startRenaming}
  >
    {#if isRenaming}
      <input
        type="text"
        bind:value={newName}
        on:keydown={handleKeyDown}
        on:blur={handleRename}
        class="bg-transparent border border-gray-300 dark:border-gray-600 dark:bg-gray-800 text-gray-900 dark:text-gray-100 p-1 rounded w-full"
        autofocus
      />
    {:else}
      <span class="flex items-center text-gray-800 dark:text-gray-200">
        {item.isDir ? (isExpanded ? '📂' : '📁') : '📄'} {item.name}
      </span>
      {#if shouldShowDeleteButton(item)}
        <button
          on:click|stopPropagation={deleteItem}
          class="delete-button ml-auto text-red-600 hover:text-red-800 dark:text-red-500 dark:hover:text-red-700"
          aria-label="Delete"
        >
          <TrashBinOutline class="icon w-4 h-4" />
        </button>
      {/if}
    {/if}
  </div>
  {#if item.isDir && isExpanded}
    <div class="children pl-4">
      {#each item.children as child}
        <svelte:self
          item={child}
          onSelectFile={onSelectFile}
          showToast={showToast}
          refreshTree={refreshTree}
        />
      {/each}
    </div>
  {/if}
  <!-- Delete Confirmation Modal -->
  <Modal bind:open={isDeleteModalOpen} size="xs" autoclose>
    <div class="text-center">
      <ExclamationCircleOutline class="mx-auto mb-4 text-gray-400 w-12 h-12" />
      <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
        Are you sure you want to delete "<strong>{item.name}</strong>"?
      </h3>
      <div class="flex justify-center gap-4">
        <Button color="red" class="me-2" on:click={confirmDelete}>Yes, I'm sure</Button>
        <Button color="alternative" on:click={cancelDelete}>No, cancel</Button>
      </div>
    </div>
  </Modal>
</div>

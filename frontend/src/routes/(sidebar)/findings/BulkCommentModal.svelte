<script lang="ts">
  import { Button, Modal, Textarea } from 'flowbite-svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  
  export let open = false;
  export let findings: Array<{id: string}> = [];
  
  let comment = '';
  let isSubmitting = false;
  let error = '';
  let success = false;

  async function addCommentToFindings() {
    if (!comment.trim()) {
      error = 'Please enter a comment';
      return;
    }

    isSubmitting = true;
    error = '';
    success = false;

    try {
      // Add the comment to each finding
      await Promise.all(findings.map(finding => 
        $pocketbase.collection('findings').update(finding.id, {
          comments: comment
        })
      ));
      
      success = true;
      setTimeout(() => {
        open = false;
        comment = '';
      }, 1500);
    } catch (err) {
      console.error('Error adding comments:', err);
      error = 'Failed to add comments. Please try again.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<Modal bind:open={open} size="md" autoclose={false}>
  <div class="text-center">
    <h3 class="mb-5 text-lg font-normal text-gray-500 dark:text-gray-400">
      Add Comment to {findings.length} Finding{findings.length !== 1 ? 's' : ''}
    </h3>
    
    <div class="mb-4">
      <Textarea
        bind:value={comment}
        rows={4}
        placeholder="Enter your comment here..."
      />
    </div>

    {#if error}
      <p class="text-red-500 mb-4">{error}</p>
    {/if}

    {#if success}
      <p class="text-green-500 mb-4">Comments added successfully!</p>
    {/if}

    <div class="flex justify-center gap-4">
      <Button color="alternative" on:click={() => open = false}>
        Cancel
      </Button>
      <Button 
        color="primary"
        on:click={addCommentToFindings}
        disabled={isSubmitting}
      >
        {isSubmitting ? 'Adding Comments...' : 'Add Comments'}
      </Button>
    </div>
  </div>
</Modal> 
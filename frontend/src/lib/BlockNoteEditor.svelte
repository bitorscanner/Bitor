<script lang="ts">
  import { onMount, onDestroy, createEventDispatcher } from 'svelte';
  import { BlockNoteEditor } from '@blocknote/core';
  import '@blocknote/core/style.css';
  import type { Block } from '@blocknote/core';
  import type { BlockNoteEditor as BlockNoteEditorType } from '@blocknote/core';

  /// <reference types="node" />

  const dispatch = createEventDispatcher();

  export let content: string = '';
  let editor: BlockNoteEditorType | null = null;
  let editorElement: HTMLElement;
  let menuElement: HTMLDivElement;
  let isMenuVisible = false;
  let menuPosition = { top: 0, left: 0 };
  let isEditorReady = false;
  let isLocalUpdate = false;
  let updateTimeout: NodeJS.Timeout | null = null;

  let outsideClickListener: (event: MouseEvent) => void;

  export let onUpdate: (editor: BlockNoteEditorType) => void = () => {};
  export let initialContent: Block[] = [];

  type MenuItem = {
    title: string;
    onItemClick: () => void;
    group?: string;
    icon?: string;
  };

  function createMenuItems(): MenuItem[] {
    return [
      // Basic blocks
      {
        title: 'Text',
        group: 'Basic blocks',
        icon: '📝',
        onItemClick: () => updateBlockType('paragraph')
      },
      {
        title: 'Heading 1',
        group: 'Basic blocks',
        icon: 'H1',
        onItemClick: () => updateBlockType('heading', { level: 1 })
      },
      {
        title: 'Heading 2',
        group: 'Basic blocks',
        icon: 'H2',
        onItemClick: () => updateBlockType('heading', { level: 2 })
      },
      {
        title: 'Heading 3',
        group: 'Basic blocks',
        icon: 'H3',
        onItemClick: () => updateBlockType('heading', { level: 3 })
      },
      // Lists
      {
        title: 'Bullet List',
        group: 'Lists',
        icon: '•',
        onItemClick: () => updateBlockType('bulletListItem')
      },
      {
        title: 'Numbered List',
        group: 'Lists',
        icon: '1.',
        onItemClick: () => updateBlockType('numberedListItem')
      },
      {
        title: 'Check List',
        group: 'Lists',
        icon: '☐',
        onItemClick: () => updateBlockType('checkListItem')
      },
      // Advanced
      {
        title: 'Code Block',
        group: 'Advanced',
        icon: '💻',
        onItemClick: () => updateBlockType('codeBlock')
      }
    ];
  }

  const menuItems = createMenuItems();

  function updateBlockType(type: string, props: Record<string, any> = {}) {
    if (editor) {
      editor.updateBlock(editor.getTextCursorPosition().block, {
        type: type as any,
        props
      });
      isMenuVisible = false;
    }
  }

  function debouncedDispatch(htmlContent: string) {
    if (updateTimeout) {
      clearTimeout(updateTimeout);
    }
    
    updateTimeout = setTimeout(() => {
      if (!isLocalUpdate) {
        dispatch('change', htmlContent);
      }
    }, 100);
  }

  async function initializeEditor() {
    try {
      editor = BlockNoteEditor.create();
      await editor.mount(editorElement);
      isEditorReady = true;

      if (content) {
        console.log('Setting initial content');
        isLocalUpdate = true;
        try {
          editor._tiptapEditor.commands.setContent(content);
        } catch (error) {
          console.error('Error setting initial content:', error);
        } finally {
          isLocalUpdate = false;
        }
      }

      editor._tiptapEditor.on('update', () => {
        if (!editor) return;
        try {
          const htmlContent = editor._tiptapEditor.getHTML();
          debouncedDispatch(htmlContent);
        } catch (error) {
          console.error('Error dispatching change event:', error);
        }
      });

    } catch (error) {
      console.error('Error initializing editor:', error);
    }
  }

  onMount(async () => {
    menuElement = document.createElement('div');
    menuElement.className = 'bn-suggestion-menu';
    document.body.appendChild(menuElement);

    await initializeEditor();

    editorElement.addEventListener('keydown', (event) => {
      if (!editor) return;

      if (event.key === '/') {
        const selection = editor._tiptapEditor.view.state.selection;
        const pos = editor._tiptapEditor.view.coordsAtPos(selection.from);
        menuPosition = { top: pos.top + 20, left: pos.left };
        isMenuVisible = true;
        event.preventDefault();
      } else if (event.key === 'Escape') {
        isMenuVisible = false;
      }
    });

    // Event listener to hide menu on outside click
    outsideClickListener = (event: MouseEvent) => {
      if (
        menuElement &&
        !menuElement.contains(event.target as Node) &&
        !editorElement.contains(event.target as Node)
      ) {
        isMenuVisible = false;
      }
    };
    document.addEventListener('mousedown', outsideClickListener);
  });

  onDestroy(() => {
    if (updateTimeout) {
      clearTimeout(updateTimeout);
    }
    if (editor) {
      editor.mount(null);
      editor = null;
    }
    if (menuElement) {
      menuElement.remove();
    }
    // Remove the outside click listener
    document.removeEventListener('mousedown', outsideClickListener);
  });

  // Handle external content updates
  $: if (editor && isEditorReady && !isLocalUpdate) {
    const currentEditorContent = editor._tiptapEditor.getHTML();
    if (content !== currentEditorContent) {
      isLocalUpdate = true;
      try {
        editor._tiptapEditor.commands.setContent(content);
      } catch (error) {
        console.error('Error updating content:', error);
      } finally {
        isLocalUpdate = false;
      }
    }
  }

  $: if (isMenuVisible && menuElement) {
    menuElement.style.display = 'block';
    menuElement.style.top = `${menuPosition.top}px`;
    menuElement.style.left = `${menuPosition.left}px`;
    menuElement.innerHTML = '';

    const groups = menuItems.reduce((acc, item) => {
      const group = item.group || 'Other';
      if (!acc[group]) acc[group] = [];
      acc[group].push(item);
      return acc;
    }, {} as Record<string, MenuItem[]>);

    Object.entries(groups).forEach(([groupName, items]) => {
      const groupDiv = document.createElement('div');
      groupDiv.className = 'menu-group';
      
      const groupTitle = document.createElement('div');
      groupTitle.className = 'menu-group-title';
      groupTitle.textContent = groupName;
      groupDiv.appendChild(groupTitle);

      items.forEach(item => {
        const button = document.createElement('button');
        button.className = 'menu-item';
        button.innerHTML = `
          <span class="menu-item-icon">${item.icon || ''}</span>
          <span class="menu-item-title">${item.title}</span>
        `;
        button.onclick = item.onItemClick;
        groupDiv.appendChild(button);
      });

      menuElement.appendChild(groupDiv);
    });
  } else if (menuElement) {
    menuElement.style.display = 'none';
  }

  function handleImageUpload(file: File): Promise<string> {
    return new Promise((resolve) => {
      const reader = new FileReader();
      reader.onload = (e) => {
        if (e.target?.result) {
          resolve(e.target.result as string);
        }
      };
      reader.readAsDataURL(file);
    });
  }
</script>

<div class="editor-container">
  <div bind:this={editorElement} class="editor-instance"></div>
</div>

<style>
  .editor-container {
    position: relative;
    width: 100%;
    height: auto;
    padding: 0;
  }

  .editor-instance {
    border: 1px solid #ccc;
    border-radius: 4px;
    min-height: 200px;
    max-height: 400px;
    overflow-y: auto;
  }

  :global(.bn-suggestion-menu) {
    position: fixed !important;
    z-index: 99999 !important;
    background: white !important;
    border: 1px solid #ccc !important;
    border-radius: 4px !important;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15) !important;
    padding: 4px !important;
    min-width: 200px !important;
  }

  :global(.menu-group) {
    padding: 4px 0;
    border-bottom: 1px solid #eee;
  }

  :global(.menu-group:last-child) {
    border-bottom: none;
  }

  :global(.menu-group-title) {
    padding: 4px 12px;
    font-size: 0.8em;
    color: #666;
    font-weight: 500;
  }

  :global(.menu-item) {
    display: flex;
    align-items: center;
    width: 100%;
    padding: 8px 12px;
    border: none;
    background: none;
    text-align: left;
    cursor: pointer;
    gap: 8px;
  }

  :global(.menu-item:hover) {
    background: #f5f5f5;
  }

  :global(.menu-item-icon) {
    width: 20px;
    text-align: center;
    font-size: 0.9em;
  }

  :global(.menu-item-title) {
    flex: 1;
  }
</style>
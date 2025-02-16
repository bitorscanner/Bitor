<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '$lib/stores/pocketbase';
  import { Accordion, AccordionItem, Button, Checkbox, Label, P, Badge, Input } from 'flowbite-svelte';
  import { SearchOutline } from 'flowbite-svelte-icons';
  import { createEventDispatcher } from 'svelte';

  interface Template {
    id: string;
    name: string;
    description?: string;
    category: string;
    path: string;
  }

  interface CategoryInfo {
    templates: Template[];
    total: number;
  }

  // Add search index interface
  interface SearchIndex {
    name: string;
    description: string;
    category: string;
    template: Template;
  }

  // Props
  export let selectedTemplates: string[] = [];
  export let useAllTemplates: boolean = false;

  // Event dispatcher
  const dispatch = createEventDispatcher<{
    change: { selectedTemplates: string[]; useAllTemplates: boolean };
  }>();

  // Add debounce function
  function debounce<T extends (...args: any[]) => any>(
    fn: T,
    wait: number
  ): (...args: Parameters<T>) => void {
    let timeoutId: ReturnType<typeof setTimeout>;
    return function (this: any, ...args: Parameters<T>) {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => fn.apply(this, args), wait);
    };
  }

  // State
  let officialTemplates: Record<string, CategoryInfo> = {
    'DNS': { templates: [], total: 0 },
    'File': { templates: [], total: 0 },
    'Headless': { templates: [], total: 0 },
    'HTTP': { templates: [], total: 0 },
    'Network': { templates: [], total: 0 },
    'SSL/TLS': { templates: [], total: 0 },
    'Workflows': { templates: [], total: 0 },
    'Uncategorized': { templates: [], total: 0 }
  };
  let customTemplates: CategoryInfo = { templates: [], total: 0 };
  let loading = false;
  let searchQuery = '';
  let expandedCategories = new Set<string>();
  let currentLimit = 10;

  // Fetch templates with search
  async function fetchTemplates(limit = 10, isLoadMore = false) {
    try {
      if (isLoadMore) {
        loading = true;
      }
      
      const searchParam = searchQuery.length >= 2 ? `&search=${searchQuery}` : '';
      const response = await fetch(
        `${import.meta.env.VITE_API_BASE_URL}/api/templates/list?limit=${limit}${searchParam}`,
        {
          headers: {
            Authorization: `Bearer ${$pocketbase.authStore.token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to fetch templates');
      }

      const data = await response.json();

      // Reset templates
      officialTemplates = {
        'DNS': { templates: [], total: 0 },
        'File': { templates: [], total: 0 },
        'Headless': { templates: [], total: 0 },
        'HTTP': { templates: [], total: 0 },
        'Network': { templates: [], total: 0 },
        'SSL/TLS': { templates: [], total: 0 },
        'Workflows': { templates: [], total: 0 },
        'Uncategorized': { templates: [], total: 0 }
      };

      // Process official templates
      if (data.official) {
        Object.entries(data.official).forEach(([category, info]: [string, CategoryInfo]) => {
          if (!officialTemplates[category]) {
            officialTemplates[category] = { templates: [], total: 0 };
          }
          officialTemplates[category] = info;
        });
      }

      // Process custom templates
      if (data.custom) {
        customTemplates = data.custom;
      }

      // Auto-expand categories with search results
      if (searchQuery.length >= 2) {
        expandedCategories = new Set(Object.keys(data.official));
      }

      // Ensure reactivity
      officialTemplates = { ...officialTemplates };
      customTemplates = { ...customTemplates };
      
    } catch (error) {
      console.error('Error fetching templates:', error);
    } finally {
      loading = false;
    }
  }

  // Debounced search with longer delay
  const debouncedSearch = debounce(() => {
    if (searchQuery.length >= 2) {
      fetchTemplates(currentLimit);
    }
  }, 1000); // Increased to 1 second

  // Watch for search changes
  $: {
    if (!searchQuery) {
      fetchTemplates(currentLimit);
    } else if (searchQuery.length >= 2) {
      debouncedSearch();
    }
  }

  onMount(() => {
    fetchTemplates(currentLimit);
  });

  // Handle template selection
  function handleTemplateSelect(templatePath: string, event: Event) {
    const checked = (event.target as HTMLInputElement)?.checked ?? false;
    if (checked) {
      selectedTemplates = [...selectedTemplates, templatePath];
    } else {
      selectedTemplates = selectedTemplates.filter(path => path !== templatePath);
    }
    dispatch('change', { selectedTemplates, useAllTemplates });
  }

  // Handle "Use All Templates" toggle
  function handleUseAllTemplates(event: Event) {
    const checked = (event.target as HTMLInputElement)?.checked ?? false;
    useAllTemplates = checked;
    if (checked) {
      // When "Use All" is selected, clear individual selections
      selectedTemplates = [];
    }
    dispatch('change', { selectedTemplates, useAllTemplates });
  }

  // Add new function to handle category selection
  function handleCategorySelect(templates: Template[], event: Event) {
    const checked = (event.target as HTMLInputElement)?.checked ?? false;
    if (checked) {
      // Add all templates from the category that aren't already selected
      const newPaths = templates.map(t => t.path).filter(path => !selectedTemplates.includes(path));
      selectedTemplates = [...selectedTemplates, ...newPaths];
    } else {
      // Remove all templates from this category
      const categoryPaths = templates.map(t => t.path);
      selectedTemplates = selectedTemplates.filter(path => !categoryPaths.includes(path));
    }
    dispatch('change', { selectedTemplates, useAllTemplates });
  }

  // Add computed property for category selection state
  function isCategorySelected(templates: Template[]): boolean {
    return templates.every(template => selectedTemplates.includes(template.path));
  }

  function isCategoryPartiallySelected(templates: Template[]): boolean {
    const selected = templates.some(template => selectedTemplates.includes(template.path));
    return selected && !isCategorySelected(templates);
  }

  // Compute selected template count
  $: selectedCount = useAllTemplates ? 'All' : selectedTemplates.length;

  // Handle category expansion
  function handleCategoryExpand(category: string) {
    if (expandedCategories.has(category)) {
      expandedCategories.delete(category);
    } else {
      expandedCategories.add(category);
    }
    expandedCategories = expandedCategories; // Trigger reactivity
  }

  function loadMore(category: string) {
    currentLimit += 10;
    fetchTemplates(currentLimit, true);
  }

  let searchTimeout: NodeJS.Timeout;
  function handleSearch() {
    if (searchTimeout) {
      clearTimeout(searchTimeout);
    }
    
    searchTimeout = setTimeout(() => {
      if (searchQuery.length >= 2 || searchQuery.length === 0) {
        currentLimit = 10;
        fetchTemplates(currentLimit);
        
        // Auto-expand categories with search results
        if (searchQuery.length >= 2) {
          expandedCategories = new Set(Object.keys(officialTemplates));
        }
      }
    }, 300);
  }
</script>

<div class="space-y-4">
  <!-- Use All Templates Option -->
  <div class="flex items-center space-x-2 p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
    <Checkbox
      id="use-all-templates"
      bind:checked={useAllTemplates}
      on:change={handleUseAllTemplates}
    />
    <Label for="use-all-templates" class="flex items-center space-x-2">
      <span>Use All Templates</span>
      <Badge color={useAllTemplates ? "green" : "dark"}>Recommended</Badge>
    </Label>
  </div>

  {#if !useAllTemplates}
    <!-- Search Bar -->
    <div class="relative">
      <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        {#if loading}
          <div class="animate-spin h-5 w-5 border-2 border-blue-500 rounded-full border-t-transparent"></div>
        {:else}
          <SearchOutline class="w-5 h-5 text-gray-500 dark:text-gray-400" />
        {/if}
      </div>
      <Input
        type="search"
        placeholder="Type at least 2 characters to search..."
        bind:value={searchQuery}
        class="pl-10 w-full"
        on:input={handleSearch}
      />
      {#if searchQuery && searchQuery.length < 2}
        <div class="text-sm text-gray-500 mt-1">Keep typing to search...</div>
      {/if}
    </div>

    <div class="space-y-4">
      <!-- Official Templates -->
      <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold">Official Templates</h3>
          {#if searchQuery}
            <Badge color="blue">
              {Object.values(officialTemplates).reduce((acc, info) => acc + info.total, 0)} results
            </Badge>
          {/if}
        </div>
        {#if loading}
          <P>Loading templates...</P>
        {:else}
          <Accordion>
            {#each Object.entries(officialTemplates) as [category, categoryInfo]}
              {#if categoryInfo.templates.length > 0}
                <AccordionItem
                  open={searchQuery ? true : expandedCategories.has(category)}
                  on:click={() => !searchQuery && handleCategoryExpand(category)}
                >
                  <span slot="header" class="flex items-center justify-between">
                    <div class="flex items-center space-x-3">
                      <Checkbox
                        id={`category-${category}`}
                        checked={isCategorySelected(categoryInfo.templates)}
                        indeterminate={isCategoryPartiallySelected(categoryInfo.templates)}
                        on:change={(e) => handleCategorySelect(categoryInfo.templates, e)}
                      />
                      <span>{category}</span>
                    </div>
                    <Badge color="dark">{categoryInfo.total}</Badge>
                  </span>
                  <div class="space-y-3 p-2">
                    {#each categoryInfo.templates as template}
                      <div class="flex items-start space-x-3 p-2 hover:bg-gray-50 dark:hover:bg-gray-700 rounded-lg">
                        <Checkbox
                          id={template.id}
                          checked={selectedTemplates.includes(template.path)}
                          on:change={(e) => handleTemplateSelect(template.path, e)}
                        />
                        <Label for={template.id} class="flex-1 cursor-pointer">
                          <span class="font-medium">{template.name}</span>
                          {#if template.description}
                            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{template.description}</p>
                          {/if}
                        </Label>
                      </div>
                    {/each}
                    {#if categoryInfo.templates.length < categoryInfo.total}
                      <Button size="sm" color="light" class="mt-2" on:click={() => loadMore(category)}>
                        Load More ({categoryInfo.templates.length} of {categoryInfo.total})
                      </Button>
                    {/if}
                  </div>
                </AccordionItem>
              {/if}
            {/each}
          </Accordion>
        {/if}
      </div>

      <!-- Custom Templates -->
      {#if customTemplates.total > 0}
        <div class="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          <div class="flex items-center justify-between mb-4">
            <h3 class="text-lg font-semibold">Custom Templates</h3>
            <div class="flex items-center space-x-3">
              <Checkbox
                id="category-custom"
                checked={customTemplates.templates.every(t => selectedTemplates.includes(t.path))}
                indeterminate={customTemplates.templates.some(t => selectedTemplates.includes(t.path)) && 
                             !customTemplates.templates.every(t => selectedTemplates.includes(t.path))}
                on:change={(e) => handleCategorySelect(customTemplates.templates, e)}
              />
              <Badge color="dark">{customTemplates.total}</Badge>
            </div>
          </div>
          <div class="space-y-3">
            {#each customTemplates.templates as template}
              <div class="flex items-start space-x-3 p-2 hover:bg-gray-50 dark:hover:bg-gray-700 rounded-lg">
                <Checkbox
                  id={template.id}
                  checked={selectedTemplates.includes(template.path)}
                  on:change={(e) => handleTemplateSelect(template.path, e)}
                />
                <Label for={template.id} class="flex-1 cursor-pointer">
                  <span class="font-medium">{template.name}</span>
                  {#if template.description}
                    <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">{template.description}</p>
                  {/if}
                </Label>
              </div>
            {/each}
            {#if customTemplates.templates.length < customTemplates.total}
              <Button size="sm" color="light" class="mt-2" on:click={() => loadMore('custom')}>
                Load More ({customTemplates.templates.length} of {customTemplates.total})
              </Button>
            {/if}
          </div>
        </div>
      {/if}
    </div>
  {/if}

  <!-- Selection Summary -->
  <div class="p-4 bg-gray-50 dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
    <P class="flex items-center space-x-2">
      <span>Selected templates:</span>
      <Badge color="blue">{selectedCount}</Badge>
    </P>
  </div>
</div>

<style>
  :global(.accordion-content) {
    max-height: none !important;
  }
</style> 
<!-- File: frontend/src/routes/(sidebar)/findings/AdvancedSearch.svelte -->
<script lang="ts">
  import { Button, Input, Select, MultiSelect, Badge, Dropdown, DropdownItem } from 'flowbite-svelte';
  import { createEventDispatcher } from 'svelte';
  import { CloseOutline, FilterSolid, SearchOutline } from 'flowbite-svelte-icons';
  import { 
    fieldOptions, 
    operatorOptions, 
    statusOptions as defaultStatusOptions,
    getValueInput,
    getValueOptions,
    getValuePlaceholder,
    getFieldLabel,
    getOperatorLabel,
    searchFields
  } from './searchHelpers';

  export let severityOptions: { value: string; name: string }[] = [];
  export let clientOptions: { value: string; name: string }[] = [];
  export let statusOptions: { value: string; name: string }[] = defaultStatusOptions;
  export let isAdmin = false;
  export let showMyFindingsOnly = !isAdmin;

  const dispatch = createEventDispatcher<{
    change: { 
      filters: Filter[];
      severityFilter: string[];
      clientFilter: string[];
      statusFilter: string[];
      showMyFindingsOnly: boolean;
      searchTerm: string;
      searchField: string;
    };
  }>();

  interface Filter {
    field: string;
    operator: string;
    value: string | string[];
    id: string;
  }

  let filters: Filter[] = [];
  let currentFilter: Filter = createEmptyFilter();
  let multiSelectValue: string[] = [];
  let searchTerm = '';
  let searchField = '';
  let dropdownOpen = false;
  let fieldSearchTerm = '';
  let showFieldSearch = false;

  // Initialize filter arrays
  let severityFilter: string[] = [];
  let clientFilter: string[] = [];
  let statusFilter: string[] = [];

  $: filteredFieldGroups = searchFields(fieldSearchTerm);
  
  // Update fieldSearchTerm when currentFilter.field changes
  $: if (currentFilter.field && !showFieldSearch) {
    fieldSearchTerm = fieldOptions.find(f => f.value === currentFilter.field)?.name || '';
  }

  const searchFieldOptions = [
    { value: '', name: 'All Fields' },
    { value: 'name', name: 'Name' },
    { value: 'host', name: 'Host' },
    { value: 'ip', name: 'IP Address' },
    { value: 'template_id', name: 'Template ID' },
    { value: 'scan_name', name: 'Scan Name' },
  ];

  function createEmptyFilter(): Filter {
    return {
      field: '',
      operator: 'equals',
      value: '',
      id: crypto.randomUUID()
    };
  }

  function addFilter() {
    if (currentFilter.field && (typeof currentFilter.value === 'string' ? currentFilter.value : currentFilter.value.length > 0)) {
      // Update the appropriate filter array based on the field
      if (currentFilter.field === 'severity' && Array.isArray(currentFilter.value)) {
        severityFilter = currentFilter.value;
      } else if (currentFilter.field === 'client' && Array.isArray(currentFilter.value)) {
        clientFilter = currentFilter.value;
      } else if (currentFilter.field === 'status' && Array.isArray(currentFilter.value)) {
        statusFilter = currentFilter.value;
      } else if (currentFilter.field === 'user') {
        showMyFindingsOnly = currentFilter.value === 'mine';
      } else {
        filters = [...filters, { ...currentFilter }];
      }
      
      currentFilter = createEmptyFilter();
      multiSelectValue = [];
      dispatchChanges();
    }
  }

  function removeFilter(id: string) {
    filters = filters.filter(f => f.id !== id);
    dispatchChanges();
  }

  function handleMultiSelectChange() {
    currentFilter.value = multiSelectValue;
  }

  function dispatchChanges() {
    dispatch('change', {
      filters,
      severityFilter,
      clientFilter,
      statusFilter,
      showMyFindingsOnly,
      searchTerm,
      searchField
    });
  }

  function handleSearch() {
    dispatchChanges();
  }

  // Watch for changes in search
  $: {
    searchTerm, searchField;
    dispatchChanges();
  }
</script>

<div class="space-y-4">
  <!-- Main Search Bar with Filter Button -->
  <form on:submit|preventDefault={handleSearch} class="w-full">
    <div class="flex gap-2 items-center">
      <div class="flex-1 relative">
        <Input
          id="searchTerm"
          bind:value={searchTerm}
          placeholder="Search findings..."
          size="lg"
          class="pr-32"
        />
        <div class="absolute right-2 top-1/2 -translate-y-1/2 flex items-center gap-2">
          {#if filters.length > 0 || severityFilter.length > 0 || clientFilter.length > 0 || statusFilter.length > 0}
            <span class="text-sm font-medium bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-300 px-2 py-0.5 rounded-full">
              {filters.length + (severityFilter.length > 0 ? 1 : 0) + (clientFilter.length > 0 ? 1 : 0) + (statusFilter.length > 0 ? 1 : 0)}
            </span>
          {/if}
          <Button
            size="sm"
            color={filters.length > 0 || severityFilter.length > 0 || clientFilter.length > 0 || statusFilter.length > 0 ? "blue" : "alternative"}
            class="!p-2"
            type="button"
            on:click={() => dropdownOpen = !dropdownOpen}
          >
            <FilterSolid class="w-4 h-4" />
          </Button>
        </div>
      </div>
      <div class="w-32">
        <Select
          id="userFilter"
          bind:value={showMyFindingsOnly}
          items={[
            { value: true, name: 'My Findings' },
            { value: false, name: 'All Findings' },
          ]}
          on:change={() => dispatchChanges()}
        />
      </div>
      <div class="w-48">
        <Select
          id="searchField"
          bind:value={searchField}
          items={searchFieldOptions}
          placeholder="All Fields"
        />
      </div>
      <Button type="submit" size="lg" class="px-8">
        <SearchOutline class="w-4 h-4 mr-2" />
        Search
      </Button>
    </div>
  </form>

  <!-- Filter Dropdown -->
  {#if dropdownOpen}
    <div class="relative z-50">
      <div class="absolute top-0 left-0 w-full mt-2 p-4 bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700">
        <!-- Filter Builder -->
        <div class="space-y-4">
          <div class="flex gap-2 items-start">
            <div class="flex-1 space-y-2">
              <div class="relative">
                <Input
                  type="text"
                  bind:value={fieldSearchTerm}
                  placeholder="Search fields..."
                  on:focus={() => showFieldSearch = true}
                />
                {#if showFieldSearch}
                  <div 
                    class="absolute top-full left-0 w-full mt-1 max-h-96 overflow-y-auto bg-white dark:bg-gray-800 rounded-lg shadow-lg border dark:border-gray-700 z-50"
                  >
                    {#each filteredFieldGroups as group}
                      {#if group.fields.length > 0}
                        <div class="p-2">
                          <div class="text-sm font-semibold text-gray-500 dark:text-gray-400 px-2 py-1">
                            {group.name}
                          </div>
                          {#each group.fields as field}
                            <button
                              class="w-full text-left px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 {currentFilter.field === field.value ? 'bg-gray-100 dark:bg-gray-700' : ''}"
                              on:click={() => {
                                currentFilter = {
                                  ...createEmptyFilter(),
                                  field: field.value,
                                  operator: 'equals'
                                };
                                multiSelectValue = [];
                                showFieldSearch = false;
                              }}
                            >
                              {field.name}
                            </button>
                          {/each}
                        </div>
                      {/if}
                    {/each}
                  </div>
                {/if}
              </div>
              <div class="flex gap-2">
                <div class="w-1/3">
                  <Select
                    items={operatorOptions}
                    bind:value={currentFilter.operator}
                    placeholder="Operator"
                  />
                </div>
                <div class="flex-1">
                  {#if getValueInput(currentFilter.field) === 'MultiSelect'}
                    <MultiSelect
                      items={getValueOptions(currentFilter.field, { severityOptions, clientOptions, statusOptions })}
                      bind:value={multiSelectValue}
                      on:change={handleMultiSelectChange}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                    />
                  {:else if getValueInput(currentFilter.field) === 'Select'}
                    <Select
                      bind:value={currentFilter.value}
                      items={getValueOptions(currentFilter.field, { severityOptions, clientOptions, statusOptions })}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                    />
                  {:else if getValueInput(currentFilter.field) === 'date'}
                    <Input
                      type="date"
                      bind:value={currentFilter.value}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                    />
                  {:else if getValueInput(currentFilter.field) === 'number'}
                    <Input
                      type="number"
                      bind:value={currentFilter.value}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                    />
                  {:else if getValueInput(currentFilter.field) === 'textarea'}
                    <Input
                      type="text"
                      bind:value={currentFilter.value}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                      class="h-24"
                    />
                  {:else}
                    <Input
                      type="text"
                      bind:value={currentFilter.value}
                      placeholder={getValuePlaceholder(currentFilter.field)}
                    />
                  {/if}
                </div>
              </div>
            </div>
            <Button class="mt-0" on:click={addFilter}>Add Filter</Button>
          </div>

          <!-- Active Filters -->
          {#if filters.length > 0 || severityFilter.length > 0 || clientFilter.length > 0 || statusFilter.length > 0}
            <div class="flex flex-wrap gap-2 mt-4">
              {#if severityFilter.length > 0}
                <Badge color="dark" class="flex items-center gap-2">
                  <span>Severity in [{severityFilter.map(v => severityOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
                  <button
                    class="ml-1 text-sm"
                    on:click={() => { severityFilter = []; dispatchChanges(); }}
                  >
                    <CloseOutline class="w-3 h-3" />
                  </button>
                </Badge>
              {/if}
              {#if clientFilter.length > 0}
                <Badge color="dark" class="flex items-center gap-2">
                  <span>Client in [{clientFilter.map(v => clientOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
                  <button
                    class="ml-1 text-sm"
                    on:click={() => { clientFilter = []; dispatchChanges(); }}
                  >
                    <CloseOutline class="w-3 h-3" />
                  </button>
                </Badge>
              {/if}
              {#if statusFilter.length > 0}
                <Badge color="dark" class="flex items-center gap-2">
                  <span>Status in [{statusFilter.map(v => statusOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
                  <button
                    class="ml-1 text-sm"
                    on:click={() => { statusFilter = []; dispatchChanges(); }}
                  >
                    <CloseOutline class="w-3 h-3" />
                  </button>
                </Badge>
              {/if}
              {#each filters as filter (filter.id)}
                <Badge color="dark" class="flex items-center gap-2">
                  <span>{getFieldLabel(filter.field)} {getOperatorLabel(filter.operator)} {filter.value}</span>
                  <button
                    class="ml-1 text-sm"
                    on:click={() => removeFilter(filter.id)}
                  >
                    <CloseOutline class="w-3 h-3" />
                  </button>
                </Badge>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
  {/if}

  <!-- Active Filters Display (when dropdown is closed) -->
  {#if !dropdownOpen && (filters.length > 0 || severityFilter.length > 0 || clientFilter.length > 0 || statusFilter.length > 0)}
    <div class="flex flex-wrap gap-2">
      {#if severityFilter.length > 0}
        <Badge color="dark" class="flex items-center gap-2">
          <span>Severity in [{severityFilter.map(v => severityOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
          <button
            class="ml-1 text-sm"
            on:click={() => { severityFilter = []; dispatchChanges(); }}
          >
            <CloseOutline class="w-3 h-3" />
          </button>
        </Badge>
      {/if}
      {#if clientFilter.length > 0}
        <Badge color="dark" class="flex items-center gap-2">
          <span>Client in [{clientFilter.map(v => clientOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
          <button
            class="ml-1 text-sm"
            on:click={() => { clientFilter = []; dispatchChanges(); }}
          >
            <CloseOutline class="w-3 h-3" />
          </button>
        </Badge>
      {/if}
      {#if statusFilter.length > 0}
        <Badge color="dark" class="flex items-center gap-2">
          <span>Status in [{statusFilter.map(v => statusOptions.find(o => o.value === v)?.name || v).join(', ')}]</span>
          <button
            class="ml-1 text-sm"
            on:click={() => { statusFilter = []; dispatchChanges(); }}
          >
            <CloseOutline class="w-3 h-3" />
          </button>
        </Badge>
      {/if}
      {#each filters as filter (filter.id)}
        <Badge color="dark" class="flex items-center gap-2">
          <span>{getFieldLabel(filter.field)} {getOperatorLabel(filter.operator)} {filter.value}</span>
          <button
            class="ml-1 text-sm"
            on:click={() => removeFilter(filter.id)}
          >
            <CloseOutline class="w-3 h-3" />
          </button>
        </Badge>
      {/each}
    </div>
  {/if}
</div> 
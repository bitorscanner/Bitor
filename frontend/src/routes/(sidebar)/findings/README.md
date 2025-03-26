# My Findings Filter Integration

This document explains how to integrate the "My Findings Only" filter into the findings page.

## Components

1. `MyFindingsFilter.svelte` - A reusable component that renders a checkbox for filtering findings by the current user.
2. `myFindings.ts` - Utility functions to help with API request filtering and cache key generation.

## Backend Changes

The backend API now supports filtering findings by the `created_by` field. When the "My Findings Only" checkbox is checked, the API will filter findings to only show those created by the current user.

## Integration Steps

1. Import the components and utilities in `+page.svelte`:

```js
import MyFindingsFilter from './MyFindingsFilter.svelte';
import { addUserFilter, updateCacheKey } from './myFindings';
```

2. Add a state variable for the filter:

```js
// User filter state
let showMyFindingsOnly = false;
```

3. Add the filter component to your form:

```svelte
<form on:submit|preventDefault={applyFilters} class="flex flex-wrap gap-4 mb-4">
  <!-- Other filters -->

  <!-- My Findings Filter -->
  <MyFindingsFilter 
    bind:checked={showMyFindingsOnly} 
    on:change={({ detail }) => {
      showMyFindingsOnly = detail.checked;
      applyFilters();
    }}
  />

  <!-- Other form elements -->
</form>
```

4. Update your API request function to include the user filter:

```js
// In your fetchGroupedFindings function:
const params = new URLSearchParams();
// Add other parameters...

// Add user filter
addUserFilter(params, showMyFindingsOnly, $pocketbase.authStore.model?.id ?? '');

// Make the API request
const response = await fetch(
  `${import.meta.env.VITE_API_BASE_URL}/api/findings/grouped?${params.toString()}`,
  {
    headers: {
      Authorization: `Bearer ${token}`,
      'Content-Type': 'application/json',
    }
  }
);
```

5. Update your cache key generation to include the user filter:

```js
// In your fetchGroupedFindings function:
const baseKey = `${currentPage}-${sortField}-${sortDirection}-${JSON.stringify(severityFilter)}-${JSON.stringify(clientFilter)}-${searchTerm}-${searchField}-${JSON.stringify(statusFilter)}`;
const cacheKey = updateCacheKey(baseKey, showMyFindingsOnly);
```

## Notes

- The `created_by` field must be populated in the findings records for this filter to work correctly.
- The backend API has been updated to support this filtering. If you encounter issues, ensure the backend is properly handling the `created_by` parameter. 
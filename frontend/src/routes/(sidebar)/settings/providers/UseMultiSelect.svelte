<script lang="ts">
    // Cast the imported MultiSelect to any so we can pass the appendToBody prop
    import { MultiSelect as OriginalMultiSelect } from 'flowbite-svelte';
    const MultiSelect: any = OriginalMultiSelect;
    import type { UseType } from './types';
  
    export let value: UseType[] = [];
    export let useDescriptions: Record<UseType, string>;
    export let onChange: (uses: UseType[]) => void;
  
    // Convert to items for MultiSelect
    $: items = Object.entries(useDescriptions).map(([val, name]) => ({
      value: val,
      name,
      label: name
    }));

    let selected = value;

    // Watch for changes to selected
    $: {
        if (selected && selected.length > 0 && JSON.stringify(selected) !== JSON.stringify(value)) {
            onChange(selected as UseType[]);
        } else if (!selected || selected.length === 0) {
            // If trying to clear selection, keep at least one value
            selected = value;
        }
    }
</script>
  
<div>
    <MultiSelect
        {items}
        bind:value={selected}
        placeholder="Select provider uses..."
        class="w-full"
        appendToBody
    />
</div>
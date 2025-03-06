<script lang="ts">
  import { onMount } from 'svelte';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { Card, Heading, Accordion, AccordionItem, Skeleton } from 'flowbite-svelte';

  interface Finding {
    id: string;
    info: { name: string };
    host: string;
    ip: string;
    severity: string;
    timestamp: string;
  }

  interface GroupedFindings {
    severity_order: number;
    severity: string;
    findings: Finding[];
  }

  let groupedFindings: GroupedFindings[] = [];
  let isLoading = false;

  // Fetch findings from the last 30 days
  async function fetchGroupedFindings() {
    try {
      isLoading = true;
      const token = $pocketbase.authStore.token;

      const response = await fetch(
        `${import.meta.env.VITE_API_BASE_URL}/api/findings/recent`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Error: ${response.statusText}`);
      }

      const data = await response.json();

      // Filter out 'info' severity groups (case-insensitive)
      const filteredData = data.filter(
        (group) => group.severity.toLowerCase() !== 'info'
      );

      groupedFindings = filteredData.map((group) => ({
        severity_order: group.severity_order,
        severity: group.severity,
        findings: group.findings,
      }));
    } catch (error) {
      console.error('Error fetching findings:', error);
    } finally {
      isLoading = false;
    }
  }

  function getSeverityColor(severity: string): string {
    switch (severity.toLowerCase()) {
      case 'critical':
        return 'bg-red-600 text-white';
      case 'high':
        return 'bg-orange-500 text-white';
      case 'medium':
        return 'bg-yellow-500 text-white';
      case 'low':
        return 'bg-green-500 text-white';
      case 'info':
        return 'bg-blue-500 text-white';
      default:
        return 'bg-gray-500 text-white';
    }
  }

  onMount(() => {
    fetchGroupedFindings();
  });
</script>

<Card size="xl" class="shadow-sm max-w-none">
  <Heading class="mb-4 text-xl font-semibold">
    Recent Findings (Last 30 Days)
  </Heading>

  {#if isLoading}
    <div class="space-y-4">
      {#each Array(5) as _, index}
        <Skeleton key={index} class="h-16 w-full" />
      {/each}
    </div>
  {:else if groupedFindings.length > 0}
    {#each groupedFindings as group}
      <Accordion flush={true}>
        <AccordionItem>
          <!-- Accordion Header -->
          <div slot="header" class="flex items-center space-x-2">
            <span
              class={`px-2 py-1 rounded ${getSeverityColor(group.severity)}`}
            >
              {group.severity}
            </span>
            <span class="text-sm text-gray-600">
              {group.findings.length} Findings
            </span>
          </div>

          <!-- Accordion Content -->
          <ul class="p-4 space-y-2">
            {#each group.findings as finding}
              <li class="border-b pb-2">
                <div class="font-medium">{finding.info.name}</div>
                <div class="text-sm text-gray-500">
                  {finding.host} ({finding.ip}) -
                  {new Date(finding.timestamp).toLocaleString()}
                </div>
              </li>
            {/each}
          </ul>
        </AccordionItem>
      </Accordion>
    {/each}
  {:else}
    <p>No findings in the last 30 days.</p>
  {/if}
</Card>

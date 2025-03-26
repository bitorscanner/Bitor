<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { pocketbase } from '@lib/stores/pocketbase';
  import { Card, Heading, Checkbox } from 'flowbite-svelte';
  import { Bar } from 'svelte-chartjs';
  import type { ChartConfiguration } from 'chart.js';
  import {
    Chart,
    Title,
    Tooltip,
    Legend,
    BarElement,
    CategoryScale,
    LinearScale,
  } from 'chart.js';

  Chart.register(
    Title,
    Tooltip,
    Legend,
    BarElement,
    CategoryScale,
    LinearScale
  );

  interface VulnerabilityItem {
    clientId: string;
    clientName: string;
    clientFavicon: string | null;
    critical: number;
    high: number;
    medium: number;
    low: number;
    unknown: number;
    total: number;
  }

  let vulnerabilitiesByClient: VulnerabilityItem[] = [];
  let chartData: ChartConfiguration;
  let error = '';
  let isLoading = true;
  let showMyDataOnly = !$pocketbase.authStore.isAdmin;
  let currentUserId = $pocketbase.authStore.model?.id ?? '';

  async function fetchVulnerabilitiesByClient() {
    try {
      const token = get(pocketbase).authStore.token;
      const params = new URLSearchParams();
      
      // Always apply user filter for non-admin users
      if (!$pocketbase.authStore.isAdmin) {
        params.append('user_id', currentUserId);
      }

      const response = await fetch(
        `${import.meta.env.VITE_API_BASE_URL}/api/findings/by-client?${params.toString()}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to fetch data');
      }

      const data = await response.json();

      // If no data is returned, set vulnerabilitiesByClient to empty array
      if (!data || data.length === 0) {
        vulnerabilitiesByClient = [];
        error = '';
        return;
      }

      // Fetch full client data for each client
      const clientsData = await Promise.all(
        data.map(async (item) => {
          try {
            const client = await get(pocketbase).collection('clients').getOne(item.clientId);
            return {
              ...item,
              clientFavicon: client.favicon ? get(pocketbase).getFileUrl(client, client.favicon) : null
            };
          } catch (error) {
            console.error(`Error fetching client ${item.clientId}:`, error);
            return item;
          }
        })
      );

      vulnerabilitiesByClient = clientsData.map((item) => ({
        clientId: item.clientId,
        clientName: item.clientName,
        clientFavicon: item.clientFavicon,
        critical: item.critical || 0,
        high: item.high || 0,
        medium: item.medium || 0,
        low: item.low || 0,
        unknown: item.unknown || 0,
        total: item.total || 0,
      }));

      error = '';
      prepareChartData();
    } catch (err) {
      console.error('Error fetching vulnerabilities by client:', err);
      error = showMyDataOnly ? 'No vulnerabilities found for your account.' : 'No vulnerabilities found.';
    } finally {
      isLoading = false;
    }
  }

  function prepareChartData() {
    // Create empty labels since we're showing favicons above
    const labels = vulnerabilitiesByClient.map(() => '');

    const datasets = [
      {
        label: 'Critical',
        data: vulnerabilitiesByClient.map((item) => item.critical),
        backgroundColor: '#8B5CF6', // Purple
      },
      {
        label: 'High',
        data: vulnerabilitiesByClient.map((item) => item.high),
        backgroundColor: '#EF4444', // Red
      },
      {
        label: 'Medium',
        data: vulnerabilitiesByClient.map((item) => item.medium),
        backgroundColor: '#F59E0B', // Yellow
      },
      {
        label: 'Low',
        data: vulnerabilitiesByClient.map((item) => item.low),
        backgroundColor: '#10B981', // Green
      },
      {
        label: 'Unknown',
        data: vulnerabilitiesByClient.map((item) => item.unknown),
        backgroundColor: '#6B7280', // Gray
      },
    ];

    chartData = {
      type: 'bar',
      data: {
        labels,
        datasets,
      },
      options: {
        responsive: true,
        maintainAspectRatio: true,
        layout: {
          padding: {
            bottom: 0
          }
        },
        plugins: {
          tooltip: {
            mode: 'index',
            intersect: false,
            callbacks: {
              title: function(context) {
                const index = context[0].dataIndex;
                return vulnerabilitiesByClient[index].clientName;
              }
            }
          },
          legend: {
            position: 'top',
          }
        },
        scales: {
          x: {
            stacked: true,
            grid: {
              display: false
            },
            ticks: {
              padding: 0
            }
          },
          y: {
            stacked: true,
            beginAtZero: true,
          },
        },
      },
    };
  }

  onMount(() => {
    fetchVulnerabilitiesByClient();
  });

  $: {
    // Refetch data when filter changes
    showMyDataOnly;
    fetchVulnerabilitiesByClient();
  }
</script>

<Card size="xl" class="w-full max-w-none">
  <div class="mb-4">
    <div class="flex justify-between items-center">
      <div>
        <Heading tag="h3" class="text-2xl">Vulnerabilities by Client</Heading>
        <p class="text-base font-light text-gray-500">
          Clients sorted by the number of open vulnerabilities
        </p>
      </div>
      {#if !$pocketbase.authStore.isAdmin}
        <div class="flex items-center">
          <Checkbox 
            bind:checked={showMyDataOnly}
            class="mr-2"
          >
            <span class="ml-2 text-sm font-medium text-gray-900 dark:text-gray-300">
              Show My Data Only
            </span>
          </Checkbox>
        </div>
      {/if}
    </div>
  </div>

  {#if isLoading}
    <p class="text-center text-gray-500">Loading data...</p>
  {:else if error}
    <p class="text-center text-gray-500">{error}</p>
  {:else if vulnerabilitiesByClient.length > 0}
    <div class="my-4">
      <div class="h-[350px] mb-0">
        <Bar data={chartData.data} options={chartData.options} />
      </div>
      <!-- Display client favicons below the chart -->
      <div class="flex justify-between px-12 -mt-4">
        {#each vulnerabilitiesByClient as client}
          <div class="flex flex-col items-center">
            {#if client.clientFavicon}
              <img src={client.clientFavicon} alt="{client.clientName}" class="h-5 w-5" />
            {:else}
              <span class="text-xs">{client.clientName}</span>
            {/if}
          </div>
        {/each}
      </div>
    </div>
  {:else}
    <p class="text-center text-gray-500">No vulnerabilities found {showMyDataOnly ? 'for your account' : ''}.</p>
  {/if}
</Card>

<style>
  /* Hide x-axis labels since we're showing favicons above */
  :global(.chartjs-axis-bottom .chartjs-tick-label) {
    display: none;
  }
</style>
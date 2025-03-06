<script lang="ts">
    import { onMount } from 'svelte';
    import { pocketbase } from '@lib/stores/pocketbase';
    import { Card, Chart, Heading } from 'flowbite-svelte';
    import type { ApexOptions } from 'apexcharts';
    
    // Import ApexCharts only in browser
    let ApexCharts: any;
    onMount(async () => {
      const module = await import('apexcharts');
      ApexCharts = module.default;
    });

    interface ChartSeries {
      name: string;
      data: number[];
    }
  
    interface ChartOptions {
      chart: {
        type: string;
        height: number;
        toolbar: {
          show: boolean;
        };
      };
      plotOptions: {
        bar: {
          horizontal: boolean;
          columnWidth: string;
          borderRadius: number;
        };
      };
      dataLabels: {
        enabled: boolean;
      };
      stroke: {
        show: boolean;
        width: number;
        colors: string[];
      };
      xaxis: {
        categories: string[];
      };
      yaxis: {
        title: {
          text: string;
        };
      };
      fill: {
        opacity: number;
      };
      tooltip: {
        y: {
          formatter: (value: number) => string;
        };
      };
      series: ChartSeries[];
    }
  
    let chartOptions: ChartOptions = {
      chart: {
        type: 'bar',
        height: 350,
        toolbar: {
          show: false
        }
      },
      plotOptions: {
        bar: {
          horizontal: false,
          columnWidth: '55%',
          borderRadius: 4
        }
      },
      dataLabels: {
        enabled: false
      },
      stroke: {
        show: true,
        width: 2,
        colors: ['transparent']
      },
      xaxis: {
        categories: ['Critical', 'High', 'Medium', 'Unknown']
      },
      yaxis: {
        title: {
          text: 'Number of Vulnerabilities'
        }
      },
      fill: {
        opacity: 1
      },
      tooltip: {
        y: {
          formatter: (value: number) => `${value} vulnerabilities`
        }
      },
      series: [{
        name: 'Open Vulnerabilities',
        data: [0, 0, 0, 0]
      }]
    };
  
    let chart: any;
  
    async function fetchOpenVulnerabilities() {
      try {
        const result = await $pocketbase.collection('nuclei_findings').getFullList(200, {
          filter: `
            (severity = "critical" || severity = "high" || severity = "medium" || severity = "unknown")
            && (status = "open" || status = "")
          `
        });
  
        const criticalCount = result.filter(item => item.severity === 'critical').length;
        const highCount = result.filter(item => item.severity === 'high').length;
        const mediumCount = result.filter(item => item.severity === 'medium').length;
        const unknownCount = result.filter(item => item.severity === 'unknown').length;
  
        if (chartOptions.series && chartOptions.series[0]) {
          chartOptions.series[0].data = [criticalCount, highCount, mediumCount, unknownCount];
          if (chart) {
            chart.updateOptions(chartOptions);
          }
        }
      } catch (error) {
        console.error('Error fetching open vulnerabilities:', error);
      }
    }
  
    onMount(async () => {
      if (typeof window !== 'undefined') {
        const chartElement = document.querySelector('#openVulnerabilitiesChart');
        if (chartElement && ApexCharts) {
          chart = new ApexCharts(chartElement, chartOptions);
          await chart.render();
          await fetchOpenVulnerabilities();
        }
      }
    });
  </script>
  
  <Card size="xl" class="w-full max-w-none 2xl:col-span-2">
    <div class="mb-4 flex items-center justify-between">
      <div class="flex-shrink-0">
        <Heading tag="h3" class="text-2xl">All Open Vulnerabilities</Heading>
        <p class="text-base font-light text-gray-500 dark:text-gray-400">
          Vulnerabilities not acknowledged or marked as false positive
        </p>
      </div>
    </div>
  
    <div id="openVulnerabilitiesChart" class="w-full h-96"></div>
  </Card>
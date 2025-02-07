<script lang="ts">
  import { onMount } from 'svelte';
  import { Card, Button, Toggle, Input, Label, Toast, Breadcrumb, BreadcrumbItem } from 'flowbite-svelte';
  import { pocketbase } from '$lib/stores/pocketbase';
  import { CheckCircleSolid, ExclamationCircleSolid } from 'flowbite-svelte-icons';
  import { goto } from '$app/navigation';

  interface Settings {
    scan_concurrency: number;
    auto_scan_enabled: boolean;
    auto_scan_interval: number;
    retention_period: number;
    debug_mode: boolean;
    stale_threshold_days: number;
    max_cost_per_month: number;
    sender_name: string;
    sender_address: string;
    smtp_host: string;
    smtp_port: number;
    smtp_username: string;
    smtp_password: string;
    smtp_encryption: string;
    app_name: string;
    app_url: string;
  }

  let settings: Settings = {
    scan_concurrency: 10,
    auto_scan_enabled: false,
    auto_scan_interval: 24,
    retention_period: 30,
    debug_mode: false,
    stale_threshold_days: 30,
    max_cost_per_month: 50,
    sender_name: '',
    sender_address: '',
    smtp_host: '',
    smtp_port: 587,
    smtp_username: '',
    smtp_password: '',
    smtp_encryption: 'tls',
    app_name: 'Orbit',
    app_url: ''
  };

  let saveMessage = '';
  let loading = true;
  let showToast = false;
  let isError = false;
  let isSuperAdmin = false;

  function handleToggle(event: CustomEvent<boolean>, key: keyof Settings) {
    if (typeof settings[key] === 'boolean') {
      settings = {
        ...settings,
        [key]: event.detail
      };
    }
  }

  onMount(async () => {
    try {
      // Check if user is super admin
      isSuperAdmin = $pocketbase.authStore.isAdmin;

      // Fetch system settings
      const record = await $pocketbase.collection('system_settings').getFirstListItem('id != ""');
      if (record) {
        settings = {
          ...settings,
          scan_concurrency: record.scan_concurrency || 10,
          auto_scan_enabled: record.auto_scan_enabled || false,
          auto_scan_interval: record.auto_scan_interval || 24,
          retention_period: record.retention_period || 30,
          debug_mode: record.debug_mode || false,
          stale_threshold_days: record.stale_threshold_days || 30,
          max_cost_per_month: record.max_cost_per_month || 50
        };
      }

      // Only fetch mail settings if super admin
      if (isSuperAdmin) {
        try {
          const mailSettings = await $pocketbase.settings.getAll();
          if (mailSettings) {
            settings = {
              ...settings,
              sender_name: mailSettings.meta?.senderName || '',
              sender_address: mailSettings.meta?.senderAddress || '',
              smtp_host: mailSettings.smtp?.host || '',
              smtp_port: mailSettings.smtp?.port || 587,
              smtp_username: mailSettings.smtp?.username || '',
              smtp_password: '', // Don't load the password for security
              smtp_encryption: mailSettings.smtp?.tls ? 'tls' : 'none',
              app_name: mailSettings.meta?.appName || 'Orbit',
              app_url: mailSettings.meta?.appUrl || ''
            };
          }
        } catch (err) {
          console.log('Error fetching mail settings:', err);
        }
      }
    } catch (error: any) {
      console.error('Error fetching settings:', error);
      if (error.status === 401) {
        goto('/authentication/sign-in');
      }
    } finally {
      loading = false;
    }
  });

  async function saveSettings() {
    try {
      // Prepare system settings data
      const settingsData = {
        scan_concurrency: settings.scan_concurrency,
        auto_scan_enabled: settings.auto_scan_enabled,
        auto_scan_interval: settings.auto_scan_interval,
        retention_period: settings.retention_period,
        debug_mode: settings.debug_mode,
        stale_threshold_days: settings.stale_threshold_days,
        max_cost_per_month: settings.max_cost_per_month
      };

      // Save system settings
      let record;
      try {
        record = await $pocketbase.collection('system_settings').getFirstListItem('id != ""');
        await $pocketbase.collection('system_settings').update(record.id, settingsData);
      } catch {
        await $pocketbase.collection('system_settings').create(settingsData);
      }

      // Save email settings only if super admin
      if (isSuperAdmin) {
        try {
          const currentSettings = await $pocketbase.settings.getAll();
          
          // Convert smtp_port to number
          const smtpPort = parseInt(settings.smtp_port.toString(), 10);
          if (isNaN(smtpPort)) {
            throw new Error('SMTP Port must be a valid number');
          }
          
          const updatedSettings = {
            meta: {
              ...currentSettings.meta,
              senderName: settings.sender_name,
              senderAddress: settings.sender_address,
              appName: settings.app_name,
              appUrl: settings.app_url
            },
            smtp: {
              enabled: true,
              host: settings.smtp_host,
              port: smtpPort,
              username: settings.smtp_username,
              tls: settings.smtp_encryption === 'tls',
              ...(settings.smtp_password ? { password: settings.smtp_password } : {})
            }
          };

          console.log('Saving settings:', updatedSettings);
          await $pocketbase.settings.update(updatedSettings);
        } catch (err: any) {
          console.error('Error saving settings:', err);
          saveMessage = err?.data?.message || err?.message || 'Error saving settings';
          isError = true;
          throw err;
        }
      }

      saveMessage = 'Settings saved successfully';
      isError = false;
    } catch (error: any) {
      console.error('Error saving settings:', error);
      if (error.status === 401) {
        goto('/authentication/sign-in');
        return;
      }
      saveMessage = error?.data?.message || 'Error saving settings';
      isError = true;
    }
    showToast = true;
    setTimeout(() => {
      showToast = false;
    }, 3000);
  }
</script>

<div class="container mx-auto px-4 py-6">
  <Breadcrumb class="mb-4">
    <BreadcrumbItem href="/">Home</BreadcrumbItem>
    <BreadcrumbItem href="/settings">Settings</BreadcrumbItem>
    <BreadcrumbItem>System Settings</BreadcrumbItem>
  </Breadcrumb>

  <h1 class="text-2xl font-bold mb-6">System Settings</h1>

  {#if loading}
    <p class="text-gray-600 dark:text-gray-400">Loading settings...</p>
  {:else}
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
      <!-- Cost Settings -->
      <Card>
        <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Cost Settings</h2>
        <div class="space-y-4">
          <div>
            <Label for="max_cost" class="mb-2">Maximum Monthly Cost (USD)</Label>
            <Input
              id="max_cost"
              type="number"
              min="0"
              max="1000"
              step="0.01"
              bind:value={settings.max_cost_per_month}
              class="max-w-md"
            />
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Set to 0 for no limit. Maximum allowed value is $1,000.
            </p>
          </div>
        </div>
      </Card>

      <!-- Debug Settings -->
      <Card>
        <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Debug Settings</h2>
        <div class="space-y-4">
          <div class="flex items-center space-x-2">
            <Toggle 
              bind:checked={settings.debug_mode}
            />
            <span class="text-gray-700 dark:text-gray-300">Enable Debug Mode</span>
          </div>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            Enable detailed logging and debugging features
          </p>
        </div>
      </Card>

      <!-- Scan Settings -->
      <Card>
        <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Scan Settings</h2>
        <div class="space-y-4">
          <div>
            <Label for="scan_concurrency" class="mb-2">Scan Concurrency</Label>
            <Input
              id="scan_concurrency"
              type="number"
              bind:value={settings.scan_concurrency}
              min="1"
              max="100"
              class="w-full"
            />
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Maximum number of concurrent scans (1-100)
            </p>
          </div>

          <div class="flex items-center space-x-2">
            <Toggle 
              bind:checked={settings.auto_scan_enabled}
            />
            <span class="text-gray-700 dark:text-gray-300">Enable Automatic Scanning</span>
          </div>

          <div>
            <Label for="auto_scan_interval" class="mb-2">Auto Scan Interval (hours)</Label>
            <Input
              id="auto_scan_interval"
              type="number"
              bind:value={settings.auto_scan_interval}
              min="1"
              max="168"
              class="w-full"
            />
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              How often to run automatic scans (1-168 hours)
            </p>
          </div>
        </div>
      </Card>

      <!-- Data Management -->
      <Card>
        <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Data Management</h2>
        <div class="space-y-4">
          <div>
            <Label for="retention_period" class="mb-2">Data Retention Period (days)</Label>
            <Input
              id="retention_period"
              type="number"
              bind:value={settings.retention_period}
              min="1"
              max="365"
              class="w-full"
            />
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              How long to keep scan results (1-365 days)
            </p>
          </div>

          <div>
            <Label for="stale_threshold" class="mb-2">Finding Stale Threshold (days)</Label>
            <Input
              id="stale_threshold"
              type="number"
              bind:value={settings.stale_threshold_days}
              min="1"
              max="365"
              class="w-full"
            />
            <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
              Number of days after which a finding is considered stale if not seen
            </p>
          </div>
        </div>
      </Card>

      {#if isSuperAdmin}
        <!-- Email Settings -->
        <Card>
          <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Email Settings</h2>
          <div class="space-y-4">
            <div>
              <Label for="sender_name" class="mb-2">Sender Name</Label>
              <Input
                id="sender_name"
                type="text"
                bind:value={settings.sender_name}
                class="w-full"
                placeholder="Orbit Security"
              />
            </div>

            <div>
              <Label for="sender_address" class="mb-2">Sender Address</Label>
              <Input
                id="sender_address"
                type="email"
                bind:value={settings.sender_address}
                class="w-full"
                placeholder="security@yourdomain.com"
              />
            </div>

            <div>
              <Label for="smtp_host" class="mb-2">SMTP Host</Label>
              <Input
                id="smtp_host"
                type="text"
                bind:value={settings.smtp_host}
                class="w-full"
                placeholder="smtp.gmail.com"
              />
            </div>

            <div>
              <Label for="smtp_port" class="mb-2">SMTP Port</Label>
              <Input
                id="smtp_port"
                type="number"
                bind:value={settings.smtp_port}
                class="w-full"
                placeholder="587"
              />
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                Common ports: 25 (SMTP), 465 (SMTPS), 587 (Submission)
              </p>
            </div>

            <div>
              <Label for="smtp_username" class="mb-2">SMTP Username</Label>
              <Input
                id="smtp_username"
                type="text"
                bind:value={settings.smtp_username}
                class="w-full"
                placeholder="your-email@gmail.com"
              />
            </div>

            <div>
              <Label for="smtp_password" class="mb-2">SMTP Password</Label>
              <Input
                id="smtp_password"
                type="password"
                bind:value={settings.smtp_password}
                class="w-full"
                placeholder="••••••••"
              />
            </div>

            <div>
              <Label for="smtp_encryption" class="mb-2">SMTP Encryption</Label>
              <select
                id="smtp_encryption"
                bind:value={settings.smtp_encryption}
                class="block w-full rounded-lg border border-gray-300 bg-gray-50 p-2.5 text-sm text-gray-900 focus:border-blue-500 focus:ring-blue-500 dark:border-gray-600 dark:bg-gray-700 dark:text-white dark:placeholder-gray-400 dark:focus:border-blue-500 dark:focus:ring-blue-500"
              >
                <option value="tls">TLS</option>
                <option value="none">None</option>
              </select>
            </div>
          </div>
        </Card>

        <!-- Application Settings -->
        <Card>
          <h2 class="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Application Settings</h2>
          <div class="space-y-4">
            <div>
              <Label for="app_name" class="mb-2">Application Name</Label>
              <Input
                id="app_name"
                type="text"
                bind:value={settings.app_name}
                class="w-full"
                placeholder="Orbit"
              />
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                The name of your application (appears in emails and UI)
              </p>
            </div>

            <div>
              <Label for="app_url" class="mb-2">Application URL</Label>
              <Input
                id="app_url"
                type="url"
                bind:value={settings.app_url}
                class="w-full"
                placeholder="https://your-orbit-instance.com"
              />
              <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
                The base URL of your application (used for email links)
              </p>
            </div>
          </div>
        </Card>
      {/if}

      <!-- Save Button -->
      <div class="col-span-full flex justify-end mt-4">
        <Button color="blue" on:click={saveSettings}>Save Settings</Button>
      </div>

      <!-- Toast Notification -->
      {#if showToast}
        <div class="fixed bottom-4 right-4">
          <Toast class="mb-2">
            <svelte:fragment slot="icon">
              {#if isError}
                <ExclamationCircleSolid class="w-5 h-5 text-red-600" />
              {:else}
                <CheckCircleSolid class="w-5 h-5 text-green-600" />
              {/if}
            </svelte:fragment>
            <div class="ml-3 text-sm font-normal">
              {saveMessage}
            </div>
          </Toast>
        </div>
      {/if}
    </div>
  {/if}
</div>

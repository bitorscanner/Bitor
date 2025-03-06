<script lang="ts">
    import { Modal, Button, Input, Label, Toggle } from 'flowbite-svelte';
    export let open: boolean = false;

    // Define the settings with default values
    let settings = {
      scanAllIps: false,
      automaticScan: false,
      validate: false,
      noStrictSyntax: false,
      followRedirects: false,
      storeResp: false,
      jsonl: false,
      verbose: false,
      updateTemplates: false,
      interactshServer: 'oast.pro,oast.live,oast.site,oast.online,oast.fun,oast.me', // Default value
      interactshToken: '',
      interactionsCacheSize: 5000,
      interactionsEviction: 60,
      interactionsPollDuration: 5,
      interactionsCooldownPeriod: 5,
      noInteractsh: false,
      author: '',
      tags: '',
      excludeTags: '',
      includeTags: '',
      templateId: '',
      excludeId: '',
      includeTemplates: '',
      excludeTemplates: '',
      excludeMatchers: '',
      severity: '',
      excludeSeverity: '',
      type: '',
      excludeType: '',
      templateCondition: '',
    };

    // Track which settings are enabled
    let enabledSettings = {
      scanAllIps: false,
      automaticScan: false,
      validate: false,
      noStrictSyntax: false,
      followRedirects: false,
      storeResp: false,
      jsonl: false,
      verbose: false,
      updateTemplates: false,
      interactshServer: false,
      interactshToken: false,
      interactionsCacheSize: false,
      interactionsEviction: false,
      interactionsPollDuration: false,
      interactionsCooldownPeriod: false,
      noInteractsh: false,
      author: false,
      tags: false,
      excludeTags: false,
      includeTags: false,
      templateId: false,
      excludeId: false,
      includeTemplates: false,
      excludeTemplates: false,
      excludeMatchers: false,
      severity: false,
      excludeSeverity: false,
      type: false,
      excludeType: false,
      templateCondition: false,
    };

    // Control the visibility of the INTERACTSH section
    let showInteractshSettings = false;
    let showFilteringSettings = false;

    function saveSettings() {
      console.log('Settings saved:', settings);
      open = false;
    }
</script>

<Modal bind:open>
  <div>
    <h3 class="text-lg font-semibold mb-4">Nuclei Settings</h3>
    <div class="space-y-4">
      <!-- INTERACTSH Section -->
      <div>
        <Label>
          <span>INTERACTSH Settings</span>
          <Toggle bind:checked={showInteractshSettings} />
        </Label>
        {#if showInteractshSettings}
          <div class="pl-4 space-y-4">
            <Label>
              <span>Interactsh Server URL</span>
              <Toggle bind:checked={enabledSettings.interactshServer} />
              <Input bind:value={settings.interactshServer} disabled={!enabledSettings.interactshServer} placeholder="Enter server URL" />
              <p class="text-sm text-gray-500">Interactsh server URL for self-hosted instance.</p>
            </Label>
            <Label>
              <span>Interactsh Token</span>
              <Toggle bind:checked={enabledSettings.interactshToken} />
              <Input bind:value={settings.interactshToken} disabled={!enabledSettings.interactshToken} placeholder="Enter token" />
              <p class="text-sm text-gray-500">Authentication token for self-hosted interactsh server.</p>
            </Label>
            <Label>
              <span>Interactions Cache Size</span>
              <Toggle bind:checked={enabledSettings.interactionsCacheSize} />
              <Input type="number" bind:value={settings.interactionsCacheSize} disabled={!enabledSettings.interactionsCacheSize} />
              <p class="text-sm text-gray-500">Number of requests to keep in the interactions cache.</p>
            </Label>
            <Label>
              <span>Interactions Eviction (seconds)</span>
              <Toggle bind:checked={enabledSettings.interactionsEviction} />
              <Input type="number" bind:value={settings.interactionsEviction} disabled={!enabledSettings.interactionsEviction} />
              <p class="text-sm text-gray-500">Seconds to wait before evicting requests from cache.</p>
            </Label>
            <Label>
              <span>Interactions Poll Duration (seconds)</span>
              <Toggle bind:checked={enabledSettings.interactionsPollDuration} />
              <Input type="number" bind:value={settings.interactionsPollDuration} disabled={!enabledSettings.interactionsPollDuration} />
              <p class="text-sm text-gray-500">Seconds to wait before each interaction poll request.</p>
            </Label>
            <Label>
              <span>Interactions Cooldown Period (seconds)</span>
              <Toggle bind:checked={enabledSettings.interactionsCooldownPeriod} />
              <Input type="number" bind:value={settings.interactionsCooldownPeriod} disabled={!enabledSettings.interactionsCooldownPeriod} />
              <p class="text-sm text-gray-500">Extra time for interaction polling before exiting.</p>
            </Label>
            <Label>
              <span>No Interactsh</span>
              <Toggle bind:checked={enabledSettings.noInteractsh} />
              <Toggle bind:checked={settings.noInteractsh} disabled={!enabledSettings.noInteractsh} />
              <p class="text-sm text-gray-500">Disable interactsh server for OAST testing.</p>
            </Label>
          </div>
        {/if}
      </div>

      <!-- FILTERING Section -->
      <div>
        <Label>
          <span>Filtering Settings</span>
          <Toggle bind:checked={showFilteringSettings} />
        </Label>
        {#if showFilteringSettings}
          <div class="pl-4 space-y-4">
            <Label>
              <span>Author</span>
              <Toggle bind:checked={enabledSettings.author} />
              <Input bind:value={settings.author} disabled={!enabledSettings.author} placeholder="Enter authors" />
              <p class="text-sm text-gray-500">Templates to run based on authors (comma-separated, file).</p>
            </Label>
            <Label>
              <span>Tags</span>
              <Toggle bind:checked={enabledSettings.tags} />
              <Input bind:value={settings.tags} disabled={!enabledSettings.tags} placeholder="Enter tags" />
              <p class="text-sm text-gray-500">Templates to run based on tags (comma-separated, file).</p>
            </Label>
            <Label>
              <span>Exclude Tags</span>
              <Toggle bind:checked={enabledSettings.excludeTags} />
              <Input bind:value={settings.excludeTags} disabled={!enabledSettings.excludeTags} placeholder="Enter tags to exclude" />
              <p class="text-sm text-gray-500">Templates to exclude based on tags (comma-separated, file).</p>
            </Label>
            <Label>
              <span>Include Tags</span>
              <Toggle bind:checked={enabledSettings.includeTags} />
              <Input bind:value={settings.includeTags} disabled={!enabledSettings.includeTags} placeholder="Enter tags to include" />
              <p class="text-sm text-gray-500">Tags to be executed even if they are excluded either by default or configuration.</p>
            </Label>
            <Label>
              <span>Template ID</span>
              <Toggle bind:checked={enabledSettings.templateId} />
              <Input bind:value={settings.templateId} disabled={!enabledSettings.templateId} placeholder="Enter template IDs" />
              <p class="text-sm text-gray-500">Templates to run based on template IDs (comma-separated, file, allow-wildcard).</p>
            </Label>
            <Label>
              <span>Exclude ID</span>
              <Toggle bind:checked={enabledSettings.excludeId} />
              <Input bind:value={settings.excludeId} disabled={!enabledSettings.excludeId} placeholder="Enter IDs to exclude" />
              <p class="text-sm text-gray-500">Templates to exclude based on template IDs (comma-separated, file).</p>
            </Label>
            <Label>
              <span>Include Templates</span>
              <Toggle bind:checked={enabledSettings.includeTemplates} />
              <Input bind:value={settings.includeTemplates} disabled={!enabledSettings.includeTemplates} placeholder="Enter paths to include" />
              <p class="text-sm text-gray-500">Path to template file or directory to be executed even if they are excluded either by default or configuration.</p>
            </Label>
            <Label>
              <span>Exclude Templates</span>
              <Toggle bind:checked={enabledSettings.excludeTemplates} />
              <Input bind:value={settings.excludeTemplates} disabled={!enabledSettings.excludeTemplates} placeholder="Enter paths to exclude" />
              <p class="text-sm text-gray-500">Path to template file or directory to exclude (comma-separated, file).</p>
            </Label>
            <Label>
              <span>Exclude Matchers</span>
              <Toggle bind:checked={enabledSettings.excludeMatchers} />
              <Input bind:value={settings.excludeMatchers} disabled={!enabledSettings.excludeMatchers} placeholder="Enter matchers to exclude" />
              <p class="text-sm text-gray-500">Template matchers to exclude in result.</p>
            </Label>
            <Label>
              <span>Severity</span>
              <Toggle bind:checked={enabledSettings.severity} />
              <Input bind:value={settings.severity} disabled={!enabledSettings.severity} placeholder="Enter severities" />
              <p class="text-sm text-gray-500">Templates to run based on severity. Possible values: info, low, medium, high, critical, unknown.</p>
            </Label>
            <Label>
              <span>Exclude Severity</span>
              <Toggle bind:checked={enabledSettings.excludeSeverity} />
              <Input bind:value={settings.excludeSeverity} disabled={!enabledSettings.excludeSeverity} placeholder="Enter severities to exclude" />
              <p class="text-sm text-gray-500">Templates to exclude based on severity. Possible values: info, low, medium, high, critical, unknown.</p>
            </Label>
            <Label>
              <span>Type</span>
              <Toggle bind:checked={enabledSettings.type} />
              <Input bind:value={settings.type} disabled={!enabledSettings.type} placeholder="Enter types" />
              <p class="text-sm text-gray-500">Templates to run based on type. Possible values: http, dns, file, etc.</p>
            </Label>
            <Label>
              <span>Exclude Type</span>
              <Toggle bind:checked={enabledSettings.excludeType} />
              <Input bind:value={settings.excludeType} disabled={!enabledSettings.excludeType} placeholder="Enter types to exclude" />
              <p class="text-sm text-gray-500">Templates to exclude based on type. Possible values: http, dns, file, etc.</p>
            </Label>
            <Label>
              <span>Template Condition</span>
              <Toggle bind:checked={enabledSettings.templateCondition} />
              <Input bind:value={settings.templateCondition} disabled={!enabledSettings.templateCondition} placeholder="Enter template condition" />
              <p class="text-sm text-gray-500">Template condition to be met for a template to be executed.</p>
            </Label>
          </div>
        {/if}
      </div>
    </div>
    <div class="flex justify-end mt-4">
      <Button on:click={saveSettings}>Save</Button>
    </div>
  </div>
</Modal>

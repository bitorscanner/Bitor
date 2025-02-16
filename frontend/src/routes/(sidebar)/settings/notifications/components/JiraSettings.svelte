<script lang="ts">
    import { createEventDispatcher } from 'svelte';
    import { Input, Textarea, Label, Toggle } from 'flowbite-svelte';

    export let settings: {
        enabled: boolean;
        url: string;
        username: string;
        api_token: string;
        project_key: string;
        issue_type: string;
        template: string;
    };

    const dispatch = createEventDispatcher();

    function handleChange() {
        dispatch('change', settings);
    }

    const defaultTemplate = `BHIS conducted scans on the externally accessible web applications with {{tool}}. This was automatically executed from the following IP addresses:

{{scan_ips}}

For more details on this automation, see {{jira_link}}

Tool output is attached to this ticket as a compressed archive format.

Statistics:
{{tool}} Version: {{tool_version}}
Total Targets: {{total_targets}}
Total Skipped Targets: {{skipped_targets}}
Critical Findings: {{critical_findings}}
High Findings: {{high_findings}}
Medium Findings: {{medium_findings}}
Low Findings: {{low_findings}}
Informational Findings: {{info_findings}}
Unknown Findings: {{unknown_findings}}
Total Scan Time: {{scan_time}}

{{#if findings}}
Findings Details:
{{#each findings}}
* {{severity}} - {{title}}
  Target: {{target}}
  Description: {{description}}
{{/each}}
{{/if}}

Scan Details:
- Scan ID: {{scan_id}}
- Scan Name: {{scan_name}}
- Start Time: {{start_time}}
- End Time: {{end_time}}
- Client: {{client_name}}

{{#if additional_notes}}
Additional Notes:
{{additional_notes}}
{{/if}}`;

    function resetTemplate() {
        settings.template = defaultTemplate;
        handleChange();
    }
</script>

<div class="space-y-4">
    <div class="flex items-center justify-between">
        <Label>Enable Jira Integration</Label>
        <Toggle 
            bind:checked={settings.enabled} 
            on:change={handleChange}
        />
    </div>

    {#if settings.enabled}
        <div>
            <Label for="url">Jira URL</Label>
            <Input
                id="url"
                type="url"
                placeholder="https://your-domain.atlassian.net"
                bind:value={settings.url}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">Your Jira instance URL (e.g., https://your-domain.atlassian.net)</p>
        </div>

        <div>
            <Label for="username">Username (Email)</Label>
            <Input
                id="username"
                type="email"
                placeholder="user@example.com"
                bind:value={settings.username}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">Your Jira account email address</p>
        </div>

        <div>
            <Label for="api_token">API Token</Label>
            <Input
                id="api_token"
                type="password"
                placeholder="Jira API token"
                bind:value={settings.api_token}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">
                Generate an API token from 
                <a 
                    href="https://id.atlassian.com/manage-profile/security/api-tokens" 
                    target="_blank" 
                    rel="noopener noreferrer"
                    class="text-blue-600 hover:underline"
                >
                    Atlassian Account Settings
                </a>
            </p>
        </div>

        <div>
            <Label for="project_key">Project Key</Label>
            <Input
                id="project_key"
                type="text"
                placeholder="PROJECT"
                bind:value={settings.project_key}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">The project key where issues will be created (e.g., PROJ)</p>
        </div>

        <div>
            <Label for="issue_type">Issue Type</Label>
            <Input
                id="issue_type"
                type="text"
                placeholder="Task"
                bind:value={settings.issue_type}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">The type of issue to create (e.g., Bug, Task, Story)</p>
        </div>

        <div>
            <div class="flex justify-between items-center">
                <Label for="template">Issue Template</Label>
                <button
                    class="text-sm text-blue-600 hover:underline"
                    on:click={resetTemplate}
                >
                    Reset to Default
                </button>
            </div>
            <Textarea
                id="template"
                rows={15}
                placeholder="Enter your issue template"
                bind:value={settings.template}
                on:change={handleChange}
            />
            <p class="text-sm text-gray-500 mt-1">
                Available placeholders: 
                <br>Tool: {'{{tool}}'}, {'{{tool_version}}'}
                <br>Findings: {'{{critical_findings}}'}, {'{{high_findings}}'}, {'{{medium_findings}}'}, {'{{low_findings}}'}, {'{{info_findings}}'}, {'{{unknown_findings}}'}
                <br>Scan: {'{{scan_id}}'}, {'{{scan_name}}'}, {'{{start_time}}'}, {'{{end_time}}'}, {'{{scan_time}}'}, {'{{total_targets}}'}, {'{{skipped_targets}}'}
                <br>Client: {'{{client_name}}'}, {'{{scan_ips}}'}, {'{{jira_link}}'}
                <br>Findings array: {'{{#each findings}}'} with {'{{severity}}'}, {'{{title}}'}, {'{{target}}'}, {'{{description}}'}
                <br>Optional: {'{{#if additional_notes}}'}, {'{{additional_notes}}'}
            </p>
        </div>
    {/if}
</div> 
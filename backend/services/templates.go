package services

// ScanStartedTemplate is the template for scan started notifications
const ScanStartedTemplate = `A new scan has been initiated in Orbit.

Scan Details:
- Scan ID: {{.scan_id}}
- Scan Name: {{.scan_name}}
- Start Time: {{.time}}
- Client: {{.client_name}}
- Tool: {{.tool}} v{{.tool_version}}
- Total Targets: {{.total_targets}}

For more details on this automation, see {{.jira_link}}`

// ScanFinishedTemplate is the template for scan finished notifications
const ScanFinishedTemplate = `A scan was conducted on externally accessible web applications using {{.tool}}. This scan was automatically executed from the following IP addresses:

{{.scan_ips}}

For more details on this automation, see {{.jira_link}}

Tool output is attached to this ticket as a compressed archive format.

Statistics:
{{.tool}} Version: {{.tool_version}}
Total Targets: {{.total_targets}}
Total Skipped Targets: {{.skipped_targets}}
Critical Findings: {{.critical_findings}}
High Findings: {{.high_findings}}
Medium Findings: {{.medium_findings}}
Low Findings: {{.low_findings}}
Informational Findings: {{.info_findings}}
Unknown Findings: {{.unknown_findings}}
Total Scan Time: {{.scan_time}}

{{if .findings}}
Findings Details:
{{range .findings}}
* {{.severity}} - {{.title}}
  Target: {{.target}}
  Description: {{.description}}
{{end}}
{{end}}

Scan Details:
- Scan ID: {{.scan_id}}
- Scan Name: {{.scan_name}}
- Start Time: {{.start_time}}
- End Time: {{.end_time}}
- Client: {{.client_name}}

{{if .additional_notes}}
Additional Notes:
{{.additional_notes}}
{{end}}`

// ScanFailedTemplate is the template for scan failed notifications
const ScanFailedTemplate = `⚠️ Security scan failed to complete.

Error Details:
{{.error}}

Scan Information:
- Scan Name: {{.scan_name}}
- Scan ID: {{.scan_id}}
- Client: {{.client_name}}
- Time of Failure: {{.time}}
- Tool: {{.tool}}

For more details, see {{.jira_link}}`

// ScanStoppedTemplate is the template for scan stopped notifications
const ScanStoppedTemplate = `Security scan was manually stopped.

Scan Details:
- Scan Name: {{.scan_name}}
- Scan ID: {{.scan_id}}
- Stop Time: {{.time}}
- Client: {{.client_name}}
- Tool: {{.tool}}

For more details, see {{.jira_link}}`

// FindingTemplate is the template for finding notifications
const FindingTemplate = `A new {{.severity}} severity finding has been detected:

Title: {{.title}}
Target: {{.target}}
Description: {{.description}}

Scan Details:
- Scan ID: {{.scan_id}}
- Scan Name: {{.scan_name}}
- Client: {{.client_name}}
- Detection Time: {{.time}}

For more details, see {{.jira_link}}`

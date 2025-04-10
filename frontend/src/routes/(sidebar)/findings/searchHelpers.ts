// File: frontend/src/routes/(sidebar)/findings/searchHelpers.ts

interface FieldGroup {
  name: string;
  fields: Array<{ value: string; name: string }>;
}

export const fieldGroups: FieldGroup[] = [
  {
    name: 'Basic Information',
    fields: [
      { value: 'name', name: 'Name' },
      { value: 'description', name: 'Description' },
      { value: 'severity', name: 'Severity' },
      { value: 'severity_order', name: 'Severity Order' },
      { value: 'type', name: 'Type' },
      { value: 'tool', name: 'Tool' }
    ]
  },
  {
    name: 'Target Information',
    fields: [
      { value: 'host', name: 'Host' },
      { value: 'ip', name: 'IP' },
      { value: 'port', name: 'Port' },
      { value: 'protocol', name: 'Protocol' },
      { value: 'url', name: 'URL' }
    ]
  },
  {
    name: 'Finding Details',
    fields: [
      { value: 'template_id', name: 'Template ID' },
      { value: 'matched_at', name: 'Matched At' },
      { value: 'matcher_name', name: 'Matcher Name' },
      { value: 'curl_command', name: 'Curl Command' },
      { value: 'request', name: 'Request' },
      { value: 'response', name: 'Response' },
      { value: 'extracted_results', name: 'Extracted Results' }
    ]
  },
  {
    name: 'Status & Management',
    fields: [
      { value: 'status', name: 'Status' },
      { value: 'acknowledged', name: 'Acknowledged' },
      { value: 'false_positive', name: 'False Positive' },
      { value: 'remediated', name: 'Remediated' },
      { value: 'notes', name: 'Notes' }
    ]
  },
  {
    name: 'Metadata',
    fields: [
      { value: 'client', name: 'Client' },
      { value: 'scan_id', name: 'Scan ID' },
      { value: 'created_by', name: 'Created By' },
      { value: 'first_seen', name: 'First Seen' },
      { value: 'timestamp', name: 'Timestamp' },
      { value: 'user', name: 'User' }
    ]
  },
  {
    name: 'Additional Info',
    fields: [
      { value: 'info.name', name: 'Finding Name' },
      { value: 'info.tags', name: 'Tags' },
      { value: 'info.description', name: 'Info Description' },
      { value: 'info.reference', name: 'References' },
      { value: 'info.author', name: 'Author' },
      { value: 'info.severity', name: 'Original Severity' }
    ]
  },
  {
    name: 'Classification',
    fields: [
      { value: 'info.classification.cve-id', name: 'CVE ID' },
      { value: 'info.classification.cwe-id', name: 'CWE ID' },
      { value: 'info.classification.cvss-metrics', name: 'CVSS Metrics' }
    ]
  }
];

// Flatten field options for backward compatibility
export const fieldOptions = fieldGroups.flatMap(group => group.fields);

export const operatorOptions = [
  { value: 'equals', name: 'Equals' },
  { value: 'contains', name: 'Contains' },
  { value: 'in', name: 'In' },
  { value: 'not_equals', name: 'Not Equals' },
  { value: 'greater_than', name: 'Greater Than' },
  { value: 'less_than', name: 'Less Than' },
  { value: 'exists', name: 'Exists' },
  { value: 'not_exists', name: 'Not Exists' },
  { value: 'starts_with', name: 'Starts With' },
  { value: 'ends_with', name: 'Ends With' }
];

export const statusOptions = [
  { value: 'open', name: 'Open' },
  { value: 'closed', name: 'Closed' },
  { value: 'in_progress', name: 'In Progress' },
  { value: 'false_positive', name: 'False Positive' },
  { value: 'acknowledged', name: 'Acknowledged' },
  { value: 'remediated', name: 'Remediated' }
];

export function getValueInput(field: string) {
  switch (field) {
    case 'severity':
    case 'info.tags':
    case 'extracted_results':
      return 'MultiSelect';
    case 'client':
      return 'MultiSelect';
    case 'status':
      return 'MultiSelect';
    case 'acknowledged':
    case 'false_positive':
    case 'remediated':
      return 'Select';
    case 'user':
      return 'Select';
    case 'first_seen':
    case 'timestamp':
      return 'date';
    case 'port':
    case 'severity_order':
      return 'number';
    case 'request':
    case 'response':
    case 'notes':
    case 'description':
      return 'textarea';
    default:
      return 'text';
  }
}

export function getValueOptions(field: string, { severityOptions, clientOptions, statusOptions }: { 
  severityOptions: Array<{ value: string; name: string }>;
  clientOptions: Array<{ value: string; name: string }>;
  statusOptions: Array<{ value: string; name: string }>;
}) {
  switch (field) {
    case 'severity':
      return severityOptions;
    case 'client':
      return clientOptions;
    case 'status':
      return statusOptions;
    case 'acknowledged':
    case 'false_positive':
    case 'remediated':
      return [
        { value: 'true', name: 'Yes' },
        { value: 'false', name: 'No' }
      ];
    case 'user':
      return [
        { value: 'all', name: 'All Findings' },
        { value: 'mine', name: 'My Findings Only' }
      ];
    default:
      return [];
  }
}

export function getValuePlaceholder(field: string): string {
  switch (field) {
    case 'severity':
      return 'Select severity';
    case 'client':
      return 'Select client';
    case 'status':
      return 'Select status';
    case 'template_id':
      return 'Enter template ID';
    case 'host':
      return 'Enter hostname';
    case 'ip':
      return 'Enter IP address';
    case 'port':
      return 'Enter port number';
    case 'protocol':
      return 'Enter protocol';
    case 'type':
      return 'Enter finding type';
    case 'tool':
      return 'Enter tool name';
    case 'scan_id':
      return 'Enter scan ID';
    case 'name':
      return 'Enter finding name';
    case 'description':
      return 'Enter description';
    case 'matched_at':
      return 'Enter match location';
    case 'matcher_name':
      return 'Enter matcher name';
    case 'url':
      return 'Enter URL';
    case 'created_by':
      return 'Enter creator ID';
    case 'first_seen':
      return 'Enter date (YYYY-MM-DD)';
    case 'info.name':
      return 'Enter finding name';
    case 'info.tags':
      return 'Enter tags';
    case 'info.description':
      return 'Enter info description';
    case 'info.reference':
      return 'Enter references';
    case 'info.author':
      return 'Enter author';
    case 'info.severity':
      return 'Enter original severity';
    case 'info.classification.cve-id':
      return 'Enter CVE ID';
    case 'info.classification.cwe-id':
      return 'Enter CWE ID';
    case 'info.classification.cvss-metrics':
      return 'Enter CVSS metrics';
    case 'notes':
      return 'Enter notes';
    case 'request':
      return 'Enter request details';
    case 'response':
      return 'Enter response details';
    case 'curl_command':
      return 'Enter curl command';
    case 'extracted_results':
      return 'Enter extracted results';
    case 'severity_order':
      return 'Enter severity order (1-5)';
    case 'timestamp':
      return 'Enter timestamp';
    default:
      return 'Enter value';
  }
}

export function getFieldLabel(field: string): string {
  return fieldOptions.find(f => f.value === field)?.name || field;
}

export function getOperatorLabel(operator: string): string {
  return operatorOptions.find(o => o.value === operator)?.name || operator;
}

export function searchFields(query: string): FieldGroup[] {
  if (!query) return fieldGroups;
  
  const lowerQuery = query.toLowerCase();
  return fieldGroups
    .map(group => ({
      name: group.name,
      fields: group.fields.filter(field => 
        field.name.toLowerCase().includes(lowerQuery) || 
        field.value.toLowerCase().includes(lowerQuery)
      )
    }))
    .filter(group => group.fields.length > 0);
} 
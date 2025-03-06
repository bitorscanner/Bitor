export interface Client {
  id: string;
  name: string;
  api_key: string | null;
  favicon: string;
}

export interface Provider {
  id: string;
  name: string;
  provider_type?: string;
  use?: string[];
}

export interface ScanData {
  name: string;
  nuclei_targets: string;
  scan_profile: string;
  nuclei_profile: string;
  client: string;
  frequency: "one-time" | "scheduled";
  cron: string | null;
  startImmediately: boolean;
  status: string;
}

export interface ScanFormData {
  name: string;
  nuclei_targets: string;
  scan_profile: string;
  nuclei_profile: string;
  nuclei_interact: string;
  vm_provider: string;
  state_bucket: string;
  scan_bucket: string;
  client: string;
  frequency: "one-time" | "scheduled";
  cron?: string | null;
  startImmediately: boolean;
  status: string;
  use_all_templates: boolean;
  selected_templates: string[];
}

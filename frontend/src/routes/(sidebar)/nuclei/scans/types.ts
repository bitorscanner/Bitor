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
  id: string;
  name: string;
  status: string;
  destroyed: boolean;
  start_time?: string;
  end_time?: string;
  progress?: number;
  vm_provider?: string;
  vm_provider_name?: string;
  nuclei_profile?: string;
  nuclei_profile_name?: string;
  nuclei_targets?: string;
  nuclei_interact?: string;
  nuclei_interact_name?: string;
  client?: {
    id: string | null;
    name: string;
    api_key: string | null;
    favicon: string;
  };
  ansible_logs?: any[];
  state_bucket?: string;
  scan_bucket?: string;
  ip_address?: string;
  cost?: number;
  vm_size?: string;
  archived?: boolean;
  vm_start_time?: string;
  vm_stop_time?: string;
  cost_per_hour?: number;
  created_by?: string;
  created_by_name?: string;
  start_time_display?: string;
  end_time_display?: string;
  scan_profile?: string;
  frequency?: string;
  cron?: string;
  startImmediately?: boolean;
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

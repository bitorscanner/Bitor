export interface DigitalOceanSettings {
  region: string;
  do_project: string;
  size: string;
  tags: string[];
}

export interface AWSSettings {
  region: string;
  account_id: string;
}

export interface S3Settings {
  endpoint: string;
  bucket: string;
  region: string;
  use_path_style: boolean;
  statefile_path: string;
  scans_path: string;
}

export interface EmailSettings {
  smtp_host: string;
  smtp_port: number;
  from_address: string;
  encryption: 'none' | 'tls' | 'starttls';
}

export interface WebhookSettings {
  webhook_url: string;
}

export interface TelegramSettings {
  bot_token: string;
  chat_id: string;
}

export interface JiraClientMapping {
  client_id: string;
  organization_id: string;
}

export interface JiraSettings {
  jira_url: string;
  project_key: string;
  jira_project?: string;
  issue_type: string;
  client_mappings: JiraClientMapping[];
}

export type ProviderSettings = 
  | DigitalOceanSettings 
  | AWSSettings 
  | S3Settings 
  | EmailSettings 
  | WebhookSettings 
  | TelegramSettings
  | JiraSettings;

export type ProviderType = 
  | 'aws' 
  | 'digitalocean' 
  | 's3' 
  | 'email' 
  | 'slack' 
  | 'teams' 
  | 'discord' 
  | 'telegram'
  | 'jira';

export interface Provider {
  id?: string;
  name: string;
  provider_type: ProviderType;
  enabled: boolean;
  uses: string[];
  settings: ProviderSettings;
  created?: string;
  updated?: string;
}

export interface ApiKey {
  id: string;
  key: string;
  key_type: string;
  provider: string;
}

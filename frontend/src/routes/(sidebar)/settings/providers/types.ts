export interface DigitalOceanSettings {
  region: string;
  do_project: string;
  size: string;
  tags: string[];
}

export interface TailscaleSettings {
  api_key: string;
  tailnet: string;
  tags: string[];
  subnet_routes: string[];
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
  | JiraSettings
  | TailscaleSettings;

export type AIProviderType = 
  | 'openai'
  | 'anthropic'
  | 'google'
  | 'mistral'
  | 'ollama'
  | 'cohere';

export type ProviderType = 
  | 'aws' 
  | 'digitalocean' 
  | 's3' 
  | 'email' 
  | 'slack' 
  | 'teams' 
  | 'discord' 
  | 'telegram'
  | 'jira'
  | 'tailscale'
  | AIProviderType
  | DiscoveryServiceType;

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

export type UseType = string;

export type DiscoveryServiceType = 
  | 'alienvault'
  | 'binaryedge'
  | 'bufferover'
  | 'censys'
  | 'certspotter'
  | 'chaos'
  | 'github'
  | 'intelx'
  | 'passivetotal'
  | 'securitytrails'
  | 'shodan'
  | 'virustotal'
  | 'whoisxml';

export interface DiscoveryService {
  id?: string;
  name: string;
  service_type: DiscoveryServiceType;
  api_key: string;
  enabled: boolean;
  status?: 'active' | 'inactive' | 'error';
  last_tested?: string;
  created?: string;
  updated?: string;
}

export const DISCOVERY_SERVICES = {
  alienvault: {
    name: 'AlienVault',
    description: 'Threat intelligence platform providing insights into potential security threats'
  },
  binaryedge: {
    name: 'BinaryEdge',
    description: 'Internet security scanning platform'
  },
  bufferover: {
    name: 'BufferOver',
    description: 'DNS data and subdomain enumeration'
  },
  censys: {
    name: 'Censys',
    description: 'Internet security and attack surface management platform'
  },
  certspotter: {
    name: 'CertSpotter',
    description: 'SSL/TLS certificate monitoring'
  },
  chaos: {
    name: 'Chaos',
    description: 'Project Discovery\'s Chaos dataset API'
  },
  github: {
    name: 'GitHub',
    description: 'Source code and repository scanning'
  },
  intelx: {
    name: 'IntelX',
    description: 'Intelligence data search platform'
  },
  passivetotal: {
    name: 'PassiveTotal',
    description: 'RiskIQ\'s threat intelligence platform'
  },
  securitytrails: {
    name: 'SecurityTrails',
    description: 'Security intelligence data platform'
  },
  shodan: {
    name: 'Shodan',
    description: 'Internet-connected device search engine'
  },
  virustotal: {
    name: 'VirusTotal',
    description: 'File and URL analysis platform'
  },
  whoisxml: {
    name: 'WhoisXML API',
    description: 'IP netblocks and WHOIS intelligence platform'
  }
} as const;

export const AI_SERVICES = {
  openai: {
    name: 'OpenAI',
    description: 'GPT-4, GPT-3.5 and other OpenAI models'
  },
  anthropic: {
    name: 'Anthropic',
    description: 'Claude and other Anthropic models'
  },
  google: {
    name: 'Google AI',
    description: 'Gemini and other Google AI models'
  },
  mistral: {
    name: 'Mistral AI',
    description: 'Mistral large language models'
  },
  ollama: {
    name: 'Ollama',
    description: 'Self-hosted open source models'
  },
  cohere: {
    name: 'Cohere',
    description: 'Cohere language models'
  }
} as const;

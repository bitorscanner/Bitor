export type ProviderType =
  | "aws"
  | "digitalocean"
  | "s3"
  | "email"
  | "slack"
  | "teams"
  | "discord"
  | "telegram"
  | "jira";

export interface DigitalOceanSettings {
  region?: string;
  do_project?: string;
  size?: string;
  tags: string[];
  dns_domain?: string;
  token?: string;
}

export interface AWSSettings {
  region?: string;
  vpc?: string;
  subnet?: string;
  instance_type?: string;
  tags: string[];
}

export interface Provider {
  id: string;
  name: string;
  provider_type: ProviderType;
  enabled: boolean;
  uses?: string[];
  settings: DigitalOceanSettings | AWSSettings | Record<string, unknown>;
  created?: string;
  updated?: string;
}

export interface ApiKey {
  id: string;
  key: string;
  key_type: string;
  provider: string;
}

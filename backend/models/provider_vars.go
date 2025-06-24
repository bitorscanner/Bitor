package models

type ProviderVars struct {
	ProviderType string   // Type of provider (aws, digitalocean, s3, etc)
	Name         string   // Provider name
	Key          string   // API key
	SecretKey    string   // Secret key (if applicable)
	Settings     Settings // Provider-specific settings
	Uses         []string // What the provider is used for (compute, terraform_storage, scan_storage)
}

type Settings struct {
	// S3-specific settings
	Bucket        string `json:"bucket,omitempty"`
	Endpoint      string `json:"endpoint,omitempty"`
	Region        string `json:"region,omitempty"`
	UsePathStyle  bool   `json:"use_path_style,omitempty"`
	StatefilePath string `json:"statefile_path,omitempty"`
	ScansPath     string `json:"scans_path,omitempty"`

	// DigitalOcean-specific settings
	Project string   `json:"project,omitempty"`
	Tags    []string `json:"tags,omitempty"`

	// AWS-specific settings
	AccountID    string   `json:"account_id,omitempty"`
	VPC          string   `json:"vpc,omitempty"`
	Subnet       string   `json:"subnet,omitempty"`
	InstanceType string   `json:"instance_type,omitempty"`
	AWSRegion    string   `json:"aws_region,omitempty"` // Added to distinguish from S3 region
	AWSTags      []string `json:"aws_tags,omitempty"`   // AWS-specific tags
}

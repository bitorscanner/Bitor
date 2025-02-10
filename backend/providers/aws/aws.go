package aws

import (
	"context"
	"fmt"
	"orbit/utils/crypto"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
)

// getAWSCredentials retrieves and decrypts AWS credentials from the database
func getAWSCredentials(app *pocketbase.PocketBase, providerID string) (accessKeyID string, secretAccessKey string, err error) {
	// Get access key for this provider
	accessKeyRecord, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "access_key",
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find access key: %w", err)
	}

	// Get secret key for this provider
	secretKeyRecord, err := app.Dao().FindFirstRecordByFilter(
		"api_keys",
		"provider = {:provider} && key_type = {:key_type}",
		dbx.Params{
			"provider": providerID,
			"key_type": "secret_key",
		},
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to find secret key: %w", err)
	}

	// Decrypt the access key
	encryptedAccessKey := accessKeyRecord.GetString("key")
	decryptedAccessKey, err := crypto.Decrypt(encryptedAccessKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt access key: %w", err)
	}

	// Decrypt the secret key
	encryptedSecretKey := secretKeyRecord.GetString("key")
	decryptedSecretKey, err := crypto.Decrypt(encryptedSecretKey, app.Settings().RecordAuthToken.Secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt secret key: %w", err)
	}

	return string(decryptedAccessKey), string(decryptedSecretKey), nil
}

// createAWSConfig creates an AWS config with the provided credentials and region
func createAWSConfig(ctx context.Context, accessKeyID, secretAccessKey, region string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}
	return cfg, nil
}

// FetchRegions fetches all available AWS regions
func FetchRegions(app *pocketbase.PocketBase, providerID string) ([]map[string]interface{}, error) {
	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, "us-east-1") // Use us-east-1 to list regions
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	regions, err := ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(false), // Only fetch enabled regions
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	var result []map[string]interface{}
	for _, region := range regions.Regions {
		result = append(result, map[string]interface{}{
			"id":       aws.ToString(region.RegionName),
			"name":     aws.ToString(region.RegionName),
			"endpoint": aws.ToString(region.Endpoint),
		})
	}

	return result, nil
}

// FetchVPCs fetches all VPCs in the specified region
func FetchVPCs(app *pocketbase.PocketBase, providerID string, region string) ([]map[string]interface{}, error) {
	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	vpcs, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe VPCs: %w", err)
	}

	var result []map[string]interface{}
	for _, vpc := range vpcs.Vpcs {
		name := ""
		for _, tag := range vpc.Tags {
			if aws.ToString(tag.Key) == "Name" {
				name = aws.ToString(tag.Value)
				break
			}
		}

		result = append(result, map[string]interface{}{
			"id":   aws.ToString(vpc.VpcId),
			"name": name,
			"cidr": aws.ToString(vpc.CidrBlock),
		})
	}

	return result, nil
}

// FetchSubnets fetches all subnets in the specified VPC
func FetchSubnets(app *pocketbase.PocketBase, providerID string, region string, vpcID string) ([]map[string]interface{}, error) {
	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	subnets, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe subnets: %w", err)
	}

	var result []map[string]interface{}
	for _, subnet := range subnets.Subnets {
		name := ""
		for _, tag := range subnet.Tags {
			if aws.ToString(tag.Key) == "Name" {
				name = aws.ToString(tag.Value)
				break
			}
		}

		result = append(result, map[string]interface{}{
			"id":                aws.ToString(subnet.SubnetId),
			"name":              name,
			"cidr":              aws.ToString(subnet.CidrBlock),
			"availability_zone": aws.ToString(subnet.AvailabilityZone),
		})
	}

	return result, nil
}

// FetchSecurityGroups fetches all security groups in the specified VPC
func FetchSecurityGroups(app *pocketbase.PocketBase, providerID string, region string, vpcID string) ([]map[string]interface{}, error) {
	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	ec2Client := ec2.NewFromConfig(cfg)
	securityGroups, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe security groups: %w", err)
	}

	var result []map[string]interface{}
	for _, sg := range securityGroups.SecurityGroups {
		result = append(result, map[string]interface{}{
			"id":          aws.ToString(sg.GroupId),
			"name":        aws.ToString(sg.GroupName),
			"description": aws.ToString(sg.Description),
		})
	}

	return result, nil
}

// ValidateCredentials validates the AWS credentials by attempting to get the account ID
func ValidateCredentials(app *pocketbase.PocketBase, providerID string) (string, error) {
	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, "us-east-1")
	if err != nil {
		return "", err
	}

	stsClient := sts.NewFromConfig(cfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("failed to validate credentials: %w", err)
	}

	return aws.ToString(identity.Account), nil
}

// FetchInstanceTypes fetches available EC2 instance types for a region
func FetchInstanceTypes(app *pocketbase.PocketBase, providerID string, region string) ([]map[string]interface{}, error) {
	if region == "" {
		return nil, fmt.Errorf("region is required")
	}

	accessKeyID, secretAccessKey, err := getAWSCredentials(app, providerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS credentials: %w", err)
	}

	ctx := context.Background()
	cfg, err := createAWSConfig(ctx, accessKeyID, secretAccessKey, region)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// Get instance types without filters but with pagination
	input := &ec2.DescribeInstanceTypesInput{
		MaxResults: aws.Int32(100), // Limit to 100 instance types
	}

	fmt.Printf("Fetching instance types for region %s\n", region)

	result, err := ec2Client.DescribeInstanceTypes(ctx, input)
	if err != nil {
		fmt.Printf("Error describing instance types: %v\n", err)
		return []map[string]interface{}{}, fmt.Errorf("failed to describe instance types: %w", err)
	}

	if result == nil {
		fmt.Println("Received nil result from AWS API")
		return []map[string]interface{}{}, nil
	}

	fmt.Printf("Found %d instance types\n", len(result.InstanceTypes))

	// Define the instance families and sizes we want to include
	wantedTypes := map[string]bool{
		// t2 family - burstable, good for development
		"t2.micro":  true,
		"t2.small":  true,
		"t2.medium": true,
		"t2.large":  true,

		// t3 family - newer generation burstable
		"t3.micro":  true,
		"t3.small":  true,
		"t3.medium": true,
		"t3.large":  true,

		// m5/m6 family - general purpose
		"m5.large":   true,
		"m5.xlarge":  true,
		"m6a.large":  true,
		"m6a.xlarge": true,

		// c5/c6 family - compute optimized
		"c5.large":   true,
		"c5.xlarge":  true,
		"c6a.large":  true,
		"c6a.xlarge": true,

		// r5/r6 family - memory optimized
		"r5.large":   true,
		"r5.xlarge":  true,
		"r6a.large":  true,
		"r6a.xlarge": true,
	}

	// Approximate hourly prices (US East prices as reference)
	prices := map[string]float64{
		"t2.micro":   0.0116,
		"t2.small":   0.023,
		"t2.medium":  0.0464,
		"t2.large":   0.0928,
		"t3.micro":   0.0104,
		"t3.small":   0.0208,
		"t3.medium":  0.0416,
		"t3.large":   0.0832,
		"m5.large":   0.096,
		"m5.xlarge":  0.192,
		"m6a.large":  0.0864,
		"m6a.xlarge": 0.1728,
		"c5.large":   0.085,
		"c5.xlarge":  0.17,
		"c6a.large":  0.0765,
		"c6a.xlarge": 0.153,
		"r5.large":   0.126,
		"r5.xlarge":  0.252,
		"r6a.large":  0.1134,
		"r6a.xlarge": 0.2268,
	}

	instanceTypes := make([]map[string]interface{}, 0)
	for _, instanceType := range result.InstanceTypes {
		typeName := string(instanceType.InstanceType)

		// Skip if not in our wanted types
		if !wantedTypes[typeName] {
			continue
		}

		vcpus := "N/A"
		if instanceType.VCpuInfo != nil && instanceType.VCpuInfo.DefaultVCpus != nil {
			vcpus = fmt.Sprintf("%d vCPUs", *instanceType.VCpuInfo.DefaultVCpus)
		}

		memory := "N/A"
		if instanceType.MemoryInfo != nil && instanceType.MemoryInfo.SizeInMiB != nil {
			memory = fmt.Sprintf("%.1f GiB", float64(*instanceType.MemoryInfo.SizeInMiB)/1024)
		}

		price := prices[typeName]
		description := fmt.Sprintf("%s, %s - $%.3f/hour", vcpus, memory, price)

		instanceTypes = append(instanceTypes, map[string]interface{}{
			"id":          typeName,
			"name":        typeName,
			"description": description,
			"price":       price,
		})
		fmt.Printf("Added instance type: %s (%s)\n", typeName, description)
	}

	// Sort instance types by price
	sort.Slice(instanceTypes, func(i, j int) bool {
		return instanceTypes[i]["price"].(float64) < instanceTypes[j]["price"].(float64)
	})

	return instanceTypes, nil
}

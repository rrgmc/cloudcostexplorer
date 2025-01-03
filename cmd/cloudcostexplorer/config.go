package main

import (
	"context"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/rrgmc/cloudcostexplorer"
	aws2 "github.com/rrgmc/cloudcostexplorer/cloud/aws"
	gcp2 "github.com/rrgmc/cloudcostexplorer/cloud/gcp"
)

type Config map[string]ConfigItem

type ConfigItem struct {
	Disabled bool   `toml:"disabled"`
	Cloud    string `toml:"cloud"`
	// AWS
	Profile string `toml:"profile"`
	Region  string `toml:"region"`
	// GCP
	ProjectID     string `toml:"project_id"`
	DefaultTable  string `toml:"default_table"`
	ResourceTable string `toml:"resource_table"`
}

func LoadConfig() (Config, error) {
	f, err := os.Open("cloudcostexplorer.conf")
	if err != nil {
		return nil, fmt.Errorf("error loading config file: %w", err)
	}
	defer f.Close()

	var config Config
	_, err = toml.NewDecoder(f).Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return config, nil
}

func CreateCloud(ctx context.Context, item ConfigItem) (cloudcostexplorer.Cloud, error) {
	switch item.Cloud {
	case "AWS":
		var optns []func(*config.LoadOptions) error
		if item.Profile != "" {
			optns = append(optns, config.WithSharedConfigProfile(item.Profile))
		}
		if item.Region != "" {
			optns = append(optns, config.WithRegion(item.Region))
		}

		cfg, err := config.LoadDefaultConfig(ctx, optns...)
		if err != nil {
			return nil, fmt.Errorf("unable to load AWS SDK config: %v", err)
		}

		return aws2.New(ctx,
			aws2.WithCloudConfig(cfg),
		)
	case "GCP":
		var optns []gcp2.CloudOption
		if item.ProjectID != "" {
			optns = append(optns, gcp2.WithProjectID(item.ProjectID))
		}
		if item.DefaultTable != "" {
			optns = append(optns, gcp2.WithDefaultTableName(item.DefaultTable))
		}
		if item.ResourceTable != "" {
			optns = append(optns, gcp2.WithResourceTableName(item.ResourceTable))
		}

		return gcp2.New(ctx, optns...)
	default:
		return nil, fmt.Errorf("cloud %s not supported", item.Cloud)
	}
}

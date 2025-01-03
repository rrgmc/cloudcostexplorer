package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	"github.com/rrgmc/cloudcostexplorer"
)

type Cloud struct {
	projectID         string
	bigQueryClient    *bigquery.Client
	defaultTableName  string
	resourceTableName string
	blankKeyValue     string

	parameters      cloudcostexplorer.Parameters
	parameterValues map[string]parameterValues
}

var _ cloudcostexplorer.Cloud = (*Cloud)(nil)

func New(ctx context.Context, options ...CloudOption) (*Cloud, error) {
	ret := &Cloud{
		defaultTableName:  "billing_export.gcp_billing_export_v1",
		resourceTableName: "billing_export.gcp_billing_export_resource_v1",
		parameterValues:   make(map[string]parameterValues),
		blankKeyValue:     fmt.Sprintf("blank_value_%s", uuid.New().String()),
	}
	ret.load()
	for _, opt := range options {
		opt(ret)
	}

	bqClient, err := bigquery.NewClient(ctx, ret.projectID)
	if err != nil {
		return nil, fmt.Errorf("error creating BigQuery client: %v", err)
	}

	ret.bigQueryClient = bqClient

	return ret, nil
}

func (c *Cloud) DaysDelay() int {
	return 1
}

func (c *Cloud) MaxGroupBy() int {
	return 4 // no limit
}

func (c *Cloud) Parameters() cloudcostexplorer.Parameters {
	return c.parameters
}

func (c *Cloud) setParameterValue(parameter, value, title string) {
	if _, ok := c.parameterValues[parameter]; !ok {
		c.parameterValues[parameter] = parameterValues{
			values: make(map[string]string),
		}
	}
	if _, ok := c.parameterValues[parameter].values[value]; !ok {
		c.parameterValues[parameter].values[value] = title
	}
}

func (c *Cloud) ParameterTitle(id string, defaultValue string) string {
	d, ok := c.parameterValues[id]
	if !ok {
		return defaultValue
	}
	return d.values[defaultValue]
}

func (c *Cloud) load() {
	c.parameters = cloudcostexplorer.Parameters{
		{
			ID:              "PROJECT",
			Name:            "Project",
			DefaultPriority: 1,
			IsGroup:         true,
			IsGroupFilter:   true,
			IsFilter:        true,
		},
		{
			ID:              "SERVICE",
			Name:            "Service",
			DefaultPriority: 2,
			IsGroup:         true,
			IsGroupFilter:   true,
			IsFilter:        true,
		},
		{
			ID:            "REGION",
			Name:          "Region",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:              "SKU",
			Name:            "SKU",
			DefaultPriority: 3,
			IsGroup:         true,
			IsGroupFilter:   true,
			IsFilter:        true,
		},
		{
			ID:            "COSTTYPE",
			Name:          "Cost type",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "RESOURCE",
			Name:          "Resource",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "LABEL",
			Name:          "Label",
			IsGroup:       true,
			IsGroupFilter: false,
			IsFilter:      true,
			HasData:       true,
		},
		{
			ID:            "SYSLABEL",
			Name:          "Syslabel",
			IsGroup:       true,
			IsGroupFilter: false,
			IsFilter:      true,
			HasData:       true,
		},
		{
			ID:            "TAGS",
			Name:          "Tags",
			IsGroup:       true,
			IsGroupFilter: false,
			IsFilter:      true,
			HasData:       true,
		},
	}
}

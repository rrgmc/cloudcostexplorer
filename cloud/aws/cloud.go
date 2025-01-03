package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/organizations"
	"github.com/rrgmc/cloudcostexplorer"
)

type Cloud struct {
	cfg                *aws.Config
	costExplorerClient *costexplorer.Client

	parameters     cloudcostexplorer.Parameters
	linkedAccounts map[string]string
}

var _ cloudcostexplorer.Cloud = (*Cloud)(nil)

func New(ctx context.Context, options ...CloudOption) (*Cloud, error) {
	ret := &Cloud{}
	for _, opt := range options {
		opt(ret)
	}
	ret.load(ctx)
	if ret.cfg == nil {
		ret.costExplorerClient = costexplorer.New(costexplorer.Options{})
	} else {
		ret.costExplorerClient = costexplorer.NewFromConfig(*ret.cfg)
	}
	return ret, nil
}

func (c *Cloud) DaysDelay() int {
	return 1
}

func (c *Cloud) MaxGroupBy() int {
	return 2 // AWS cost explorer supports a maximum of 2 groups
}

func (c *Cloud) Parameters() cloudcostexplorer.Parameters {
	return c.parameters
}

func (c *Cloud) ParameterTitle(id string, defaultValue string) string {
	if id != "LINKED_ACCOUNT" {
		return defaultValue
	}

	if la, ok := c.linkedAccounts[defaultValue]; ok {
		return la
	}

	return defaultValue
}

func (c *Cloud) load(ctx context.Context) {
	c.parameters = cloudcostexplorer.Parameters{
		{
			ID:              "LINKED_ACCOUNT",
			Name:            "Linked account",
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
			ID:              "USAGE_TYPE",
			Name:            "Usage type",
			DefaultPriority: 3,
			IsGroup:         true,
			IsGroupFilter:   true,
			IsFilter:        true,
		},
		{
			ID:            "USAGE_TYPE_GROUP",
			Name:          "Usage type group",
			IsGroup:       false,
			IsGroupFilter: false,
			IsFilter:      true,
		},
		{
			// max period: 14 days
			ID:            "RESOURCE_ID",
			Name:          "Resource ID",
			MenuTitle:     "Resource ID (only last 14 days)",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "INSTANCE_TYPE",
			Name:          "Instance type",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "INSTANCE_TYPE_FAMILY",
			Name:          "Instance type family",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "PURCHASE_TYPE",
			Name:          "Purchase type",
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "OPERATION",
			Name:          "Operation",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "PLATFORM",
			Name:          "Platform",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "TENANCY",
			Name:          "Tenancy",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "DEPLOYMENT_OPTION",
			Name:          "Deployment option",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "RESERVATION_ID",
			Name:          "Reservation ID",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "RECORD_TYPE",
			Name:          "Record type",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "AZ",
			Name:          "AZ",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "SAVINGS_PLANS_TYPE",
			Name:          "Savings plans type",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "SAVINGS_PLAN_ARN",
			Name:          "Savings plan ARN",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
		},
		{
			ID:            "TAG",
			Name:          "Tag",
			IsGroup:       true,
			IsGroupFilter: true,
			IsFilter:      true,
			HasData:       true,
			DataRequired:  true,
		},
	}

	c.loadLinkedAccounts(ctx)
}

func (c *Cloud) loadLinkedAccounts(ctx context.Context) {
	var orgsClient *organizations.Client
	if c.cfg != nil {
		orgsClient = organizations.NewFromConfig(*c.cfg)
	} else {
		orgsClient = organizations.New(organizations.Options{})
	}

	c.linkedAccounts = map[string]string{}

	for pageOrgs, err := range awsAPIIteratorInput(ctx, &organizations.ListAccountsInput{}, func(ctx context.Context, input *organizations.ListAccountsInput) (*organizations.ListAccountsOutput, error) {
		return orgsClient.ListAccounts(ctx, input)
	}) {
		if err != nil {
			fmt.Printf("error listing accounts: %v\n", err)
			break
		}

		for _, org := range pageOrgs.Accounts {
			c.linkedAccounts[*org.Id] = *org.Name
		}
	}
}

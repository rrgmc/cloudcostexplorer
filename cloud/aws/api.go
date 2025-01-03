package aws

import (
	"context"
	"fmt"
	"iter"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/davecgh/go-spew/spew"
)

type costAndUsageIterResult struct {
	timePeriod *types.DateInterval
	group      types.Group
}

type costAndUsageIter iter.Seq2[costAndUsageIterResult, error]

// costAndUsage calls the AWS cost and usage API with the passed filters and returns an iterator.
func costAndUsage(ctx context.Context, costexplorerClient *costexplorer.Client, costmetric string, start, end string,
	filters *types.Expression, groupBy []types.GroupDefinition) costAndUsageIter {
	return func(yield func(costAndUsageIterResult, error) bool) {
		for data, err := range awsAPIIteratorInput(ctx,
			&costexplorer.GetCostAndUsageInput{
				Granularity: types.GranularityDaily,
				Metrics: []string{
					costmetric,
				},
				Filter: filters,
				TimePeriod: &types.DateInterval{
					Start: aws.String(start),
					End:   aws.String(end),
				},
				GroupBy: groupBy,
			},
			func(ctx context.Context, input *costexplorer.GetCostAndUsageInput) (*costexplorer.GetCostAndUsageOutput, error) {
				return costexplorerClient.GetCostAndUsage(ctx, input)
			}) {
			if err != nil {
				yield(costAndUsageIterResult{}, err)
				return
			}
			if len(data.DimensionValueAttributes) > 0 {
				hasLinkedAccount := false
				for _, gd := range data.GroupDefinitions {
					if gd.Type == types.GroupDefinitionTypeDimension && *gd.Key == "LINKED_ACCOUNT" {
						hasLinkedAccount = true
					}
				}
				if !hasLinkedAccount {
					fmt.Printf("DimensionValueAttributes: %s\n", spew.Sdump(data.DimensionValueAttributes))
				}
			}

			for _, group := range data.ResultsByTime {
				for _, v := range group.Groups {
					if !yield(costAndUsageIterResult{
						timePeriod: group.TimePeriod,
						group:      v,
					}, nil) {
						return
					}
				}
			}
		}
	}
}

// costAndUsageWithResources calls the AWS cost and usage with resources API with the passed filters and returns an iterator.
func costAndUsageWithResources(ctx context.Context, costexplorerClient *costexplorer.Client, costmetric string, start, end string,
	filters *types.Expression, groupBy []types.GroupDefinition) costAndUsageIter {
	return func(yield func(costAndUsageIterResult, error) bool) {
		for data, err := range awsAPIIteratorInput(ctx,
			&costexplorer.GetCostAndUsageWithResourcesInput{
				Granularity: types.GranularityDaily,
				Metrics: []string{
					costmetric,
				},
				Filter: filters,
				TimePeriod: &types.DateInterval{
					Start: aws.String(start),
					End:   aws.String(end),
				},
				GroupBy: groupBy,
			},
			func(ctx context.Context, input *costexplorer.GetCostAndUsageWithResourcesInput) (*costexplorer.GetCostAndUsageWithResourcesOutput, error) {
				return costexplorerClient.GetCostAndUsageWithResources(ctx, input)
			}) {
			if err != nil {
				yield(costAndUsageIterResult{}, err)
				return
			}

			for _, group := range data.ResultsByTime {
				for _, v := range group.Groups {
					if !yield(costAndUsageIterResult{
						timePeriod: group.TimePeriod,
						group:      v,
					}, nil) {
						return
					}
				}
			}
		}
	}
}

// dimensionValues returns the values of a cost explorer dimension based on a filter.
func dimensionValues(ctx context.Context, costexplorerClient *costexplorer.Client,
	start, end string, dimension types.Dimension, tagContext types.Context, filters *types.Expression) iter.Seq2[types.DimensionValuesWithAttributes, error] {
	return func(yield func(types.DimensionValuesWithAttributes, error) bool) {
		for data, err := range awsAPIIteratorInput(ctx, &costexplorer.GetDimensionValuesInput{
			Dimension: dimension,
			Context:   tagContext,
			Filter:    filters,
			TimePeriod: &types.DateInterval{
				Start: aws.String(start),
				End:   aws.String(end),
			},
		}, func(ctx context.Context, input *costexplorer.GetDimensionValuesInput) (*costexplorer.GetDimensionValuesOutput, error) {
			return costexplorerClient.GetDimensionValues(ctx, input)
		}) {
			if err != nil {
				yield(types.DimensionValuesWithAttributes{}, err)
				return
			}
			for _, value := range data.DimensionValues {
				if !yield(value, nil) {
					return
				}
			}
		}
	}
}

// tags returns the possible values for a cost explorer tag based on a filter.
func tags(ctx context.Context, costexplorerClient *costexplorer.Client,
	start, end string, filters *types.Expression, tagKey *string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		for data, err := range awsAPIIteratorInput(ctx, &costexplorer.GetTagsInput{
			Filter: filters,
			TimePeriod: &types.DateInterval{
				Start: aws.String(start),
				End:   aws.String(end),
			},
			TagKey: tagKey,
		}, func(ctx context.Context, input *costexplorer.GetTagsInput) (*costexplorer.GetTagsOutput, error) {
			return costexplorerClient.GetTags(ctx, input)
		}) {
			if err != nil {
				yield("", err)
				return
			}

			for _, value := range data.Tags {
				if !yield(value, nil) {
					return
				}
			}
		}
	}
}

type tagValue struct {
	Name   string
	Values []string
}

// tagsWithValues returns a list of cost explorer tags and their possible values based on a filter.
func tagsWithValues(ctx context.Context, costexplorerClient *costexplorer.Client,
	start, end string, filters *types.Expression) iter.Seq2[tagValue, error] {
	return func(yield func(tagValue, error) bool) {
		for tagName, err := range tags(ctx, costexplorerClient, start, end, filters, nil) {
			if err != nil {
				yield(tagValue{}, fmt.Errorf("couldn't fetch tag data: %w", err))
				return
			}
			curTag := tagValue{
				Name: tagName,
			}
			for tv, err := range tags(ctx, costexplorerClient, start, end, filters, &tagName) {
				if err != nil {
					yield(tagValue{}, fmt.Errorf("couldn't fetch tag '%s' values: %w", tagName, err))
					return
				}
				curTag.Values = append(curTag.Values, tv)
			}
			if !yield(curTag, nil) {
				return
			}
		}
	}
}

type tagValueItem struct {
	value tagValue
	err   error
}

// tagsWithValuesFuture returns tagsWithValues as a channel.
func tagsWithValuesFuture(ctx context.Context, costexplorerClient *costexplorer.Client,
	start, end string, filters *types.Expression) chan tagValueItem {
	c := make(chan tagValueItem, 100)
	go func() {
		defer close(c)
		for tag, err := range tagsWithValues(ctx, costexplorerClient, start, end, filters) {
			if err != nil {
				select {
				case c <- tagValueItem{err: fmt.Errorf("couldn't fetch tag data: %w", err)}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case c <- tagValueItem{value: tag}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return c
}

type usageTagGroupsItem struct {
	value types.DimensionValuesWithAttributes
	err   error
}

// usageTagGroupsFuture returns possible values for the cost explorer "usage type group" dimension as a channel.
func usageTagGroupsFuture(ctx context.Context, costexplorerClient *costexplorer.Client,
	start, end string, filters []types.Expression) chan usageTagGroupsItem {
	c := make(chan usageTagGroupsItem, 100)
	go func() {
		defer close(c)
		utgfilters := slices.DeleteFunc(slices.Clone(filters), func(values types.Expression) bool {
			return values.Dimensions != nil && values.Dimensions.Key == types.DimensionUsageTypeGroup
		})
		for dims, err := range dimensionValues(ctx, costexplorerClient, start, end, types.DimensionUsageTypeGroup,
			types.ContextCostAndUsage, buildCostExplorerFilter(utgfilters)) {
			if err != nil {
				select {
				case c <- usageTagGroupsItem{err: fmt.Errorf("couldn't fetch tag data: %w", err)}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case c <- usageTagGroupsItem{value: dims}:
			case <-ctx.Done():
				return
			}
		}
	}()
	return c
}

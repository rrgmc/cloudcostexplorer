package aws

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/invzhi/timex"
	"github.com/rrgmc/cloudcostexplorer"
)

func (c *Cloud) Query(ctx context.Context, options ...cloudcostexplorer.QueryOption) iter.Seq2[cloudcostexplorer.CloudQueryItem, error] {
	return func(yield func(cloudcostexplorer.CloudQueryItem, error) bool) {
		optns, err := cloudcostexplorer.ParseQueryOptions(options...)
		if err != nil {
			yield(cloudcostexplorer.CloudQueryItem{}, err)
			return
		}

		if len(optns.Groups) > 2 {
			yield(cloudcostexplorer.CloudQueryItem{}, errors.New("AWS Cost Explorer only supports up to 2 groups"))
			return
		}

		var isResource bool
		var isFilter bool
		costmetric := "UnblendedCost"

		start := optns.Start.String()
		// end time is exclusive in cost explorer, must use next day
		end := optns.End.AddDays(1).String()

		var filters []types.Expression
		var groups []types.GroupDefinition

		// FILTERS

		for _, filter := range optns.Filters {
			if filter.ID == "TAG" {
				lkey, lval, _ := strings.Cut(filter.Value, cloudcostexplorer.DataSeparator)
				filters = append(filters, types.Expression{
					Tags: &types.TagValues{
						Key:    cloudcostexplorer.Ptr(lkey),
						Values: []string{lval},
					},
				})
			} else {
				if filter.ID != "LINKED_ACCOUNT" {
					isFilter = true
				}
				filters = append(filters, types.Expression{
					Dimensions: &types.DimensionValues{
						Key:    types.Dimension(strings.ToUpper(filter.ID)),
						Values: []string{filter.Value},
					},
				})
			}
		}

		extraDataCtx, extraDataCancel := context.WithCancel(ctx)
		defer extraDataCancel()

		var usageTypeGroupsFuture chan usageTagGroupsItem
		var tagsFuture chan tagValueItem

		isExtraData := isFilter && optns.ExtraDataCallback != nil

		if isExtraData {
			usageTypeGroupsFuture = usageTagGroupsFuture(extraDataCtx, c.costExplorerClient, start, end, filters)
			tagsFuture = tagsWithValuesFuture(extraDataCtx, c.costExplorerClient, start, end,
				buildCostExplorerFilter(filters))
		}

		// GROUPS

		for _, group := range optns.Groups {
			kgroup, kok := c.parameters.FindById(group.ID)
			if !kok || !kgroup.IsGroup {
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("invalid group '%s'", group.ID))
				return
			}

			if group.ID == "RESOURCE_ID" {
				isResource = true
			}

			if group.ID == "TAG" {
				groups = append(groups, types.GroupDefinition{
					Key:  aws.String(group.Data),
					Type: types.GroupDefinitionTypeTag,
				})
			} else {
				groups = append(groups, types.GroupDefinition{
					Key:  aws.String(group.ID),
					Type: types.GroupDefinitionTypeDimension,
				})
			}
		}

		if isResource {
			// resource grouping have a max of 14 days
			minStart := timex.Today(time.UTC).AddDays(-13)
			if optns.Start.Before(minStart) {
				start = minStart.String()
				end = optns.End.AddDays(cloudcostexplorer.DefaultSkipDays).String()
			}
		}

		var costIter costAndUsageIter
		if isResource {
			costIter = costAndUsageWithResources(ctx, c.costExplorerClient, costmetric, start, end,
				buildCostExplorerFilter(filters), groups)
		} else {
			costIter = costAndUsage(ctx, c.costExplorerClient, costmetric, start, end,
				buildCostExplorerFilter(filters), groups)
		}

		for groupValue, err := range costIter {
			if err != nil {
				yield(cloudcostexplorer.CloudQueryItem{}, err)
				return
			}

			cost, err := strconv.ParseFloat(*groupValue.group.Metrics[costmetric].Amount, 64)
			if err != nil {
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("error parsing cost value '%s' (%s): %w",
					*groupValue.group.Metrics[costmetric].Amount, costmetric, err))
				return
			}

			groupStart, err := timex.ParseDate("YYYY-MM-DD", *groupValue.timePeriod.Start)
			if err != nil {
				yield(cloudcostexplorer.CloudQueryItem{}, fmt.Errorf("error parsing start date '%s': %w", *groupValue.timePeriod.Start, err))
				return
			}

			var itemKeys []cloudcostexplorer.ItemKey
			for groupIdx, group := range optns.Groups {
				groupName := groupValue.group.Keys[groupIdx]

				key := cloudcostexplorer.ItemKey{
					ID:    groupName,
					Value: groupName,
				}

				switch group.ID {
				case "LINKED_ACCOUNT":
					if la, ok := c.linkedAccounts[groupName]; ok {
						key.Value = la
					}
				case "TAG":
					if tn, tv, ok := strings.Cut(groupName, "$"); ok {
						key.ID = fmt.Sprintf("%s%s%s", tn, cloudcostexplorer.DataSeparator, tv)
						key.Value = tv
					}
				}

				itemKeys = append(itemKeys, key)
			}

			if !yield(cloudcostexplorer.CloudQueryItem{
				Date:  groupStart,
				Keys:  itemKeys,
				Value: cost,
			}, nil) {
				return
			}
		}

		if isExtraData {
			if usageTypeGroupsFuture != nil {
				ed := &extraDataUsageTypeGroups{}
				for dims := range usageTypeGroupsFuture {
					if dims.err != nil {
						ed.err = fmt.Errorf("error getting usage type groups: %w", dims.err)
						break
					}
					ad := ""
					if len(dims.value.Attributes) > 1 {
						ad = fmt.Sprintf("%s", spew.Sdump(dims.value.Attributes))
					}
					ed.data = append(ed.data, extraDataUsageTypeGroup{
						Unit:          dims.value.Attributes["unit"],
						Value:         *dims.value.Value,
						AttributeDump: ad,
					})
				}
				optns.ExtraDataCallback(ed)
			}

			if tagsFuture != nil {
				ed := &extraDataTags{
					data: map[string]*extraDataTag{},
				}
				for tt := range tagsFuture {
					if tt.err != nil {
						ed.err = fmt.Errorf("error getting tags: %w", tt.err)
						break
					}
					tag := tt.value
					ed.data[tag.Name] = &extraDataTag{
						Name:   tag.Name,
						Values: tag.Values,
					}
				}
				optns.ExtraDataCallback(ed)
			}
		}
	}
}

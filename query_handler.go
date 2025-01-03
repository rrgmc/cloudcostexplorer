package cloudcostexplorer

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
)

// QueryHandler handles calling [Cloud.Query] while supporting multiple periods.
func QueryHandler(ctx context.Context, cloud Cloud, options ...QueryHandlerOption) (*QueryResult, error) {
	optns, err := parseQueryHandlerOptions(options...)
	if err != nil {
		return nil, err
	}

	items := map[string]*Item{}

	ret := QueryResult{
		PeriodsSameDuration: true,
	}
	lastPeriodDurationDays := -1
	for _, periodList := range optns.periodLists {
		for _, period := range periodList.Periods {
			ret.Periods = append(ret.Periods, QueryResultPeriod{
				QueryPeriod: period,
			})
			durationDays := period.End.Sub(period.Start)
			if lastPeriodDurationDays == -1 {
				lastPeriodDurationDays = durationDays
			} else if durationDays != lastPeriodDurationDays {
				ret.PeriodsSameDuration = false
			}
		}
	}
	if len(ret.Periods) == 0 {
		return nil, errors.New("at least one period is required")
	}

	for _, group := range optns.groups {
		kgroup, kok := cloud.Parameters().FindById(group.ID)
		if !kok || !kgroup.IsGroup {
			return nil, fmt.Errorf("invalid group '%s'", group.ID)
		}

		ret.Groups = append(ret.Groups, QueryResultGroup{
			Parameter: kgroup,
			Data:      group.Data,
		})
	}

	var extraData []QueryExtraData

	periodStart := 0
	for _, periodList := range optns.periodLists {
		if len(periodList.Periods) == 0 {
			continue
		}

		ok, start, end := periodList.Range()
		if !ok {
			continue
		}

		isSinglePeriod := len(periodList.Periods) == 1

		qopts := []QueryOption{
			WithQueryDates(start, end),
			WithQueryGroups(optns.groups...),
			WithQueryFilters(optns.filters...),
			WithQueryExtraData(func(data QueryExtraData) {
				extraData = append(extraData, data)
			}),
		}

		if !isSinglePeriod {
			qopts = append(qopts, WithQueryGroupByDate(true))
		}

		for item, err := range cloud.Query(ctx, qopts...) {
			if err != nil {
				return nil, err
			}

			if optns.filterKeys != nil && !optns.filterKeys(item.Keys) {
				continue
			}

			itemHash := optns.itemKeysHash(item.Keys)
			if _, ok := items[itemHash]; !ok {
				items[itemHash] = NewItem(item.Keys, len(ret.Periods))
			}
			ret.TotalValue += item.Value

			periodMatches := 0
			for periodIdx, period := range periodList.Periods {
				if !isSinglePeriod && !DateBetweenDates(item.Date, period.Start, period.End) {
					continue
				}
				items[itemHash].Values[periodStart+periodIdx] += item.Value
				ret.Periods[periodStart+periodIdx].TotalValue += item.Value
				periodMatches++
			}

			if periodMatches != 1 && optns.onPeriodMatchError != nil {
				err := optns.onPeriodMatchError(item, periodStart)
				if err != nil {
					return nil, err
				}
			}
		}

		periodStart += len(periodList.Periods)
	}

	ret.Items = slices.Collect(maps.Values(items))
	ret.ExtraOutput = cloud.QueryExtraOutput(ctx, extraData)

	return &ret, nil
}

type QueryHandlerOption func(options *queryHandlerOptions)

func parseQueryHandlerOptions(options ...QueryHandlerOption) (queryHandlerOptions, error) {
	var optns queryHandlerOptions
	for _, opt := range options {
		opt(&optns)
	}

	if len(optns.groups) == 0 {
		return queryHandlerOptions{}, errors.New("at least one group is required")
	}
	if optns.itemKeysHash == nil {
		optns.itemKeysHash = DefaultItemKeysHash
	}
	return optns, nil
}

// WithQueryHandlerPeriodLists sets a list of period lists to query.
func WithQueryHandlerPeriodLists(periodList ...QueryPeriodList) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.periodLists = append(options.periodLists, periodList...)
	}
}

// WithQueryHandlerPeriods adds a list of periods to query.
func WithQueryHandlerPeriods(periods ...QueryPeriod) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.periodLists = append(options.periodLists, QueryPeriodList{
			Periods: periods,
		})
	}
}

// WithQueryHandlerGroups sets the groups to use for querying.
func WithQueryHandlerGroups(groups ...QueryGroup) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.groups = groups
	}
}

// WithQueryHandlerFilters sets the filters to use for querying.
func WithQueryHandlerFilters(filters ...QueryFilter) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.filters = append(options.filters, filters...)
	}
}

// WithQueryHandlerItemKeysHash sets the function to use for hashing the item keys. The default is [DefaultItemKeysHash].
func WithQueryHandlerItemKeysHash(f func(keys []ItemKey) string) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.itemKeysHash = f
	}
}

// WithQueryHandlerFilterKeys sets a function to use for filtering the item keys, if needed.
func WithQueryHandlerFilterKeys(f func(keys []ItemKey) bool) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.filterKeys = f
	}
}

// WithQueryHandlerOnPeriodMatchError sets a function to handle period mismatches. If the function returns an error,
// processing is stopped and the error is returned.
func WithQueryHandlerOnPeriodMatchError(onPeriodMatchError func(item CloudQueryItem, matchCount int) error) QueryHandlerOption {
	return func(options *queryHandlerOptions) {
		options.onPeriodMatchError = onPeriodMatchError
	}
}

type queryHandlerOptions struct {
	periodLists        []QueryPeriodList
	groups             []QueryGroup
	filters            []QueryFilter
	filterKeys         func(keys []ItemKey) bool
	itemKeysHash       func(keys []ItemKey) string
	onPeriodMatchError func(item CloudQueryItem, matchCount int) error
}

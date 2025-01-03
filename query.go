package cloudcostexplorer

import (
	"errors"
	"fmt"
	"iter"

	"github.com/invzhi/timex"
)

type CloudQueryItem struct {
	Date  timex.Date
	Keys  []ItemKey
	Value float64
}

type QueryResult struct {
	Items               []*Item
	TotalValue          float64
	Groups              []QueryResultGroup
	PeriodsSameDuration bool
	Periods             []QueryResultPeriod
	ExtraOutput         QueryExtraOutput
}

// QueryFilter is the ID and value of a filter.
type QueryFilter struct {
	ID    string
	Value string
}

// QueryGroup is the ID and optional value of a group.
type QueryGroup struct {
	ID   string
	Data string // only if group has data
}

// QueryPeriodList is a list of periods to query.
type QueryPeriodList struct {
	Periods []QueryPeriod
}

func (l QueryPeriodList) Range() (bool, timex.Date, timex.Date) {
	var start, end *timex.Date
	for _, period := range l.Periods {
		if start == nil || period.Start.Before(*start) {
			start = Ptr(period.Start)
		}
		if end == nil || period.End.After(*end) {
			end = Ptr(period.End)
		}
	}
	if start == nil || end == nil {
		return false, timex.Date{}, timex.Date{}
	}
	return true, *start, *end
}

// QueryPeriod is a single period to query.
type QueryPeriod struct {
	ID         string // optional ID that can be used by the caller to identify the returned period. Not used by the library.
	Start, End timex.Date
}

// QueryResultGroup is a group that was used to query the results.
type QueryResultGroup struct {
	Parameter
	Data string
}

func (q QueryResultGroup) String() string {
	return q.Title(false)
}

func (q QueryResultGroup) Title(isMenu bool) string {
	ret := q.Parameter.Name
	if isMenu && q.Parameter.MenuTitle != "" {
		ret = q.Parameter.MenuTitle
	}
	if q.Data != "" {
		ret = fmt.Sprintf("%s[%s]", ret, q.Data)
	}
	return ret
}

// QueryResultPeriod is a period that was used to query the results.
type QueryResultPeriod struct {
	QueryPeriod
	TotalValue float64
}

// String returns a short string representation of the period.
func (q QueryPeriod) String() string {
	if q.Start.Equal(q.End) {
		return FormatShortDate(q.Start)
	}
	return fmt.Sprintf("%s-%s", FormatShortDate(q.Start), FormatShortDate(q.End))
}

// StringFilter returns a string representation of the period to be used as a filtering value.
func (q QueryPeriod) StringFilter() string {
	ret := fmt.Sprintf("T%s", q.Start.Format("YYYY-MM-DD"))
	if q.Start.Equal(q.End) {
		return ret
	}
	return fmt.Sprintf("%s%s%s", ret, DataSeparator, q.End.Format("YYYY-MM-DD"))
}

// StringWithDuration may append the duration to the String result.
func (q QueryPeriod) StringWithDuration(showDuration bool) string {
	s := q.String()
	if showDuration {
		s += fmt.Sprintf(" (%d days)", q.End.Sub(q.Start)+1)
	}
	return s
}

func NewQueryPeriod(start, end timex.Date) QueryPeriod {
	return QueryPeriod{
		Start: start,
		End:   end,
	}
}

// QueryExtraData is possible extra data to be shown after the main cost table.
type QueryExtraData interface {
	ExtraDataType() string
}

// QueryExtraOutput returns a list of possible extra output to be shown after the main cost table.
type QueryExtraOutput interface {
	Close()
	ExtraOutputs() iter.Seq2[ValueOutput, error]
}

type QueryOption func(options *QueryOptions)

// ParseQueryOptions parses the default query options.
func ParseQueryOptions(options ...QueryOption) (QueryOptions, error) {
	var optns QueryOptions
	for _, opt := range options {
		opt(&optns)
	}

	if optns.Start.IsZero() || optns.End.IsZero() {
		return QueryOptions{}, errors.New("start and end times are required")
	}
	if len(optns.Groups) == 0 {
		return QueryOptions{}, errors.New("at least one group is required")
	}
	return optns, nil
}

// WithQueryDates sets the date range to query.
func WithQueryDates(start, end timex.Date) QueryOption {
	return func(options *QueryOptions) {
		options.Start = start
		options.End = end
	}
}

// WithQueryGroupByDate sets whether to group by date (day only), ignoring any possible time value.
func WithQueryGroupByDate(groupByDate bool) QueryOption {
	return func(options *QueryOptions) {
		options.GroupByDate = groupByDate
	}
}

// WithQueryGroups sets the grouping to use for the query.
func WithQueryGroups(groups ...QueryGroup) QueryOption {
	return func(options *QueryOptions) {
		options.Groups = groups
	}
}

// WithQueryFilters sets the filters to use for the query.
func WithQueryFilters(filters ...QueryFilter) QueryOption {
	return func(options *QueryOptions) {
		options.Filters = append(options.Filters, filters...)
	}
}

// WithQueryExtraData sets a callback to receive any possible extra data that the query may return.
// A list of these values should be sent to [Cloud.QueryExtraOutput] after the main cost query finishes.
func WithQueryExtraData(extraDataCallback func(data QueryExtraData)) QueryOption {
	return func(options *QueryOptions) {
		options.ExtraDataCallback = extraDataCallback
	}
}

type QueryOptions struct {
	Start, End        timex.Date
	GroupByDate       bool
	Groups            []QueryGroup
	Filters           []QueryFilter
	ExtraDataCallback func(data QueryExtraData)
}

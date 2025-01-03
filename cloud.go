package cloudcostexplorer

import (
	"context"
	"iter"
)

// Cloud is a cloud service abstraction.
type Cloud interface {
	// DaysDelay returns the number of days that the cloud service takes to process cost data. Usually the data
	// is only reliable on the next day.
	DaysDelay() int
	// MaxGroupBy returns the maximum number of groups that the cloud service supports.
	MaxGroupBy() int
	// Parameters returns the list of possible filtering and grouping parameter.
	Parameters() Parameters
	// ParameterTitle returns the string value of a parameter, or the passed value if unknown.
	ParameterTitle(id string, defaultValue string) string
	// Query executes the cost explorer query and returns an iterator for the data.
	Query(ctx context.Context, options ...QueryOption) iter.Seq2[CloudQueryItem, error]
	// QueryExtraOutput may return any extra output to be shown after the query data, like extra filters
	// not available in fields from the main query.
	QueryExtraOutput(ctx context.Context, extraData []QueryExtraData) QueryExtraOutput
}

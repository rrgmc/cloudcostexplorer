package cloudcostexplorer

import (
	"context"
)

// ValueOutput allows outputting custom values for cost table columns.
type ValueOutput interface {
	Output(ctx context.Context, vctx ValueContext, uq *URLQuery) (string, error)
}

type ValueContext interface {
	GroupParamName() string           // the group parameter of the column.
	FilterParamName(id string) string // the URL query field name that should be used for filtering.
	Flush()                           // flushes the HTTP output.
}

// EmptyValue implements ValueOutput always returning "[EMPTY VALUE]".
type EmptyValue struct {
}

func (v EmptyValue) Output(ctx context.Context, vctx ValueContext, uq *URLQuery) (string, error) {
	return "[EMPTY VALUE]", nil
}

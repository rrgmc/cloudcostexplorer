package gcp

import (
	"fmt"

	"cloud.google.com/go/bigquery"
)

// parameterValues caches parameter values from previous runs.
type parameterValues struct {
	values map[string]string
}

// bigQueryStringValue returns the string value of a bigQuery result field.
func bigQueryStringValue(row map[string]bigquery.Value, fieldName string) string {
	v, ok := row[fieldName]
	if !ok {
		panic(fmt.Sprintf("missing field %s", fieldName))
	}

	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}

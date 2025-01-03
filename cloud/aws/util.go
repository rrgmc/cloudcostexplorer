package aws

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// buildCostExplorerFilter creates a cost explorer [types.Expression] from a list of [types.Expression].
func buildCostExplorerFilter(filter []types.Expression) *types.Expression {
	if len(filter) == 0 {
		return nil
	}
	if len(filter) == 1 {
		return &filter[0]
	}
	return &types.Expression{
		And: filter,
	}
}

// awsAPIIteratorInput iterates on AWS APIs which takes a single input struct and returns a single output struct.
func awsAPIIteratorInput[I, O any](ctx context.Context, input *I, nextPage func(ctx context.Context,
	input *I) (*O, error)) iter.Seq2[*O, error] {
	return awsAPIIterator[O](ctx, func(ctx context.Context, nextPageToken *string) (*O, error) {
		err := awsAPISetNextPageToken(input, nextPageToken)
		if err != nil {
			return nil, err
		}
		return nextPage(ctx, input)
	})
}

// awsAPIIterator is a generic iterator for AWS APIs.
func awsAPIIterator[T any](ctx context.Context, nextPage func(ctx context.Context,
	nextPageToken *string) (*T, error)) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		var nextPageToken *string
		for {
			data, err := nextPage(ctx, nextPageToken)
			if err != nil {
				yield(nil, fmt.Errorf("error calling AWS API: %w", err))
				return
			}
			if !yield(data, nil) {
				return
			}
			nextToken, err := awsAPIGetNextPageToken(data)
			if err != nil {
				yield(nil, err)
				return
			}
			if nextToken == "" {
				return
			}
			nextPageToken = &nextToken
		}
	}
}

// awsAPIGetNextPageToken gets the next page token from any AWS output struct using reflection.
func awsAPIGetNextPageToken(out any) (string, error) {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Pointer || v.Type().Elem().Kind() != reflect.Struct {
		return "", fmt.Errorf("expected *struct got %s", v.Kind().String())
	}
	if v.IsNil() {
		return "", errors.New("output cannot be nil")
	}
	v = v.Elem()
	npt := v.FieldByName("NextPageToken")
	if !npt.IsValid() {
		npt = v.FieldByName("NextToken")
	}
	if !npt.IsValid() {
		return "", errors.New("field 'NextPageToken' or 'NextToken' not found")
	}
	if npt.Kind() != reflect.Pointer || npt.Type().Elem().Kind() != reflect.String {
		return "", fmt.Errorf("field 'NextPageToken' not of expected '*string' type")
	}
	if npt.IsNil() {
		return "", nil
	}
	return npt.Elem().String(), nil
}

// awsAPISetNextPageToken sets the next page token on any AWS output struct using reflection.
func awsAPISetNextPageToken(in any, token *string) error {
	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Pointer || v.Type().Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected *struct got %s", v.Kind().String())
	}
	if v.IsNil() {
		return errors.New("input cannot be nil")
	}
	v = v.Elem()
	npt := v.FieldByName("NextPageToken")
	if !npt.IsValid() {
		npt = v.FieldByName("NextToken")
	}
	if !npt.IsValid() {
		return errors.New("field 'NextPageToken' or 'NextToken' not found")
	}
	if npt.Kind() != reflect.Pointer || npt.Type().Elem().Kind() != reflect.String {
		return errors.New("field 'NextPageToken' not of expected '*string' type")
	}
	npt.Set(reflect.ValueOf(token))
	return nil
}

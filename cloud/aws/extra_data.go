package aws

import (
	"context"
	"errors"
	"slices"

	"github.com/rrgmc/cloudcostexplorer"
)

type extraDataUsageTypeGroups struct {
	err  error
	data []extraDataUsageTypeGroup
}

func (e *extraDataUsageTypeGroups) ExtraDataType() string {
	return "USAGE_TYPE_GROUP"
}

func (e *extraDataUsageTypeGroups) merge(other *extraDataUsageTypeGroups) {
	if other.err != nil {
		e.err = errors.Join(e.err, other.err)
	} else {
		for _, od := range other.data {
			if slices.Contains(e.data, od) {
				continue
			}
			e.data = append(e.data, od)
		}
	}
}

type extraDataTags struct {
	err  error
	data map[string]*extraDataTag
}

func (e *extraDataTags) ExtraDataType() string {
	return "TAG"
}

func (e *extraDataTags) merge(other *extraDataTags) {
	if other.err != nil {
		e.err = errors.Join(e.err, other.err)
	} else {
		for tn, tv := range other.data {
			curv, ok := e.data[tn]
			if !ok {
				e.data[tn] = tv
			} else {
				for _, v := range tv.Values {
					if slices.Contains(curv.Values, v) {
						continue
					}
					curv.Values = append(curv.Values, v)
				}
			}
		}
	}
}

type extraDataUsageTypeGroup struct {
	Unit          string
	Value         string
	AttributeDump string
}

type extraDataTag struct {
	Name   string
	Values []string
}

// QueryExtraOutput outputs a list of usage type group values and tags available for the current filter.
func (c *Cloud) QueryExtraOutput(ctx context.Context, extraData []cloudcostexplorer.QueryExtraData) cloudcostexplorer.QueryExtraOutput {
	out := &extraOutput{}

	var edUsageTypeGroups extraDataUsageTypeGroups
	edTags := extraDataTags{
		data: make(map[string]*extraDataTag),
	}

	for _, data := range extraData {
		switch dt := data.(type) {
		case *extraDataUsageTypeGroups:
			edUsageTypeGroups.merge(dt)
		case *extraDataTags:
			edTags.merge(dt)
		}
	}

	if edUsageTypeGroups.err != nil || len(edUsageTypeGroups.data) > 0 {
		out.usageTypeGroups = &extraOutputUsageTypeGroups{data: edUsageTypeGroups}
	}

	if edTags.err != nil || len(edTags.data) > 0 {
		out.tags = &extraOutputTags{data: edTags}
	}

	return out
}

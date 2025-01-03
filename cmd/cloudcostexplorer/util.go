package main

import (
	"cmp"
	"fmt"
	"math"
	"net/http"

	"github.com/rrgmc/cloudcostexplorer"
)

func costDiffPct(cost1, cost2 float64) float64 {
	var costDiff float64
	if cost2 != 0.0 {
		costDiff = (cost1 - cost2) / math.Abs(cost2) * 100.0
	} else {
		costDiff = cost1
	}
	return costDiff
}

func itemCostDiffGet(idx int, item *cloudcostexplorer.Item) (float64, float64) {
	if idx < 1 || idx >= len(item.Values) {
		return 0, 0
	}
	return item.Values[idx] - item.Values[idx-1], costDiffPct(item.Values[idx], item.Values[idx-1])
}

func periodCostDiffGet(idx int, items []cloudcostexplorer.QueryResultPeriod) (float64, float64) {
	if idx < 1 || idx >= len(items) {
		return 0, 0
	}
	return items[idx].TotalValue - items[idx-1].TotalValue, costDiffPct(items[idx].TotalValue, items[idx-1].TotalValue)
}

func itemCostDiff(idx int, item *cloudcostexplorer.Item) float64 {
	d, _ := itemCostDiffGet(idx, item)
	return d
}

func itemCostDiffPct(idx int, item *cloudcostexplorer.Item) float64 {
	_, d := itemCostDiffGet(idx, item)
	return d
}

func compare[T cmp.Ordered](x, y T, reverse bool) int {
	if reverse {
		return cmp.Compare(y, x)
	}
	return cmp.Compare(x, y)
}

type activeFilter struct {
	parameter  cloudcostexplorer.Parameter
	title      string
	paramNames []string
}

type valueContext struct {
	flusher        http.Flusher
	groupParamName string
}

func newValueContext(groupParamName string, flusher any) valueContext {
	ret := valueContext{
		groupParamName: groupParamName,
	}
	if f, ok := flusher.(http.Flusher); ok {
		ret.flusher = f
	}
	return ret
}

func (v valueContext) Flush() {
	if v.flusher != nil {
		v.flusher.Flush()
	}
}

func (v valueContext) GroupParamName() string {
	return v.groupParamName
}

func (v valueContext) FilterParamName(id string) string {
	return fmt.Sprintf("f%s", id)
}

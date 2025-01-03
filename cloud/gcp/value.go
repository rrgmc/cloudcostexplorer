package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/rrgmc/cloudcostexplorer"
)

// LabelValue outputs a list of labels and their values.
type LabelValue struct {
	Labels []Label

	raw       string
	paramName string
}

func NewLabelValue(paramName string, value string) *LabelValue {
	ret := &LabelValue{
		paramName: paramName,
	}

	if err := json.Unmarshal([]byte(value), &ret.Labels); err != nil {
		ret.raw = cloudcostexplorer.IndentJSON(value)
	}
	return ret
}

func (o *LabelValue) Output(ctx context.Context, vctx cloudcostexplorer.ValueContext, uq *cloudcostexplorer.URLQuery) (string, error) {
	if o.raw != "" {
		return fmt.Sprintf(`<pre>%s</pre>`, o.raw), nil
	}

	if len(o.Labels) == 0 {
		return "", nil
	}

	var sb strings.Builder

	labels := slices.SortedFunc(slices.Values(o.Labels), func(label Label, label2 Label) int {
		return strings.Compare(label.Key, label2.Key)
	})
	_, _ = sb.WriteString(`<table class="table table-sm table-bordered"><tbody>`)
	for _, label := range labels {
		_, _ = sb.WriteString(fmt.Sprintf(`<tr><td><strong><a href="%s">%s</a></strong></td><td><a href="%s">%s</a></td></tr>`,
			uq.Clone().Set(vctx.GroupParamName(), fmt.Sprintf("%s%s%s", o.paramName, cloudcostexplorer.DataSeparator, label.Key)),
			label.Key,
			uq.Clone().Set(vctx.FilterParamName(o.paramName), fmt.Sprintf("%s%s%s", label.Key, cloudcostexplorer.DataSeparator, label.Value)),
			label.Value))
	}
	_, _ = sb.WriteString(`</tbody></table>`)

	return sb.String(), nil
}

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
